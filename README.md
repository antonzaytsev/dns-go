# Simple DNS Proxy Server

A lightweight DNS proxy server written in Go that forwards queries to upstream DNS servers (Google DNS and Cloudflare DNS by default).

## Features

- **Simple and Fast**: Minimal overhead DNS proxy
- **Multiple Upstreams**: Supports multiple upstream DNS servers with failover
- **Configurable**: Command-line flags for easy configuration
- **Docker Ready**: Includes Docker and Docker Compose support
- **Comprehensive Logging**: Built-in detailed request/response logging with file and console output
- **Detailed Metrics**: Timing information, upstream server tracking, and answer logging

## Default Upstream Servers

- **Google DNS**: 8.8.8.8, 8.8.4.4
- **Cloudflare DNS**: 1.1.1.1, 1.0.0.1

## Logging Features

The DNS server provides **structured JSON logging** for every request and response with **unique request tracking**:

- **Request UUID**: Each DNS request gets a unique 8-character identifier for easy correlation
- **Structured JSON**: All request information consolidated into a single JSON object
- **Complete Lifecycle**: Request, upstream attempts, response, and answers in one log entry
- **Performance Metrics**: Response times (RTT and total duration) for each upstream attempt
- **Error Handling**: Failed upstream attempts and timeout logging within the same JSON structure
- **Easy Parsing**: JSON format perfect for log aggregation tools (ELK stack, Loki, etc.)
- **Dual Output**: Logs to both console and file simultaneously (when file logging enabled)

### JSON Log Structure

Each DNS request produces a single comprehensive JSON log entry:

```json
{
  "timestamp": "2025-06-03T10:40:38.623412391Z",
  "uuid": "cd1e2dda",
  "request": {
    "client": "192.168.148.1:39729",
    "query": "google.com.",
    "type": "A",
    "id": 48130
  },
  "upstreams": [
    {
      "server": "8.8.8.8:53",
      "attempt": 1,
      "rtt_ms": 43.891259,
      "duration_ms": 44.03878
    }
  ],
  "response": {
    "upstream": "8.8.8.8:53",
    "rcode": "NOERROR",
    "answer_count": 6,
    "rtt_ms": 43.891259
  },
  "answers": [
    ["google.com.", "181", "IN", "A", "173.194.222.139"],
    ["google.com.", "181", "IN", "A", "173.194.222.113"],
    ["google.com.", "181", "IN", "A", "173.194.222.138"],
    ["google.com.", "181", "IN", "A", "173.194.222.100"],
    ["google.com.", "181", "IN", "A", "173.194.222.101"],
    ["google.com.", "181", "IN", "A", "173.194.222.102"]
  ],
  "ip_addresses": [
    "173.194.222.139",
    "173.194.222.113",
    "173.194.222.138",
    "173.194.222.100",
    "173.194.222.101",
    "173.194.222.102"
  ],
  "status": "success",
  "total_duration_ms": 44.078161
}
```

**IPv6 (AAAA) response:**
```json
{
  "timestamp": "2025-06-03T10:40:38.676657383Z",
  "uuid": "9d8189ad",
  "request": {
    "client": "192.168.148.1:59747",
    "query": "cloudflare.com.",
    "type": "AAAA",
    "id": 63430
  },
  "response": {
    "upstream": "8.8.8.8:53",
    "rcode": "NOERROR",
    "answer_count": 2,
    "rtt_ms": 56.983809
  },
  "answers": [
    ["cloudflare.com.", "300", "IN", "AAAA", "2606:4700::6810:85e5"],
    ["cloudflare.com.", "300", "IN", "AAAA", "2606:4700::6810:84e5"]
  ],
  "ip_addresses": [
    "2606:4700::6810:85e5",
    "2606:4700::6810:84e5"
  ],
  "status": "success",
  "total_duration_ms": 57.176045
}
```

**Non-IP record type (MX) - no ip_addresses field:**
```json
{
  "request": {
    "query": "google.com.",
    "type": "MX"
  },
  "response": {
    "upstream": "8.8.8.8:53",
    "rcode": "NOERROR",
    "answer_count": 1,
    "rtt_ms": 48.433
  },
  "answers": [
    ["google.com.", "47", "IN", "MX", "10", "smtp.google.com."]
  ],
  "status": "success",
  "total_duration_ms": 48.617
}
```

**NXDOMAIN (non-existent domain) response:**
```json
{
  "timestamp": "2025-06-03T10:33:57.53317319Z",
  "uuid": "21a9f843",
  "request": {
    "client": "192.168.148.1:40266",
    "query": "really-nonexistent-domain.invalid.",
    "type": "A",
    "id": 31143
  },
  "upstreams": [
    {
      "server": "8.8.8.8:53",
      "attempt": 1,
      "rtt": "53.520346ms",
      "duration": "53.645345ms"
    }
  ],
  "response": {
    "upstream": "8.8.8.8:53",
    "rcode": "NXDOMAIN",
    "answer_count": 0,
    "rtt": "53.520346ms"
  },
  "status": "success",
  "total_duration": "53.66522ms"
}
```

**Upstream failure scenario (would show multiple attempts):**
```json
{
  "upstreams": [
    {
      "server": "8.8.8.8:53",
      "attempt": 1,
      "error": "timeout",
      "duration": "5.0s"
    },
    {
      "server": "8.8.4.4:53",
      "attempt": 2,
      "rtt": "45ms",
      "duration": "46ms"
    }
  ],
  "status": "success"
}
```

### JSON Field Definitions

