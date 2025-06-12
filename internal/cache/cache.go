package cache

import (
	"sync"
	"time"

	"github.com/miekg/dns"
)

// Entry represents a cached DNS response with expiration
type Entry struct {
	Response  *dns.Msg
	ExpiresAt time.Time
}

// Cache provides a thread-safe DNS response cache with TTL
type Cache struct {
	mu         sync.RWMutex
	entries    map[string]*Entry
	maxSize    int
	defaultTTL time.Duration
}

// New creates a new DNS cache with the specified maximum size and default TTL
func New(maxSize int, defaultTTL time.Duration) *Cache {
	return &Cache{
		entries:    make(map[string]*Entry),
		maxSize:    maxSize,
		defaultTTL: defaultTTL,
	}
}

// generateKey creates a cache key from DNS question
func (c *Cache) generateKey(question dns.Question) string {
	return question.Name + ":" + dns.TypeToString[question.Qtype]
}

// Get retrieves a cached response if it exists and hasn't expired
func (c *Cache) Get(question dns.Question) (*dns.Msg, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := c.generateKey(question)
	entry, exists := c.entries[key]
	if !exists {
		return nil, false
	}

	if time.Now().After(entry.ExpiresAt) {
		// Entry expired, will be cleaned up later
		return nil, false
	}

	// Create a copy of the response to avoid race conditions
	response := entry.Response.Copy()
	return response, true
}

// Set stores a DNS response in the cache
func (c *Cache) Set(question dns.Question, response *dns.Msg) {
	if response == nil || len(response.Answer) == 0 {
		return // Don't cache empty responses
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// If cache is full, remove oldest entries
	if len(c.entries) >= c.maxSize {
		c.evictOldest()
	}

	key := c.generateKey(question)

	// Calculate TTL from the response, use default if no TTL found
	ttl := c.calculateTTL(response)

	c.entries[key] = &Entry{
		Response:  response.Copy(),
		ExpiresAt: time.Now().Add(ttl),
	}
}

// calculateTTL determines the appropriate TTL for a DNS response
func (c *Cache) calculateTTL(response *dns.Msg) time.Duration {
	if len(response.Answer) == 0 {
		return c.defaultTTL
	}

	// Use the minimum TTL from all answer records
	minTTL := uint32(^uint32(0)) // Max uint32
	for _, answer := range response.Answer {
		if answer.Header().Ttl < minTTL {
			minTTL = answer.Header().Ttl
		}
	}

	if minTTL == 0 {
		return c.defaultTTL
	}

	ttl := time.Duration(minTTL) * time.Second
	if ttl > c.defaultTTL {
		return c.defaultTTL
	}
	return ttl
}

// evictOldest removes expired entries or oldest entries if cache is full
func (c *Cache) evictOldest() {
	now := time.Now()

	// First, remove expired entries
	for key, entry := range c.entries {
		if now.After(entry.ExpiresAt) {
			delete(c.entries, key)
		}
	}

	// If still full, remove some entries (simple FIFO-like approach)
	if len(c.entries) >= c.maxSize {
		count := 0
		for key := range c.entries {
			delete(c.entries, key)
			count++
			if count >= c.maxSize/4 { // Remove 25% of entries
				break
			}
		}
	}
}

// Cleanup removes expired entries from the cache
func (c *Cache) Cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, entry := range c.entries {
		if now.After(entry.ExpiresAt) {
			delete(c.entries, key)
		}
	}
}

// Stats returns cache statistics
func (c *Cache) Stats() (size int, maxSize int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries), c.maxSize
}

// StartCleanupTimer starts a background goroutine that periodically cleans up expired entries
func (c *Cache) StartCleanupTimer(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			c.Cleanup()
		}
	}()
}
