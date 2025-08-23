// Package main provides the web dashboard server for DNS proxy monitoring.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"dns-go/internal/webserver"
	"dns-go/pkg/version"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Web dashboard failed: %v", err)
	}
}

func run() error {
	// Parse command line flags
	var (
		showVersion = flag.Bool("version", false, "Show version information and exit")
		showHelp    = flag.Bool("help", false, "Show help information and exit")
		port        = flag.String("port", "8080", "Web server port")
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
		fmt.Printf("DNS Server Web Dashboard - %s\n\n", version.Get().Short())
		fmt.Println("A web dashboard for monitoring DNS proxy server metrics and performance.")
		fmt.Println("\nUsage:")
		flag.PrintDefaults()
		fmt.Println("\nEnvironment Variables:")
		fmt.Println("  WEB_PORT        Web server port (default: 8080)")
		fmt.Println("  DNS_LOG_FILE    Path to DNS server log file")
		return nil
	}

	// Get port from environment variable if not set via flag
	webPort := webserver.GetPortFromEnv(*port)

	// Get log file path from environment if not set via flag
	logFilePath := *logFile
	if logFilePath == "" {
		if envLogFile := os.Getenv("DNS_LOG_FILE"); envLogFile != "" {
			logFilePath = envLogFile
		}
	}

	// Create web server configuration
	config := webserver.Config{
		Port:        webPort,
		LogFilePath: logFilePath,
	}

	// Create web server
	server, err := webserver.NewWebServer(config)
	if err != nil {
		return fmt.Errorf("failed to create web server: %w", err)
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
	fmt.Printf("DNS Web Dashboard - %s\n", version.Get().String())
	fmt.Printf("Starting web server on port %s\n", webPort)
	if logFilePath != "" {
		fmt.Printf("Loading historical data from: %s\n", logFilePath)
	}
	fmt.Printf("Dashboard URL: http://localhost:%s\n", webPort)
	fmt.Printf("API URL: http://localhost:%s/api/metrics\n", webPort)
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
	fmt.Println("Shutting down web server...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown failed: %w", err)
	}

	fmt.Println("Web server shutdown complete")
	return nil
}