- **timestamp**: ISO8601 timestamp when request was received
- **uuid**: Unique 8-character identifier for request correlation
- **request**: Client info, query details, and DNS message ID
- **upstreams**: Array of all upstream attempts (successful and failed)
  - **rtt_ms**: Round-trip time in decimal milliseconds (when successful)
  - **duration_ms**: Total time spent on this upstream attempt in decimal milliseconds
- **response**: Details of successful response (if any)
  - **rtt_ms**: Round-trip time in decimal milliseconds
- **answers**: Array of arrays, where each DNS record is broken into components:
  - For A records: `["name", "ttl", "class", "type", "ip_address"]`
  - For MX records: `["name", "ttl", "class", "type", "priority", "mail_server"]`
  - For AAAA records: `["name", "ttl", "class", "type", "ipv6_address"]`
- **ip_addresses**: Array of IP addresses extracted from A and AAAA records (only present for IP queries)
- **status**: Overall request status (`success`, `all_upstreams_failed`, `malformed_query`)
- **total_duration_ms**: End-to-end processing time in decimal milliseconds

### Log Analysis Examples

```bash
# View formatted JSON logs
tail -f logs/dns-server.log | sed 's/^[0-9\/: .]*{/{/' | jq .

# Filter by specific UUID
grep "da998633" logs/dns-server.log | jq .

# Find all NXDOMAIN responses
grep '"rcode":"NXDOMAIN"' logs/dns-server.log | jq .

# Find slow queries (>100ms)
grep -o '{.*}' logs/dns-server.log | jq 'select(.total_duration_ms > 100)'

# Count queries by type
grep -o '{.*}' logs/dns-server.log | jq -r '.request.type' | sort | uniq -c

# Extract all IP addresses from A/AAAA queries
grep -o '{.*}' logs/dns-server.log | jq -r '.ip_addresses[]?' | sort | uniq

# Find queries that returned specific IP address
grep -o '{.*}' logs/dns-server.log | jq 'select(.ip_addresses[]? == "173.194.222.139")'

# Get statistics on query response times
grep -o '{.*}' logs/dns-server.log | jq -r '.total_duration_ms' | awk '{sum+=$1; count++} END {print "Avg response time:", sum/count, "ms"}'

# Extract IP addresses from structured answers (5th element in A records)
grep -o '{.*}' logs/dns-server.log | jq -r '.answers[]? | select(.[3] == "A") | .[4]' | sort | uniq

# Find MX record priorities and mail servers
grep -o '{.*}' logs/dns-server.log | jq -r '.answers[]? | select(.[3] == "MX") | .[4] + " " + .[5]'
```

## Usage

### Option 1: Run with Go (Development)

```bash
# Install dependencies
go mod tidy

# Run with default settings (console logging only)
sudo go run main.go

# Run with file logging
go run main.go -port 5353 -log ./logs/dns.log

# Run with custom settings
go run main.go -listen 127.0.0.1 -port 5353 -upstreams "8.8.8.8:53,1.1.1.1:53" -log ./dns.log
```

### Option 2: Build and Run Binary

```bash
# Build the binary
go build -o dns-server main.go

# Run (requires sudo for port 53)
sudo ./dns-server

# Run with file logging on non-privileged port
./dns-server -port 5353 -log ./logs/dns.log
```

### Option 3: Docker Compose (Recommended)

```bash
# Build and start the DNS server (includes automatic file logging)
docker-compose up -d

# View real-time logs (console output)
docker-compose logs -f

# View log file content
cat logs/dns-server.log

# Stop the server
docker-compose down
```

### Option 4: Docker (Manual)

```bash
# Build the image
docker build -t dns2-server .

# Run with volume mount for logs
docker run -d --name dns2 -p 53:53/udp -v $(pwd)/logs:/logs dns2-server
```

## Configuration

### Command Line Flags

- `-listen`: Listen address (default: `0.0.0.0`)
- `-port`: Listen port (default: `53`)
- `-upstreams`: Comma-separated upstream DNS servers (default: `8.8.8.8:53,1.1.1.1:53`)
- `-log`: Log file path (optional, logs to console if not specified)

### Examples

```bash
# Listen only on localhost, port 5353, with file logging
./dns-server -listen 127.0.0.1 -port 5353 -log ./dns.log

# Use only Cloudflare DNS with logging
./dns-server -upstreams "1.1.1.1:53,1.0.0.1:53" -log ./cloudflare-dns.log

# Use custom DNS servers with detailed logging
./dns-server -upstreams "9.9.9.9:53,149.112.112.112:53" -log ./quad9-dns.log
```

## Testing

Test your DNS server with `dig` or `nslookup`:

```bash
# Test with dig (default port 53)
dig @localhost google.com

# Test with custom port
dig @localhost -p 5353 google.com

# Test different record types
dig @localhost -p 5353 google.com MX
dig @localhost -p 5353 cloudflare.com AAAA

# Test with nslookup
nslookup google.com localhost
```

## Docker Compose Configuration

The `docker-compose.yml` includes:
- Automatic restart policy
- UDP port 53 mapping
- Volume mount for persistent log storage (`./logs:/logs`)
- Multiple upstream DNS servers for redundancy
- Automatic file logging to `/logs/dns-server.log`

## Log File Location

- **Docker Compose**: Logs are saved to `./logs/dns-server.log` on the host
- **Manual Docker**: Mount a volume to `/logs` in the container
- **Direct Go execution**: Specify path with `-log` flag

## Requirements

- Go 1.21 or later
- Docker and Docker Compose (for containerized deployment)
- Root privileges (for binding to port 53)

## Dependencies

- `github.com/miekg/dns`: High-performance DNS library for Go

## License

This project is open source and available under the MIT License. 