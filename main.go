package main

import (
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/miekg/dns"
)

// LogEntry represents a complete DNS request/response cycle
type LogEntry struct {
	Timestamp   time.Time         `json:"timestamp"`
	UUID        string            `json:"uuid"`
	Request     RequestInfo       `json:"request"`
	Upstreams   []UpstreamAttempt `json:"upstreams"`
	Response    *ResponseInfo     `json:"response,omitempty"`
	Answers     [][]string        `json:"answers,omitempty"`
	IPAddresses []string          `json:"ip_addresses,omitempty"`
	Status      string            `json:"status"`
	Duration    float64           `json:"total_duration_ms"`
}

// RequestInfo contains information about the DNS request
type RequestInfo struct {
	Client string `json:"client"`
	Query  string `json:"query"`
	Type   string `json:"type"`
	ID     uint16 `json:"id"`
}

// UpstreamAttempt represents an attempt to query an upstream server
type UpstreamAttempt struct {
	Server   string   `json:"server"`
	Attempt  int      `json:"attempt"`
	Error    *string  `json:"error,omitempty"`
	RTT      *float64 `json:"rtt_ms,omitempty"`
	Duration float64  `json:"duration_ms"`
}

// ResponseInfo contains information about the successful response
type ResponseInfo struct {
	Upstream    string  `json:"upstream"`
	Rcode       string  `json:"rcode"`
	AnswerCount int     `json:"answer_count"`
	RTT         float64 `json:"rtt_ms"`
}

// Config holds the DNS server configuration
type Config struct {
	ListenAddress string
	Port          string
	UpstreamDNS   []string
	LogFile       string
}

// DNSServer represents our DNS proxy server
type DNSServer struct {
	config Config
	client *dns.Client
}

