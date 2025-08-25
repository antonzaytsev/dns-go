package metrics

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"

	"dns-go/internal/types"
)

// Metrics holds aggregated DNS server statistics
type Metrics struct {
	mu                sync.RWMutex
	startTime         time.Time
	totalRequests     int64
	cacheHits         int64
	cacheMisses       int64
	successfulQueries int64
	failedQueries     int64
	rateLimited       int64
	malformedQueries  int64

	// Time-based metrics
	requestsLastHour map[int64]int64 // timestamp -> count
	requestsLastDay  map[int64]int64 // timestamp -> count

	// Client statistics
	clientStats map[string]*ClientStats

	// Query type statistics
	queryTypeStats map[string]int64

	// Upstream statistics
	upstreamStats map[string]*UpstreamStats

	// Response time statistics
	responseTimeSum   float64
	responseTimeCount int64

	// Recent requests for real-time display
	recentRequests []types.LogEntry
	maxRecentSize  int
}

// ClientStats holds statistics for a specific client
type ClientStats struct {
	TotalRequests     int64     `json:"total_requests"`
	LastSeen          time.Time `json:"last_seen"`
	CacheHits         int64     `json:"cache_hits"`
	SuccessfulQueries int64     `json:"successful_queries"`
	FailedQueries     int64     `json:"failed_queries"`
}

// UpstreamStats holds statistics for upstream servers
type UpstreamStats struct {
	TotalQueries      int64     `json:"total_queries"`
	SuccessfulQueries int64     `json:"successful_queries"`
	FailedQueries     int64     `json:"failed_queries"`
	AverageRTT        float64   `json:"average_rtt_ms"`
	LastUsed          time.Time `json:"last_used"`
	RTTSum            float64   `json:"-"`
	RTTCount          int64     `json:"-"`
}

// DashboardMetrics represents the metrics data structure for the web dashboard
type DashboardMetrics struct {
	Overview        OverviewMetrics           `json:"overview"`
	TimeSeriesData  TimeSeriesData            `json:"time_series"`
	TopClients      []ClientMetric            `json:"top_clients"`
	QueryTypes      map[string]int64          `json:"query_types"`
	UpstreamServers map[string]*UpstreamStats `json:"upstream_servers"`
	SystemInfo      SystemInfo                `json:"system_info"`
}

// OverviewMetrics provides high-level statistics
type OverviewMetrics struct {
	Uptime              string  `json:"uptime"`
	TotalRequests       int64   `json:"total_requests"`
	RequestsPerSecond   float64 `json:"requests_per_second"`
	CacheHitRate        float64 `json:"cache_hit_rate"`
	SuccessRate         float64 `json:"success_rate"`
	AverageResponseTime float64 `json:"average_response_time_ms"`
	ActiveClients       int     `json:"active_clients"`
}

// TimeSeriesData holds time-based metrics for charts
type TimeSeriesData struct {
	RequestsLastHour []TimePoint `json:"requests_last_hour"`
	RequestsLastDay  []TimePoint `json:"requests_last_day"`
}

// TimePoint represents a data point in time series
type TimePoint struct {
	Timestamp int64 `json:"timestamp"`
	Value     int64 `json:"value"`
}

// ClientMetric represents client statistics for the dashboard
type ClientMetric struct {
	IP           string    `json:"ip"`
	Requests     int64     `json:"requests"`
	CacheHitRate float64   `json:"cache_hit_rate"`
	SuccessRate  float64   `json:"success_rate"`
	LastSeen     time.Time `json:"last_seen"`
}

// SystemInfo provides system-level information
type SystemInfo struct {
	Version   string `json:"version"`
	StartTime string `json:"start_time"`
}

// NewMetrics creates a new metrics collector
func NewMetrics() *Metrics {
	return &Metrics{
		startTime:        time.Now(),
		requestsLastHour: make(map[int64]int64),
		requestsLastDay:  make(map[int64]int64),
		clientStats:      make(map[string]*ClientStats),
		queryTypeStats:   make(map[string]int64),
		upstreamStats:    make(map[string]*UpstreamStats),
		recentRequests:   make([]types.LogEntry, 0),
		maxRecentSize:    100, // Keep last 100 requests
	}
}

