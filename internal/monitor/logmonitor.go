package monitor

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"dns-go/internal/metrics"
	"dns-go/internal/types"
)

// LogMonitor watches DNS log files and feeds metrics to the collector
type LogMonitor struct {
	logFilePath string
	metrics     *metrics.Metrics
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewLogMonitor creates a new log monitor
func NewLogMonitor(logFilePath string, metricsCollector *metrics.Metrics) *LogMonitor {
	ctx, cancel := context.WithCancel(context.Background())

	return &LogMonitor{
		logFilePath: logFilePath,
		metrics:     metricsCollector,
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start begins monitoring the log file
func (lm *LogMonitor) Start() error {
	if lm.logFilePath == "" {
		return fmt.Errorf("log file path is empty")
	}

	// Load existing data first
	if err := lm.loadExistingData(); err != nil {
		fmt.Printf("Warning: Could not load existing log data: %v\n", err)
	}

	// Start watching for new entries
	go lm.watchLogFile()

	fmt.Printf("📊 Log Monitor Started\n")
	fmt.Printf("  File: %s\n", lm.logFilePath)
	fmt.Printf("  Status: ✅ Monitoring active\n")
	return nil
}

// Stop stops the log monitor
func (lm *LogMonitor) Stop() {
	lm.cancel()
}

// loadExistingData loads historical data from the log file
func (lm *LogMonitor) loadExistingData() error {
	file, err := os.Open(lm.logFilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	count := 0

	// Only load entries from the last 24 hours to avoid overwhelming memory
	cutoff := time.Now().Add(-24 * time.Hour)

	for scanner.Scan() {
		var entry types.LogEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue // Skip invalid JSON lines
		}

		// Only process recent entries
		if entry.Timestamp.After(cutoff) {
			lm.metrics.RecordRequest(entry)
			count++
		}
	}

	if count > 0 {
		fmt.Printf("📈 Loaded %d historical log entries from last 24h\n", count)
	} else {
		fmt.Printf("📝 No recent log entries found (last 24h)\n")
	}

	return scanner.Err()
}

// watchLogFile continuously monitors the log file for new entries
func (lm *LogMonitor) watchLogFile() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var lastSize int64
	var file *os.File

	for {
		select {
		case <-lm.ctx.Done():
			if file != nil {
				file.Close()
			}
			return
		case <-ticker.C:
			if err := lm.checkForNewEntries(&file, &lastSize); err != nil {
				fmt.Printf("Error monitoring log file: %v\n", err)
				// Continue monitoring despite errors
			}
		}
	}
}

// checkForNewEntries checks if the log file has new entries and processes them
func (lm *LogMonitor) checkForNewEntries(file **os.File, lastSize *int64) error {
	stat, err := os.Stat(lm.logFilePath)
	if err != nil {
		// File might not exist yet or be temporarily unavailable
		if *file != nil {
			(*file).Close()
			*file = nil
		}
		*lastSize = 0
		return nil
	}

	currentSize := stat.Size()

	// If file is smaller than before, it might have been rotated
	if currentSize < *lastSize {
		if *file != nil {
			(*file).Close()
			*file = nil
		}
		*lastSize = 0
	}

	// If no new data, return
	if currentSize == *lastSize {
		return nil
	}

	// Open file if not already open
	if *file == nil {
		f, err := os.Open(lm.logFilePath)
		if err != nil {
			return err
		}
		*file = f

		// Seek to the last known position
		if *lastSize > 0 {
			(*file).Seek(*lastSize, 0)
		}
	}

	// Read new entries
	scanner := bufio.NewScanner(*file)
	for scanner.Scan() {
		var entry types.LogEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue // Skip invalid JSON lines
		}

		// Record the entry in metrics
		lm.metrics.RecordRequest(entry)
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	// Update last known size
	*lastSize = currentSize

	return nil
}

// GetLogFilePath returns the path to the log file being monitored
func (lm *LogMonitor) GetLogFilePath() string {
	return lm.logFilePath
}

// SearchLogs searches through all DNS logs for entries matching the search term
func (lm *LogMonitor) SearchLogs(searchTerm string, limit, offset int) ([]types.LogEntry, int) {
	if lm.logFilePath == "" {
		return []types.LogEntry{}, 0
	}

	file, err := os.Open(lm.logFilePath)
	if err != nil {
		fmt.Printf("Error opening log file for search: %v\n", err)
		return []types.LogEntry{}, 0
	}
	defer file.Close()

	var allMatches []types.LogEntry
	scanner := bufio.NewScanner(file)
	searchLower := strings.ToLower(searchTerm)

	// Read all lines and filter
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var entry types.LogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue // Skip malformed entries
		}

		// If no search term, include all entries
		if searchTerm == "" {
			allMatches = append(allMatches, entry)
			continue
		}

		// Search in various fields
		if lm.matchesSearchTerm(entry, searchLower) {
			allMatches = append(allMatches, entry)
		}
	}

	total := len(allMatches)

	// Apply pagination
	start := offset
	if start > total {
		start = total
	}

	end := start + limit
	if end > total {
		end = total
	}

	if start >= end {
		return []types.LogEntry{}, total
	}

	// Return results in reverse order (newest first)
	results := make([]types.LogEntry, 0, end-start)
	for i := total - 1 - start; i >= total-end; i-- {
		if i >= 0 && i < len(allMatches) {
			results = append(results, allMatches[i])
		}
	}

	return results, total
}

// matchesSearchTerm checks if a DNS log entry matches the search term
func (lm *LogMonitor) matchesSearchTerm(entry types.LogEntry, searchLower string) bool {
	// Search in query domain
	if strings.Contains(strings.ToLower(entry.Request.Query), searchLower) {
		return true
	}

	// Search in client IP
	if strings.Contains(strings.ToLower(entry.Request.Client), searchLower) {
		return true
	}

	// Search in query type
	if strings.Contains(strings.ToLower(entry.Request.Type), searchLower) {
		return true
	}

	// Search in status
	if strings.Contains(strings.ToLower(entry.Status), searchLower) {
		return true
	}

	// Search in upstream server
	if entry.Response != nil && strings.Contains(strings.ToLower(entry.Response.Upstream), searchLower) {
		return true
	}

	// Search in IP addresses
	for _, ip := range entry.IPAddresses {
		if strings.Contains(strings.ToLower(ip), searchLower) {
			return true
		}
	}

	// Search in UUID
	if strings.Contains(strings.ToLower(entry.UUID), searchLower) {
		return true
	}

	return false
}

// FindLogFile attempts to find the DNS log file in common locations
func FindLogFile() string {
	// Common log file locations
	locations := []string{
		"/logs/dns-requests.log",
		"./logs/dns-requests.log",
		"/var/log/dns-requests.log",
		"/tmp/dns-requests.log",
	}

	// Check environment variable first
	if envPath := os.Getenv("DNS_LOG_FILE"); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			return envPath
		}
	}

	// Check common locations
	for _, location := range locations {
		if _, err := os.Stat(location); err == nil {
			return location
		}
	}

	// Try to find any .log files in logs directory
	logsDir := "./logs"
	if entries, err := os.ReadDir(logsDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && filepath.Ext(entry.Name()) == ".log" {
				fullPath := filepath.Join(logsDir, entry.Name())
				if stat, err := os.Stat(fullPath); err == nil && stat.Size() > 0 {
					return fullPath
				}
			}
		}
	}

	return ""
}
