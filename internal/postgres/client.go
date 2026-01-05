package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"dns-go/internal/migrations"
	"dns-go/internal/types"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	DefaultHost     = "localhost"
	DefaultPort     = "5432"
	DefaultDatabase = "dns_logs"
	DefaultUser     = "postgres"
	DefaultPassword = "postgres"
)

// Client wraps the PostgreSQL client with DNS-specific functionality
type Client struct {
	db *gorm.DB
}

// Config holds PostgreSQL configuration
type Config struct {
	Host     string
	Port     string
	Database string
	User     string
	Password string
}

// NewClient creates a new PostgreSQL client using GORM
func NewClient(cfg Config) (*Client, error) {
	host := getEnvOrDefault("POSTGRES_HOST", cfg.Host)
	if host == "" {
		host = DefaultHost
	}

	port := getEnvOrDefault("POSTGRES_PORT", cfg.Port)
	if port == "" {
		port = DefaultPort
	}

	database := getEnvOrDefault("POSTGRES_DB", cfg.Database)
	if database == "" {
		database = DefaultDatabase
	}

	user := getEnvOrDefault("POSTGRES_USER", cfg.User)
	if user == "" {
		user = DefaultUser
	}

	password := getEnvOrDefault("POSTGRES_PASSWORD", cfg.Password)
	if password == "" {
		password = DefaultPassword
	}

	// Build DSN for GORM
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
		host, user, password, database, port)

	// Try to connect to the target database
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		// If connection fails, try to create the database
		defaultDSN := fmt.Sprintf("host=%s user=%s password=%s dbname=postgres port=%s sslmode=disable TimeZone=UTC",
			host, user, password, port)

		defaultDB, err := sql.Open("postgres", defaultDSN)
		if err != nil {
			return nil, fmt.Errorf("failed to open connection to default database: %w", err)
		}
		defer defaultDB.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Test connection to default database
		if err := defaultDB.PingContext(ctx); err != nil {
			return nil, fmt.Errorf("failed to connect to PostgreSQL server: %w", err)
		}

		// Check if database exists before creating
		var dbExists int
		checkDBSQL := "SELECT 1 FROM pg_database WHERE datname = $1"
		err = defaultDB.QueryRowContext(ctx, checkDBSQL, database).Scan(&dbExists)
		dbExistsCheck := err == nil

		if !dbExistsCheck {
			// Create the database if it doesn't exist
			quotedDB := fmt.Sprintf(`"%s"`, strings.ReplaceAll(database, `"`, `""`))
			createDBSQL := fmt.Sprintf("CREATE DATABASE %s", quotedDB)
			_, err = defaultDB.ExecContext(ctx, createDBSQL)
			if err != nil {
				if !isDatabaseExistsError(err) {
					return nil, fmt.Errorf("failed to create database %s: %w", database, err)
				}
			}
		}

		// Now connect to the newly created database
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err != nil {
			return nil, fmt.Errorf("failed to open database connection after creation: %w", err)
		}
	}

	client := &Client{
		db: db,
	}

	// Run migrations using GORM AutoMigrate
	if err := client.initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return client, nil
}

// isDatabaseExistsError checks if the error indicates the database already exists
func isDatabaseExistsError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "already exists") ||
		strings.Contains(errStr, "duplicate_database") ||
		strings.Contains(errStr, "42p04")
}

// initialize runs database migrations and ensures models are up to date
func (c *Client) initialize() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Check and run file-based migrations first
	migrator := migrations.NewMigrator(c.db)
	if err := migrator.Run(ctx); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Then run GORM AutoMigrate to ensure models match current code
	// This handles any schema changes that might be needed
	if err := c.db.WithContext(ctx).AutoMigrate(&DNSLog{}, &DNSMapping{}); err != nil {
		// If AutoMigrate fails on constraint issues with existing tables, continue
		if strings.Contains(err.Error(), "constraint") || strings.Contains(err.Error(), "does not exist") {
			fmt.Printf("⚠️  Warning: AutoMigrate encountered constraint issues (tables exist, continuing): %v\n", err)
			// Verify tables exist
			if !c.db.Migrator().HasTable(&DNSLog{}) || !c.db.Migrator().HasTable(&DNSMapping{}) {
				return fmt.Errorf("required tables missing after migration: %w", err)
			}
		} else {
			return fmt.Errorf("failed to run auto migrate: %w", err)
		}
	}

	return nil
}

