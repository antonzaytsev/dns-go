package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"dns-go/internal/config"
	"dns-go/internal/metrics"
	"dns-go/internal/monitor"
	"dns-go/internal/postgres"
	"dns-go/pkg/version"
)

// Server provides REST API endpoints for DNS server metrics
type Server struct {
	server     *http.Server
	metrics    *metrics.Metrics
	logMonitor *monitor.LogMonitor
	pgClient   *postgres.Client
	config     *config.Config
	port       string
}

// Config holds API server configuration
type Config struct {
	Port        string
	LogFilePath string
	DNSConfig   *config.Config
}

// NewServer creates a new API server instance
func NewServer(cfg Config) (*Server, error) {
	metricsCollector := metrics.NewMetrics()

	// Try to find log file if not specified
	logFilePath := cfg.LogFilePath
	if logFilePath == "" {
		logFilePath = monitor.FindLogFile()
	}

	var logMonitor *monitor.LogMonitor
	if logFilePath != "" {
		logMonitor = monitor.NewLogMonitor(logFilePath, metricsCollector)
		if err := logMonitor.Start(); err != nil {
			fmt.Printf("Warning: Could not start log monitor: %v\n", err)
		}
	} else {
		fmt.Println("Warning: No DNS log file found. Real-time metrics will not be available.")
	}

	// Initialize PostgreSQL client if configuration is provided
	var pgClient *postgres.Client
	pgHost := os.Getenv("POSTGRES_HOST")
	pgPort := os.Getenv("POSTGRES_PORT")
	pgDB := os.Getenv("POSTGRES_DB")
	pgUser := os.Getenv("POSTGRES_USER")
	pgPassword := os.Getenv("POSTGRES_PASSWORD")

	if pgHost != "" || pgPort != "" || pgDB != "" {
		pgConfig := postgres.Config{
			Host:     pgHost,
			Port:     pgPort,
			Database: pgDB,
			User:     pgUser,
			Password: pgPassword,
		}

		if client, err := postgres.NewClient(pgConfig); err == nil {
			pgClient = client
			fmt.Println("✅ PostgreSQL client initialized successfully")

			// Migrate DNS mappings from JSON file to PostgreSQL if needed
			const customDNSConfigFile = "custom-dns.json"
			if err := pgClient.MigrateDNSMappingsFromJSON(customDNSConfigFile); err != nil {
				fmt.Printf("⚠️  Warning: Failed to migrate DNS mappings from JSON: %v\n", err)
			} else {
				fmt.Println("✅ DNS mappings migration completed")
			}
		} else {
			fmt.Printf("⚠️  Warning: Failed to initialize PostgreSQL client: %v\n", err)
		}
	} else {
		fmt.Println("📝 No PostgreSQL configuration provided")
	}

	s := &Server{
		metrics:    metricsCollector,
		logMonitor: logMonitor,
		pgClient:   pgClient,
		config:     cfg.DNSConfig,
		port:       cfg.Port,
	}

	// Setup HTTP routes
	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("/api/metrics", s.handleMetrics)
	mux.HandleFunc("/api/clients", s.handleClients)
	mux.HandleFunc("/api/search", s.handleSearch)
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/version", s.handleVersion)
	mux.HandleFunc("/api/dns-mappings", s.handleDNSMappings)
	mux.HandleFunc("/api/log-counts", s.handleLogCounts)

	// CORS middleware
	handler := s.corsMiddleware(s.loggingMiddleware(mux))

	s.server = &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s, nil
}