// RecordRequest records a DNS request in the metrics
func (m *Metrics) RecordRequest(entry types.LogEntry) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.totalRequests++

	// Time-based metrics (rounded to minutes for aggregation)
	hourKey := entry.Timestamp.Truncate(time.Minute).Unix()
	dayKey := entry.Timestamp.Truncate(time.Hour).Unix()

	m.requestsLastHour[hourKey]++
	m.requestsLastDay[dayKey]++

	// Clean old data (keep last hour and day)
	m.cleanOldTimeData()

	// Client statistics
	clientIP := types.ExtractIPFromAddr(entry.Request.Client)
	if stats, exists := m.clientStats[clientIP]; exists {
		stats.TotalRequests++
		stats.LastSeen = entry.Timestamp
	} else {
		m.clientStats[clientIP] = &ClientStats{
			TotalRequests: 1,
			LastSeen:      entry.Timestamp,
		}
	}

	// Query type statistics
	m.queryTypeStats[entry.Request.Type]++

	// Status-based metrics
	switch entry.Status {
	case "cache_hit":
		m.cacheHits++
		m.clientStats[clientIP].CacheHits++
	case "success":
		m.cacheMisses++
		m.successfulQueries++
		m.clientStats[clientIP].SuccessfulQueries++

		// Record response time
		m.responseTimeSum += entry.Duration
		m.responseTimeCount++

		// Upstream statistics
		if entry.Response != nil && entry.Response.Upstream != "cache" {
			upstream := entry.Response.Upstream
			if stats, exists := m.upstreamStats[upstream]; exists {
				stats.TotalQueries++
				stats.SuccessfulQueries++
				stats.RTTSum += entry.Response.RTT
				stats.RTTCount++
				stats.AverageRTT = stats.RTTSum / float64(stats.RTTCount)
				stats.LastUsed = entry.Timestamp
			} else {
				m.upstreamStats[upstream] = &UpstreamStats{
					TotalQueries:      1,
					SuccessfulQueries: 1,
					AverageRTT:        entry.Response.RTT,
					LastUsed:          entry.Timestamp,
					RTTSum:            entry.Response.RTT,
					RTTCount:          1,
				}
			}
		}
	case "all_upstreams_failed":
		m.cacheMisses++
		m.failedQueries++
		m.clientStats[clientIP].FailedQueries++

		// Record failed upstream attempts
		for _, upstream := range entry.Upstreams {
			if stats, exists := m.upstreamStats[upstream.Server]; exists {
				stats.TotalQueries++
				stats.FailedQueries++
				stats.LastUsed = entry.Timestamp
			} else {
				m.upstreamStats[upstream.Server] = &UpstreamStats{
					TotalQueries:  1,
					FailedQueries: 1,
					LastUsed:      entry.Timestamp,
				}
			}
		}
	case "malformed_query":
		m.malformedQueries++
	}

	// Add to recent requests
	m.recentRequests = append(m.recentRequests, entry)
	if len(m.recentRequests) > m.maxRecentSize {
		m.recentRequests = m.recentRequests[1:]
	}
}

// RecordRateLimited records a rate-limited request
func (m *Metrics) RecordRateLimited(clientIP string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.rateLimited++

	cleanIP := types.ExtractIPFromAddr(clientIP)

	// Update client stats
	if stats, exists := m.clientStats[cleanIP]; exists {
		stats.TotalRequests++
		stats.LastSeen = time.Now()
	} else {
		m.clientStats[cleanIP] = &ClientStats{
			TotalRequests: 1,
			LastSeen:      time.Now(),
		}
	}
}

// GetDashboardMetrics returns formatted metrics for the web dashboard
func (m *Metrics) GetDashboardMetrics(version string) DashboardMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	uptime := time.Since(m.startTime)

	// Calculate rates
	var cacheHitRate, successRate, avgResponseTime, requestsPerSecond float64

	if m.totalRequests > 0 {
		cacheHitRate = float64(m.cacheHits) / float64(m.totalRequests) * 100
		successRate = float64(m.successfulQueries+m.cacheHits) / float64(m.totalRequests) * 100
		requestsPerSecond = float64(m.totalRequests) / uptime.Seconds()
	}

	if m.responseTimeCount > 0 {
		avgResponseTime = m.responseTimeSum / float64(m.responseTimeCount)
	}

	// Get time series data
	timeSeriesData := m.getTimeSeriesData()

	// Get top clients
	topClients := m.getTopClients()

	// Count active clients (seen in last hour)
	activeClients := 0
	oneHourAgo := time.Now().Add(-time.Hour)
	for _, stats := range m.clientStats {
		if stats.LastSeen.After(oneHourAgo) {
			activeClients++
		}
	}

	return DashboardMetrics{
		Overview: OverviewMetrics{
			Uptime:              formatDuration(uptime),
			TotalRequests:       m.totalRequests,
			RequestsPerSecond:   requestsPerSecond,
			CacheHitRate:        cacheHitRate,
			SuccessRate:         successRate,
			AverageResponseTime: avgResponseTime,
			ActiveClients:       activeClients,
		},
		TimeSeriesData:  timeSeriesData,
		TopClients:      topClients,
		QueryTypes:      m.queryTypeStats,
		UpstreamServers: m.upstreamStats,
		SystemInfo: SystemInfo{
			Version:   version,
			StartTime: m.startTime.Format(time.RFC3339),
		},
	}
}