// NewDNSServer creates a new DNS server instance
func NewDNSServer(config Config) *DNSServer {
	return &DNSServer{
		config: config,
		client: &dns.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// setupLogging configures logging to file and/or console
func setupLogging(logFile string) (*os.File, error) {
	if logFile == "" {
		// Only console logging
		log.SetOutput(os.Stdout)
		return nil, nil
	}

	// Open log file
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	// Setup multi-writer to write to both file and console
	multiWriter := io.MultiWriter(os.Stdout, file)
	log.SetOutput(multiWriter)

	// Set log format with timestamp
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	log.Printf("Logging initialized - File: %s", logFile)
	return file, nil
}

// generateRequestUUID creates a unique identifier for each request
func generateRequestUUID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

// durationToMilliseconds converts time.Duration to milliseconds as float64
func durationToMilliseconds(d time.Duration) float64 {
	return float64(d.Nanoseconds()) / 1e6
}

// handleDNSRequest processes incoming DNS queries
func (s *DNSServer) handleDNSRequest(w dns.ResponseWriter, r *dns.Msg) {
	start := time.Now()
	clientAddr := w.RemoteAddr().String()
	requestUUID := generateRequestUUID()

	// Initialize log entry
	logEntry := LogEntry{
		Timestamp: start,
		UUID:      requestUUID,
		Upstreams: make([]UpstreamAttempt, 0),
		Status:    "unknown",
	}

	// Handle malformed queries
	if len(r.Question) == 0 {
		logEntry.Request = RequestInfo{
			Client: clientAddr,
			Query:  "MALFORMED",
			Type:   "UNKNOWN",
			ID:     r.Id,
		}
		logEntry.Status = "malformed_query"
		logEntry.Duration = durationToMilliseconds(time.Since(start))

		// Log and return error
		s.logJSON(logEntry)
		msg := &dns.Msg{}
		msg.SetRcode(r, dns.RcodeFormatError)
		w.WriteMsg(msg)
		return
	}

	// Set request information
	question := r.Question[0]
	logEntry.Request = RequestInfo{
		Client: clientAddr,
		Query:  question.Name,
		Type:   dns.TypeToString[question.Qtype],
		ID:     r.Id,
	}

	// Try each upstream DNS server until we get a response
	for i, upstream := range s.config.UpstreamDNS {
		upstreamStart := time.Now()

		resp, rtt, err := s.client.Exchange(r, upstream)
		upstreamDuration := time.Since(upstreamStart)

		// Record upstream attempt
		attempt := UpstreamAttempt{
			Server:   upstream,
			Attempt:  i + 1,
			Duration: durationToMilliseconds(upstreamDuration),
		}

		if err != nil {
			errStr := err.Error()
			attempt.Error = &errStr
			logEntry.Upstreams = append(logEntry.Upstreams, attempt)
			continue
		}

		if resp != nil {
			rttStr := durationToMilliseconds(rtt)
			attempt.RTT = &rttStr
			logEntry.Upstreams = append(logEntry.Upstreams, attempt)

			// Set response information
			logEntry.Response = &ResponseInfo{
				Upstream:    upstream,
				Rcode:       dns.RcodeToString[resp.Rcode],
				AnswerCount: len(resp.Answer),
				RTT:         rttStr,
			}

			// Collect answer records
			logEntry.Answers = make([][]string, len(resp.Answer))
			for j, answer := range resp.Answer {
				// Parse the answer record into components
				answerStr := answer.String()
				parts := strings.Fields(answerStr)

				// Ensure we have at least the basic components
				if len(parts) >= 4 {
					logEntry.Answers[j] = parts
				} else {
					// Fallback to the original string as a single element
					logEntry.Answers[j] = []string{answerStr}
				}
			}

			// Collect IP addresses from A and AAAA records
			logEntry.IPAddresses = make([]string, 0)
			for _, answer := range resp.Answer {
				switch rr := answer.(type) {
				case *dns.A:
					logEntry.IPAddresses = append(logEntry.IPAddresses, rr.A.String())
				case *dns.AAAA:
					logEntry.IPAddresses = append(logEntry.IPAddresses, rr.AAAA.String())
				}
			}

			logEntry.Status = "success"
			logEntry.Duration = durationToMilliseconds(time.Since(start))

			// Log successful response
			s.logJSON(logEntry)

			// Forward the response back to the client
			if err := w.WriteMsg(resp); err != nil {
				// Log write error separately as it's after the main request processing
				errorLog := map[string]interface{}{
					"timestamp": time.Now(),
					"uuid":      requestUUID,
					"error":     "failed_to_write_response",
					"message":   err.Error(),
					"client":    clientAddr,
				}
				s.logJSON(errorLog)
			}
			return
		} else {
			logEntry.Upstreams = append(logEntry.Upstreams, attempt)
		}
	}

	// If all upstreams failed, return SERVFAIL
	logEntry.Status = "all_upstreams_failed"
	logEntry.Duration = durationToMilliseconds(time.Since(start))

	s.logJSON(logEntry)

	msg := &dns.Msg{}
	msg.SetRcode(r, dns.RcodeServerFailure)
	if err := w.WriteMsg(msg); err != nil {
		// Log write error separately
		errorLog := map[string]interface{}{
			"timestamp": time.Now(),
			"uuid":      requestUUID,
			"error":     "failed_to_write_servfail",
			"message":   err.Error(),
			"client":    clientAddr,
		}
		s.logJSON(errorLog)
	}
}

// logJSON outputs a structured log entry as JSON
func (s *DNSServer) logJSON(entry interface{}) {
	jsonBytes, err := json.Marshal(entry)
	if err != nil {
		log.Printf("Error marshaling log entry: %v", err)
		return
	}
	log.Println(string(jsonBytes))
}

// Start begins the DNS server
func (s *DNSServer) Start() error {
	// Create DNS handler
	dns.HandleFunc(".", s.handleDNSRequest)

	// Setup UDP server
	server := &dns.Server{
		Addr: net.JoinHostPort(s.config.ListenAddress, s.config.Port),
		Net:  "udp",
	}

	log.Printf("Starting DNS server on %s:%s", s.config.ListenAddress, s.config.Port)
	log.Printf("Upstream DNS servers: %s", strings.Join(s.config.UpstreamDNS, ", "))

	return server.ListenAndServe()
}

func main() {
	// Command line flags
	listenAddr := flag.String("listen", "0.0.0.0", "Listen address")
	port := flag.String("port", "53", "Listen port")
	upstreams := flag.String("upstreams", "8.8.8.8:53,1.1.1.1:53", "Comma-separated list of upstream DNS servers")
	logFile := flag.String("log", "", "Log file path (optional, logs to console if not specified)")
	flag.Parse()

	// Setup logging
	var file *os.File
	if *logFile != "" {
		var err error
		file, err = setupLogging(*logFile)
		if err != nil {
			log.Fatalf("Failed to setup logging: %v", err)
		}
		// Ensure file is closed when program exits
		if file != nil {
			defer func() {
				log.Println("Closing log file...")
				file.Close()
			}()
		}
	} else {
		// Setup console-only logging
		log.SetFlags(log.LstdFlags | log.Lmicroseconds)
		log.Println("Logging to console only")
	}

	// Parse upstream servers
	upstreamList := strings.Split(*upstreams, ",")
	for i, upstream := range upstreamList {
		upstreamList[i] = strings.TrimSpace(upstream)
	}

	// Create configuration
	config := Config{
		ListenAddress: *listenAddr,
		Port:          *port,
		UpstreamDNS:   upstreamList,
		LogFile:       *logFile,
	}

	// Create and start DNS server
	server := NewDNSServer(config)

	log.Printf("DNS Proxy Server starting...")
	log.Printf("Configuration:")
	log.Printf("  Listen: %s:%s", config.ListenAddress, config.Port)
	log.Printf("  Upstreams: %v", config.UpstreamDNS)
	if config.LogFile != "" {
		log.Printf("  Log File: %s", config.LogFile)
	} else {
		log.Printf("  Log File: Console only")
	}

	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start DNS server: %v", err)
	}
}
