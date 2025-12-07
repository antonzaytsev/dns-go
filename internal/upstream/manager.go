package upstream

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/miekg/dns"
)

// ServerState represents the health state of an upstream server
type ServerState int

const (
	StateHealthy ServerState = iota
	StateUnhealthy
	StateRecovering
)

// Protocol represents the DNS protocol type
type Protocol int

const (
	ProtocolDNS Protocol = iota // Standard DNS (UDP/TCP)
	ProtocolDoT                 // DNS over TLS
	ProtocolDoH                 // DNS over HTTPS
)

// Server represents an upstream DNS server with health tracking
type Server struct {
	Address      string
	Protocol     Protocol
	DoHURL       string // For DoH servers, the full URL
	State        int64  // atomic ServerState
	FailureCount int64  // atomic
	LastCheck    int64  // atomic time.Unix()
	LastSuccess  int64  // atomic time.Unix()
	ResponseTime int64  // atomic time in nanoseconds
}

// Manager handles multiple upstream DNS servers with health checking
type Manager struct {
	servers    []*Server
	client     *dns.Client
	dotClient  *dns.Client // DNS over TLS client
	httpClient *http.Client
	timeout    time.Duration
	maxRetries int

	// Circuit breaker settings
	failureThreshold  int
	recoveryTimeout   time.Duration
	healthCheckTicker *time.Ticker

	mu sync.RWMutex
}

// QueryResult represents the result of a DNS query attempt
type QueryResult struct {
	Response *dns.Msg
	RTT      time.Duration
	Server   string
	Error    error
}

// parseUpstreamAddress parses an upstream address and determines the protocol
func parseUpstreamAddress(addr string) (protocol Protocol, address string, dohURL string, err error) {
	addr = strings.TrimSpace(addr)

	// Check for DoH URL (https://)
	if strings.HasPrefix(addr, "https://") {
		protocol = ProtocolDoH
		parsedURL, err := url.Parse(addr)
		if err != nil {
			return ProtocolDNS, "", "", fmt.Errorf("invalid DoH URL: %w", err)
		}
		// Ensure path ends with /dns-query if not specified
		if parsedURL.Path == "" || parsedURL.Path == "/" {
			parsedURL.Path = "/dns-query"
		}
		dohURL = parsedURL.String()
		address = parsedURL.Host
		return protocol, address, dohURL, nil
	}

	// Check for DoT (tls:// or dot://)
	if strings.HasPrefix(addr, "tls://") || strings.HasPrefix(addr, "dot://") {
		protocol = ProtocolDoT
		address = strings.TrimPrefix(strings.TrimPrefix(addr, "tls://"), "dot://")
		// Ensure port is specified (default to 853 for DoT)
		if !strings.Contains(address, ":") {
			address = net.JoinHostPort(address, "853")
		}
		return protocol, address, "", nil
	}

	// Check for explicit doh:// prefix
	if strings.HasPrefix(addr, "doh://") {
		protocol = ProtocolDoH
		// Convert doh:// to https://
		httpsURL := strings.Replace(addr, "doh://", "https://", 1)
		parsedURL, err := url.Parse(httpsURL)
		if err != nil {
			return ProtocolDNS, "", "", fmt.Errorf("invalid DoH URL: %w", err)
		}
		if parsedURL.Path == "" || parsedURL.Path == "/" {
			parsedURL.Path = "/dns-query"
		}
		dohURL = parsedURL.String()
		address = parsedURL.Host
		return protocol, address, dohURL, nil
	}

	// Default to standard DNS
	protocol = ProtocolDNS
	address = addr
	// Ensure port is specified (default to 53 for DNS)
	if !strings.Contains(address, ":") {
		address = net.JoinHostPort(address, "53")
	}
	return protocol, address, "", nil
}

// New creates a new upstream manager
func New(addresses []string, timeout time.Duration, maxRetries int) *Manager {
	servers := make([]*Server, 0, len(addresses))
	for _, addr := range addresses {
		protocol, address, dohURL, err := parseUpstreamAddress(addr)
		if err != nil {
			// Log error but continue with other servers
			continue
		}

		server := &Server{
			Address:     address,
			Protocol:    protocol,
			DoHURL:      dohURL,
			State:       int64(StateHealthy),
			LastCheck:   time.Now().Unix(),
			LastSuccess: time.Now().Unix(),
		}
		servers = append(servers, server)
	}

	// Create DNS client for standard DNS
	dnsClient := &dns.Client{Timeout: timeout}

	// Create DoT client with TLS config
	dotClient := &dns.Client{
		Net:     "tcp-tls",
		Timeout: timeout,
		TLSConfig: &tls.Config{
			ServerName:         "",
			InsecureSkipVerify: false,
		},
	}

	// Create HTTP client for DoH
	httpClient := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: false,
			},
		},
	}

	return &Manager{
		servers:          servers,
		client:           dnsClient,
		dotClient:        dotClient,
		httpClient:       httpClient,
		timeout:          timeout,
		maxRetries:       maxRetries,
		failureThreshold: 3,
		recoveryTimeout:  30 * time.Second,
	}
}

