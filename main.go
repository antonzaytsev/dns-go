package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"dns-go/internal/cache"
	"dns-go/internal/config"
	"dns-go/internal/logging"
	"dns-go/internal/types"
	"dns-go/internal/upstream"

	"github.com/miekg/dns"
)

// DNSServer represents our improved DNS proxy server
type DNSServer struct {
	config         *config.Config
	logger         *logging.Logger
	upstreamMgr    *upstream.Manager
	cache          *cache.Cache
	requestLimiter chan struct{}
	wg             sync.WaitGroup
	shutdown       chan struct{}
}

// NewDNSServer creates a new DNS server instance with all improvements
func NewDNSServer(cfg *config.Config, logger *logging.Logger) *DNSServer {
	// Create upstream manager with concurrent query support
	upstreamMgr := upstream.New(cfg.UpstreamDNS, cfg.Timeout, cfg.RetryAttempts)

	// Create DNS cache
	dnsCache := cache.New(cfg.CacheSize, cfg.CacheTTL)

	// Create request limiter channel
	requestLimiter := make(chan struct{}, cfg.MaxConcurrent)

	server := &DNSServer{
		config:         cfg,
		logger:         logger,
		upstreamMgr:    upstreamMgr,
		cache:          dnsCache,
		requestLimiter: requestLimiter,
		shutdown:       make(chan struct{}),
	}

	return server
}

// handleDNSRequest processes incoming DNS queries with caching and concurrent upstream queries
func (s *DNSServer) handleDNSRequest(w dns.ResponseWriter, r *dns.Msg) {
	// Rate limiting
	select {
	case s.requestLimiter <- struct{}{}:
		defer func() { <-s.requestLimiter }()
	default:
		// Too many concurrent requests, return SERVFAIL
		s.logger.Warn("Request rate limited", map[string]interface{}{
			"client": w.RemoteAddr().String(),
		})
		msg := &dns.Msg{}
		msg.SetRcode(r, dns.RcodeServerFailure)
		w.WriteMsg(msg)
		return
	}

	start := time.Now()
	clientAddr := w.RemoteAddr().String()
	requestUUID := types.GenerateRequestUUID()

	// Initialize log entry
	logEntry := types.LogEntry{
		Timestamp: start,
		UUID:      requestUUID,
		Upstreams: make([]types.UpstreamAttempt, 0),
		Status:    "unknown",
	}

	// Handle malformed queries
	if len(r.Question) == 0 {
		logEntry.Request = types.RequestInfo{
			Client: clientAddr,
			Query:  "MALFORMED",
			Type:   "UNKNOWN",
			ID:     r.Id,
		}
		logEntry.Status = "malformed_query"
		logEntry.Duration = types.DurationToMilliseconds(time.Since(start))

		s.logger.LogJSON(logEntry)
		s.logger.LogRequestResponse(requestUUID, clientAddr, "MALFORMED", "UNKNOWN",
			"malformed_query", types.DurationToMilliseconds(time.Since(start)), false, "none")
		msg := &dns.Msg{}
		msg.SetRcode(r, dns.RcodeFormatError)
		w.WriteMsg(msg)
		return
	}

	// Set request information
	question := r.Question[0]
	logEntry.Request = types.RequestInfo{
		Client: clientAddr,
		Query:  question.Name,
		Type:   dns.TypeToString[question.Qtype],
		ID:     r.Id,
	}

	// Check cache first
	if cachedResp, found := s.cache.Get(question); found {
		logEntry.CacheHit = true
		logEntry.Status = "cache_hit"
		logEntry.Duration = types.DurationToMilliseconds(time.Since(start))

		// Update response ID to match request
		cachedResp.Id = r.Id

		// Set response info for cache hit
		logEntry.Response = &types.ResponseInfo{
			Upstream:    "cache",
			Rcode:       dns.RcodeToString[cachedResp.Rcode],
			AnswerCount: len(cachedResp.Answer),
			RTT:         0, // Cache hit, no network RTT
		}

		logEntry.Answers = types.ExtractAnswers(cachedResp.Answer)
		logEntry.IPAddresses = types.ExtractIPAddresses(cachedResp.Answer)

		s.logger.LogJSON(logEntry)
		s.logger.LogRequestResponse(requestUUID, clientAddr, question.Name,
			dns.TypeToString[question.Qtype], "cache_hit",
			types.DurationToMilliseconds(time.Since(start)), true, "cache")
		w.WriteMsg(cachedResp)
		return
	}

	// Query upstream servers concurrently
	ctx, cancel := context.WithTimeout(context.Background(), s.config.Timeout)
	defer cancel()

	result, allResults := s.upstreamMgr.QueryConcurrent(ctx, r)

	// Convert upstream results to log format
	for i, upstreamResult := range allResults {
		attempt := types.UpstreamAttempt{
			Server:   upstreamResult.Server,
			Attempt:  i + 1,
			Duration: types.DurationToMilliseconds(upstreamResult.RTT),
		}

		if upstreamResult.Error != nil {
			errStr := upstreamResult.Error.Error()
			attempt.Error = &errStr
		} else {
			rttMs := types.DurationToMilliseconds(upstreamResult.RTT)
			attempt.RTT = &rttMs
		}

		logEntry.Upstreams = append(logEntry.Upstreams, attempt)
	}

	if result.Error == nil && result.Response != nil {
		// Successful response
		logEntry.Response = &types.ResponseInfo{
			Upstream:    result.Server,
			Rcode:       dns.RcodeToString[result.Response.Rcode],
			AnswerCount: len(result.Response.Answer),
			RTT:         types.DurationToMilliseconds(result.RTT),
		}

		logEntry.Answers = types.ExtractAnswers(result.Response.Answer)
		logEntry.IPAddresses = types.ExtractIPAddresses(result.Response.Answer)
		logEntry.Status = "success"
		logEntry.Duration = types.DurationToMilliseconds(time.Since(start))

		// Cache the response
		s.cache.Set(question, result.Response)

		s.logger.LogJSON(logEntry)
		s.logger.LogRequestResponse(requestUUID, clientAddr, question.Name,
			dns.TypeToString[question.Qtype], "success",
			types.DurationToMilliseconds(time.Since(start)), false, result.Server)

		// Forward the response back to the client
		if err := w.WriteMsg(result.Response); err != nil {
			s.logger.Error("Failed to write response", map[string]interface{}{
				"uuid":   requestUUID,
				"client": clientAddr,
				"error":  err.Error(),
			})
		}
		return
	}

	// All upstreams failed
	logEntry.Status = "all_upstreams_failed"
	logEntry.Duration = types.DurationToMilliseconds(time.Since(start))
	s.logger.LogJSON(logEntry)
	s.logger.LogRequestResponse(requestUUID, clientAddr, question.Name,
		dns.TypeToString[question.Qtype], "all_upstreams_failed",
		types.DurationToMilliseconds(time.Since(start)), false, "none")

	msg := &dns.Msg{}
	msg.SetRcode(r, dns.RcodeServerFailure)
	if err := w.WriteMsg(msg); err != nil {
		s.logger.Error("Failed to write SERVFAIL", map[string]interface{}{
			"uuid":   requestUUID,
			"client": clientAddr,
			"error":  err.Error(),
		})
	}
}