// Start starts the API server
func (s *Server) Start() error {
	fmt.Printf("\n🚀 DNS API Server Starting\n")
	fmt.Printf("========================\n")
	fmt.Printf("Port: %s\n", s.port)
	fmt.Printf("Time: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Printf("\n📡 Available Endpoints:\n")
	fmt.Printf("  🔍 GET /api/metrics      - DNS server metrics and statistics\n")
	fmt.Printf("  👥 GET /api/clients      - DNS clients and statistics\n")
	fmt.Printf("  🔎 GET /api/search       - Search through DNS logs\n")
	fmt.Printf("  ❤️  GET /api/health       - Health check endpoint\n")
	fmt.Printf("  ℹ️  GET /api/version      - Version and build information\n")
	fmt.Printf("  🌐 GET/PUT/POST/DELETE /api/dns-mappings - Manage custom DNS mappings\n")
	fmt.Printf("\n🌐 Access URLs:\n")
	fmt.Printf("  Local:    http://localhost:%s/api\n", s.port)
	fmt.Printf("  Network:  http://0.0.0.0:%s/api\n", s.port)
	fmt.Printf("\n📊 Log storage: %s\n", func() string {
		if s.pgClient != nil {
			return "✅ PostgreSQL"
		}
		return "❌ None"
	}())
	fmt.Printf("🔍 Log search: %s\n", func() string {
		if s.pgClient != nil {
			return "✅ PostgreSQL"
		} else if s.logMonitor != nil {
			return "📁 File-based (fallback)"
		}
		return "❌ Disabled"
	}())
	fmt.Printf("========================\n\n")

	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the API server
func (s *Server) Shutdown(ctx context.Context) error {
	// Stop log monitor first
	if s.logMonitor != nil {
		s.logMonitor.Stop()
	}

	// Close PostgreSQL client
	if s.pgClient != nil {
		s.pgClient.Close()
	}

	return s.server.Shutdown(ctx)
}

// GetMetrics returns the metrics collector for external use
func (s *Server) GetMetrics() *metrics.Metrics {
	return s.metrics
}

// HTTP Handlers

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if s.pgClient == nil {
		http.Error(w, "PostgreSQL not connected", http.StatusServiceUnavailable)
		return
	}

	// Build dashboard metrics from PostgreSQL
	dashboardMetrics, err := s.buildDashboardMetricsFromPostgres()
	if err != nil {
		http.Error(w, "Failed to build metrics: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(dashboardMetrics); err != nil {
		http.Error(w, "Failed to encode metrics", http.StatusInternalServerError)
		return
	}
}

// buildDashboardMetricsFromPostgres builds dashboard metrics by aggregating data from PostgreSQL
func (s *Server) buildDashboardMetricsFromPostgres() (*metrics.DashboardMetrics, error) {
	if s.pgClient == nil {
		return nil, fmt.Errorf("PostgreSQL client not available")
	}

	// Get all data in parallel
	overviewStats, err := s.pgClient.GetOverviewStats()
	if err != nil {
		return nil, fmt.Errorf("failed to get overview stats: %w", err)
	}

	timeSeriesData, err := s.pgClient.GetTimeSeriesData()
	if err != nil {
		return nil, fmt.Errorf("failed to get time series data: %w", err)
	}

	topClients, err := s.pgClient.GetTopClients(20)
	if err != nil {
		return nil, fmt.Errorf("failed to get top clients: %w", err)
	}

	topQueryTypes, err := s.pgClient.GetTopQueryTypes(8)
	if err != nil {
		return nil, fmt.Errorf("failed to get query types: %w", err)
	}

	recentRequests, err := s.pgClient.GetRecentRequests(100)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent requests: %w", err)
	}

	// Convert PostgreSQL types to metrics types
	overview := metrics.OverviewMetrics{
		Uptime:              "N/A", // We don't track uptime from DB
		TotalRequests:       overviewStats.TotalRequests,
		RequestsPerSecond:   0, // Calculate from time window if needed
		CacheHitRate:        0,
		SuccessRate:         0,
		AverageResponseTime: overviewStats.AverageResponseTime,
		Clients:             overviewStats.ActiveClients,
	}

	if overviewStats.TotalRequests > 0 {
		overview.CacheHitRate = float64(overviewStats.CacheHits) / float64(overviewStats.TotalRequests) * 100
		overview.SuccessRate = float64(overviewStats.SuccessfulQueries) / float64(overviewStats.TotalRequests) * 100
	}

	// Convert time series data
	timeSeries := metrics.TimeSeriesData{
		RequestsLastHour:  convertTimeSeriesPoints(timeSeriesData["requests_last_hour"]),
		RequestsLastDay:   convertTimeSeriesPoints(timeSeriesData["requests_last_day"]),
		RequestsLastWeek:  convertTimeSeriesPoints(timeSeriesData["requests_last_week"]),
		RequestsLastMonth: convertTimeSeriesPoints(timeSeriesData["requests_last_month"]),
	}

	// Convert top clients
	clientMetrics := make([]metrics.ClientMetric, len(topClients))
	for i, client := range topClients {
		clientMetrics[i] = metrics.ClientMetric{
			IP:           client.IP,
			Requests:     client.Requests,
			CacheHitRate: client.CacheHitRate,
			SuccessRate:  client.SuccessRate,
			LastSeen:     client.LastSeen,
		}
	}

	// Convert query types
	queryTypeMetrics := make([]metrics.QueryTypeMetric, len(topQueryTypes))
	for i, qt := range topQueryTypes {
		queryTypeMetrics[i] = metrics.QueryTypeMetric{
			Type:  qt.Type,
			Count: qt.Count,
		}
	}

	// Build upstream servers stats (empty for now, can be added later)
	upstreamServers := make(map[string]*metrics.UpstreamStats)

	return &metrics.DashboardMetrics{
		Overview:        overview,
		TimeSeriesData:  timeSeries,
		TopClients:      clientMetrics,
		QueryTypes:      queryTypeMetrics,
		UpstreamServers: upstreamServers,
		Requests:        recentRequests,
		SystemInfo: metrics.SystemInfo{
			Version:   version.Get().Short(),
			StartTime: time.Now().Format(time.RFC3339), // Could track from DB if needed
		},
	}, nil
}

// convertTimeSeriesPoints converts PostgreSQL time series points to metrics format
func convertTimeSeriesPoints(points []postgres.TimeSeriesPoint) []metrics.TimePoint {
	result := make([]metrics.TimePoint, len(points))
	for i, point := range points {
		// PostgreSQL returns Unix timestamp in seconds, frontend expects milliseconds
		result[i] = metrics.TimePoint{
			Timestamp: point.Timestamp * 1000,
			Value:     point.Value,
		}
	}
	return result
}

func (s *Server) handleClients(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if s.pgClient == nil {
		http.Error(w, "PostgreSQL not connected", http.StatusServiceUnavailable)
		return
	}

	// Get all clients from PostgreSQL
	pgClients, err := s.pgClient.GetTopClients(1000) // Get many clients
	if err != nil {
		http.Error(w, "Failed to get clients: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert to metrics.ClientMetric format
	clients := make([]metrics.ClientMetric, len(pgClients))
	for i, client := range pgClients {
		clients[i] = metrics.ClientMetric{
			IP:           client.IP,
			Requests:     client.Requests,
			CacheHitRate: client.CacheHitRate,
			SuccessRate:  client.SuccessRate,
			LastSeen:     client.LastSeen,
		}
	}

	response := map[string]interface{}{
		"clients": clients,
		"total":   len(clients),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode clients", http.StatusInternalServerError)
		return
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"version":   version.Get().Short(),
		"uptime":    time.Since(time.Now()).String(), // This will be updated with actual start time
	}

	if r.Method == http.MethodGet {
		json.NewEncoder(w).Encode(health)
	}
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Parse query parameters
	query := r.URL.Query()
	searchTerm := query.Get("q")
	limitStr := query.Get("limit")
	offsetStr := query.Get("offset")
	sinceStr := query.Get("since")

	// Set defaults
	limit := 100
	offset := 0
	var since *time.Time

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Parse and validate since parameter (only format: 2024-01-02T15:04:05Z)
	if sinceStr != "" {
		parsedTime, err := time.Parse("2006-01-02T15:04:05Z", sinceStr)
		if err != nil {
			http.Error(w, "Invalid since parameter: must be in format 2024-01-02T15:04:05Z", http.StatusBadRequest)
			return
		}

		// Validate that the timestamp is not in the future
		if parsedTime.After(time.Now()) {
			http.Error(w, "Invalid since parameter: timestamp cannot be in the future", http.StatusBadRequest)
			return
		}

		since = &parsedTime
	}

	// Use PostgreSQL for search
	if s.pgClient == nil {
		http.Error(w, "Search service unavailable: PostgreSQL not connected", http.StatusServiceUnavailable)
		return
	}

	// Search in PostgreSQL
	searchResult, err := s.pgClient.SearchLogs(searchTerm, limit, offset, since)
	if err != nil {
		fmt.Printf("PostgreSQL search failed: %v\n", err)
		http.Error(w, "Search failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"results": searchResult.Results,
		"total":   searchResult.Total,
		"limit":   limit,
		"offset":  offset,
		"query":   searchTerm,
		"since":   since,
		"source":  "postgres",
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode search results", http.StatusInternalServerError)
		return
	}
}

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	versionInfo := version.Get()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"version":    versionInfo.Version,
		"git_commit": versionInfo.GitCommit,
		"build_date": versionInfo.BuildDate,
		"go_version": versionInfo.GoVersion,
	})
}

func (s *Server) handleLogCounts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	response := map[string]interface{}{
		"postgres": nil,
	}

	// Get PostgreSQL count
	if s.pgClient != nil {
		pgCount, err := s.pgClient.GetLogCount()
		if err != nil {
			response["postgres"] = map[string]interface{}{
				"count": nil,
				"error": err.Error(),
			}
		} else {
			response["postgres"] = map[string]interface{}{
				"count": pgCount,
				"error": nil,
			}
		}
	} else {
		response["postgres"] = map[string]interface{}{
			"count": nil,
			"error": "PostgreSQL not connected",
		}
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode log counts", http.StatusInternalServerError)
		return
	}
}

