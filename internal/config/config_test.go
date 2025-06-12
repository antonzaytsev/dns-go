package config

import (
	"flag"
	"os"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.ListenAddress != defaultListenAddress {
		t.Errorf("Expected ListenAddress %s, got %s", defaultListenAddress, cfg.ListenAddress)
	}

	if cfg.Port != defaultPort {
		t.Errorf("Expected Port %s, got %s", defaultPort, cfg.Port)
	}

	if len(cfg.UpstreamDNS) != len(defaultUpstreamDNS) {
		t.Errorf("Expected %d upstream DNS servers, got %d", len(defaultUpstreamDNS), len(cfg.UpstreamDNS))
	}

	if cfg.LogLevel != defaultLogLevel {
		t.Errorf("Expected LogLevel %s, got %s", defaultLogLevel, cfg.LogLevel)
	}

	if cfg.CacheTTL != defaultCacheTTL {
		t.Errorf("Expected CacheTTL %v, got %v", defaultCacheTTL, cfg.CacheTTL)
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid config",
			config:  DefaultConfig(),
			wantErr: false,
		},
		{
			name: "empty port",
			config: &Config{
				Port:          "",
				UpstreamDNS:   []string{"8.8.8.8:53"},
				CacheSize:     1000,
				MaxConcurrent: 100,
				RetryAttempts: 3,
				Timeout:       5 * time.Second,
				CacheTTL:      5 * time.Minute,
				LogLevel:      "info",
			},
			wantErr: true,
			errMsg:  "port cannot be empty",
		},
		{
			name: "no upstream DNS servers",
			config: &Config{
				Port:          "53",
				UpstreamDNS:   []string{},
				CacheSize:     1000,
				MaxConcurrent: 100,
				RetryAttempts: 3,
				Timeout:       5 * time.Second,
				CacheTTL:      5 * time.Minute,
				LogLevel:      "info",
			},
			wantErr: true,
			errMsg:  "at least one upstream DNS server must be specified",
		},
		{
			name: "negative cache size",
			config: &Config{
				Port:          "53",
				UpstreamDNS:   []string{"8.8.8.8:53"},
				CacheSize:     -1,
				MaxConcurrent: 100,
				RetryAttempts: 3,
				Timeout:       5 * time.Second,
				CacheTTL:      5 * time.Minute,
				LogLevel:      "info",
			},
			wantErr: true,
			errMsg:  "cache size must be positive",
		},
		{
			name: "zero max concurrent",
			config: &Config{
				Port:          "53",
				UpstreamDNS:   []string{"8.8.8.8:53"},
				CacheSize:     1000,
				MaxConcurrent: 0,
				RetryAttempts: 3,
				Timeout:       5 * time.Second,
				CacheTTL:      5 * time.Minute,
				LogLevel:      "info",
			},
			wantErr: true,
			errMsg:  "max concurrent requests must be positive",
		},
		{
			name: "negative retry attempts",
			config: &Config{
				Port:          "53",
				UpstreamDNS:   []string{"8.8.8.8:53"},
				CacheSize:     1000,
				MaxConcurrent: 100,
				RetryAttempts: -1,
				Timeout:       5 * time.Second,
				CacheTTL:      5 * time.Minute,
				LogLevel:      "info",
			},
			wantErr: true,
			errMsg:  "retry attempts must be non-negative",
		},
		{
			name: "zero timeout",
			config: &Config{
				Port:          "53",
				UpstreamDNS:   []string{"8.8.8.8:53"},
				CacheSize:     1000,
				MaxConcurrent: 100,
				RetryAttempts: 3,
				Timeout:       0,
				CacheTTL:      5 * time.Minute,
				LogLevel:      "info",
			},
			wantErr: true,
			errMsg:  "timeout must be positive",
		},
		{
			name: "invalid log level",
			config: &Config{
				Port:          "53",
				UpstreamDNS:   []string{"8.8.8.8:53"},
				CacheSize:     1000,
				MaxConcurrent: 100,
				RetryAttempts: 3,
				Timeout:       5 * time.Second,
				CacheTTL:      5 * time.Minute,
				LogLevel:      "invalid",
			},
			wantErr: true,
			errMsg:  "invalid log level",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.errMsg != "" && !containsString(err.Error(), tt.errMsg) {
					t.Errorf("Expected error message to contain %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestConfig_String(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ListenAddress = "127.0.0.1"
	cfg.Port = "5353"
	cfg.LogLevel = "debug"

	str := cfg.String()
	expectedParts := []string{
		"127.0.0.1:5353",
		"debug",
		"8.8.8.8:53",
		"1.1.1.1:53",
	}

	for _, part := range expectedParts {
		if !containsString(str, part) {
			t.Errorf("Expected string representation to contain %q, got %q", part, str)
		}
	}
}

func TestLoadFromFlags_MockFlags(t *testing.T) {
	// Save original command line args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Reset flag package state
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// Mock command line arguments
	os.Args = []string{
		"test",
		"-listen=127.0.0.1",
		"-port=5353",
		"-upstreams=1.1.1.1:53,9.9.9.9:53",
		"-log-level=debug",
		"-cache-size=5000",
		"-max-concurrent=50",
	}

	cfg, err := LoadFromFlags()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if cfg.ListenAddress != "127.0.0.1" {
		t.Errorf("Expected ListenAddress 127.0.0.1, got %s", cfg.ListenAddress)
	}

	if cfg.Port != "5353" {
		t.Errorf("Expected Port 5353, got %s", cfg.Port)
	}

	expectedUpstreams := []string{"1.1.1.1:53", "9.9.9.9:53"}
	if len(cfg.UpstreamDNS) != len(expectedUpstreams) {
		t.Errorf("Expected %d upstreams, got %d", len(expectedUpstreams), len(cfg.UpstreamDNS))
	}

	if cfg.LogLevel != "debug" {
		t.Errorf("Expected LogLevel debug, got %s", cfg.LogLevel)
	}

	if cfg.CacheSize != 5000 {
		t.Errorf("Expected CacheSize 5000, got %d", cfg.CacheSize)
	}

	if cfg.MaxConcurrent != 50 {
		t.Errorf("Expected MaxConcurrent 50, got %d", cfg.MaxConcurrent)
	}
}

func TestUpstreamParsing(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single upstream",
			input:    "8.8.8.8:53",
			expected: []string{"8.8.8.8:53"},
		},
		{
			name:     "multiple upstreams",
			input:    "8.8.8.8:53,1.1.1.1:53,9.9.9.9:53",
			expected: []string{"8.8.8.8:53", "1.1.1.1:53", "9.9.9.9:53"},
		},
		{
			name:     "upstreams with spaces",
			input:    " 8.8.8.8:53 , 1.1.1.1:53 , 9.9.9.9:53 ",
			expected: []string{"8.8.8.8:53", "1.1.1.1:53", "9.9.9.9:53"},
		},
		{
			name:     "empty entries filtered",
			input:    "8.8.8.8:53,,1.1.1.1:53,  ,9.9.9.9:53",
			expected: []string{"8.8.8.8:53", "1.1.1.1:53", "9.9.9.9:53"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original args
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()

			// Reset flag package
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

			os.Args = []string{"test", "-upstreams=" + tt.input}

			cfg, err := LoadFromFlags()
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(cfg.UpstreamDNS) != len(tt.expected) {
				t.Errorf("Expected %d upstreams, got %d", len(tt.expected), len(cfg.UpstreamDNS))
				return
			}

			for i, expected := range tt.expected {
				if cfg.UpstreamDNS[i] != expected {
					t.Errorf("Expected upstream[%d] = %s, got %s", i, expected, cfg.UpstreamDNS[i])
				}
			}
		})
	}
}

// containsString checks if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