// Start begins the DNS server with all improvements
func (s *DNSServer) Start() error {
	// Start background services
	s.upstreamMgr.StartHealthChecks(s.config.HealthCheckInterval)
	s.cache.StartCleanupTimer(5 * time.Minute)

	// Setup DNS handler
	dns.HandleFunc(".", s.handleDNSRequest)

	// Setup UDP server
	server := &dns.Server{
		Addr: net.JoinHostPort(s.config.ListenAddress, s.config.Port),
		Net:  "udp",
	}

	s.logger.Info("Starting DNS server", map[string]interface{}{
		"address":   s.config.ListenAddress,
		"port":      s.config.Port,
		"upstreams": strings.Join(s.config.UpstreamDNS, ", "),
	})

	// Start server in goroutine
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if err := server.ListenAndServe(); err != nil {
			s.logger.Error("DNS server error", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}()

	// Wait for shutdown signal
	<-s.shutdown

	s.logger.Info("Shutting down DNS server", nil)

	// Stop background services
	s.upstreamMgr.StopHealthChecks()

	// Shutdown server
	if err := server.Shutdown(); err != nil {
		s.logger.Error("Error shutting down server", map[string]interface{}{
			"error": err.Error(),
		})
	}

	s.wg.Wait()
	return nil
}

// Shutdown gracefully stops the DNS server
func (s *DNSServer) Shutdown() {
	close(s.shutdown)
}

// GetStats returns server statistics
func (s *DNSServer) GetStats() map[string]interface{} {
	cacheSize, maxCacheSize := s.cache.Stats()
	upstreamStats := s.upstreamMgr.GetStats()

	return map[string]interface{}{
		"cache": map[string]interface{}{
			"size":     cacheSize,
			"max_size": maxCacheSize,
		},
		"upstreams": upstreamStats,
	}
}

func main() {
	// Load configuration
	cfg, err := config.LoadFromFlags()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	// Setup logging
	logger, jsonFile, humanFile, err := logging.NewFromConfig(cfg.LogFile, cfg.LogLevel)
	if err != nil {
		log.Fatalf("Failed to setup logging: %v", err)
	}
	if jsonFile != nil {
		defer func() {
			logger.Info("Closing log files", nil)
			jsonFile.Close()
		}()
	}
	if humanFile != nil {
		defer humanFile.Close()
	}

	// Create and configure DNS server
	server := NewDNSServer(cfg, logger)

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("Received shutdown signal", nil)
		server.Shutdown()
	}()

	logger.Info("DNS Proxy Server starting", map[string]interface{}{
		"config": map[string]interface{}{
			"listen":         cfg.ListenAddress + ":" + cfg.Port,
			"upstreams":      cfg.UpstreamDNS,
			"log_file":       cfg.LogFile,
			"log_level":      cfg.LogLevel,
			"cache_size":     cfg.CacheSize,
			"cache_ttl":      cfg.CacheTTL.String(),
			"max_concurrent": cfg.MaxConcurrent,
			"timeout":        cfg.Timeout.String(),
		},
	})

	// Start server
	if err := server.Start(); err != nil {
		logger.Error("Failed to start DNS server", map[string]interface{}{
			"error": err.Error(),
		})
		os.Exit(1)
	}
}
