package cache

import (
	"testing"
	"time"

	"github.com/miekg/dns"
)

func TestCacheBasicOperations(t *testing.T) {
	cache := New(10, 5*time.Minute)

	// Create a test question
	question := dns.Question{
		Name:   "example.com.",
		Qtype:  dns.TypeA,
		Qclass: dns.ClassINET,
	}

	// Test cache miss
	_, found := cache.Get(question)
	if found {
		t.Error("Expected cache miss, got hit")
	}

	// Create a test response
	response := &dns.Msg{}
	response.SetQuestion(question.Name, question.Qtype)

	// Add an answer record
	rr, _ := dns.NewRR("example.com. 300 IN A 192.0.2.1")
	response.Answer = []dns.RR{rr}

	// Set in cache
	cache.Set(question, response)

	// Test cache hit
	cached, found := cache.Get(question)
	if !found {
		t.Error("Expected cache hit, got miss")
	}

	if len(cached.Answer) != 1 {
		t.Errorf("Expected 1 answer, got %d", len(cached.Answer))
	}
}

func TestCacheExpiration(t *testing.T) {
	cache := New(10, 50*time.Millisecond)

	question := dns.Question{
		Name:   "example.com.",
		Qtype:  dns.TypeA,
		Qclass: dns.ClassINET,
	}

	response := &dns.Msg{}
	response.SetQuestion(question.Name, question.Qtype)
	rr, _ := dns.NewRR("example.com. 1 IN A 192.0.2.1") // Very short TTL
	response.Answer = []dns.RR{rr}

	cache.Set(question, response)

	// Should be in cache immediately
	_, found := cache.Get(question)
	if !found {
		t.Error("Expected cache hit immediately after set")
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should be expired now
	_, found = cache.Get(question)
	if found {
		t.Error("Expected cache miss after expiration")
	}
}

func TestCacheStats(t *testing.T) {
	cache := New(5, 5*time.Minute)

	size, maxSize := cache.Stats()
	if size != 0 {
		t.Errorf("Expected empty cache size 0, got %d", size)
	}
	if maxSize != 5 {
		t.Errorf("Expected max size 5, got %d", maxSize)
	}

	// Add one entry
	question := dns.Question{
		Name:   "example.com.",
		Qtype:  dns.TypeA,
		Qclass: dns.ClassINET,
	}

	response := &dns.Msg{}
	response.SetQuestion(question.Name, question.Qtype)
	rr, _ := dns.NewRR("example.com. 300 IN A 192.0.2.1")
	response.Answer = []dns.RR{rr}

	cache.Set(question, response)

	size, _ = cache.Stats()
	if size != 1 {
		t.Errorf("Expected cache size 1 after adding entry, got %d", size)
	}
}
