package webserver

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"dns-go/internal/metrics"
	"dns-go/internal/monitor"
	"dns-go/pkg/version"
)

// WebServer provides HTTP endpoints for DNS server metrics and dashboard
type WebServer struct {
	server     *http.Server
	metrics    *metrics.Metrics
	logMonitor *monitor.LogMonitor
	port       string
}

// Config holds web server configuration
type Config struct {
	Port        string
	LogFilePath string
}

// NewWebServer creates a new web server instance
func NewWebServer(cfg Config) (*WebServer, error) {
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

	ws := &WebServer{
		metrics:    metricsCollector,
		logMonitor: logMonitor,
		port:       cfg.Port,
	}

	// Setup HTTP routes
	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("/api/metrics", ws.handleMetrics)
	mux.HandleFunc("/api/health", ws.handleHealth)

	// Dashboard UI
	mux.HandleFunc("/", ws.handleDashboard)
	mux.HandleFunc("/dashboard", ws.handleDashboard)

	// Static assets (embedded)
	mux.HandleFunc("/static/", ws.handleStatic)

	// WebSocket for real-time updates (future enhancement)
	// mux.HandleFunc("/ws", ws.handleWebSocket)

	ws.server = &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      ws.corsMiddleware(ws.loggingMiddleware(mux)),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return ws, nil
}

// Start starts the web server
func (ws *WebServer) Start() error {
	fmt.Printf("Starting web server on port %s\n", ws.port)
	fmt.Printf("Dashboard available at: http://localhost:%s\n", ws.port)
	fmt.Printf("API available at: http://localhost:%s/api/metrics\n", ws.port)

	return ws.server.ListenAndServe()
}

// Shutdown gracefully shuts down the web server
func (ws *WebServer) Shutdown(ctx context.Context) error {
	// Stop log monitor first
	if ws.logMonitor != nil {
		ws.logMonitor.Stop()
	}

	return ws.server.Shutdown(ctx)
}

// GetMetrics returns the metrics collector for external use
func (ws *WebServer) GetMetrics() *metrics.Metrics {
	return ws.metrics
}

// HTTP Handlers

func (ws *WebServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	dashboardMetrics := ws.metrics.GetDashboardMetrics(version.Get().Short())

	if err := json.NewEncoder(w).Encode(dashboardMetrics); err != nil {
		http.Error(w, "Failed to encode metrics", http.StatusInternalServerError)
		return
	}
}

func (ws *WebServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"version":   version.Get().Short(),
	}

	json.NewEncoder(w).Encode(health)
}

func (ws *WebServer) handleDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	tmpl := template.Must(template.New("dashboard").Parse(dashboardHTML))

	data := struct {
		Title   string
		Version string
		Port    string
	}{
		Title:   "DNS Server Dashboard",
		Version: version.Get().Short(),
		Port:    ws.port,
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Failed to render dashboard", http.StatusInternalServerError)
		return
	}
}

func (ws *WebServer) handleStatic(w http.ResponseWriter, r *http.Request) {
	// Extract the file path
	path := r.URL.Path[len("/static/"):]

	switch filepath.Ext(path) {
	case ".css":
		w.Header().Set("Content-Type", "text/css")
		if path == "dashboard.css" {
			w.Write([]byte(dashboardCSS))
		}
	case ".js":
		w.Header().Set("Content-Type", "application/javascript")
		if path == "dashboard.js" {
			w.Write([]byte(dashboardJS))
		}
	default:
		http.NotFound(w, r)
	}
}

// Middleware

func (ws *WebServer) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (ws *WebServer) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap ResponseWriter to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)
		fmt.Printf("[%s] %s %s %d %v\n",
			start.Format("2006-01-02 15:04:05"),
			r.Method,
			r.URL.Path,
			wrapped.statusCode,
			duration,
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

// GetPortFromEnv gets the web server port from environment variable or returns default
func GetPortFromEnv(defaultPort string) string {
	if port := os.Getenv("WEB_PORT"); port != "" {
		// Validate port number
		if portNum, err := strconv.Atoi(port); err == nil && portNum > 0 && portNum <= 65535 {
			return port
		}
	}
	return defaultPort
}
