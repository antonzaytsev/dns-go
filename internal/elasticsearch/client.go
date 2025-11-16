package elasticsearch

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"dns-go/internal/types"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

const (
	DefaultHost  = "localhost"
	DefaultPort  = "9200"
	DefaultIndex = "dns-logs"
)

// Client wraps the Elasticsearch client with DNS-specific functionality
type Client struct {
	es    *elasticsearch.Client
	index string
}

// Config holds Elasticsearch configuration
type Config struct {
	Host  string
	Port  string
	URL   string // Deprecated: use Host and Port instead
	Index string
}

// NewClient creates a new Elasticsearch client
func NewClient(cfg Config) (*Client, error) {
	// Determine URL from host/port or fallback to URL field
	var url string
	if cfg.Host != "" || cfg.Port != "" {
		// Use host and port
		host := getEnvOrDefault("ELASTICSEARCH_HOST", cfg.Host)
		if host == "" {
			host = DefaultHost
		}
		port := getEnvOrDefault("ELASTICSEARCH_PORT", cfg.Port)
		if port == "" {
			port = DefaultPort
		}
		url = fmt.Sprintf("http://%s:%s", host, port)
	} else {
		// Fallback to URL field for backward compatibility
		url = cfg.URL
		if url == "" {
			url = getEnvOrDefault("ELASTICSEARCH_URL", fmt.Sprintf("http://%s:%s", DefaultHost, DefaultPort))
		}
	}

	if cfg.Index == "" {
		cfg.Index = getEnvOrDefault("ELASTICSEARCH_INDEX", DefaultIndex)
	}

	// Configure Elasticsearch client
	esCfg := elasticsearch.Config{
		Addresses: []string{url},
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	es, err := elasticsearch.NewClient(esCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Elasticsearch client: %w", err)
	}

	client := &Client{
		es:    es,
		index: cfg.Index,
	}

	// Test connection and create index if needed
	if err := client.initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize Elasticsearch: %w", err)
	}

	return client, nil
}

// initialize checks connection and sets up the index
func (c *Client) initialize() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test connection
	res, err := c.es.Info(c.es.Info.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("failed to connect to Elasticsearch: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("Elasticsearch returned error: %s", res.String())
	}

	// Create index with mapping if it doesn't exist
	return c.createIndexIfNotExists(ctx)
}

// createIndexIfNotExists creates the DNS logs index with proper mapping
func (c *Client) createIndexIfNotExists(ctx context.Context) error {
	// Check if index exists
	res, err := c.es.Indices.Exists([]string{c.index})
	if err != nil {
		return fmt.Errorf("failed to check if index exists: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode == 200 {
		// Index already exists
		return nil
	}

	// Create index with mapping optimized for DNS logs
	mapping := map[string]interface{}{
		"mappings": map[string]interface{}{
			"properties": map[string]interface{}{
				"timestamp": map[string]interface{}{
					"type": "date",
				},
				"uuid": map[string]interface{}{
					"type": "keyword",
				},
				"request": map[string]interface{}{
					"properties": map[string]interface{}{
						"client": map[string]interface{}{
							"type": "keyword",
						},
						"query": map[string]interface{}{
							"type": "text",
							"fields": map[string]interface{}{
								"keyword": map[string]interface{}{
									"type":         "keyword",
									"ignore_above": 256,
								},
							},
						},
						"type": map[string]interface{}{
							"type": "keyword",
						},
						"id": map[string]interface{}{
							"type": "integer",
						},
					},
				},
				"response": map[string]interface{}{
					"properties": map[string]interface{}{
						"upstream": map[string]interface{}{
							"type": "keyword",
						},
						"rcode": map[string]interface{}{
							"type": "keyword",
						},
						"answer_count": map[string]interface{}{
							"type": "integer",
						},
						"rtt_ms": map[string]interface{}{
							"type": "float",
						},
					},
				},
				"ip_addresses": map[string]interface{}{
					"type": "ip",
				},
				"status": map[string]interface{}{
					"type": "keyword",
				},
				"total_duration_ms": map[string]interface{}{
					"type": "float",
				},
			},
		},
		"settings": map[string]interface{}{
			"number_of_shards":   1,
			"number_of_replicas": 0,
		},
	}

	mappingBytes, err := json.Marshal(mapping)
	if err != nil {
		return fmt.Errorf("failed to marshal index mapping: %w", err)
	}

	req := esapi.IndicesCreateRequest{
		Index: c.index,
		Body:  strings.NewReader(string(mappingBytes)),
	}

	res, err = req.Do(ctx, c.es)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		// Check if it's just a "resource_already_exists_exception" which is fine
		if res.StatusCode == 400 && strings.Contains(res.String(), "resource_already_exists_exception") {
			// Index already exists, this is fine
			return nil
		}
		return fmt.Errorf("failed to create index: %s", res.String())
	}

	return nil
}