// GetHealthyServers returns a list of currently healthy servers
func (m *Manager) GetHealthyServers() []*Server {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var healthy []*Server
	for _, server := range m.servers {
		state := ServerState(atomic.LoadInt64(&server.State))
		if state == StateHealthy || state == StateRecovering {
			healthy = append(healthy, server)
		}
	}
	return healthy
}

// QueryConcurrent performs concurrent queries to multiple upstream servers
func (m *Manager) QueryConcurrent(ctx context.Context, msg *dns.Msg) (*QueryResult, []QueryResult) {
	healthyServers := m.GetHealthyServers()
	if len(healthyServers) == 0 {
		// Fallback to all servers if none are healthy
		healthyServers = m.servers
	}

	resultChan := make(chan QueryResult, len(healthyServers))
	var wg sync.WaitGroup

	// Start concurrent queries
	for _, server := range healthyServers {
		wg.Add(1)
		go func(srv *Server) {
			defer wg.Done()
			result := m.querySingle(ctx, srv, msg)
			select {
			case resultChan <- result:
			case <-ctx.Done():
			}
		}(server)
	}

	// Close channel when all queries complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	var firstSuccess *QueryResult
	var allResults []QueryResult

	// Process results as they arrive, return immediately on first success
	for result := range resultChan {
		allResults = append(allResults, result)

		if result.Error == nil && firstSuccess == nil {
			firstSuccess = &result
			// Return immediately on first success to avoid waiting for slower upstreams
			// Remaining results will continue to be collected in background for logging
			break
		}
	}

	if firstSuccess != nil {
		return firstSuccess, allResults
	}

	// If no successful response, return the first result (which will be an error)
	if len(allResults) > 0 {
		return &allResults[0], allResults
	}

	return &QueryResult{
		Error: fmt.Errorf("no upstream servers available"),
	}, allResults
}

// querySingle performs a single DNS query to an upstream server
func (m *Manager) querySingle(ctx context.Context, server *Server, msg *dns.Msg) QueryResult {
	start := time.Now()
	var resp *dns.Msg
	var rtt time.Duration
	var err error

	switch server.Protocol {
	case ProtocolDoH:
		resp, rtt, err = m.queryDoH(ctx, server, msg)
	case ProtocolDoT:
		resp, rtt, err = m.queryDoT(ctx, server, msg)
	case ProtocolDNS:
		fallthrough
	default:
		resp, rtt, err = m.client.ExchangeContext(ctx, msg, server.Address)
	}

	duration := time.Since(start)
	if rtt == 0 {
		rtt = duration
	}

	displayAddr := server.Address
	if server.Protocol == ProtocolDoH && server.DoHURL != "" {
		displayAddr = server.DoHURL
	}

	result := QueryResult{
		Response: resp,
		RTT:      rtt,
		Server:   displayAddr,
		Error:    err,
	}

	// Update server statistics
	if err != nil {
		m.recordFailure(server)
	} else {
		m.recordSuccess(server, duration)
	}

	return result
}

// queryDoT performs a DNS over TLS query
func (m *Manager) queryDoT(ctx context.Context, server *Server, msg *dns.Msg) (*dns.Msg, time.Duration, error) {
	// Extract hostname for TLS SNI
	host, _, err := net.SplitHostPort(server.Address)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid DoT address: %w", err)
	}

	// Create a DoT client with proper SNI configuration
	dotClient := &dns.Client{
		Net:     "tcp-tls",
		Timeout: m.timeout,
		TLSConfig: &tls.Config{
			ServerName:         host,
			InsecureSkipVerify: false,
		},
	}

	return dotClient.ExchangeContext(ctx, msg, server.Address)
}

// queryDoH performs a DNS over HTTPS query (tries POST first, then GET as fallback)
func (m *Manager) queryDoH(ctx context.Context, server *Server, msg *dns.Msg) (*dns.Msg, time.Duration, error) {
	if server.DoHURL == "" {
		return nil, 0, fmt.Errorf("DoH URL not configured")
	}

	// Pack DNS message
	packed, err := msg.Pack()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to pack DNS message: %w", err)
	}

	// Try POST first (RFC 8484 standard)
	resp, rtt, err := m.queryDoHPost(ctx, server.DoHURL, packed)
	if err == nil {
		return resp, rtt, nil
	}

	// Fallback to GET if POST fails
	return m.queryDoHGet(ctx, server.DoHURL, packed)
}

// queryDoHPost performs a DNS over HTTPS query using POST method
func (m *Manager) queryDoHPost(ctx context.Context, dohURL string, packed []byte) (*dns.Msg, time.Duration, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", dohURL, bytes.NewReader(packed))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/dns-message")
	req.Header.Set("Accept", "application/dns-message")

	start := time.Now()
	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, time.Since(start), fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		rtt := time.Since(start)
		return nil, rtt, fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		rtt := time.Since(start)
		return nil, rtt, fmt.Errorf("failed to read response body: %w", err)
	}

	dnsResp := new(dns.Msg)
	if err := dnsResp.Unpack(body); err != nil {
		rtt := time.Since(start)
		return nil, rtt, fmt.Errorf("failed to unpack DNS response: %w", err)
	}

	rtt := time.Since(start)
	return dnsResp, rtt, nil
}

