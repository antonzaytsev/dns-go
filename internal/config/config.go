// Package config provides configuration management for the DNS proxy server.
// It handles command-line flag parsing, validation, and default values.
package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	// Default configuration values
	defaultListenAddress       = "0.0.0.0"
	defaultPort                = "53"
	defaultLogLevel            = "info"
	defaultCacheTTL            = 5 * time.Minute
	defaultCacheSize           = 10000
	defaultMaxConcurrent       = 100
	defaultTimeout             = 5 * time.Second
	defaultRetryAttempts       = 3
	defaultHealthCheckInterval = 30 * time.Second
	customDNSConfigFile        = "custom-dns.json"
)

var (
	// defaultUpstreamDNS contains the default DNS servers to use
	defaultUpstreamDNS = []string{"8.8.8.8:53", "1.1.1.1:53"}
)

// Config holds the DNS server configuration
type Config struct {
	ListenAddress       string            `json:"listen_address"`
	Port                string            `json:"port"`
	UpstreamDNS         []string          `json:"upstream_dns"`
	CustomDNS           map[string]string `json:"custom_dns,omitempty"`
	LogFile             string            `json:"log_file,omitempty"`
	LogLevel            string            `json:"log_level"`
	CacheTTL            time.Duration     `json:"cache_ttl"`
	CacheSize           int               `json:"cache_size"`
	MaxConcurrent       int               `json:"max_concurrent"`
	Timeout             time.Duration     `json:"timeout"`
	RetryAttempts       int               `json:"retry_attempts"`
	HealthCheckInterval time.Duration     `json:"health_check_interval"`

	// File watching for hot reload
	customDNSPath    string
	customDNSModTime time.Time
	mutex            sync.RWMutex
}

// CustomDNSConfig represents the structure of the custom DNS configuration file
type CustomDNSConfig struct {
	Mappings map[string]string `json:"mappings"`
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		ListenAddress:       defaultListenAddress,
		Port:                defaultPort,
		UpstreamDNS:         append([]string(nil), defaultUpstreamDNS...), // Copy slice
		CustomDNS:           make(map[string]string),
		LogLevel:            defaultLogLevel,
		CacheTTL:            defaultCacheTTL,
		CacheSize:           defaultCacheSize,
		MaxConcurrent:       defaultMaxConcurrent,
		Timeout:             defaultTimeout,
		RetryAttempts:       defaultRetryAttempts,
		HealthCheckInterval: defaultHealthCheckInterval,
	}
}

