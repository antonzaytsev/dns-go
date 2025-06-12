package upstream

import (
	"context"
	"fmt"
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

// Server represents an upstream DNS server with health tracking
type Server struct {
	Address      string
	State        int64 // atomic ServerState
	FailureCount int64 // atomic
	LastCheck    int64 // atomic time.Unix()
	LastSuccess  int64 // atomic time.Unix()
	ResponseTime int64 // atomic time in nanoseconds
}

// Manager handles multiple upstream DNS servers with health checking
type Manager struct {
	servers    []*Server
	client     *dns.Client
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

// New creates a new upstream manager
func New(addresses []string, timeout time.Duration, maxRetries int) *Manager {
	servers := make([]*Server, len(addresses))
	for i, addr := range addresses {
		servers[i] = &Server{
			Address:     addr,
			State:       int64(StateHealthy),
			LastCheck:   time.Now().Unix(),
			LastSuccess: time.Now().Unix(),
		}
	}

	return &Manager{
		servers:          servers,
		client:           &dns.Client{Timeout: timeout},
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

	// Wait for first successful response or all failures
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	var firstSuccess *QueryResult
	var allResults []QueryResult

	for result := range resultChan {
		allResults = append(allResults, result)

		if result.Error == nil && firstSuccess == nil {
			firstSuccess = &result
			// Don't return immediately, collect all results for logging
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

	resp, rtt, err := m.client.ExchangeContext(ctx, msg, server.Address)
	duration := time.Since(start)

	result := QueryResult{
		Response: resp,
		RTT:      rtt,
		Server:   server.Address,
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
