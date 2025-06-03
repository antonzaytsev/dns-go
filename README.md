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

The DNS server provides detailed logging for every request and response with **unique request tracking**:

- **Request UUID**: Each DNS request gets a unique 8-character identifier for easy correlation
- **Request Logging**: Client IP, query name, query type, request ID
- **Upstream Tracking**: Which upstream server was used for each query
- **Performance Metrics**: Response times (RTT and total duration)
- **Response Details**: Number of answers, response codes, individual answer records
- **Error Handling**: Failed upstream attempts and timeout logging
- **Dual Output**: Logs to both console and file simultaneously (when file logging enabled)

### Log Format Examples

Each request is tracked with a unique UUID that appears in all related log entries:

```
[REQUEST] UUID: 815760ac | Client: 192.168.1.100:12345 | Query: google.com. | Type: A | ID: 12345
[UPSTREAM] UUID: 815760ac | Trying upstream 1/4: 8.8.8.8:53
[RESPONSE] UUID: 815760ac | Success | Upstream: 8.8.8.8:53 | RTT: 45ms | Total: 46ms | Answers: 6 | Rcode: NOERROR | ID: 12345
[ANSWER] UUID: 815760ac | google.com. 300 IN A 172.217.164.110
```

**Multiple record responses:**
```
[REQUEST] UUID: b4749e73 | Client: 192.168.1.100:12346 | Query: stackoverflow.com. | Type: TXT | ID: 9539
[UPSTREAM] UUID: b4749e73 | Trying upstream 1/4: 8.8.8.8:53
[RESPONSE] UUID: b4749e73 | Success | Upstream: 8.8.8.8:53 | RTT: 60ms | Total: 61ms | Answers: 15 | Rcode: NOERROR | ID: 9539
[ANSWER] UUID: b4749e73 | stackoverflow.com. 300 IN TXT "google-site-verification=..."
[ANSWER] UUID: b4749e73 | stackoverflow.com. 300 IN TXT "v=spf1 ip4:198.252.206.71..."
... (13 more TXT records with same UUID)
```

**Error responses:**
```
[REQUEST] UUID: df61b1b0 | Client: 192.168.1.100:12347 | Query: nonexistent.invalid. | Type: A | ID: 22938
[UPSTREAM] UUID: df61b1b0 | Trying upstream 1/4: 8.8.8.8:53
[RESPONSE] UUID: df61b1b0 | Success | Upstream: 8.8.8.8:53 | RTT: 42ms | Total: 42ms | Answers: 0 | Rcode: NXDOMAIN | ID: 22938
```

### Request Correlation Benefits

- **Easy Filtering**: `grep "UUID: 815760ac" logs/dns-server.log` shows all entries for one request
- **Performance Analysis**: Track end-to-end timing for specific queries
- **Debugging**: Follow the complete lifecycle of problematic requests
- **Monitoring**: Identify patterns in failed requests or slow responses

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