// IndexLogEntry indexes a DNS log entry
func (c *Client) IndexLogEntry(entry types.LogEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal log entry: %w", err)
	}

	req := esapi.IndexRequest{
		Index:      c.index,
		DocumentID: entry.UUID,
		Body:       strings.NewReader(string(data)),
		Refresh:    "false", // Don't refresh immediately for performance
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := req.Do(ctx, c.es)
	if err != nil {
		return fmt.Errorf("failed to index log entry: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("Elasticsearch indexing error: %s", res.String())
	}

	return nil
}

// SearchLogEntry represents a search result
type SearchResult struct {
	Results []types.LogEntry `json:"results"`
	Total   int64            `json:"total"`
}

// SearchLogs searches through DNS logs stored in Elasticsearch
func (c *Client) SearchLogs(searchTerm string, limit, offset int, since *time.Time) (*SearchResult, error) {
	query := c.buildSearchQuery(searchTerm, since)

	searchBody := map[string]interface{}{
		"query": query,
		"sort": []map[string]interface{}{
			{
				"timestamp": map[string]interface{}{
					"order": "desc",
				},
			},
		},
		"from": offset,
		"size": limit,
	}

	searchBytes, err := json.Marshal(searchBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal search query: %w", err)
	}

	req := esapi.SearchRequest{
		Index: []string{c.index},
		Body:  strings.NewReader(string(searchBytes)),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := req.Do(ctx, c.es)
	if err != nil {
		return nil, fmt.Errorf("failed to search logs: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("Elasticsearch search error: %s", res.String())
	}

	var response struct {
		Hits struct {
			Total struct {
				Value int64 `json:"value"`
			} `json:"total"`
			Hits []struct {
				Source types.LogEntry `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}

	results := make([]types.LogEntry, len(response.Hits.Hits))
	for i, hit := range response.Hits.Hits {
		results[i] = hit.Source
	}

	return &SearchResult{
		Results: results,
		Total:   response.Hits.Total.Value,
	}, nil
}

// buildSearchQuery constructs an Elasticsearch query based on search term and time filter
func (c *Client) buildSearchQuery(searchTerm string, since *time.Time) map[string]interface{} {
	var query map[string]interface{}

	// Build the main query (text search)
	if searchTerm == "" {
		query = map[string]interface{}{
			"match_all": map[string]interface{}{},
		}
	} else {
		shouldClauses := []map[string]interface{}{
			// Substring search in query domain using wildcard
			{
				"wildcard": map[string]interface{}{
					"request.query": map[string]interface{}{
						"value":            fmt.Sprintf("*%s*", strings.ToLower(searchTerm)),
						"case_insensitive": true,
					},
				},
			},
			// Exact and partial match for query domain
			{
				"match": map[string]interface{}{
					"request.query": map[string]interface{}{
						"query":     searchTerm,
						"fuzziness": "AUTO",
					},
				},
			},
			// Substring search in client address
			{
				"wildcard": map[string]interface{}{
					"request.client": map[string]interface{}{
						"value":            fmt.Sprintf("*%s*", searchTerm),
						"case_insensitive": true,
					},
				},
			},
			// Exact match for request type
			{
				"term": map[string]interface{}{
					"request.type": map[string]interface{}{
						"value": strings.ToUpper(searchTerm),
					},
				},
			},
			// Exact match for status
			{
				"term": map[string]interface{}{
					"status": map[string]interface{}{
						"value": strings.ToLower(searchTerm),
					},
				},
			},
			// Substring search in upstream server
			{
				"wildcard": map[string]interface{}{
					"response.upstream": map[string]interface{}{
						"value":            fmt.Sprintf("*%s*", searchTerm),
						"case_insensitive": true,
					},
				},
			},
			// Exact match for UUID
			{
				"term": map[string]interface{}{
					"uuid": map[string]interface{}{
						"value": searchTerm,
					},
				},
			},
		}

		// Add IP address search if it looks like an IP or partial IP
		if isValidIP(searchTerm) || isPartialIP(searchTerm) {
			shouldClauses = append(shouldClauses, map[string]interface{}{
				"wildcard": map[string]interface{}{
					"ip_addresses": map[string]interface{}{
						"value": fmt.Sprintf("*%s*", searchTerm),
					},
				},
			})
		}

		query = map[string]interface{}{
			"bool": map[string]interface{}{
				"should":               shouldClauses,
				"minimum_should_match": 1,
			},
		}
	}

	// Add time filter if specified
	if since != nil {
		now := time.Now()
		timeFilter := map[string]interface{}{
			"range": map[string]interface{}{
				"timestamp": map[string]interface{}{
					"gte": since.Format(time.RFC3339),
					"lte": now.Format(time.RFC3339),
				},
			},
		}

		// Combine text search query with time filter
		if query != nil {
			query = map[string]interface{}{
				"bool": map[string]interface{}{
					"must": []map[string]interface{}{
						query,
						timeFilter,
					},
				},
			}
		} else {
			query = timeFilter
		}
	}

	return query
}

// isValidIP checks if a string looks like an IP address
func isValidIP(str string) bool {
	parts := strings.Split(str, ".")
	if len(parts) != 4 {
		return false
	}
	for _, part := range parts {
		if len(part) == 0 || len(part) > 3 {
			return false
		}
		for _, char := range part {
			if char < '0' || char > '9' {
				return false
			}
		}
	}
	return true
}

// isPartialIP checks if a string looks like a partial IP address
func isPartialIP(str string) bool {
	parts := strings.Split(str, ".")
	if len(parts) > 4 || len(parts) == 0 {
		return false
	}

	for _, part := range parts {
		if part == "" {
			continue // Allow empty parts for partial IPs like "192.168."
		}
		if len(part) > 3 {
			return false
		}

		// Check if part is numeric
		if _, err := strconv.Atoi(part); err != nil {
			return false
		}
	}

	// Must contain at least one digit and a dot, or be all digits
	return strings.Contains(str, ".") || (len(str) > 0 && str[0] >= '0' && str[0] <= '9')
}

// GetLogCount returns the total number of log entries in Elasticsearch
func (c *Client) GetLogCount() (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Force refresh before counting to ensure all indexed documents are searchable
	refreshReq := esapi.IndicesRefreshRequest{
		Index: []string{c.index},
	}
	refreshRes, err := refreshReq.Do(ctx, c.es)
	if err != nil {
		return 0, fmt.Errorf("failed to refresh index: %w", err)
	}
	refreshRes.Body.Close()

	searchBody := map[string]interface{}{
		"query": map[string]interface{}{
			"match_all": map[string]interface{}{},
		},
		"size": 0,
	}

	searchBytes, err := json.Marshal(searchBody)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal search query: %w", err)
	}

	req := esapi.SearchRequest{
		Index: []string{c.index},
		Body:  strings.NewReader(string(searchBytes)),
	}

	res, err := req.Do(ctx, c.es)
	if err != nil {
		return 0, fmt.Errorf("failed to search Elasticsearch: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return 0, fmt.Errorf("Elasticsearch search error: %s", res.String())
	}

	var response struct {
		Hits struct {
			Total struct {
				Value int64 `json:"value"`
			} `json:"total"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return 0, fmt.Errorf("failed to decode search response: %w", err)
	}

	return response.Hits.Total.Value, nil
}

// HealthCheck checks if Elasticsearch is healthy
func (c *Client) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := c.es.Cluster.Health(c.es.Cluster.Health.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("failed to check Elasticsearch health: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("Elasticsearch health check failed: %s", res.String())
	}

	return nil
}

// GetClient returns the underlying Elasticsearch client
func (c *Client) GetClient() *elasticsearch.Client {
	return c.es
}

// Close closes the Elasticsearch client
func (c *Client) Close() error {
	// The v8 client doesn't have a Close method, connection cleanup is automatic
	return nil
}

// getEnvOrDefault returns environment variable value or default if not set
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
