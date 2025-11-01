package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"dns-go/internal/types"

	"github.com/lib/pq"
	_ "github.com/lib/pq"
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
	db *sql.DB
}

// Config holds PostgreSQL configuration
type Config struct {
	Host     string
	Port     string
	Database string
	User     string
	Password string
}

// NewClient creates a new PostgreSQL client
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

	// First, try to connect to the target database
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, database)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	client := &Client{
		db: db,
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.db.PingContext(ctx); err != nil {
		// If connection fails, try to create the database
		client.db.Close()

		// Connect to default 'postgres' database to create the target database
		defaultDSN := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=postgres sslmode=disable",
			host, port, user, password)

		defaultDB, err := sql.Open("postgres", defaultDSN)
		if err != nil {
			return nil, fmt.Errorf("failed to open connection to default database: %w", err)
		}
		defer defaultDB.Close()

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
			// Use pq.QuoteIdentifier for safe SQL identifier quoting
			quotedDB := pq.QuoteIdentifier(database)
			createDBSQL := fmt.Sprintf("CREATE DATABASE %s", quotedDB)
			_, err = defaultDB.ExecContext(ctx, createDBSQL)
			if err != nil {
				// Check if error is because database already exists (race condition)
				if !isDatabaseExistsError(err) {
					return nil, fmt.Errorf("failed to create database %s: %w", database, err)
				}
				// Database might have been created between our attempts, continue
			}
		}

		// Now connect to the newly created database
		db, err = sql.Open("postgres", dsn)
		if err != nil {
			return nil, fmt.Errorf("failed to open database connection after creation: %w", err)
		}

		client.db = db

		// Test connection again
		if err := client.db.PingContext(ctx); err != nil {
			return nil, fmt.Errorf("failed to ping database after creation: %w", err)
		}
	}

	// Create table and indices if they don't exist
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
	// PostgreSQL error codes: 42P04 = duplicate_database
	return strings.Contains(errStr, "already exists") ||
		strings.Contains(errStr, "duplicate_database") ||
		strings.Contains(errStr, "42p04")
}

// initialize creates the table and indices if they don't exist
func (c *Client) initialize() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	createTableSQL := `
	CREATE TABLE IF NOT EXISTS dns_logs (
		id SERIAL PRIMARY KEY,
		uuid VARCHAR(255) UNIQUE NOT NULL,
		timestamp TIMESTAMP NOT NULL,
		client_ip INET NOT NULL,
		query VARCHAR(255) NOT NULL,
		query_type VARCHAR(10) NOT NULL,
		query_id INTEGER,
		status VARCHAR(50) NOT NULL,
		cache_hit BOOLEAN DEFAULT FALSE,
		duration_ms DOUBLE PRECISION,
		response_upstream VARCHAR(255),
		response_rcode VARCHAR(10),
		response_answer_count INTEGER,
		response_rtt_ms DOUBLE PRECISION,
		upstreams JSONB,
		answers JSONB,
		ip_addresses INET[],
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	`

	if _, err := c.db.ExecContext(ctx, createTableSQL); err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	// Create indices for efficient aggregation queries
	indices := []string{
		// Time-based indices for aggregation (minute, hour, day, week)
		"CREATE INDEX IF NOT EXISTS idx_dns_logs_timestamp ON dns_logs(timestamp)",
		"CREATE INDEX IF NOT EXISTS idx_dns_logs_timestamp_date ON dns_logs(DATE_TRUNC('day', timestamp))",
		"CREATE INDEX IF NOT EXISTS idx_dns_logs_timestamp_hour ON dns_logs(DATE_TRUNC('hour', timestamp))",
		"CREATE INDEX IF NOT EXISTS idx_dns_logs_timestamp_minute ON dns_logs(DATE_TRUNC('minute', timestamp))",

		// Client IP index for client-based aggregation
		"CREATE INDEX IF NOT EXISTS idx_dns_logs_client_ip ON dns_logs(client_ip)",

		// Composite indices for common queries
		"CREATE INDEX IF NOT EXISTS idx_dns_logs_timestamp_client ON dns_logs(timestamp, client_ip)",
		"CREATE INDEX IF NOT EXISTS idx_dns_logs_status ON dns_logs(status)",
		"CREATE INDEX IF NOT EXISTS idx_dns_logs_cache_hit ON dns_logs(cache_hit)",
		"CREATE INDEX IF NOT EXISTS idx_dns_logs_query_type ON dns_logs(query_type)",

		// UUID index for lookups
		"CREATE INDEX IF NOT EXISTS idx_dns_logs_uuid ON dns_logs(uuid)",

		// Indices for search queries (ILIKE performance)
		"CREATE INDEX IF NOT EXISTS idx_dns_logs_query ON dns_logs(query)",
		"CREATE INDEX IF NOT EXISTS idx_dns_logs_response_upstream ON dns_logs(response_upstream)",
	}

	for _, indexSQL := range indices {
		if _, err := c.db.ExecContext(ctx, indexSQL); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}