// toDNSLog converts types.LogEntry to DNSLog model
func toDNSLog(entry types.LogEntry) *DNSLog {
	clientIP := types.ExtractIPFromAddr(entry.Request.Client)

	// Convert upstreams to JSONB (as array)
	upstreamsJSON := make(JSONB, len(entry.Upstreams))
	for i, upstream := range entry.Upstreams {
		upstreamData := map[string]interface{}{
			"server":   upstream.Server,
			"attempt":  upstream.Attempt,
			"duration": upstream.Duration,
		}
		if upstream.Error != nil {
			upstreamData["error"] = *upstream.Error
		}
		if upstream.RTT != nil {
			upstreamData["rtt_ms"] = *upstream.RTT
		}
		upstreamsJSON[i] = upstreamData
	}

	// Convert answers to JSONB (as array)
	answersJSON := make(JSONB, len(entry.Answers))
	for i, answer := range entry.Answers {
		answersJSON[i] = answer
	}

	queryID := int(entry.Request.ID)
	durationMs := entry.Duration

	log := &DNSLog{
		UUID:        entry.UUID,
		Timestamp:   entry.Timestamp,
		ClientIP:    clientIP,
		Query:       entry.Request.Query,
		QueryType:   entry.Request.Type,
		QueryID:     &queryID,
		Status:      entry.Status,
		DurationMs:  &durationMs,
		Upstreams:   upstreamsJSON,
		Answers:     answersJSON,
		IPAddresses: StringArray(entry.IPAddresses),
	}

	if entry.Response != nil {
		log.ResponseUpstream = &entry.Response.Upstream
		log.ResponseRcode = &entry.Response.Rcode
		answerCount := entry.Response.AnswerCount
		log.ResponseAnswerCount = &answerCount
		rtt := entry.Response.RTT
		log.ResponseRTTMs = &rtt
	}

	return log
}

// toLogEntry converts DNSLog model to types.LogEntry
func toLogEntry(log *DNSLog) types.LogEntry {
	entry := types.LogEntry{
		Timestamp: log.Timestamp,
		UUID:      log.UUID,
		Request: types.RequestInfo{
			Client: log.ClientIP,
			Query:  log.Query,
			Type:   log.QueryType,
		},
		Status: log.Status,
	}

	if log.QueryID != nil {
		entry.Request.ID = uint16(*log.QueryID)
	}

	if log.DurationMs != nil {
		entry.Duration = *log.DurationMs
	}

	// Convert JSONB upstreams back to []UpstreamAttempt
	if log.Upstreams != nil && len(log.Upstreams) > 0 {
		upstreams := make([]types.UpstreamAttempt, 0, len(log.Upstreams))
		for _, val := range log.Upstreams {
			if data, ok := val.(map[string]interface{}); ok {
				attempt := types.UpstreamAttempt{
					Server:   getString(data, "server"),
					Attempt:  getInt(data, "attempt"),
					Duration: getFloat64(data, "duration"),
				}
				if errStr := getStringPtr(data, "error"); errStr != nil {
					attempt.Error = errStr
				}
				if rtt := getFloat64Ptr(data, "rtt_ms"); rtt != nil {
					attempt.RTT = rtt
				}
				upstreams = append(upstreams, attempt)
			}
		}
		entry.Upstreams = upstreams
	}

	// Convert JSONB answers back to [][]string
	if log.Answers != nil && len(log.Answers) > 0 {
		answers := make([][]string, 0, len(log.Answers))
		for _, val := range log.Answers {
			if arr, ok := val.([]interface{}); ok {
				strArr := make([]string, 0, len(arr))
				for _, v := range arr {
					if str, ok := v.(string); ok {
						strArr = append(strArr, str)
					}
				}
				answers = append(answers, strArr)
			} else if arr, ok := val.([]string); ok {
				answers = append(answers, arr)
			}
		}
		entry.Answers = answers
	}

	// Convert StringArray to []string
	if log.IPAddresses != nil {
		entry.IPAddresses = []string(log.IPAddresses)
	}

	// Set Response if available
	if log.ResponseUpstream != nil {
		entry.Response = &types.ResponseInfo{
			Upstream: *log.ResponseUpstream,
		}
		if log.ResponseRcode != nil {
			entry.Response.Rcode = *log.ResponseRcode
		}
		if log.ResponseAnswerCount != nil {
			entry.Response.AnswerCount = *log.ResponseAnswerCount
		}
		if log.ResponseRTTMs != nil {
			entry.Response.RTT = *log.ResponseRTTMs
		}
	}

	return entry
}

