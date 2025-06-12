package types

import (
	"crypto/rand"
	"fmt"
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
	CacheHit    bool              `json:"cache_hit,omitempty"`
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

// GenerateRequestUUID creates a unique identifier for each request
func GenerateRequestUUID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

// DurationToMilliseconds converts time.Duration to milliseconds as float64
func DurationToMilliseconds(d time.Duration) float64 {
	return float64(d.Nanoseconds()) / 1e6
}

// ExtractAnswers parses DNS answer records into structured format
func ExtractAnswers(answers []dns.RR) [][]string {
	result := make([][]string, len(answers))
	for i, answer := range answers {
		answerStr := answer.String()
		parts := strings.Fields(answerStr)
		if len(parts) >= 4 {
			result[i] = parts
		} else {
			result[i] = []string{answerStr}
		}
	}
	return result
}

// ExtractIPAddresses extracts IP addresses from A and AAAA records
func ExtractIPAddresses(answers []dns.RR) []string {
	var ips []string
	for _, answer := range answers {
		switch rr := answer.(type) {
		case *dns.A:
			ips = append(ips, rr.A.String())
		case *dns.AAAA:
			ips = append(ips, rr.AAAA.String())
		}
	}
	return ips
}