// InsertLogEntry inserts a DNS log entry into PostgreSQL
func (c *Client) InsertLogEntry(entry types.LogEntry) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	clientIP := types.ExtractIPFromAddr(entry.Request.Client)

	// Convert upstreams to JSONB
	upstreamsJSON, err := json.Marshal(entry.Upstreams)
	if err != nil {
		return fmt.Errorf("failed to marshal upstreams: %w", err)
	}

	// Convert answers to JSONB
	answersJSON, err := json.Marshal(entry.Answers)
	if err != nil {
		return fmt.Errorf("failed to marshal answers: %w", err)
	}

	// Convert IP addresses array - use pq.Array for proper PostgreSQL array handling
	var ipAddressesArray interface{}
	if len(entry.IPAddresses) > 0 {
		ipAddressesArray = pq.Array(entry.IPAddresses)
	}

	// Prepare SQL statement
	insertSQL := `
	INSERT INTO dns_logs (
		uuid, timestamp, client_ip, query, query_type, query_id,
		status, cache_hit, duration_ms,
		response_upstream, response_rcode, response_answer_count, response_rtt_ms,
		upstreams, answers, ip_addresses
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	ON CONFLICT (uuid) DO NOTHING;
	`

	var responseUpstream, responseRcode sql.NullString
	var responseAnswerCount sql.NullInt32
	var responseRTT sql.NullFloat64

	if entry.Response != nil {
		responseUpstream = sql.NullString{String: entry.Response.Upstream, Valid: true}
		responseRcode = sql.NullString{String: entry.Response.Rcode, Valid: true}
		responseAnswerCount = sql.NullInt32{Int32: int32(entry.Response.AnswerCount), Valid: true}
		responseRTT = sql.NullFloat64{Float64: entry.Response.RTT, Valid: true}
	}

	_, err = c.db.ExecContext(ctx, insertSQL,
		entry.UUID,
		entry.Timestamp,
		clientIP,
		entry.Request.Query,
		entry.Request.Type,
		entry.Request.ID,
		entry.Status,
		entry.CacheHit,
		entry.Duration,
		responseUpstream,
		responseRcode,
		responseAnswerCount,
		responseRTT,
		upstreamsJSON,
		answersJSON,
		ipAddressesArray,
	)

	if err != nil {
		return fmt.Errorf("failed to insert log entry: %w", err)
	}

	return nil
}

// SearchLogs searches through DNS logs stored in PostgreSQL
type SearchResult struct {
	Results []types.LogEntry `json:"results"`
	Total   int64            `json:"total"`
}