// LoadFromFlags parses command line flags and returns configuration.
// It returns an error if the configuration is invalid.
func LoadFromFlags() (*Config, error) {
	cfg := DefaultConfig()

	listenAddr := flag.String("listen", cfg.ListenAddress, "Listen address")
	port := flag.String("port", cfg.Port, "Listen port")
	upstreams := flag.String("upstreams", strings.Join(cfg.UpstreamDNS, ","), "Comma-separated list of upstream DNS servers")
	customDNS := flag.String("custom-dns", "", "Custom DNS mappings in format: domain1=ip1,domain2=ip2 (e.g., server.local=192.168.0.30)")
	logFile := flag.String("log", cfg.LogFile, "Log file path (optional)")
	logLevel := flag.String("log-level", cfg.LogLevel, "Log level (debug, info, warn, error)")
	cacheTTL := flag.Duration("cache-ttl", cfg.CacheTTL, "DNS cache TTL")
	cacheSize := flag.Int("cache-size", cfg.CacheSize, "DNS cache size")
	maxConcurrent := flag.Int("max-concurrent", cfg.MaxConcurrent, "Maximum concurrent requests")
	timeout := flag.Duration("timeout", cfg.Timeout, "Upstream server timeout")
	retryAttempts := flag.Int("retry-attempts", cfg.RetryAttempts, "Number of retry attempts")

	flag.Parse()

	cfg.ListenAddress = strings.TrimSpace(*listenAddr)
	cfg.Port = strings.TrimSpace(*port)
	cfg.LogFile = strings.TrimSpace(*logFile)
	cfg.LogLevel = strings.ToLower(strings.TrimSpace(*logLevel))
	cfg.CacheTTL = *cacheTTL
	cfg.CacheSize = *cacheSize
	cfg.MaxConcurrent = *maxConcurrent
	cfg.Timeout = *timeout
	cfg.RetryAttempts = *retryAttempts

	// Parse upstream servers
	if strings.TrimSpace(*upstreams) != "" {
		upstreamList := strings.Split(*upstreams, ",")
		cfg.UpstreamDNS = make([]string, 0, len(upstreamList))
		for _, upstream := range upstreamList {
			if trimmed := strings.TrimSpace(upstream); trimmed != "" {
				cfg.UpstreamDNS = append(cfg.UpstreamDNS, trimmed)
			}
		}
	}

	// Parse custom DNS mappings
	if strings.TrimSpace(*customDNS) != "" {
		mappingList := strings.Split(*customDNS, ",")
		cfg.CustomDNS = make(map[string]string)
		for _, mapping := range mappingList {
			mapping = strings.TrimSpace(mapping)
			if mapping == "" {
				continue
			}
			parts := strings.SplitN(mapping, "=", 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid custom DNS mapping format: %s (expected domain=ip)", mapping)
			}
			domain := strings.TrimSpace(parts[0])
			ip := strings.TrimSpace(parts[1])
			if domain == "" || ip == "" {
				return nil, fmt.Errorf("invalid custom DNS mapping format: %s (domain and IP cannot be empty)", mapping)
			}
			// Ensure domain ends with a dot for DNS processing
			if !strings.HasSuffix(domain, ".") {
				domain += "."
			}
			cfg.CustomDNS[domain] = ip
		}
	}

	// Load custom DNS mappings from file if it exists
	if err := cfg.loadCustomDNS(); err != nil {
		return nil, fmt.Errorf("failed to load custom DNS configuration: %w", err)
	}

	return cfg, cfg.Validate()
}

// Validate checks the configuration for errors and returns an error if any are found.
func (c *Config) Validate() error {
	if c.Port == "" {
		return fmt.Errorf("port cannot be empty")
	}

	if len(c.UpstreamDNS) == 0 {
		return fmt.Errorf("at least one upstream DNS server must be specified")
	}

	if c.CacheSize <= 0 {
		return fmt.Errorf("cache size must be positive, got %d", c.CacheSize)
	}

	if c.MaxConcurrent <= 0 {
		return fmt.Errorf("max concurrent requests must be positive, got %d", c.MaxConcurrent)
	}

	if c.RetryAttempts < 0 {
		return fmt.Errorf("retry attempts must be non-negative, got %d", c.RetryAttempts)
	}

	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive, got %v", c.Timeout)
	}

	if c.CacheTTL <= 0 {
		return fmt.Errorf("cache TTL must be positive, got %v", c.CacheTTL)
	}

	// Validate log level
	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[c.LogLevel] {
		return fmt.Errorf("invalid log level %q, must be one of: debug, info, warn, error", c.LogLevel)
	}

	return nil
}

