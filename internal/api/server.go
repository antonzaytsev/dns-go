package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"dns-go/internal/metrics"
	"dns-go/internal/monitor"
	"dns-go/pkg/version"
)

// Server provides REST API endpoints for DNS server metrics
type Server struct {
	server     *http.Server
	metrics    *metrics.Metrics
	logMonitor *monitor.LogMonitor
	port       string
}

// Config holds API server configuration
type Config struct {
	Port        string
	LogFilePath string
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

	s := &Server{
		metrics:    metricsCollector,
		logMonitor: logMonitor,
		port:       cfg.Port,
	}

	// Setup HTTP routes
	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("/api/metrics", s.handleMetrics)
	mux.HandleFunc("/api/search", s.handleSearch)
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/version", s.handleVersion)

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
	fmt.Printf("  🔍 GET /api/metrics  - DNS server metrics and statistics\n")
	fmt.Printf("  🔎 GET /api/search   - Search through DNS logs\n")
	fmt.Printf("  ❤️  GET /api/health   - Health check endpoint\n")
	fmt.Printf("  ℹ️  GET /api/version  - Version and build information\n")
	fmt.Printf("\n🌐 Access URLs:\n")
	fmt.Printf("  Local:    http://localhost:%s/api\n", s.port)
	fmt.Printf("  Network:  http://0.0.0.0:%s/api\n", s.port)
	fmt.Printf("\n📊 Log monitoring: %s\n", func() string {
		if s.logMonitor != nil {
			return "✅ Active"
		}
		return "❌ No log file found"
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

	// Set defaults
	limit := 100
	offset := 0

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

	// If no log monitor, return empty results
	if s.logMonitor == nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"results": []interface{}{},
			"total":   0,
			"limit":   limit,
			"offset":  offset,
			"query":   searchTerm,
		})
		return
	}

	// Perform search
	results, total := s.logMonitor.SearchLogs(searchTerm, limit, offset)

	response := map[string]interface{}{
		"results": results,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
		"query":   searchTerm,
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