// SearchLogs searches DNS logs with pagination and optional filters
func (c *Client) SearchLogs(searchTerm string, limit, offset int, since *time.Time) (*SearchResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var whereClauses []string
	var args []interface{}
	argIndex := 1

	// Build WHERE clause based on search term
	if searchTerm != "" {
		// Check if search term looks like an IP address or partial IP
		searchPattern := "%" + searchTerm + "%"
		whereClause := fmt.Sprintf(`(
			query ILIKE $%d OR
			client_ip::text ILIKE $%d OR
			query_type ILIKE $%d OR
			status ILIKE $%d OR
			response_upstream ILIKE $%d OR
			uuid = $%d`,
			argIndex, argIndex, argIndex, argIndex, argIndex, argIndex)

		// Add IP address array search if search term contains dots or numbers
		if strings.Contains(searchTerm, ".") || (len(searchTerm) > 0 && searchTerm[0] >= '0' && searchTerm[0] <= '9') {
			whereClause += fmt.Sprintf(` OR EXISTS (
				SELECT 1 FROM unnest(ip_addresses) AS ip WHERE ip::text ILIKE $%d
			)`, argIndex)
		}
		whereClause += ")"

		whereClauses = append(whereClauses, whereClause)
		args = append(args, searchPattern)
		argIndex++
	}

	// Add time filter if specified
	if since != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("timestamp >= $%d AND timestamp <= $%d", argIndex, argIndex+1))
		args = append(args, *since, time.Now())
		argIndex += 2
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = "WHERE " + fmt.Sprintf("(%s)", whereClauses[0])
		for i := 1; i < len(whereClauses); i++ {
			whereSQL += " AND " + whereClauses[i]
		}
	}

	// Count total results
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM dns_logs %s", whereSQL)
	var total int64
	if err := c.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("failed to count results: %w", err)
	}

	// Fetch paginated results
	selectSQL := fmt.Sprintf(`
		SELECT uuid, timestamp, client_ip, query, query_type, query_id,
			status, cache_hit, duration_ms,
			response_upstream, response_rcode, response_answer_count, response_rtt_ms,
			upstreams, answers, ip_addresses
		FROM dns_logs
		%s
		ORDER BY timestamp DESC
		LIMIT $%d OFFSET $%d
	`, whereSQL, argIndex, argIndex+1)

	args = append(args, limit, offset)

	rows, err := c.db.QueryContext(ctx, selectSQL, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query logs: %w", err)
	}
	defer rows.Close()

	var results []types.LogEntry
	for rows.Next() {
		var entry types.LogEntry
		var clientIP string
		var upstreamsJSON, answersJSON []byte
		var ipAddressesArray pq.StringArray
		var responseUpstream, responseRcode sql.NullString
		var responseAnswerCount sql.NullInt32
		var responseRTT sql.NullFloat64

		err := rows.Scan(
			&entry.UUID,
			&entry.Timestamp,
			&clientIP,
			&entry.Request.Query,
			&entry.Request.Type,
			&entry.Request.ID,
			&entry.Status,
			&entry.CacheHit,
			&entry.Duration,
			&responseUpstream,
			&responseRcode,
			&responseAnswerCount,
			&responseRTT,
			&upstreamsJSON,
			&answersJSON,
			&ipAddressesArray,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Reconstruct entry
		entry.Request.Client = clientIP
		if responseUpstream.Valid {
			entry.Response = &types.ResponseInfo{
				Upstream:    responseUpstream.String,
				Rcode:       responseRcode.String,
				AnswerCount: int(responseAnswerCount.Int32),
				RTT:         responseRTT.Float64,
			}
		}

		if err := json.Unmarshal(upstreamsJSON, &entry.Upstreams); err != nil {
			return nil, fmt.Errorf("failed to unmarshal upstreams: %w", err)
		}

		if err := json.Unmarshal(answersJSON, &entry.Answers); err != nil {
			return nil, fmt.Errorf("failed to unmarshal answers: %w", err)
		}

		// Convert pq.StringArray to []string
		if ipAddressesArray != nil {
			entry.IPAddresses = []string(ipAddressesArray)
		} else {
			entry.IPAddresses = []string{}
		}

		results = append(results, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
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
	err := c.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM dns_logs").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count logs: %w", err)
	}

	return count, nil
}

// TimeSeriesPoint represents a time series data point
type TimeSeriesPoint struct {
	Timestamp int64
	Value     int64
}

