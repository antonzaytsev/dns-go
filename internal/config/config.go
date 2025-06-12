// Package config provides configuration management for the DNS proxy server.
// It handles command-line flag parsing, validation, and default values.
package config

import (
	"flag"
	"fmt"
	"strings"
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
)

var (
	// defaultUpstreamDNS contains the default DNS servers to use
	defaultUpstreamDNS = []string{"8.8.8.8:53", "1.1.1.1:53"}
)

// Config holds the DNS server configuration
type Config struct {
	ListenAddress       string        `json:"listen_address"`
	Port                string        `json:"port"`
	UpstreamDNS         []string      `json:"upstream_dns"`
	LogFile             string        `json:"log_file,omitempty"`
	LogLevel            string        `json:"log_level"`
	CacheTTL            time.Duration `json:"cache_ttl"`
	CacheSize           int           `json:"cache_size"`
	MaxConcurrent       int           `json:"max_concurrent"`
	Timeout             time.Duration `json:"timeout"`
	RetryAttempts       int           `json:"retry_attempts"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		ListenAddress:       defaultListenAddress,
		Port:                defaultPort,
		UpstreamDNS:         append([]string(nil), defaultUpstreamDNS...), // Copy slice
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

// String returns a string representation of the configuration (excluding sensitive data).
func (c *Config) String() string {
	return fmt.Sprintf("Config{Listen: %s:%s, Upstreams: %v, LogLevel: %s, CacheSize: %d}",
		c.ListenAddress, c.Port, c.UpstreamDNS, c.LogLevel, c.CacheSize)
}
