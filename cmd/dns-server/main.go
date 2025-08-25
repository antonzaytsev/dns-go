// Package main provides the DNS proxy server application.
package main

import (
	"context"
	"flag"
	"fmt"
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
	"dns-go/internal/resolver"
	"dns-go/internal/types"
	"dns-go/internal/upstream"
	"dns-go/pkg/version"

	"github.com/miekg/dns"
)

// DNSServer represents our improved DNS proxy server
type DNSServer struct {
	config         *config.Config
	logger         *logging.Logger
	resolver       *resolver.LocalResolver
	upstreamMgr    *upstream.Manager
	cache          *cache.Cache
	requestLimiter chan struct{}
	wg             sync.WaitGroup
	shutdown       chan struct{}
	server         *dns.Server
}

// NewDNSServer creates a new DNS server instance with all improvements
func NewDNSServer(cfg *config.Config, logger *logging.Logger) *DNSServer {
	// Create local resolver for custom DNS mappings
	localResolver := resolver.New(cfg.CustomDNS)

	// Create upstream manager with concurrent query support
	upstreamMgr := upstream.New(cfg.UpstreamDNS, cfg.Timeout, cfg.RetryAttempts)

	// Create DNS cache
	dnsCache := cache.New(cfg.CacheSize, cfg.CacheTTL)

	// Create request limiter channel
	requestLimiter := make(chan struct{}, cfg.MaxConcurrent)

	server := &DNSServer{
		config:         cfg,
		logger:         logger,
		resolver:       localResolver,
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
			"client": types.ExtractIPFromAddr(w.RemoteAddr().String()),
		})
		msg := &dns.Msg{}
		msg.SetRcode(r, dns.RcodeServerFailure)
		w.WriteMsg(msg)
		return
	}

	start := time.Now()
	clientAddr := types.ExtractIPFromAddr(w.RemoteAddr().String())
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

		s.logger.LogDNSEntry(logEntry)
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

	// Check custom resolver first
	if customResp := s.resolver.Resolve(question); customResp != nil {
		logEntry.Status = "custom_resolution"
		logEntry.Duration = types.DurationToMilliseconds(time.Since(start))

		// Update response ID to match request
		customResp.Id = r.Id

		// Set response info for custom resolution
		logEntry.Response = &types.ResponseInfo{
			Upstream:    "custom",
			Rcode:       dns.RcodeToString[customResp.Rcode],
			AnswerCount: len(customResp.Answer),
			RTT:         0, // Custom resolution, no network RTT
		}

		logEntry.Answers = types.ExtractAnswers(customResp.Answer)
		logEntry.IPAddresses = types.ExtractIPAddresses(customResp.Answer)

		s.logger.LogDNSEntry(logEntry)
		s.logger.LogRequestResponse(requestUUID, clientAddr, question.Name,
			dns.TypeToString[question.Qtype], "custom_resolution",
			types.DurationToMilliseconds(time.Since(start)), false, "custom")
		w.WriteMsg(customResp)
		return
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

		s.logger.LogDNSEntry(logEntry)
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

		s.logger.LogDNSEntry(logEntry)
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
func (s *DNSServer) Start(ctx context.Context) error {
	// Start background services
	s.upstreamMgr.StartHealthChecks(s.config.HealthCheckInterval)
	s.cache.StartCleanupTimer(5 * time.Minute)

	// Setup DNS handler
	dns.HandleFunc(".", s.handleDNSRequest)

	// Setup UDP server
	s.server = &dns.Server{
		Addr: net.JoinHostPort(s.config.ListenAddress, s.config.Port),
		Net:  "udp",
	}

	s.logger.Info("Starting DNS server", map[string]interface{}{
		"address":   s.config.ListenAddress,
		"port":      s.config.Port,
		"upstreams": strings.Join(s.config.UpstreamDNS, ", "),
		"version":   version.Get().Short(),
	})

	// Start server in goroutine
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if err := s.server.ListenAndServe(); err != nil {
			s.logger.Error("DNS server error", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}()

	// Wait for context cancellation or shutdown signal
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.shutdown:
		return nil
	}
}

// Shutdown gracefully stops the DNS server
func (s *DNSServer) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down DNS server", nil)

	// Stop background services
	s.upstreamMgr.StopHealthChecks()

	// Shutdown server with timeout
	if s.server != nil {
		if err := s.server.ShutdownContext(ctx); err != nil {
			s.logger.Error("Error shutting down server", map[string]interface{}{
				"error": err.Error(),
			})
			return err
		}
	}

	// Signal shutdown to other goroutines
	close(s.shutdown)

	// Wait for all goroutines to finish with timeout
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// GetStats returns server statistics
func (s *DNSServer) GetStats() map[string]interface{} {
	cacheSize, maxCacheSize := s.cache.Stats()
	upstreamStats := s.upstreamMgr.GetStats()

	return map[string]interface{}{
		"version": version.Get().Short(),
		"cache": map[string]interface{}{
			"size":     cacheSize,
			"max_size": maxCacheSize,
		},
		"upstreams": upstreamStats,
	}
}

// run is the main application logic
func run() error {
	// Parse command line flags
	var (
		showVersion = flag.Bool("version", false, "Show version information and exit")
		showHelp    = flag.Bool("help", false, "Show help information and exit")
	)

	// Load configuration (this will parse the remaining flags)
	cfg, err := config.LoadFromFlags()
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	// Handle version flag
	if *showVersion {
		fmt.Println(version.Get().String())
		return nil
	}

	// Handle help flag
	if *showHelp {
		fmt.Printf("DNS Proxy Server - %s\n\n", version.Get().Short())
		fmt.Println("A high-performance DNS proxy server with caching and health monitoring.")
		fmt.Println("\nUsage:")
		flag.PrintDefaults()
		return nil
	}

	// Setup logging
	logger, jsonFile, humanFile, err := logging.NewFromConfig(cfg.LogFile, cfg.LogLevel)
	if err != nil {
		return fmt.Errorf("failed to setup logging: %w", err)
	}

	// Ensure log files are closed on exit
	defer func() {
		if jsonFile != nil {
			logger.Info("Closing log files", nil)
			jsonFile.Close()
		}
		if humanFile != nil {
			humanFile.Close()
		}
	}()

	// Create and configure DNS server
	server := NewDNSServer(cfg, logger)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Handle shutdown signals
	go func() {
		sig := <-sigChan
		logger.Info("Received shutdown signal", map[string]interface{}{
			"signal": sig.String(),
		})
		cancel()

		// Force shutdown after 30 seconds
		time.AfterFunc(30*time.Second, func() {
			logger.Error("Force shutdown after timeout", nil)
			os.Exit(1)
		})
	}()

	// Log startup information
	versionInfo := version.Get()
	startupConfig := map[string]interface{}{
		"listen":         cfg.ListenAddress + ":" + cfg.Port,
		"upstreams":      cfg.UpstreamDNS,
		"log_file":       cfg.LogFile,
		"log_level":      cfg.LogLevel,
		"cache_size":     cfg.CacheSize,
		"cache_ttl":      cfg.CacheTTL.String(),
		"max_concurrent": cfg.MaxConcurrent,
		"timeout":        cfg.Timeout.String(),
	}

	// Add custom DNS mappings if present
	if len(cfg.CustomDNS) > 0 {
		startupConfig["custom_dns_mappings"] = cfg.CustomDNS
	}

	logger.Info("DNS Proxy Server starting", map[string]interface{}{
		"version": versionInfo.String(),
		"config":  startupConfig,
	})

	// Start server
	if err := server.Start(ctx); err != nil && err != context.Canceled {
		return fmt.Errorf("server failed: %w", err)
	}

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown failed: %w", err)
	}

	logger.Info("DNS server shutdown complete", nil)
	return nil
}

func main() {
	if err := run(); err != nil {
		log.Fatalf("Application failed: %v", err)
		os.Exit(1)
	}
}