// GetTimeSeriesData returns aggregated time series data from PostgreSQL
func (c *Client) GetTimeSeriesData() (map[string][]TimeSeriesPoint, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result := make(map[string][]TimeSeriesPoint)

	// Last hour - per minute (75 minutes)
	hourSQL := `
		SELECT 
			EXTRACT(EPOCH FROM DATE_TRUNC('minute', timestamp))::BIGINT as ts,
			COUNT(*)::BIGINT as count
		FROM dns_logs
		WHERE timestamp >= NOW() - INTERVAL '75 minutes'
		GROUP BY DATE_TRUNC('minute', timestamp)
		ORDER BY ts ASC
		LIMIT 75
	`
	rows, err := c.db.QueryContext(ctx, hourSQL)
	if err != nil {
		return nil, fmt.Errorf("failed to query hourly data: %w", err)
	}
	defer rows.Close()

	var hourData []TimeSeriesPoint
	for rows.Next() {
		var point TimeSeriesPoint
		if err := rows.Scan(&point.Timestamp, &point.Value); err != nil {
			return nil, fmt.Errorf("failed to scan hour data: %w", err)
		}
		hourData = append(hourData, point)
	}
	result["requests_last_hour"] = hourData

	// Last day - per hour (75 hours)
	daySQL := `
		SELECT 
			EXTRACT(EPOCH FROM DATE_TRUNC('hour', timestamp))::BIGINT as ts,
			COUNT(*)::BIGINT as count
		FROM dns_logs
		WHERE timestamp >= NOW() - INTERVAL '75 hours'
		GROUP BY DATE_TRUNC('hour', timestamp)
		ORDER BY ts ASC
		LIMIT 75
	`
	rows, err = c.db.QueryContext(ctx, daySQL)
	if err != nil {
		return nil, fmt.Errorf("failed to query daily data: %w", err)
	}
	defer rows.Close()

	var dayData []TimeSeriesPoint
	for rows.Next() {
		var point TimeSeriesPoint
		if err := rows.Scan(&point.Timestamp, &point.Value); err != nil {
			return nil, fmt.Errorf("failed to scan day data: %w", err)
		}
		dayData = append(dayData, point)
	}
	result["requests_last_day"] = dayData

	// Last week - per day (75 days)
	weekSQL := `
		SELECT 
			EXTRACT(EPOCH FROM DATE_TRUNC('day', timestamp))::BIGINT as ts,
			COUNT(*)::BIGINT as count
		FROM dns_logs
		WHERE timestamp >= NOW() - INTERVAL '75 days'
		GROUP BY DATE_TRUNC('day', timestamp)
		ORDER BY ts ASC
		LIMIT 75
	`
	rows, err = c.db.QueryContext(ctx, weekSQL)
	if err != nil {
		return nil, fmt.Errorf("failed to query weekly data: %w", err)
	}
	defer rows.Close()

	var weekData []TimeSeriesPoint
	for rows.Next() {
		var point TimeSeriesPoint
		if err := rows.Scan(&point.Timestamp, &point.Value); err != nil {
			return nil, fmt.Errorf("failed to scan week data: %w", err)
		}
		weekData = append(weekData, point)
	}
	result["requests_last_week"] = weekData

	// Last month - per day (75 days)
	monthSQL := `
		SELECT 
			EXTRACT(EPOCH FROM DATE_TRUNC('day', timestamp))::BIGINT as ts,
			COUNT(*)::BIGINT as count
		FROM dns_logs
		WHERE timestamp >= NOW() - INTERVAL '75 days'
		GROUP BY DATE_TRUNC('day', timestamp)
		ORDER BY ts ASC
		LIMIT 75
	`
	rows, err = c.db.QueryContext(ctx, monthSQL)
	if err != nil {
		return nil, fmt.Errorf("failed to query monthly data: %w", err)
	}
	defer rows.Close()

	var monthData []TimeSeriesPoint
	for rows.Next() {
		var point TimeSeriesPoint
		if err := rows.Scan(&point.Timestamp, &point.Value); err != nil {
			return nil, fmt.Errorf("failed to scan month data: %w", err)
		}
		monthData = append(monthData, point)
	}
	result["requests_last_month"] = monthData

	return result, nil
}

// ClientMetric represents aggregated client statistics
type ClientMetric struct {
	IP           string
	Requests     int64
	CacheHitRate float64
	SuccessRate  float64
	LastSeen     time.Time
}