func (s *Server) handleDNSMappings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Require PostgreSQL client for DNS mappings
	if s.pgClient == nil {
		http.Error(w, "PostgreSQL not connected", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Return current DNS mappings from PostgreSQL
		mappings, err := s.pgClient.GetAllDNSMappings()
		if err != nil {
			http.Error(w, "Failed to get DNS mappings: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Remove trailing dots from domains for display (user-friendly format)
		displayMappings := make(map[string]string)
		for domain, ip := range mappings {
			displayDomain := strings.TrimSuffix(domain, ".")
			displayMappings[displayDomain] = ip
		}

		response := map[string]interface{}{
			"mappings": displayMappings,
			"count":    len(displayMappings),
		}
		json.NewEncoder(w).Encode(response)

	case http.MethodPost:
		// Add a single DNS mapping
		var requestBody struct {
			Domain string `json:"domain"`
			IP     string `json:"ip"`
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		if err := json.Unmarshal(body, &requestBody); err != nil {
			http.Error(w, "Invalid JSON format", http.StatusBadRequest)
			return
		}

		domain := strings.TrimSpace(requestBody.Domain)
		ip := strings.TrimSpace(requestBody.IP)

		if domain == "" || ip == "" {
			http.Error(w, "Domain and IP are required", http.StatusBadRequest)
			return
		}

		// Ensure domain ends with a dot for DNS processing
		if !strings.HasSuffix(domain, ".") {
			domain += "."
		}

		// Check if domain already exists
		existingMappings, err := s.pgClient.GetAllDNSMappings()
		if err != nil {
			http.Error(w, "Failed to check existing mappings: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if _, exists := existingMappings[domain]; exists {
			http.Error(w, "Domain mapping already exists. Delete first to update.", http.StatusConflict)
			return
		}

		// Create the mapping in PostgreSQL
		if err := s.pgClient.CreateDNSMapping(domain, ip); err != nil {
			http.Error(w, "Failed to create DNS mapping: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Update in-memory config if available
		if s.config != nil {
			if s.config.CustomDNS == nil {
				s.config.CustomDNS = make(map[string]string)
			}
			s.config.CustomDNS[domain] = ip
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "DNS mapping added successfully",
			"domain":  strings.TrimSuffix(domain, "."),
			"ip":      ip,
		})

	case http.MethodDelete:
		// Delete a specific DNS mapping
		domain := r.URL.Query().Get("domain")
		if domain == "" {
			http.Error(w, "Domain parameter is required", http.StatusBadRequest)
			return
		}

		// Ensure domain ends with a dot for DNS processing
		if !strings.HasSuffix(domain, ".") {
			domain += "."
		}

		// Delete from PostgreSQL
		if err := s.pgClient.DeleteDNSMapping(domain); err != nil {
			if strings.Contains(err.Error(), "not found") {
				http.Error(w, "Domain mapping not found", http.StatusNotFound)
			} else {
				http.Error(w, "Failed to delete DNS mapping: "+err.Error(), http.StatusInternalServerError)
			}
			return
		}

		// Update in-memory config if available
		if s.config != nil && s.config.CustomDNS != nil {
			delete(s.config.CustomDNS, domain)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "DNS mapping deleted successfully",
			"domain":  strings.TrimSuffix(domain, "."),
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Middleware

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow requests from any origin for development
		// In production, you should restrict this to specific domains
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap ResponseWriter to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)

		// Color-coded status logging
		statusColor := ""
		switch {
		case wrapped.statusCode >= 500:
			statusColor = "\033[31m" // Red
		case wrapped.statusCode >= 400:
			statusColor = "\033[33m" // Yellow
		case wrapped.statusCode >= 300:
			statusColor = "\033[36m" // Cyan
		case wrapped.statusCode >= 200:
			statusColor = "\033[32m" // Green
		default:
			statusColor = "\033[37m" // White
		}
		reset := "\033[0m"

		fmt.Printf("[%s] %s%s %s %d%s %v %s\n",
			start.Format("15:04:05"),
			statusColor,
			r.Method,
			r.URL.Path,
			wrapped.statusCode,
			reset,
			duration,
			r.RemoteAddr,
		)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// GetPortFromEnv gets the API server port from environment variable or returns default
func GetPortFromEnv(defaultPort string) string {
	if port := os.Getenv("API_PORT"); port != "" {
		// Validate port number
		if portNum, err := strconv.Atoi(port); err == nil && portNum > 0 && portNum <= 65535 {
			return port
		}
	}
	return defaultPort
}
