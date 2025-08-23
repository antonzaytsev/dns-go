package logging

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"dns-go/internal/elasticsearch"
	"dns-go/internal/types"
)

// LogLevel represents different logging levels
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

// String returns string representation of LogLevel
func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Logger provides structured logging with different levels and dual output
type Logger struct {
	mu          sync.Mutex
	level       LogLevel
	output      io.Writer
	jsonEncoder *json.Encoder
	humanLogger *log.Logger
	jsonFile    *os.File
	humanFile   *os.File
	esClient    *elasticsearch.Client
}

// New creates a new structured logger
func New(output io.Writer, level LogLevel) *Logger {
	logger := &Logger{
		level:       level,
		output:      output,
		jsonEncoder: json.NewEncoder(output),
	}
	return logger
}

// NewFromConfig creates a logger from configuration with dual file support
func NewFromConfig(logFile string, logLevel string) (*Logger, *os.File, *os.File, error) {
	level := parseLogLevel(logLevel)

	if logFile == "" {
		// Console only
		logger := New(os.Stdout, level)
		logger.humanLogger = log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds)
		return logger, nil, nil, nil
	}

	// Create log directory if it doesn't exist
	logDir := filepath.Dir(logFile)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, nil, nil, err
	}

	// Open JSON log file (for requests/responses only)
	jsonFile, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, nil, nil, err
	}

	// Open human-readable log file
	humanLogFile := logFile + ".human"
	if strings.HasSuffix(logFile, "dns-requests.log") {
		// For dns-requests.log, create dns-server.log for human-readable logs
		humanLogFile = strings.Replace(logFile, "dns-requests.log", "dns-server.log", 1)
	}
	humanFile, err := os.OpenFile(humanLogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		jsonFile.Close()
		return nil, nil, nil, err
	}

	// Create multi-writer for console output (human-readable only)
	humanMultiWriter := io.MultiWriter(os.Stdout, humanFile)

	logger := &Logger{
		level:       level,
		output:      jsonFile, // JSON goes only to file
		jsonEncoder: json.NewEncoder(jsonFile),
		humanLogger: log.New(humanMultiWriter, "", log.LstdFlags|log.Lmicroseconds),
		jsonFile:    jsonFile,
		humanFile:   humanFile,
	}

	// Try to initialize Elasticsearch client with retry logic
	if esURL := os.Getenv("ELASTICSEARCH_URL"); esURL != "" {
		esIndex := os.Getenv("ELASTICSEARCH_INDEX")
		if esIndex == "" {
			esIndex = "dns-logs"
		}

		cfg := elasticsearch.Config{
			URL:   esURL,
			Index: esIndex,
		}

		// Retry connecting to Elasticsearch with exponential backoff
		maxRetries := 5
		for i := 0; i < maxRetries; i++ {
			if esClient, err := elasticsearch.NewClient(cfg); err == nil {
				logger.esClient = esClient
				if logger.humanLogger != nil {
					logger.humanLogger.Printf("✅ DNS server Elasticsearch client initialized successfully")
				} else {
					log.Printf("✅ DNS server Elasticsearch client initialized successfully")
				}
				break
			} else {
				if i < maxRetries-1 {
					waitTime := time.Duration(1<<uint(i)) * time.Second // 1s, 2s, 4s, 8s
					if logger.humanLogger != nil {
						logger.humanLogger.Printf("⏳ DNS server Elasticsearch connection attempt %d/%d failed: %v. Retrying in %v...",
							i+1, maxRetries, err, waitTime)
					} else {
						log.Printf("⏳ DNS server Elasticsearch connection attempt %d/%d failed: %v. Retrying in %v...",
							i+1, maxRetries, err, waitTime)
					}
					time.Sleep(waitTime)
				} else {
					// Log ES initialization failure but don't fail the logger
					if logger.humanLogger != nil {
						logger.humanLogger.Printf("⚠️  Warning: DNS server failed to initialize Elasticsearch client after %d attempts: %v", maxRetries, err)
					} else {
						log.Printf("⚠️  Warning: DNS server failed to initialize Elasticsearch client after %d attempts: %v", maxRetries, err)
					}
				}
			}
		}
	}

	return logger, jsonFile, humanFile, nil
}

// parseLogLevel converts string to LogLevel
func parseLogLevel(level string) LogLevel {
	switch level {
	case "debug":
		return DEBUG
	case "info":
		return INFO
	case "warn":
		return WARN
	case "error":
		return ERROR
	default:
		return INFO
	}
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// shouldLog checks if the message should be logged at the current level
func (l *Logger) shouldLog(level LogLevel) bool {
	return level >= l.level
}

// log writes a log entry at the specified level (human-readable only)
func (l *Logger) log(level LogLevel, message string, fields map[string]interface{}) {
	if !l.shouldLog(level) {
		return
	}

	// Format human-readable log message
	msg := fmt.Sprintf("[%s] %s", level.String(), message)
	if len(fields) > 0 {
		for k, v := range fields {
			msg += fmt.Sprintf(" %s=%v", k, v)
		}
	}

	if l.humanLogger != nil {
		l.humanLogger.Println(msg)
	} else {
		// Fallback to standard logging
		log.Printf("[%s] %s", level.String(), message)
	}
}

// Debug logs at DEBUG level
func (l *Logger) Debug(message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(DEBUG, message, f)
}

// Info logs at INFO level
func (l *Logger) Info(message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(INFO, message, f)
}

// Warn logs at WARN level
func (l *Logger) Warn(message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(WARN, message, f)
}

// Error logs at ERROR level
func (l *Logger) Error(message string, fields ...map[string]interface{}) {
	var f map[string]interface{}
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(ERROR, message, f)
}

// LogJSON logs arbitrary JSON data (for DNS requests/responses only)
func (l *Logger) LogJSON(data interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if err := l.jsonEncoder.Encode(data); err != nil {
		if l.humanLogger != nil {
			l.humanLogger.Printf("JSON logging error: %v", err)
		} else {
			log.Printf("JSON logging error: %v", err)
		}
	}
}

// LogRequestResponse logs a human-readable version of the DNS request/response
func (l *Logger) LogRequestResponse(uuid, client, query, qtype, status string, duration float64, cacheHit bool, upstream string) {
	var msg string
	if cacheHit {
		msg = fmt.Sprintf("REQ %s from %s: %s %s -> CACHE HIT (%.2fms)",
			uuid, client, qtype, query, duration)
	} else {
		msg = fmt.Sprintf("REQ %s from %s: %s %s -> %s via %s (%.2fms)",
			uuid, client, qtype, query, status, upstream, duration)
	}

	if l.humanLogger != nil {
		l.humanLogger.Println(msg)
	} else {
		log.Println(msg)
	}
}

// LogDNSEntry logs a complete DNS log entry to both file and Elasticsearch
func (l *Logger) LogDNSEntry(entry types.LogEntry) {
	// Log to JSON file
	l.LogJSON(entry)

	// Log to Elasticsearch if available
	if l.esClient != nil {
		go func() {
			if err := l.esClient.IndexLogEntry(entry); err != nil {
				// Log error but don't block the main logging flow
				if l.humanLogger != nil {
					l.humanLogger.Printf("Warning: Failed to index log entry to Elasticsearch: %v", err)
				} else {
					log.Printf("Warning: Failed to index log entry to Elasticsearch: %v", err)
				}
			}
		}()
	}
}
