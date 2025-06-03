package main

import (
	"flag"
	"log"
	"net"
	"strings"
	"time"

	"github.com/miekg/dns"
)

// Config holds the DNS server configuration
type Config struct {
	ListenAddress string
	Port          string
	UpstreamDNS   []string
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

// handleDNSRequest processes incoming DNS queries
func (s *DNSServer) handleDNSRequest(w dns.ResponseWriter, r *dns.Msg) {
	// Log the query
	if len(r.Question) > 0 {
		log.Printf("Query: %s %s", r.Question[0].Name, dns.TypeToString[r.Question[0].Qtype])
	}

	// Try each upstream DNS server until we get a response
	for _, upstream := range s.config.UpstreamDNS {
		resp, _, err := s.client.Exchange(r, upstream)
		if err != nil {
			log.Printf("Failed to query upstream %s: %v", upstream, err)
			continue
		}

		if resp != nil {
			// Forward the response back to the client
			w.WriteMsg(resp)
			return
		}
	}

	// If all upstreams failed, return SERVFAIL
	msg := &dns.Msg{}
	msg.SetRcode(r, dns.RcodeServerFailure)
	w.WriteMsg(msg)
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
	flag.Parse()

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
	}

	// Create and start DNS server
	server := NewDNSServer(config)

	log.Printf("DNS Proxy Server starting...")
	log.Printf("Configuration:")
	log.Printf("  Listen: %s:%s", config.ListenAddress, config.Port)
	log.Printf("  Upstreams: %v", config.UpstreamDNS)

	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start DNS server: %v", err)
	}
}