// Helper functions for type conversion
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getStringPtr(m map[string]interface{}, key string) *string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return &str
		}
	}
	return nil
}

func getInt(m map[string]interface{}, key string) int {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case int:
			return v
		case float64:
			return int(v)
		case int64:
			return int(v)
		}
	}
	return 0
}

func getFloat64(m map[string]interface{}, key string) float64 {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case float64:
			return v
		case int:
			return float64(v)
		case int64:
			return float64(v)
		}
	}
	return 0
}

func getFloat64Ptr(m map[string]interface{}, key string) *float64 {
	if val, ok := m[key]; ok {
		var f float64
		switch v := val.(type) {
		case float64:
			f = v
		case int:
			f = float64(v)
		case int64:
			f = float64(v)
		default:
			return nil
		}
		return &f
	}
	return nil
}

// InsertLogEntry inserts a DNS log entry into PostgreSQL
func (c *Client) InsertLogEntry(entry types.LogEntry) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log := toDNSLog(entry)

	// Use GORM's FirstOrCreate to handle ON CONFLICT (do nothing if exists)
	result := c.db.WithContext(ctx).Where("uuid = ?", log.UUID).FirstOrCreate(log)
	if result.Error != nil {
		return fmt.Errorf("failed to insert log entry: %w", result.Error)
	}

	return nil
}

// SearchLogs searches through DNS logs stored in PostgreSQL
type SearchResult struct {
	Results []types.LogEntry `json:"results"`
	Total   int64            `json:"total"`
}

// SearchLogs searches DNS logs with pagination and optional filters
func (c *Client) SearchLogs(domain, clientIP string, limit, offset int, since *time.Time) (*SearchResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := c.db.WithContext(ctx).Model(&DNSLog{})

	// Add domain filter if specified
	if domain != "" {
		domainPattern := "%" + domain + "%"
		query = query.Where("query ILIKE ?", domainPattern)
	}

	// Add client IP filter if specified
	if clientIP != "" {
		clientPattern := "%" + clientIP + "%"
		query = query.Where("client_ip::text ILIKE ?", clientPattern)
	}

	// Add time filter if specified
	if since != nil {
		query = query.Where("timestamp >= ? AND timestamp <= ?", *since, time.Now())
	}

	// Count total results
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count results: %w", err)
	}

	// Fetch paginated results
	var logs []DNSLog
	if err := query.Order("timestamp DESC").Limit(limit).Offset(offset).Find(&logs).Error; err != nil {
		return nil, fmt.Errorf("failed to query logs: %w", err)
	}

	// Convert to LogEntry
	results := make([]types.LogEntry, len(logs))
	for i, log := range logs {
		results[i] = toLogEntry(&log)
	}

	return &SearchResult{
		Results: results,
		Total:   total,
	}, nil
}

// GetLogCount returns the total number of log entries in PostgreSQL
func (c *Client) GetLogCount() (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var count int64
	if err := c.db.WithContext(ctx).Model(&DNSLog{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count logs: %w", err)
	}

	return count, nil
}

// DeleteOldLogs deletes DNS logs older than the specified retention period
func (c *Client) DeleteOldLogs(retentionDays int) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)

	result := c.db.WithContext(ctx).
		Where("timestamp < ?", cutoffTime).
		Delete(&DNSLog{})

	if result.Error != nil {
		return 0, fmt.Errorf("failed to delete old logs: %w", result.Error)
	}

	return result.RowsAffected, nil
}

// DomainCount represents a domain with its request count
type DomainCount struct {
	Domain string `json:"domain"`
	Count  int64  `json:"count"`
}

