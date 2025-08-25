// Package resolver provides local DNS resolution for custom domain mappings.
package resolver

import (
	"net"
	"strings"

	"github.com/miekg/dns"
)

// LocalResolver handles custom DNS mappings
type LocalResolver struct {
	mappings map[string]string
}

// New creates a new LocalResolver with the given custom DNS mappings
func New(customMappings map[string]string) *LocalResolver {
	// Copy the mappings to avoid external modification
	mappings := make(map[string]string, len(customMappings))
	for domain, ip := range customMappings {
		mappings[domain] = ip
	}

	return &LocalResolver{
		mappings: mappings,
	}
}

// Resolve attempts to resolve a DNS question using custom mappings.
// Returns a DNS response if a mapping exists, nil otherwise.
func (r *LocalResolver) Resolve(question dns.Question) *dns.Msg {
	// Normalize the domain name (ensure it ends with a dot)
	domain := question.Name
	if !strings.HasSuffix(domain, ".") {
		domain += "."
	}

	// Check if we have a custom mapping for this domain
	ip, exists := r.mappings[domain]
	if !exists {
		return nil
	}

	// Create DNS response
	msg := &dns.Msg{}
	msg.SetReply(&dns.Msg{Question: []dns.Question{question}})
	msg.Authoritative = true

	// Handle different query types
	switch question.Qtype {
	case dns.TypeA:
		// IPv4 address query
		if parsedIP := net.ParseIP(ip); parsedIP != nil && parsedIP.To4() != nil {
			rr := &dns.A{
				Hdr: dns.RR_Header{
					Name:   domain,
					Rrtype: dns.TypeA,
					Class:  dns.ClassINET,
					Ttl:    300, // 5 minutes TTL
				},
				A: parsedIP.To4(),
			}
			msg.Answer = append(msg.Answer, rr)
		} else {
			// Invalid IPv4 address, return NXDOMAIN
			msg.SetRcode(&dns.Msg{Question: []dns.Question{question}}, dns.RcodeNameError)
		}

	case dns.TypeAAAA:
		// IPv6 address query
		if parsedIP := net.ParseIP(ip); parsedIP != nil && parsedIP.To16() != nil && parsedIP.To4() == nil {
			rr := &dns.AAAA{
				Hdr: dns.RR_Header{
					Name:   domain,
					Rrtype: dns.TypeAAAA,
					Class:  dns.ClassINET,
					Ttl:    300, // 5 minutes TTL
				},
				AAAA: parsedIP.To16(),
			}
			msg.Answer = append(msg.Answer, rr)
		} else {
			// No IPv6 address or invalid, return NXDOMAIN
			msg.SetRcode(&dns.Msg{Question: []dns.Question{question}}, dns.RcodeNameError)
		}

	case dns.TypePTR:
		// Reverse DNS lookup - not supported for custom mappings
		msg.SetRcode(&dns.Msg{Question: []dns.Question{question}}, dns.RcodeNameError)

	default:
		// For other query types, return NXDOMAIN if we have an IP mapping
		// This indicates the domain exists but the requested record type doesn't
		msg.SetRcode(&dns.Msg{Question: []dns.Question{question}}, dns.RcodeNameError)
	}

	return msg
}

// HasMapping returns true if the resolver has a custom mapping for the given domain
func (r *LocalResolver) HasMapping(domain string) bool {
	// Normalize the domain name (ensure it ends with a dot)
	if !strings.HasSuffix(domain, ".") {
		domain += "."
	}
	_, exists := r.mappings[domain]
	return exists
}

// GetMappings returns a copy of all current mappings
func (r *LocalResolver) GetMappings() map[string]string {
	mappings := make(map[string]string, len(r.mappings))
	for domain, ip := range r.mappings {
		mappings[domain] = ip
	}
	return mappings
}

// UpdateMappings replaces all current mappings with the provided ones
func (r *LocalResolver) UpdateMappings(newMappings map[string]string) {
	r.mappings = make(map[string]string, len(newMappings))
	for domain, ip := range newMappings {
		r.mappings[domain] = ip
	}
}