// loadCustomDNS loads custom DNS mappings from the configuration file if it exists
func (c *Config) loadCustomDNS() error {
	// Get the path to the custom DNS configuration file
	configPath := customDNSConfigFile

	// Check if running from a different directory, try to find the config file relative to executable
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Try to find config file in the same directory as the executable
		execPath, execErr := os.Executable()
		if execErr == nil {
			configPath = filepath.Join(filepath.Dir(execPath), customDNSConfigFile)
		}
	}

	// Store the resolved path for hot reload
	c.customDNSPath = configPath

	// Check if the config file exists
	fileInfo, err := os.Stat(configPath)
	if os.IsNotExist(err) {
		// File doesn't exist, which is fine - custom DNS feature is disabled
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to stat custom DNS config file %s: %w", configPath, err)
	}

	// Store modification time for hot reload tracking
	c.customDNSModTime = fileInfo.ModTime()

	// Read the configuration file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read custom DNS config file %s: %w", configPath, err)
	}

	// Parse the JSON configuration
	var customDNSConfig CustomDNSConfig
	if err := json.Unmarshal(data, &customDNSConfig); err != nil {
		return fmt.Errorf("failed to parse custom DNS config file %s: %w", configPath, err)
	}

	// Initialize CustomDNS map if it doesn't exist
	if c.CustomDNS == nil {
		c.CustomDNS = make(map[string]string)
	}

	// Process and normalize the mappings from the config file
	for domain, ip := range customDNSConfig.Mappings {
		domain = strings.TrimSpace(domain)
		ip = strings.TrimSpace(ip)

		if domain == "" || ip == "" {
			return fmt.Errorf("invalid custom DNS mapping in config file: empty domain or IP")
		}

		// Ensure domain ends with a dot for DNS processing
		if !strings.HasSuffix(domain, ".") {
			domain += "."
		}

		// Config file mappings take precedence over command line
		c.CustomDNS[domain] = ip
	}

	return nil
}

// String returns a string representation of the configuration (excluding sensitive data).
func (c *Config) String() string {
	return fmt.Sprintf("Config{Listen: %s:%s, Upstreams: %v, LogLevel: %s, CacheSize: %d}",
		c.ListenAddress, c.Port, c.UpstreamDNS, c.LogLevel, c.CacheSize)
}

// HasCustomDNSFileChanged checks if the custom DNS configuration file has been modified
func (c *Config) HasCustomDNSFileChanged() (bool, error) {
	c.mutex.RLock()
	configPath := c.customDNSPath
	lastModTime := c.customDNSModTime
	c.mutex.RUnlock()

	// If no path is stored, the file wasn't loaded initially
	if configPath == "" {
		return false, nil
	}

	// Check current modification time
	fileInfo, err := os.Stat(configPath)
	if os.IsNotExist(err) {
		// File was deleted - this is a change
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to stat custom DNS config file: %w", err)
	}

	// Compare modification times
	return fileInfo.ModTime().After(lastModTime), nil
}

// ReloadCustomDNS reloads the custom DNS configuration from file and returns the new mappings
func (c *Config) ReloadCustomDNS() (map[string]string, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// If no path is stored, nothing to reload
	if c.customDNSPath == "" {
		return nil, nil
	}

	// Check if file exists
	fileInfo, err := os.Stat(c.customDNSPath)
	if os.IsNotExist(err) {
		// File was deleted - clear mappings
		c.customDNSModTime = time.Time{}
		c.CustomDNS = make(map[string]string)
		return c.CustomDNS, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to stat custom DNS config file: %w", err)
	}

	// Update modification time
	c.customDNSModTime = fileInfo.ModTime()

	// Read the configuration file
	data, err := os.ReadFile(c.customDNSPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read custom DNS config file: %w", err)
	}

	// Parse the JSON configuration
	var customDNSConfig CustomDNSConfig
	if err := json.Unmarshal(data, &customDNSConfig); err != nil {
		return nil, fmt.Errorf("failed to parse custom DNS config file: %w", err)
	}

	// Create new mappings
	newMappings := make(map[string]string)

	// Process and normalize the mappings from the config file
	for domain, ip := range customDNSConfig.Mappings {
		domain = strings.TrimSpace(domain)
		ip = strings.TrimSpace(ip)

		if domain == "" || ip == "" {
			return nil, fmt.Errorf("invalid custom DNS mapping in config file: empty domain or IP")
		}

		// Ensure domain ends with a dot for DNS processing
		if !strings.HasSuffix(domain, ".") {
			domain += "."
		}

		newMappings[domain] = ip
	}

	// Update the config's custom DNS mappings
	c.CustomDNS = newMappings

	return newMappings, nil
}

// GetCustomDNS returns a thread-safe copy of the current custom DNS mappings
func (c *Config) GetCustomDNS() map[string]string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	mappings := make(map[string]string, len(c.CustomDNS))
	for domain, ip := range c.CustomDNS {
		mappings[domain] = ip
	}
	return mappings
}
