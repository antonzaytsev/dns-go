// Package main provides the DNS metrics API server.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"dns-go/internal/api"
	"dns-go/internal/config"
	"dns-go/pkg/version"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("API server failed: %v", err)
	}
}

func run() error {
	// Parse command line flags
	var (
		showVersion = flag.Bool("version", false, "Show version information and exit")
		showHelp    = flag.Bool("help", false, "Show help information and exit")
		port        = flag.String("port", "8080", "API server port")
		logFile     = flag.String("log-file", "", "Path to DNS server log file for historical data")
	)
	flag.Parse()

	// Handle version flag
	if *showVersion {
		fmt.Println(version.Get().String())
		return nil
	}

	// Handle help flag
	if *showHelp {
		fmt.Printf("DNS Server API - %s\n\n", version.Get().Short())
		fmt.Println("A REST API server for DNS proxy server metrics and monitoring.")
		fmt.Println("\nUsage:")
		flag.PrintDefaults()
		fmt.Println("\nEnvironment Variables:")
		fmt.Println("  API_PORT        API server port (default: 8080)")
		fmt.Println("  DNS_LOG_FILE    Path to DNS server log file")
		fmt.Println("\nAPI Endpoints:")
		fmt.Println("  GET /api/metrics  - DNS server metrics and statistics")
		fmt.Println("  GET /api/health   - Health check endpoint")
		fmt.Println("  GET /api/version  - Version information")
		return nil
	}

	// Get port from environment variable if not set via flag
	apiPort := api.GetPortFromEnv(*port)

	// Get log file path from environment if not set via flag
	logFilePath := *logFile
	if logFilePath == "" {
		if envLogFile := os.Getenv("DNS_LOG_FILE"); envLogFile != "" {
			logFilePath = envLogFile
		}
	}

	// Load DNS configuration to enable DNS mappings management (without flag parsing)
	dnsConfig := config.DefaultConfig()
	// Load custom DNS mappings from file without flag parsing
	if err := loadCustomDNSOnly(dnsConfig); err != nil {
		fmt.Printf("Warning: Could not load custom DNS mappings: %v\n", err)
	}

	// Create API server configuration
	apiConfig := api.Config{
		Port:        apiPort,
		LogFilePath: logFilePath,
		DNSConfig:   dnsConfig,
	}

	// Create API server
	server, err := api.NewServer(apiConfig)
	if err != nil {
		return fmt.Errorf("failed to create API server: %w", err)
	}

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		if err := server.Start(); err != nil {
			serverErr <- err
		}
	}()

	// Log startup information
	fmt.Printf("DNS API Server - %s\n", version.Get().String())
	fmt.Printf("Starting API server on port %s\n", apiPort)
	if logFilePath != "" {
		fmt.Printf("Loading historical data from: %s\n", logFilePath)
	}
	fmt.Printf("API URL: http://localhost:%s/api\n", apiPort)
	fmt.Println("Press Ctrl+C to stop...")

	// Wait for shutdown signal or server error
	select {
	case sig := <-sigChan:
		fmt.Printf("\nReceived signal: %s\n", sig)
	case err := <-serverErr:
		if err != nil {
			return fmt.Errorf("server error: %w", err)
		}
	}

	// Graceful shutdown
	fmt.Println("Shutting down API server...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown failed: %w", err)
	}

	fmt.Println("API server shutdown complete")
	return nil
}

// loadCustomDNSOnly loads custom DNS mappings from file without flag parsing
func loadCustomDNSOnly(cfg *config.Config) error {
	const customDNSConfigFile = "custom-dns.json"

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

	// Check if the config file exists
	_, err := os.Stat(configPath)
	if os.IsNotExist(err) {
		// File doesn't exist, which is fine - custom DNS feature is disabled
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to stat custom DNS config file %s: %w", configPath, err)
	}

	// Read the configuration file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read custom DNS config file %s: %w", configPath, err)
	}

	// Parse the JSON configuration
	var customDNSConfig struct {
		Mappings map[string]string `json:"mappings"`
	}
	if err := json.Unmarshal(data, &customDNSConfig); err != nil {
		return fmt.Errorf("failed to parse custom DNS config file %s: %w", configPath, err)
	}

	// Initialize CustomDNS map if it doesn't exist
	if cfg.CustomDNS == nil {
		cfg.CustomDNS = make(map[string]string)
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

		cfg.CustomDNS[domain] = ip
	}

	return nil
}