// queryDoHGet performs a DNS over HTTPS query using GET method (RFC 8484)
func (m *Manager) queryDoHGet(ctx context.Context, dohURL string, packed []byte) (*dns.Msg, time.Duration, error) {
	// Base64url encode the DNS message
	encoded := base64.RawURLEncoding.EncodeToString(packed)

	// Parse URL and add dns parameter
	parsedURL, err := url.Parse(dohURL)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid DoH URL: %w", err)
	}

	// Add dns parameter for GET request
	params := parsedURL.Query()
	params.Set("dns", encoded)
	parsedURL.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", parsedURL.String(), nil)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Accept", "application/dns-message")

	start := time.Now()
	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, time.Since(start), fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		rtt := time.Since(start)
		return nil, rtt, fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		rtt := time.Since(start)
		return nil, rtt, fmt.Errorf("failed to read response body: %w", err)
	}

	dnsResp := new(dns.Msg)
	if err := dnsResp.Unpack(body); err != nil {
		rtt := time.Since(start)
		return nil, rtt, fmt.Errorf("failed to unpack DNS response: %w", err)
	}

	rtt := time.Since(start)
	return dnsResp, rtt, nil
}

// recordSuccess updates server state after a successful query
func (m *Manager) recordSuccess(server *Server, rtt time.Duration) {
	atomic.StoreInt64(&server.LastSuccess, time.Now().Unix())
	atomic.StoreInt64(&server.ResponseTime, int64(rtt))
	atomic.StoreInt64(&server.FailureCount, 0)

	// Restore to healthy state if recovering
	currentState := ServerState(atomic.LoadInt64(&server.State))
	if currentState == StateRecovering {
		atomic.StoreInt64(&server.State, int64(StateHealthy))
	}
}

// recordFailure updates server state after a failed query
func (m *Manager) recordFailure(server *Server) {
	failures := atomic.AddInt64(&server.FailureCount, 1)

	if failures >= int64(m.failureThreshold) {
		atomic.StoreInt64(&server.State, int64(StateUnhealthy))
	}
}

// StartHealthChecks begins periodic health checking of upstream servers
func (m *Manager) StartHealthChecks(interval time.Duration) {
	m.healthCheckTicker = time.NewTicker(interval)
	go m.healthCheckLoop()
}

// StopHealthChecks stops the health checking routine
func (m *Manager) StopHealthChecks() {
	if m.healthCheckTicker != nil {
		m.healthCheckTicker.Stop()
	}
}

// healthCheckLoop runs periodic health checks on upstream servers
func (m *Manager) healthCheckLoop() {
	for range m.healthCheckTicker.C {
		for _, server := range m.servers {
			go m.healthCheck(server)
		}
	}
}

// healthCheck performs a health check on a single server
func (m *Manager) healthCheck(server *Server) {
	currentState := ServerState(atomic.LoadInt64(&server.State))

	// Skip health check for healthy servers
	if currentState == StateHealthy {
		return
	}

	// Create a simple DNS query for health check
	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn("health.check"), dns.TypeA)

	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()

	result := m.querySingle(ctx, server, msg)
	atomic.StoreInt64(&server.LastCheck, time.Now().Unix())

	if result.Error == nil {
		// Server is responding, move to recovering state
		if currentState == StateUnhealthy {
			atomic.StoreInt64(&server.State, int64(StateRecovering))
		}
	}
}

// GetStats returns statistics for all upstream servers
func (m *Manager) GetStats() []ServerStats {
	stats := make([]ServerStats, len(m.servers))
	for i, server := range m.servers {
		stats[i] = ServerStats{
			Address:      server.Address,
			State:        ServerState(atomic.LoadInt64(&server.State)),
			FailureCount: atomic.LoadInt64(&server.FailureCount),
			LastCheck:    time.Unix(atomic.LoadInt64(&server.LastCheck), 0),
			LastSuccess:  time.Unix(atomic.LoadInt64(&server.LastSuccess), 0),
			ResponseTime: time.Duration(atomic.LoadInt64(&server.ResponseTime)),
		}
	}
	return stats
}

// ServerStats represents statistics for an upstream server
type ServerStats struct {
	Address      string
	State        ServerState
	FailureCount int64
	LastCheck    time.Time
	LastSuccess  time.Time
	ResponseTime time.Duration
}

// String returns a string representation of ServerState
func (s ServerState) String() string {
	switch s {
	case StateHealthy:
		return "healthy"
	case StateUnhealthy:
		return "unhealthy"
	case StateRecovering:
		return "recovering"
	default:
		return "unknown"
	}
}