// GetDomainCounts returns aggregated domain counts filtered by time range, domain name, and client IP
func (c *Client) GetDomainCounts(since *time.Time, domainFilter, clientIP string) ([]DomainCount, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get raw database connection for direct sql.Scan
	sqlDB, err := c.db.WithContext(ctx).DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	// Build SQL query with filters
	var queryBuilder strings.Builder
	queryBuilder.WriteString(`
		SELECT 
			query as domain,
			COUNT(*)::BIGINT as count
		FROM dns_logs
		WHERE 1=1
	`)

	var args []interface{}
	argIndex := 1

	// Add time filter if specified
	if since != nil {
		queryBuilder.WriteString(fmt.Sprintf(" AND timestamp >= $%d AND timestamp <= $%d", argIndex, argIndex+1))
		args = append(args, *since, time.Now())
		argIndex += 2
	}

	// Add domain name filter if specified
	if domainFilter != "" {
		queryBuilder.WriteString(fmt.Sprintf(" AND query ILIKE $%d", argIndex))
		args = append(args, "%"+domainFilter+"%")
		argIndex++
	}

	// Add client IP filter if specified
	if clientIP != "" {
		queryBuilder.WriteString(fmt.Sprintf(" AND client_ip::text ILIKE $%d", argIndex))
		args = append(args, "%"+clientIP+"%")
		argIndex++
	}

	queryBuilder.WriteString(" GROUP BY query ORDER BY count DESC")

	// Execute raw query
	rows, err := sqlDB.QueryContext(ctx, queryBuilder.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query domain counts: %w", err)
	}
	defer rows.Close()

	var results []DomainCount
	for rows.Next() {
		var dc DomainCount
		if err := rows.Scan(&dc.Domain, &dc.Count); err != nil {
			return nil, fmt.Errorf("failed to scan domain count: %w", err)
		}
		results = append(results, dc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating domain counts: %w", err)
	}

	return results, nil
}

// TimeSeriesPoint represents a time series data point
// Field names must match SQL column aliases for GORM Raw().Scan() to map correctly
type TimeSeriesPoint struct {
	Ts    int64 `gorm:"column:ts" json:"ts"`
	Count int64 `gorm:"column:count" json:"count"`
}

// GetTimeSeriesData returns aggregated time series data from PostgreSQL
func (c *Client) GetTimeSeriesData() (map[string][]TimeSeriesPoint, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result := make(map[string][]TimeSeriesPoint)

	// Get raw database connection for direct sql.Scan
	sqlDB, err := c.db.WithContext(ctx).DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	// Last hour - per minute (75 minutes)
	var hourData []TimeSeriesPoint
	rows, err := sqlDB.QueryContext(ctx, `
		SELECT 
			EXTRACT(EPOCH FROM DATE_TRUNC('minute', timestamp))::BIGINT as ts,
			COUNT(*)::BIGINT as count
		FROM dns_logs
		WHERE timestamp >= NOW() - INTERVAL '75 minutes'
		GROUP BY DATE_TRUNC('minute', timestamp)
		ORDER BY ts ASC
		LIMIT 75
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query hourly data: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var point TimeSeriesPoint
		if err := rows.Scan(&point.Ts, &point.Count); err != nil {
			return nil, fmt.Errorf("failed to scan hourly data: %w", err)
		}
		hourData = append(hourData, point)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating hourly data: %w", err)
	}

	result["requests_last_hour"] = fillTimeSeriesSlots(hourData, time.Minute, 75)

	// Last day - per hour (75 hours)
	var dayData []TimeSeriesPoint
	rows, err = sqlDB.QueryContext(ctx, `
		SELECT 
			EXTRACT(EPOCH FROM DATE_TRUNC('hour', timestamp))::BIGINT as ts,
			COUNT(*)::BIGINT as count
		FROM dns_logs
		WHERE timestamp >= NOW() - INTERVAL '75 hours'
		GROUP BY DATE_TRUNC('hour', timestamp)
		ORDER BY ts ASC
		LIMIT 75
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query daily data: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var point TimeSeriesPoint
		if err := rows.Scan(&point.Ts, &point.Count); err != nil {
			return nil, fmt.Errorf("failed to scan daily data: %w", err)
		}
		dayData = append(dayData, point)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating daily data: %w", err)
	}

	result["requests_last_day"] = fillTimeSeriesSlots(dayData, time.Hour, 75)

	// Last week - per day (75 days)
	var weekData []TimeSeriesPoint
	rows, err = sqlDB.QueryContext(ctx, `
		SELECT 
			EXTRACT(EPOCH FROM DATE_TRUNC('day', timestamp))::BIGINT as ts,
			COUNT(*)::BIGINT as count
		FROM dns_logs
		WHERE timestamp >= NOW() - INTERVAL '75 days'
		GROUP BY DATE_TRUNC('day', timestamp)
		ORDER BY ts ASC
		LIMIT 75
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query weekly data: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var point TimeSeriesPoint
		if err := rows.Scan(&point.Ts, &point.Count); err != nil {
			return nil, fmt.Errorf("failed to scan weekly data: %w", err)
		}
		weekData = append(weekData, point)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating weekly data: %w", err)
	}

	result["requests_last_week"] = fillTimeSeriesSlots(weekData, 24*time.Hour, 75)

	// Last month - per day (75 days, aggregated into weeks on frontend)
	var monthData []TimeSeriesPoint
	rows, err = sqlDB.QueryContext(ctx, `
		SELECT 
			EXTRACT(EPOCH FROM DATE_TRUNC('day', timestamp))::BIGINT as ts,
			COUNT(*)::BIGINT as count
		FROM dns_logs
		WHERE timestamp >= NOW() - INTERVAL '525 days'
		GROUP BY DATE_TRUNC('day', timestamp)
		ORDER BY ts ASC
		LIMIT 525
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query monthly data: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var point TimeSeriesPoint
		if err := rows.Scan(&point.Ts, &point.Count); err != nil {
			return nil, fmt.Errorf("failed to scan monthly data: %w", err)
		}
		monthData = append(monthData, point)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating monthly data: %w", err)
	}

	// For monthly/weekly view, we return daily data that will be aggregated into weeks by the frontend
	result["requests_last_month"] = monthData

	return result, nil
}

// fillTimeSeriesSlots fills in missing time slots with zero values to ensure exactly count slots
func fillTimeSeriesSlots(data []TimeSeriesPoint, duration time.Duration, count int) []TimeSeriesPoint {
	if len(data) == 0 {
		// If no data at all, return all zeros
		return generateEmptyTimeSeries(duration, count)
	}

	now := time.Now()
	dataMap := make(map[int64]int64)
	for _, point := range data {
		dataMap[point.Ts] = point.Count
	}

	result := make([]TimeSeriesPoint, count)
	for i := 0; i < count; i++ {
		slotTime := now.Add(-time.Duration(count-1-i) * duration)
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
		value := int64(0)
		if v, exists := dataMap[timestamp]; exists {
			value = v
		}

		result[i] = TimeSeriesPoint{
			Ts:    timestamp,
			Count: value,
		}
	}

	return result
}

// generateEmptyTimeSeries generates a time series with all zero values
func generateEmptyTimeSeries(duration time.Duration, count int) []TimeSeriesPoint {
	now := time.Now()
	result := make([]TimeSeriesPoint, count)
	for i := 0; i < count; i++ {
		slotTime := now.Add(-time.Duration(count-1-i) * duration)
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

		result[i] = TimeSeriesPoint{
			Ts:    truncatedTime.Unix(),
			Count: 0,
		}
	}
	return result
}

// ClientMetric represents aggregated client statistics
type ClientMetric struct {
	IP          string
	Requests    int64
	SuccessRate float64
	LastSeen    time.Time
}

// GetTopClients returns top clients aggregated from PostgreSQL
func (c *Client) GetTopClients(limit int) ([]ClientMetric, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	type ClientAggregate struct {
		ClientIP      string    `gorm:"column:client_ip"`
		TotalRequests int64     `gorm:"column:total_requests"`
		Successful    int64     `gorm:"column:successful"`
		LastSeen      time.Time `gorm:"column:last_seen"`
	}

	var aggregates []ClientAggregate
	if err := c.db.WithContext(ctx).Raw(`
		SELECT 
			client_ip,
			COUNT(*)::BIGINT as total_requests,
			COUNT(*) FILTER (WHERE status = 'success')::BIGINT as successful,
			MAX(timestamp) as last_seen
		FROM dns_logs
		GROUP BY client_ip
		ORDER BY total_requests DESC
		LIMIT ?
	`, limit).Scan(&aggregates).Error; err != nil {
		return nil, fmt.Errorf("failed to query top clients: %w", err)
	}

	clients := make([]ClientMetric, len(aggregates))
	for i, agg := range aggregates {
		clients[i] = ClientMetric{
			IP:       agg.ClientIP,
			Requests: agg.TotalRequests,
			LastSeen: agg.LastSeen,
		}
		if agg.TotalRequests > 0 {
			clients[i].SuccessRate = float64(agg.Successful) / float64(agg.TotalRequests) * 100
		}
	}

	return clients, nil
}

// QueryTypeMetric represents aggregated query type statistics
type QueryTypeMetric struct {
	Type  string
	Count int64
}

// GetTopQueryTypes returns top query types aggregated from PostgreSQL
func (c *Client) GetTopQueryTypes(limit int) ([]QueryTypeMetric, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	type QueryTypeAggregate struct {
		QueryType string `gorm:"column:query_type"`
		Count     int64  `gorm:"column:count"`
	}

	var aggregates []QueryTypeAggregate
	if err := c.db.WithContext(ctx).Raw(`
		SELECT 
			query_type,
			COUNT(*)::BIGINT as count
		FROM dns_logs
		GROUP BY query_type
		ORDER BY count DESC
		LIMIT ?
	`, limit).Scan(&aggregates).Error; err != nil {
		return nil, fmt.Errorf("failed to query query types: %w", err)
	}

	queryTypes := make([]QueryTypeMetric, len(aggregates))
	for i, agg := range aggregates {
		queryTypes[i] = QueryTypeMetric{
			Type:  agg.QueryType,
			Count: agg.Count,
		}
	}

	return queryTypes, nil
}

// OverviewStats represents overview statistics
type OverviewStats struct {
	TotalRequests       int64
	SuccessfulQueries   int64
	AverageResponseTime float64
	ActiveClients       int
}

// GetOverviewStats returns overview statistics from PostgreSQL
func (c *Client) GetOverviewStats() (*OverviewStats, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stats := &OverviewStats{}

	type StatsAggregate struct {
		TotalRequests   int64           `gorm:"column:total_requests"`
		Successful      int64           `gorm:"column:successful"`
		AvgResponseTime sql.NullFloat64 `gorm:"column:avg_response_time"`
	}

	var agg StatsAggregate
	if err := c.db.WithContext(ctx).Raw(`
		SELECT 
			COUNT(*)::BIGINT as total_requests,
			COUNT(*) FILTER (WHERE status = 'success')::BIGINT as successful,
			AVG(duration_ms) as avg_response_time
		FROM dns_logs
	`).Scan(&agg).Error; err != nil {
		return nil, fmt.Errorf("failed to query overview stats: %w", err)
	}

	stats.TotalRequests = agg.TotalRequests
	stats.SuccessfulQueries = agg.Successful
	if agg.AvgResponseTime.Valid {
		stats.AverageResponseTime = agg.AvgResponseTime.Float64
	}

	// Get active clients (seen in last hour)
	var activeClients int
	if err := c.db.WithContext(ctx).Raw(`
		SELECT COUNT(DISTINCT client_ip)::INTEGER
		FROM dns_logs
		WHERE timestamp >= NOW() - INTERVAL '1 hour'
	`).Scan(&activeClients).Error; err != nil {
		// If this fails, we can still return the other stats
		activeClients = 0
	}
	stats.ActiveClients = activeClients

	return stats, nil
}

// GetRecentRequests returns recent requests for display
func (c *Client) GetRecentRequests(limit int) ([]types.LogEntry, error) {
	result, err := c.SearchLogs("", "", limit, 0, nil)
	if err != nil {
		return nil, err
	}
	return result.Results, nil
}

// HealthCheck checks if PostgreSQL is healthy
func (c *Client) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sqlDB, err := c.db.WithContext(ctx).DB()
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}

	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	return nil
}

