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
	successfulQueries int64
	failedQueries     int64
	rateLimited       int64
	malformedQueries  int64

	// Time-based metrics
	requestsLastHour  map[int64]int64 // timestamp -> count (per minute)
	requestsLastDay   map[int64]int64 // timestamp -> count (per hour)
	requestsLastWeek  map[int64]int64 // timestamp -> count (per day)
	requestsLastMonth map[int64]int64 // timestamp -> count (per day)

	// Client statistics
	clientStats map[string]*ClientStats

	// Query type statistics
	queryTypeStats map[string]int64

	// Upstream statistics
	upstreamStats map[string]*UpstreamStats

	// Response time statistics
	responseTimeSum   float64
	responseTimeCount int64

	// Requests for real-time display
	requests      []types.LogEntry
	maxRecentSize int
}

// ClientStats holds statistics for a specific client
type ClientStats struct {
	TotalRequests     int64     `json:"total_requests"`
	LastSeen          time.Time `json:"last_seen"`
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

// QueryTypeMetric represents a query type with its count (sorted by backend)
type QueryTypeMetric struct {
	Type  string `json:"type"`
	Count int64  `json:"count"`
}

// DashboardMetrics represents the metrics data structure for the web dashboard
type DashboardMetrics struct {
	Overview        OverviewMetrics           `json:"overview"`
	TimeSeriesData  TimeSeriesData            `json:"time_series"`
	TopClients      []ClientMetric            `json:"top_clients"`
	QueryTypes      []QueryTypeMetric         `json:"query_types"` // Pre-sorted, top 8 query types
	UpstreamServers map[string]*UpstreamStats `json:"upstream_servers"`
	Requests        []types.LogEntry          `json:"requests"` // Requests for real-time display
	SystemInfo      SystemInfo                `json:"system_info"`
}

// OverviewMetrics provides high-level statistics
type OverviewMetrics struct {
	Uptime              string  `json:"uptime"`
	TotalRequests       int64   `json:"total_requests"`
	RequestsPerSecond   float64 `json:"requests_per_second"`
	SuccessRate         float64 `json:"success_rate"`
	AverageResponseTime float64 `json:"average_response_time_ms"`
	Clients             int     `json:"clients"`
}

type TimeSeriesData struct {
	RequestsLastHour  []TimePoint `json:"requests_last_hour"`
	RequestsLastDay   []TimePoint `json:"requests_last_day"`
	RequestsLastWeek  []TimePoint `json:"requests_last_week"`
	RequestsLastMonth []TimePoint `json:"requests_last_month"`
}

// TimePoint represents a data point in time series
type TimePoint struct {
	Timestamp int64 `json:"timestamp"`
	Value     int64 `json:"value"`
}

// ClientMetric represents client statistics for the dashboard
type ClientMetric struct {
	IP          string    `json:"ip"`
	Requests    int64     `json:"requests"`
	SuccessRate float64   `json:"success_rate"`
	LastSeen    time.Time `json:"last_seen"`
}

// SystemInfo provides system-level information
type SystemInfo struct {
	Version   string `json:"version"`
	StartTime string `json:"start_time"`
}

// NewMetrics creates a new metrics collector
func NewMetrics() *Metrics {
	return &Metrics{
		startTime:         time.Now(),
		requestsLastHour:  make(map[int64]int64),
		requestsLastDay:   make(map[int64]int64),
		requestsLastWeek:  make(map[int64]int64),
		requestsLastMonth: make(map[int64]int64),
		clientStats:       make(map[string]*ClientStats),
		queryTypeStats:    make(map[string]int64),
		upstreamStats:     make(map[string]*UpstreamStats),
		requests:          make([]types.LogEntry, 0),
		maxRecentSize:     100, // Keep last 100 requests
	}
}

// RecordRequest records a DNS request in the metrics
func (m *Metrics) RecordRequest(entry types.LogEntry) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.totalRequests++

	// Time-based metrics with different granularities
	minuteKey := entry.Timestamp.Truncate(time.Minute).Unix() // Per minute for last hour
	hourKey := entry.Timestamp.Truncate(time.Hour).Unix()     // Per hour for last day
	dayKey := entry.Timestamp.Truncate(24 * time.Hour).Unix() // Per day for last week/month

	m.requestsLastHour[minuteKey]++
	m.requestsLastDay[hourKey]++
	m.requestsLastWeek[dayKey]++
	m.requestsLastMonth[dayKey]++

	// Clean old data
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
	case "success":
		m.successfulQueries++
		m.clientStats[clientIP].SuccessfulQueries++

		// Record response time
		m.responseTimeSum += entry.Duration
		m.responseTimeCount++

		// Upstream statistics
		if entry.Response != nil {
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

	// Add to requests
	m.requests = append(m.requests, entry)
	if len(m.requests) > m.maxRecentSize {
		m.requests = m.requests[1:]
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
	var successRate, avgResponseTime, requestsPerSecond float64

	if m.totalRequests > 0 {
		successRate = float64(m.successfulQueries) / float64(m.totalRequests) * 100
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
			SuccessRate:         successRate,
			AverageResponseTime: avgResponseTime,
			Clients:             activeClients,
		},
		TimeSeriesData:  timeSeriesData,
		TopClients:      topClients,
		QueryTypes:      m.getTopQueryTypes(),
		UpstreamServers: m.upstreamStats,
		Requests:        m.getRequests(),
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

	// Clean minute data (keep last 80 minutes - allows for 75+ bars)
	cutoffMinute := now.Add(-80 * time.Minute).Truncate(time.Minute).Unix()
	for timestamp := range m.requestsLastHour {
		if timestamp < cutoffMinute {
			delete(m.requestsLastHour, timestamp)
		}
	}

	// Clean hour data (keep last 80 hours - allows for 75+ bars)
	cutoffHour := now.Add(-80 * time.Hour).Truncate(time.Hour).Unix()
	for timestamp := range m.requestsLastDay {
		if timestamp < cutoffHour {
			delete(m.requestsLastDay, timestamp)
		}
	}

	// Clean day data (keep last 80 days - allows for 75+ bars)
	cutoffWeek := now.Add(-80 * 24 * time.Hour).Truncate(24 * time.Hour).Unix()
	for timestamp := range m.requestsLastWeek {
		if timestamp < cutoffWeek {
			delete(m.requestsLastWeek, timestamp)
		}
	}

	// Clean month data (keep last 540 days - allows for 75+ weeks of bars when aggregated by week)
	cutoffMonth := now.Add(-540 * 24 * time.Hour).Truncate(24 * time.Hour).Unix()
	for timestamp := range m.requestsLastMonth {
		if timestamp < cutoffMonth {
			delete(m.requestsLastMonth, timestamp)
		}
	}
}

func (m *Metrics) getTimeSeriesData() TimeSeriesData {
	now := time.Now()

	minutePoints := m.generateTimeSlots(m.requestsLastHour, now, time.Minute, 75)
	hourPoints := m.generateTimeSlots(m.requestsLastDay, now, time.Hour, 75)
	dayPoints := m.generateTimeSlots(m.requestsLastWeek, now, 24*time.Hour, 75)
	weekPoints := m.generateWeeklyTimeSlots(m.requestsLastMonth, now, 75)

	return TimeSeriesData{
		RequestsLastHour:  minutePoints,
		RequestsLastDay:   hourPoints,
		RequestsLastWeek:  dayPoints,
		RequestsLastMonth: weekPoints,
	}
}

// generateTimeSlots creates exactly `count` time slots going backwards from now
func (m *Metrics) generateTimeSlots(data map[int64]int64, now time.Time, duration time.Duration, count int) []TimePoint {
	slots := make([]TimePoint, count)

	for i := 0; i < count; i++ {
		slotTime := now.Add(-time.Duration(count-1-i) * duration)

		// Truncate to the appropriate unit
		var truncatedTime time.Time
		switch duration {
		case time.Minute:
			truncatedTime = slotTime.Truncate(time.Minute)
		case time.Hour:
			truncatedTime = slotTime.Truncate(time.Hour)
		case 24 * time.Hour:
			truncatedTime = slotTime.Truncate(24 * time.Hour)
		default:
			truncatedTime = slotTime.Truncate(duration)
		}

		timestamp := truncatedTime.Unix()
		value := data[timestamp]

		slots[i] = TimePoint{
			Timestamp: timestamp * 1000, // Convert to milliseconds for JavaScript
			Value:     value,
		}
	}

	return slots
}

// generateWeeklyTimeSlots aggregates daily data into weekly buckets and returns exactly `count` week slots
func (m *Metrics) generateWeeklyTimeSlots(dailyData map[int64]int64, now time.Time, count int) []TimePoint {
	slots := make([]TimePoint, count)

	for i := 0; i < count; i++ {
		// Calculate the start of the week for this slot
		weeksAgo := count - 1 - i
		targetWeek := now.Add(-time.Duration(weeksAgo) * 7 * 24 * time.Hour)

		// Get the Monday of that week (week starts on Monday)
		weekStart := getWeekStart(targetWeek)

		// Aggregate daily data for this week
		var weekTotal int64
		for dayOffset := 0; dayOffset < 7; dayOffset++ {
			dayTime := weekStart.Add(time.Duration(dayOffset) * 24 * time.Hour)
			dayTimestamp := dayTime.Truncate(24 * time.Hour).Unix()
			weekTotal += dailyData[dayTimestamp]
		}

		slots[i] = TimePoint{
			Timestamp: weekStart.Unix() * 1000, // Convert to milliseconds for JavaScript
			Value:     weekTotal,
		}
	}

	return slots
}

// getWeekStart returns the Monday (start of week) for the given date
func getWeekStart(t time.Time) time.Time {
	// Calculate days since Monday (Monday = 1, Sunday = 0)
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday becomes 7
	}

	// Go back to Monday
	daysSinceMonday := weekday - 1
	weekStart := t.Add(-time.Duration(daysSinceMonday) * 24 * time.Hour)

	// Truncate to start of day
	return time.Date(weekStart.Year(), weekStart.Month(), weekStart.Day(), 0, 0, 0, 0, weekStart.Location())
}

func (m *Metrics) getTopClients() []ClientMetric {
	clients := make([]ClientMetric, 0, len(m.clientStats))

	for ip, stats := range m.clientStats {
		var successRate float64
		if stats.TotalRequests > 0 {
			successRate = float64(stats.SuccessfulQueries) / float64(stats.TotalRequests) * 100
		}

		clients = append(clients, ClientMetric{
			IP:          ip,
			Requests:    stats.TotalRequests,
			SuccessRate: successRate,
			LastSeen:    stats.LastSeen,
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

func (m *Metrics) GetAllClients() []ClientMetric {
	m.mu.RLock()
	defer m.mu.RUnlock()

	clients := make([]ClientMetric, 0, len(m.clientStats))

	for ip, stats := range m.clientStats {
		var successRate float64
		if stats.TotalRequests > 0 {
			successRate = float64(stats.SuccessfulQueries) / float64(stats.TotalRequests) * 100
		}

		clients = append(clients, ClientMetric{
			IP:          ip,
			Requests:    stats.TotalRequests,
			SuccessRate: successRate,
			LastSeen:    stats.LastSeen,
		})
	}

	// Sort by request count (descending)
	sort.Slice(clients, func(i, j int) bool {
		return clients[i].Requests > clients[j].Requests
	})

	return clients
}

func (m *Metrics) getTopQueryTypes() []QueryTypeMetric {
	queryTypes := make([]QueryTypeMetric, 0, len(m.queryTypeStats))

	for qtype, count := range m.queryTypeStats {
		queryTypes = append(queryTypes, QueryTypeMetric{
			Type:  qtype,
			Count: count,
		})
	}

	// Sort by count (descending)
	sort.Slice(queryTypes, func(i, j int) bool {
		return queryTypes[i].Count > queryTypes[j].Count
	})

	// Return top 8 query types
	if len(queryTypes) > 8 {
		queryTypes = queryTypes[:8]
	}

	return queryTypes
}

func (m *Metrics) getRequests() []types.LogEntry {
	// Return a copy of requests (reversed to show newest first)
	recent := make([]types.LogEntry, len(m.requests))
	for i, j := 0, len(m.requests)-1; i <= j; i, j = i+1, j-1 {
		recent[i], recent[j] = m.requests[j], m.requests[i]
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
