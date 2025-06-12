package config

import (
	"flag"
	"fmt"
	"strings"
	"time"
)

// Config holds the DNS server configuration
type Config struct {
	ListenAddress       string
	Port                string
	UpstreamDNS         []string
	LogFile             string
	LogLevel            string
	CacheTTL            time.Duration
	CacheSize           int
	MaxConcurrent       int
	Timeout             time.Duration
	RetryAttempts       int
	HealthCheckInterval time.Duration
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		ListenAddress:       "0.0.0.0",
		Port:                "53",
		UpstreamDNS:         []string{"8.8.8.8:53", "1.1.1.1:53"},
		LogLevel:            "info",
		CacheTTL:            5 * time.Minute,
		CacheSize:           10000,
		MaxConcurrent:       100,
		Timeout:             5 * time.Second,
		RetryAttempts:       3,
		HealthCheckInterval: 30 * time.Second,
	}
}

// LoadFromFlags parses command line flags and returns configuration
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

	cfg.ListenAddress = *listenAddr
	cfg.Port = *port
	cfg.LogFile = *logFile
	cfg.LogLevel = *logLevel
	cfg.CacheTTL = *cacheTTL
	cfg.CacheSize = *cacheSize
	cfg.MaxConcurrent = *maxConcurrent
	cfg.Timeout = *timeout
	cfg.RetryAttempts = *retryAttempts

	// Parse upstream servers
	if *upstreams != "" {
		upstreamList := strings.Split(*upstreams, ",")
		cfg.UpstreamDNS = make([]string, len(upstreamList))
		for i, upstream := range upstreamList {
			cfg.UpstreamDNS[i] = strings.TrimSpace(upstream)
		}
	}

	return cfg, cfg.Validate()
}

// Validate checks the configuration for errors
func (c *Config) Validate() error {
	if c.Port == "" {
		return fmt.Errorf("port cannot be empty")
	}
	if len(c.UpstreamDNS) == 0 {
		return fmt.Errorf("at least one upstream DNS server must be specified")
	}
	if c.CacheSize <= 0 {
		return fmt.Errorf("cache size must be positive")
	}
	if c.MaxConcurrent <= 0 {
		return fmt.Errorf("max concurrent requests must be positive")
	}
	return nil
}
