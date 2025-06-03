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

The DNS server provides detailed logging for every request and response:

- **Request Logging**: Client IP, query name, query type, request ID
- **Upstream Tracking**: Which upstream server was used for each query
- **Performance Metrics**: Response times (RTT and total duration)
- **Response Details**: Number of answers, response codes, individual answer records
- **Error Handling**: Failed upstream attempts and timeout logging
- **Dual Output**: Logs to both console and file simultaneously (when file logging enabled)

### Log Format Examples

```
[REQUEST] Client: 192.168.1.100:12345 | Query: google.com. | Type: A | ID: 12345
[UPSTREAM] Trying upstream 1/4: 8.8.8.8:53
[RESPONSE] Success | Upstream: 8.8.8.8:53 | RTT: 45ms | Total: 46ms | Answers: 6 | Rcode: NOERROR | ID: 12345
[ANSWER] google.com. 300 IN A 172.217.164.110
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