// Close closes the PostgreSQL connection
func (c *Client) Close() error {
	if c.db != nil {
		sqlDB, err := c.db.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}

// getEnvOrDefault returns environment variable value or default if not set
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetAllDNSMappings returns all DNS mappings from the database
func (c *Client) GetAllDNSMappings() (map[string]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var mappings []DNSMapping
	if err := c.db.WithContext(ctx).Order("domain").Find(&mappings).Error; err != nil {
		return nil, fmt.Errorf("failed to query DNS mappings: %w", err)
	}

	result := make(map[string]string, len(mappings))
	for _, m := range mappings {
		result[m.Domain] = m.IPAddress
	}

	return result, nil
}

// CreateDNSMapping creates a new DNS mapping
func (c *Client) CreateDNSMapping(domain, ipAddress string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Use raw SQL for proper ON CONFLICT handling
	result := c.db.WithContext(ctx).Exec(`
		INSERT INTO dns_mappings (domain, ip_address, updated_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT (domain) DO UPDATE
		SET ip_address = EXCLUDED.ip_address, updated_at = CURRENT_TIMESTAMP
	`, domain, ipAddress)

	if result.Error != nil {
		return fmt.Errorf("failed to create DNS mapping: %w", result.Error)
	}

	return nil
}

// DeleteDNSMapping deletes a DNS mapping by domain
func (c *Client) DeleteDNSMapping(domain string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result := c.db.WithContext(ctx).Where("domain = ?", domain).Delete(&DNSMapping{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete DNS mapping: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("DNS mapping not found")
	}

	return nil
}

// MigrateDNSMappingsFromJSON migrates DNS mappings from a JSON file to PostgreSQL
func (c *Client) MigrateDNSMappingsFromJSON(jsonFilePath string) error {
	// Check if file exists
	data, err := os.ReadFile(jsonFilePath)
	if os.IsNotExist(err) {
		// File doesn't exist, nothing to migrate
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read JSON file: %w", err)
	}

	var config struct {
		Mappings map[string]string `json:"mappings"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	if len(config.Mappings) == 0 {
		return nil
	}

	// Check if we already have mappings in the database
	existingMappings, err := c.GetAllDNSMappings()
	if err != nil {
		return fmt.Errorf("failed to check existing mappings: %w", err)
	}

	if len(existingMappings) > 0 {
		// Already migrated or has data, skip
		return nil
	}

	// Migrate all mappings
	for domain, ipAddress := range config.Mappings {
		domain = strings.TrimSpace(domain)
		ipAddress = strings.TrimSpace(ipAddress)

		if domain == "" || ipAddress == "" {
			continue
		}

		// Ensure domain ends with a dot for DNS processing
		if !strings.HasSuffix(domain, ".") {
			domain += "."
		}

		if err := c.CreateDNSMapping(domain, ipAddress); err != nil {
			return fmt.Errorf("failed to migrate mapping %s: %w", domain, err)
		}
	}

	return nil
}

// AggregatedStatsData represents the cached aggregated statistics
type AggregatedStatsData struct {
	OverviewStats  *OverviewStats               `json:"overview_stats"`
	TimeSeriesData map[string][]TimeSeriesPoint `json:"time_series_data"`
	TopClients     []ClientMetric               `json:"top_clients"`
	QueryTypes     []QueryTypeMetric            `json:"query_types"`
	UpdatedAt      time.Time                    `json:"updated_at"`
}

// CalculateAndStoreAggregatedStats calculates all aggregated statistics and stores them in cache
func (c *Client) CalculateAndStoreAggregatedStats() error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Get all the stats data
	overviewStats, err := c.GetOverviewStats()
	if err != nil {
		return fmt.Errorf("failed to get overview stats: %w", err)
	}

	timeSeriesData, err := c.GetTimeSeriesData()
	if err != nil {
		return fmt.Errorf("failed to get time series data: %w", err)
	}

	topClients, err := c.GetTopClients(20)
	if err != nil {
		return fmt.Errorf("failed to get top clients: %w", err)
	}

	topQueryTypes, err := c.GetTopQueryTypes(8)
	if err != nil {
		return fmt.Errorf("failed to get query types: %w", err)
	}

	// Prepare stats data
	statsData := AggregatedStatsData{
		OverviewStats:  overviewStats,
		TimeSeriesData: timeSeriesData,
		TopClients:     topClients,
		QueryTypes:     topQueryTypes,
		UpdatedAt:      time.Now(),
	}

	// Convert to JSONB
	statsJSON, err := json.Marshal(statsData)
	if err != nil {
		return fmt.Errorf("failed to marshal stats data: %w", err)
	}

	// Store in database using upsert
	result := c.db.WithContext(ctx).Exec(`
		INSERT INTO aggregated_stats (stats_type, stats_data, updated_at, created_at)
		VALUES ($1, $2::jsonb, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT (stats_type) 
		DO UPDATE SET 
			stats_data = EXCLUDED.stats_data,
			updated_at = CURRENT_TIMESTAMP
	`, "dashboard", string(statsJSON))

	if result.Error != nil {
		return fmt.Errorf("failed to store aggregated stats: %w", result.Error)
	}

	return nil
}

// GetCachedAggregatedStats retrieves cached aggregated statistics
func (c *Client) GetCachedAggregatedStats() (*AggregatedStatsData, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get raw database connection for direct JSONB handling
	sqlDB, err := c.db.WithContext(ctx).DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	type AggregatedStatsRow struct {
		StatsData []byte    `db:"stats_data"`
		UpdatedAt time.Time `db:"updated_at"`
	}

	var row AggregatedStatsRow
	query := `
		SELECT stats_data::text, updated_at
		FROM aggregated_stats
		WHERE stats_type = $1
		LIMIT 1
	`
	err = sqlDB.QueryRowContext(ctx, query, "dashboard").Scan(&row.StatsData, &row.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get cached stats: %w", err)
	}

	// Unmarshal the JSONB data
	var statsData AggregatedStatsData
	if err := json.Unmarshal(row.StatsData, &statsData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal stats data: %w", err)
	}

	statsData.UpdatedAt = row.UpdatedAt

	return &statsData, nil
}

// GetEarliestLogTimestamp returns the earliest timestamp from DNS logs
// This can be used to approximate when the DNS server started
func (c *Client) GetEarliestLogTimestamp() (*time.Time, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var earliestTimestamp sql.NullTime
	if err := c.db.WithContext(ctx).Raw(`
		SELECT MIN(timestamp) as earliest_timestamp
		FROM dns_logs
	`).Scan(&earliestTimestamp).Error; err != nil {
		return nil, fmt.Errorf("failed to get earliest log timestamp: %w", err)
	}

	if !earliestTimestamp.Valid {
		// No logs found, return nil to indicate no start time available
		return nil, nil
	}

	return &earliestTimestamp.Time, nil
}

const (
	// MetadataKeyDNSServerStartTime is the key for storing DNS server start time
	MetadataKeyDNSServerStartTime = "dns_server_start_time"
)

// SetDNSServerStartTime records the DNS server start time in the system metadata table
func (c *Client) SetDNSServerStartTime(startTime time.Time) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result := c.db.WithContext(ctx).Exec(`
		INSERT INTO system_metadata (metadata_key, metadata_value, updated_at, created_at)
		VALUES ($1, $2, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT (metadata_key) 
		DO UPDATE SET 
			metadata_value = EXCLUDED.metadata_value,
			updated_at = CURRENT_TIMESTAMP
	`, MetadataKeyDNSServerStartTime, startTime.Format(time.RFC3339))

	if result.Error != nil {
		return fmt.Errorf("failed to set DNS server start time: %w", result.Error)
	}

	return nil
}

// GetDNSServerStartTime retrieves the DNS server start time from the system metadata table
func (c *Client) GetDNSServerStartTime() (*time.Time, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sqlDB, err := c.db.WithContext(ctx).DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	var metadataValue string
	query := `
		SELECT metadata_value
		FROM system_metadata
		WHERE metadata_key = $1
		LIMIT 1
	`
	err = sqlDB.QueryRowContext(ctx, query, MetadataKeyDNSServerStartTime).Scan(&metadataValue)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get DNS server start time: %w", err)
	}

	if metadataValue == "" {
		return nil, nil
	}

	startTime, err := time.Parse(time.RFC3339, metadataValue)
	if err != nil {
		return nil, fmt.Errorf("failed to parse start time: %w", err)
	}

	return &startTime, nil
}