// GetTopClients returns top clients aggregated from PostgreSQL
func (c *Client) GetTopClients(limit int) ([]ClientMetric, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sql := `
		SELECT 
			client_ip,
			COUNT(*)::BIGINT as total_requests,
			COUNT(*) FILTER (WHERE cache_hit = true)::BIGINT as cache_hits,
			COUNT(*) FILTER (WHERE status = 'success' OR status = 'cache_hit')::BIGINT as successful,
			MAX(timestamp) as last_seen
		FROM dns_logs
		GROUP BY client_ip
		ORDER BY total_requests DESC
		LIMIT $1
	`

	rows, err := c.db.QueryContext(ctx, sql, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query top clients: %w", err)
	}
	defer rows.Close()

	var clients []ClientMetric
	for rows.Next() {
		var client ClientMetric
		var totalRequests, cacheHits, successful int64

		err := rows.Scan(&client.IP, &totalRequests, &cacheHits, &successful, &client.LastSeen)
		if err != nil {
			return nil, fmt.Errorf("failed to scan client data: %w", err)
		}

		client.Requests = totalRequests
		if totalRequests > 0 {
			client.CacheHitRate = float64(cacheHits) / float64(totalRequests) * 100
			client.SuccessRate = float64(successful) / float64(totalRequests) * 100
		}

		clients = append(clients, client)
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

	sql := `
		SELECT 
			query_type,
			COUNT(*)::BIGINT as count
		FROM dns_logs
		GROUP BY query_type
		ORDER BY count DESC
		LIMIT $1
	`

	rows, err := c.db.QueryContext(ctx, sql, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query query types: %w", err)
	}
	defer rows.Close()

	var queryTypes []QueryTypeMetric
	for rows.Next() {
		var qt QueryTypeMetric
		if err := rows.Scan(&qt.Type, &qt.Count); err != nil {
			return nil, fmt.Errorf("failed to scan query type data: %w", err)
		}
		queryTypes = append(queryTypes, qt)
	}

	return queryTypes, nil
}

// OverviewStats represents overview statistics
type OverviewStats struct {
	TotalRequests       int64
	CacheHits           int64
	SuccessfulQueries   int64
	AverageResponseTime float64
	ActiveClients       int
}

// GetOverviewStats returns overview statistics from PostgreSQL
func (c *Client) GetOverviewStats() (*OverviewStats, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stats := &OverviewStats{}

	// Get total requests, cache hits, and average response time
	sql := `
		SELECT 
			COUNT(*)::BIGINT as total_requests,
			COUNT(*) FILTER (WHERE cache_hit = true)::BIGINT as cache_hits,
			COUNT(*) FILTER (WHERE status = 'success' OR status = 'cache_hit')::BIGINT as successful,
			AVG(duration_ms) as avg_response_time
		FROM dns_logs
	`

	err := c.db.QueryRowContext(ctx, sql).Scan(
		&stats.TotalRequests,
		&stats.CacheHits,
		&stats.SuccessfulQueries,
		&stats.AverageResponseTime,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query overview stats: %w", err)
	}

	// Get active clients (seen in last hour)
	clientSQL := `
		SELECT COUNT(DISTINCT client_ip)::INTEGER
		FROM dns_logs
		WHERE timestamp >= NOW() - INTERVAL '1 hour'
	`

	err = c.db.QueryRowContext(ctx, clientSQL).Scan(&stats.ActiveClients)
	if err != nil {
		// If this fails, we can still return the other stats
		stats.ActiveClients = 0
	}

	return stats, nil
}

// GetRecentRequests returns recent requests for display
func (c *Client) GetRecentRequests(limit int) ([]types.LogEntry, error) {
	result, err := c.SearchLogs("", limit, 0, nil)
	if err != nil {
		return nil, err
	}
	return result.Results, nil
}

// HealthCheck checks if PostgreSQL is healthy
func (c *Client) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := c.db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	return nil
}

// Close closes the PostgreSQL connection
func (c *Client) Close() error {
	if c.db != nil {
		return c.db.Close()
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
