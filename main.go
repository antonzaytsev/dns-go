package main

import (
	"flag"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/miekg/dns"
)

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

// handleDNSRequest processes incoming DNS queries
func (s *DNSServer) handleDNSRequest(w dns.ResponseWriter, r *dns.Msg) {
	start := time.Now()
	clientAddr := w.RemoteAddr().String()

	// Log the incoming query
	if len(r.Question) > 0 {
		question := r.Question[0]
		log.Printf("[REQUEST] Client: %s | Query: %s | Type: %s | ID: %d",
			clientAddr, question.Name, dns.TypeToString[question.Qtype], r.Id)
	} else {
		log.Printf("[REQUEST] Client: %s | No questions in query | ID: %d", clientAddr, r.Id)
		// Return FORMERR for malformed queries
		msg := &dns.Msg{}
		msg.SetRcode(r, dns.RcodeFormatError)
		w.WriteMsg(msg)
		return
	}

	// Try each upstream DNS server until we get a response
	for i, upstream := range s.config.UpstreamDNS {
		log.Printf("[UPSTREAM] Trying upstream %d/%d: %s", i+1, len(s.config.UpstreamDNS), upstream)

		upstreamStart := time.Now()
		resp, rtt, err := s.client.Exchange(r, upstream)
		upstreamDuration := time.Since(upstreamStart)

		if err != nil {
			log.Printf("[ERROR] Upstream %s failed after %v: %v", upstream, upstreamDuration, err)
			continue
		}

		if resp != nil {
			totalDuration := time.Since(start)

			// Log successful response details
			log.Printf("[RESPONSE] Success | Upstream: %s | RTT: %v | Total: %v | Answers: %d | Rcode: %s | ID: %d",
				upstream, rtt, totalDuration, len(resp.Answer), dns.RcodeToString[resp.Rcode], resp.Id)

			// Log answer records if present
			if len(resp.Answer) > 0 {
				for _, answer := range resp.Answer {
					log.Printf("[ANSWER] %s", answer.String())
				}
			}

			// Forward the response back to the client
			if err := w.WriteMsg(resp); err != nil {
				log.Printf("[ERROR] Failed to write response to client %s: %v", clientAddr, err)
			}
			return
		} else {
			log.Printf("[WARNING] Upstream %s returned nil response after %v", upstream, upstreamDuration)
		}
	}

	// If all upstreams failed, return SERVFAIL
	totalDuration := time.Since(start)
	log.Printf("[FAILURE] All upstreams failed after %v | Client: %s | ID: %d", totalDuration, clientAddr, r.Id)

	msg := &dns.Msg{}
	msg.SetRcode(r, dns.RcodeServerFailure)
	if err := w.WriteMsg(msg); err != nil {
		log.Printf("[ERROR] Failed to write SERVFAIL response to client %s: %v", clientAddr, err)
	}
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