// LoadFromLogFile loads historical data from log files
func (m *Metrics) LoadFromLogFile(logFilePath string) error {
	file, err := os.Open(logFilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var entry types.LogEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue // Skip invalid JSON lines
		}

		// Only process entries from the last 24 hours
		if time.Since(entry.Timestamp) <= 24*time.Hour {
			m.RecordRequest(entry)
		}
	}

	return scanner.Err()
}

// Helper methods

func (m *Metrics) cleanOldTimeData() {
	now := time.Now()

	// Clean hour data (keep last 2 hours)
	cutoffHour := now.Add(-2 * time.Hour).Truncate(time.Minute).Unix()
	for timestamp := range m.requestsLastHour {
		if timestamp < cutoffHour {
			delete(m.requestsLastHour, timestamp)
		}
	}

	// Clean day data (keep last 2 days)
	cutoffDay := now.Add(-48 * time.Hour).Truncate(time.Hour).Unix()
	for timestamp := range m.requestsLastDay {
		if timestamp < cutoffDay {
			delete(m.requestsLastDay, timestamp)
		}
	}
}

func (m *Metrics) getTimeSeriesData() TimeSeriesData {
	// Convert hour data to sorted slice
	hourPoints := make([]TimePoint, 0, len(m.requestsLastHour))
	for timestamp, count := range m.requestsLastHour {
		hourPoints = append(hourPoints, TimePoint{
			Timestamp: timestamp * 1000, // Convert to milliseconds for JavaScript
			Value:     count,
		})
	}
	sort.Slice(hourPoints, func(i, j int) bool {
		return hourPoints[i].Timestamp < hourPoints[j].Timestamp
	})

	// Convert day data to sorted slice
	dayPoints := make([]TimePoint, 0, len(m.requestsLastDay))
	for timestamp, count := range m.requestsLastDay {
		dayPoints = append(dayPoints, TimePoint{
			Timestamp: timestamp * 1000, // Convert to milliseconds for JavaScript
			Value:     count,
		})
	}
	sort.Slice(dayPoints, func(i, j int) bool {
		return dayPoints[i].Timestamp < dayPoints[j].Timestamp
	})

	return TimeSeriesData{
		RequestsLastHour: hourPoints,
		RequestsLastDay:  dayPoints,
	}
}

func (m *Metrics) getTopClients() []ClientMetric {
	clients := make([]ClientMetric, 0, len(m.clientStats))

	for ip, stats := range m.clientStats {
		var cacheHitRate, successRate float64
		if stats.TotalRequests > 0 {
			cacheHitRate = float64(stats.CacheHits) / float64(stats.TotalRequests) * 100
			successRate = float64(stats.SuccessfulQueries+stats.CacheHits) / float64(stats.TotalRequests) * 100
		}

		clients = append(clients, ClientMetric{
			IP:           ip,
			Requests:     stats.TotalRequests,
			CacheHitRate: cacheHitRate,
			SuccessRate:  successRate,
			LastSeen:     stats.LastSeen,
		})
	}

	// Sort by request count (descending)
	sort.Slice(clients, func(i, j int) bool {
		return clients[i].Requests > clients[j].Requests
	})

	// Return top 10
	if len(clients) > 10 {
		clients = clients[:10]
	}

	return clients
}

func (m *Metrics) getRecentRequests() []types.LogEntry {
	// Return a copy of recent requests (reversed to show newest first)
	recent := make([]types.LogEntry, len(m.recentRequests))
	for i, j := 0, len(m.recentRequests)-1; i <= j; i, j = i+1, j-1 {
		recent[i], recent[j] = m.recentRequests[j], m.recentRequests[i]
	}
	return recent
}

func formatDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	} else if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	} else {
		return fmt.Sprintf("%dm", minutes)
	}
}
