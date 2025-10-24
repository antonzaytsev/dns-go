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
	"dns-go/internal/elasticsearch"
	"dns-go/internal/metrics"
	"dns-go/internal/monitor"
	"dns-go/pkg/version"
)

// Server provides REST API endpoints for DNS server metrics
type Server struct {
	server     *http.Server
	metrics    *metrics.Metrics
	logMonitor *monitor.LogMonitor
	esClient   *elasticsearch.Client
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

	// Initialize Elasticsearch client if host and port are provided
	var esClient *elasticsearch.Client
	esHost := os.Getenv("ELASTICSEARCH_HOST")
	esPort := os.Getenv("ELASTICSEARCH_PORT")

	if esHost != "" || esPort != "" {
		esIndex := os.Getenv("ELASTICSEARCH_INDEX")
		if esIndex == "" {
			esIndex = "dns-logs"
		}

		esCfg := elasticsearch.Config{
			Host:  esHost,
			Port:  esPort,
			Index: esIndex,
		}

		if client, err := elasticsearch.NewClient(esCfg); err == nil {
			esClient = client
			fmt.Println("✅ Elasticsearch client initialized successfully")
		} else {
			fmt.Printf("⚠️  Warning: Failed to initialize Elasticsearch client: %v\n", err)
			fmt.Println("📝 Falling back to file-based log search")
		}
	} else {
		// Fallback to URL for backward compatibility
		if esURL := os.Getenv("ELASTICSEARCH_URL"); esURL != "" {
			esIndex := os.Getenv("ELASTICSEARCH_INDEX")
			if esIndex == "" {
				esIndex = "dns-logs"
			}

			esCfg := elasticsearch.Config{
				URL:   esURL,
				Index: esIndex,
			}

			if client, err := elasticsearch.NewClient(esCfg); err == nil {
				esClient = client
				fmt.Println("✅ Elasticsearch client initialized successfully")
			} else {
				fmt.Printf("⚠️  Warning: Failed to initialize Elasticsearch client: %v\n", err)
				fmt.Println("📝 Falling back to file-based log search")
			}
		} else {
			fmt.Println("📝 No Elasticsearch host/port or URL provided, using file-based log search")
		}
	}

	s := &Server{
		metrics:    metricsCollector,
		logMonitor: logMonitor,
		esClient:   esClient,
		config:     cfg.DNSConfig,
		port:       cfg.Port,
	}

	// Setup HTTP routes
	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("/api/metrics", s.handleMetrics)
	mux.HandleFunc("/api/search", s.handleSearch)
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/version", s.handleVersion)
	mux.HandleFunc("/api/dns-mappings", s.handleDNSMappings)

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
	fmt.Printf("  🔎 GET /api/search       - Search through DNS logs\n")
	fmt.Printf("  ❤️  GET /api/health       - Health check endpoint\n")
	fmt.Printf("  ℹ️  GET /api/version      - Version and build information\n")
	fmt.Printf("  🌐 GET/PUT/POST/DELETE /api/dns-mappings - Manage custom DNS mappings\n")
	fmt.Printf("\n🌐 Access URLs:\n")
	fmt.Printf("  Local:    http://localhost:%s/api\n", s.port)
	fmt.Printf("  Network:  http://0.0.0.0:%s/api\n", s.port)
	fmt.Printf("\n📊 Log search: %s\n", func() string {
		if s.esClient != nil {
			return "✅ Elasticsearch"
		} else if s.logMonitor != nil {
			return "📁 File-based"
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

	// Close Elasticsearch client
	if s.esClient != nil {
		s.esClient.Close()
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

	dashboardMetrics := s.metrics.GetDashboardMetrics(version.Get().Short())

	if err := json.NewEncoder(w).Encode(dashboardMetrics); err != nil {
		http.Error(w, "Failed to encode metrics", http.StatusInternalServerError)
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

	// Only use Elasticsearch for search - no file fallback
	if s.esClient == nil {
		http.Error(w, "Search service unavailable: Elasticsearch not connected", http.StatusServiceUnavailable)
		return
	}

	searchResult, err := s.esClient.SearchLogs(searchTerm, limit, offset, since)
	if err != nil {
		fmt.Printf("Elasticsearch search failed: %v\n", err)
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
		"source":  "elasticsearch",
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

func (s *Server) handleDNSMappings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// If no DNS config is available, return an error
	if s.config == nil {
		http.Error(w, "DNS configuration not available", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// Return current DNS mappings
		mappings := s.config.GetCustomDNS()
		response := map[string]interface{}{
			"mappings": mappings,
			"count":    len(mappings),
		}
		json.NewEncoder(w).Encode(response)

	case http.MethodPut:
		// Replace all DNS mappings
		var requestBody struct {
			Mappings map[string]string `json:"mappings"`
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

		// Validate and normalize mappings
		normalizedMappings := make(map[string]string)
		for domain, ip := range requestBody.Mappings {
			domain = strings.TrimSpace(domain)
			ip = strings.TrimSpace(ip)

			if domain == "" || ip == "" {
				http.Error(w, "Domain and IP cannot be empty", http.StatusBadRequest)
				return
			}

			// Ensure domain ends with a dot for DNS processing
			if !strings.HasSuffix(domain, ".") {
				domain += "."
			}

			normalizedMappings[domain] = ip
		}

		// Save to config file
		if err := s.saveDNSMappings(normalizedMappings); err != nil {
			http.Error(w, "Failed to save DNS mappings: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Update in-memory config
		s.config.CustomDNS = normalizedMappings

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "DNS mappings updated successfully",
			"count":   len(normalizedMappings),
		})

	case http.MethodPost:
		// Add or update a single DNS mapping
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

		// Get current mappings and add/update the new one
		currentMappings := s.config.GetCustomDNS()
		currentMappings[domain] = ip

		// Save to config file
		if err := s.saveDNSMappings(currentMappings); err != nil {
			http.Error(w, "Failed to save DNS mappings: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Update in-memory config
		s.config.CustomDNS = currentMappings

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "DNS mapping added successfully",
			"domain":  domain,
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

		// Get current mappings and remove the specified domain
		currentMappings := s.config.GetCustomDNS()
		if _, exists := currentMappings[domain]; !exists {
			http.Error(w, "Domain mapping not found", http.StatusNotFound)
			return
		}

		delete(currentMappings, domain)

		// Save to config file
		if err := s.saveDNSMappings(currentMappings); err != nil {
			http.Error(w, "Failed to save DNS mappings: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Update in-memory config
		s.config.CustomDNS = currentMappings

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "DNS mapping deleted successfully",
			"domain":  domain,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// saveDNSMappings saves the DNS mappings to the custom-dns.json file
func (s *Server) saveDNSMappings(mappings map[string]string) error {
	// Create the structure that matches the JSON file format
	configData := struct {
		Mappings map[string]string `json:"mappings"`
	}{
		Mappings: make(map[string]string),
	}

	// Remove trailing dots from domains for file storage (user-friendly format)
	for domain, ip := range mappings {
		displayDomain := strings.TrimSuffix(domain, ".")
		configData.Mappings[displayDomain] = ip
	}

	// Convert to JSON
	jsonData, err := json.MarshalIndent(configData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Write to the custom-dns.json file
	configFile := "custom-dns.json"
	if err := os.WriteFile(configFile, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
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
