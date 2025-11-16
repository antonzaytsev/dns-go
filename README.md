# DNS Proxy Server

A high-performance DNS proxy server written in Go with caching, concurrent upstream queries, comprehensive monitoring, and a modern web dashboard.

## Quick Start

### Development Mode
```bash
# Run DNS server on localhost:5053 with debug logging
make run-dns-dev

# Run web dashboard on localhost:8080
make run-web-dev

# Run both DNS server and web dashboard
make run-dev
```

### Production Mode
```bash
# Build and run with optimized settings
make build
./dns-server \
  -listen=0.0.0.0 \
  -port=53 \
  -upstreams="8.8.8.8:53,1.1.1.1:53,1.0.0.1:53" \
  -log=./logs/dns-requests.log \
  -log-level=info \
  -max-concurrent=200
```

### Docker Deployment
```bash
# Quick start with Docker Compose (DNS server + Web dashboard)
make docker-run

# Access the web dashboard at http://localhost:8080
# DNS server available on port 53

# Or manually
docker build -t dns-go .
docker run -d -p 53:53/udp -v $(pwd)/logs:/logs dns-go
```

## Key Features

- **🚀 High Performance**: Concurrent upstream queries for fast resolution
- **⚡ Concurrent Queries**: Parallel upstream requests for faster failover
- **🏥 Health Monitoring**: Automatic upstream server health tracking with circuit breaker
- **📊 Rate Limiting**: Configurable concurrent request limiting
- **🔍 Dual Logging**: Clean JSON logs for analysis + human-readable logs for monitoring
- **📈 Web Dashboard**: Real-time metrics, charts, and monitoring interface
- **🐳 Production Ready**: Docker support, graceful shutdown, comprehensive configuration
- **🛡️ Resilient**: Automatic failover and recovery mechanisms

## Web Dashboard

The DNS server includes a modern web dashboard for real-time monitoring and analytics.

### Features
- **📊 Real-time Metrics**: Live statistics with auto-refresh
- **📈 Interactive Charts**: Request patterns over time
- **👥 Client Analytics**: Top clients and request patterns
- **🌐 Upstream Monitoring**: Server health and performance
- **📝 Requests**: Live request feed with details
- **🎨 Modern UI**: Responsive design with dark mode support

### Access
- **URL**: http://localhost:8080 (default)
- **API**: http://localhost:8080/api/metrics
- **Health Check**: http://localhost:8080/api/health

### Dashboard Sections
1. **Overview Cards**: Total requests, success rate, response times
2. **Time Series Charts**: Requests per minute/hour with interactive graphs
3. **Query Types**: Distribution of DNS query types (A, AAAA, MX, etc.)
4. **Top Clients**: Most active clients with success rates
5. **Upstream Servers**: Health status and performance metrics
6. **Requests**: Live feed of DNS queries with details

### Configuration
```bash
# Web dashboard configuration
./web-dashboard -port=8080 -log-file=./logs/dns-requests.log

# Environment variables
export WEB_PORT=8080
export DNS_LOG_FILE=./logs/dns-requests.log
```

## DNS Server Configuration

```bash
Usage of ./dns-server:
  -custom-dns string
        Custom DNS mappings in format: domain1=ip1,domain2=ip2 (e.g., server.local=192.168.0.30)
  -listen string
        Listen address (default "0.0.0.0")
  -log string
        Log file path (optional)
  -log-level string
        Log level (debug, info, warn, error) (default "info")
  -max-concurrent int
        Maximum concurrent requests (default 100)
  -port string
        Listen port (default "53")
  -retry-attempts int
        Number of retry attempts (default 3)
  -timeout duration
        Upstream server timeout (default 5s)
  -upstreams string
        Comma-separated list of upstream DNS servers (default "8.8.8.8:53,1.1.1.1:53")
```

### Custom DNS Configuration

The DNS server supports custom local DNS mappings through a configuration file. This is useful for resolving internal network devices or local services.

#### Configuration File Setup

1. Create a `custom-dns.json` file in the same directory as the DNS server binary
2. Add your custom domain mappings in JSON format
3. The file will be automatically loaded on server startup

**Example `custom-dns.json`:**
```json
{
  "mappings": {
    "server.local": "192.168.0.30",
    "api.local": "192.168.0.31",
    "db.local": "192.168.0.32",
    "nas.local": "192.168.0.50",
    "router.local": "192.168.0.1"
  }
}
```

#### Features

- **File-based Configuration**: Custom mappings loaded from `custom-dns.json`
- **Automatic Loading**: No restart required when file is added
- **Priority Resolution**: Custom mappings are resolved before upstream queries
- **IPv4 Support**: Currently supports A record (IPv4) resolution
- **Domain Normalization**: Automatically handles domains with or without trailing dots
- **Git Ignored**: Configuration file is automatically ignored by version control

#### Usage Examples

```bash
# With custom DNS configuration file
./dns-server  # Automatically loads custom-dns.json if present

# With command line mappings (for testing)
./dns-server --custom-dns="test.local=127.0.0.1,dev.local=192.168.1.100"

# Combined with other options
./dns-server \
  -port=53 \
  -log=./logs/dns-requests.log \
  -custom-dns="server.local=192.168.0.30"
```

#### Testing Custom DNS

```bash
# Test your custom domain resolution
dig @localhost server.local

# Should return your configured IP address
nslookup server.local 127.0.0.1
```

**Note**: Custom DNS configuration file (`custom-dns.json`) takes precedence over command-line mappings.

## Usage Examples

### Basic Usage
```bash
# Console logging only
./dns-server

# With file logging
./dns-server -log=./logs/dns-requests.log

# Custom port and upstreams
./dns-server -port=5353 -upstreams="1.1.1.1:53,9.9.9.9:53"
```

### Advanced Configuration
```bash
# High-performance setup
./dns-server \
  -max-concurrent=500 \
  -timeout=3s \
  -log-level=warn

# Development/debugging
./dns-server \
  -port=5353 \
  -log-level=debug \
```

### Testing
```bash
# Test basic functionality
dig @localhost google.com
dig @localhost -p 5353 cloudflare.com AAAA

# Performance test
time dig @localhost google.com
```

## Architecture

```
dns-go/
├── cmd/
│   ├── dns-server/     # DNS server main application
│   └── web-dashboard/  # Web dashboard main application
├── internal/
│   ├── config/         # Configuration management
│   ├── logging/        # Structured logging
│   ├── metrics/        # Metrics collection and aggregation
│   ├── monitor/        # Log file monitoring
│   ├── types/          # Shared data structures
│   ├── upstream/       # Upstream server management with health checks
│   └── webserver/      # HTTP server and dashboard UI
├── pkg/
│   └── version/        # Version information
├── Makefile           # Development tasks
└── docker-compose.yml # Container deployment
```

## Performance Features

### DNS Caching
- **Thread-safe**: Concurrent request handling
- **TTL-aware**: Respects DNS record TTL values

### Concurrent Upstream Queries
- **Parallel requests**: Query multiple upstreams simultaneously
- **First-success**: Return first successful response
- **Health tracking**: Monitor upstream server performance
- **Circuit breaker**: Automatic failover for unhealthy servers

### Rate Limiting & Resource Management
- **Request limiting**: Prevent resource exhaustion
- **Graceful degradation**: SERVFAIL when limits exceeded
- **Memory efficient**: Controlled memory usage patterns
- **CPU optimized**: Atomic operations and efficient algorithms

## Logging & Monitoring

### Dual Logging System
The DNS server creates two separate log files for different purposes:

#### 1. Human-Readable Logs (`dns-server.log`)
```
2025/06/10 16:14:13.345382 [INFO] DNS Proxy Server starting config=map[...]
2025/06/10 16:14:13.345530 [INFO] Starting DNS server address=0.0.0.0 port=5053
2025/06/10 16:14:24.355900 REQ 1ebd3f6b from [::1]:62152: A google.com. -> success via 8.8.8.8:53 (39.51ms)
2025/06/10 16:14:33.673302 REQ d85e7c92 from [::1]:54813: A google.com. -> success via 8.8.8.8:53 (12.5ms)
```

#### 2. Clean JSON Logs (`dns-requests.log`)
```json
{
  "timestamp": "2025-06-10T16:19:45.298345+03:00",
  "uuid": "5d58d23a",
  "request": {
    "client": "[::1]:61481",
    "query": "example.com.",
    "type": "A",
    "id": 40119
  },
  "response": {
    "upstream": "8.8.8.8:53",
    "rcode": "NOERROR",
    "answer_count": 6,
    "rtt_ms": 37.1
  },
  "status": "success",
  "total_duration_ms": 37.5
}
```

### Log Analysis
```bash
# Real-time human-readable monitoring
tail -f logs/dns-server.log

# Real-time JSON monitoring  
tail -f logs/dns-requests.log | jq .


# Slow queries (>100ms)
jq 'select(.total_duration_ms > 100)' logs/dns-requests.log

# Error analysis
jq 'select(.status != "success")' logs/dns-requests.log

# Request pattern analysis

# Filter by timestamp (last hour)
jq --arg hour_ago "$(date -d '1 hour ago' -Iseconds)" 'select(.timestamp > $hour_ago)' logs/dns-requests.log

# Query response time statistics
jq -r '.total_duration_ms' logs/dns-requests.log | awk '{sum+=$1; count++} END {print "Avg response time:", sum/count, "ms"}'
```

## Development

### Available Make Commands
```bash
make help          # Show all available commands
make build         # Build binary
make run-dev       # Development mode
make test          # Run tests
make fmt           # Format code
make docker-build  # Build Docker image
make clean         # Clean artifacts
```

### Adding Tests
```bash
# Create test files in internal packages
internal/config/config_test.go

# Run specific package tests
```

## Production Deployment

### System Service (systemd)
```ini
[Unit]
Description=DNS Proxy Server
After=network.target

[Service]
Type=simple
User=dns-server
ExecStart=/usr/local/bin/dns-server -log=/var/log/dns-server.log
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

### Docker Compose (Recommended)
```yaml
services:
  dns-server:
    build: .
    ports:
      - "53:53/udp"
    volumes:
      - ./logs:/logs
    restart: unless-stopped
    command: [
      "./dns-server",
      "-log", "/logs/dns-requests.log",
      "-log-level", "info",
    ]

  web-dashboard:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - ./logs:/logs
    environment:
      - WEB_PORT=8080
      - DNS_LOG_FILE=/logs/dns-requests.log
    command: [
      "./web-dashboard",
      "-port", "8080",
      "-log-file", "/logs/dns-requests.log"
    ]
    depends_on:
      - dns-server
    restart: unless-stopped
```

**Note**: This creates both `./logs/dns-requests.log` (JSON) and `./logs/dns-server.log` (readable) files.

### Health Monitoring
```bash
# Check server status
dig @localhost health.check

# Monitor upstream health
grep "upstream.*state" /var/log/dns-server.log

# Performance metrics
grep "rtt_ms\|total_duration_ms" /var/log/dns-server.log
```

## Requirements

- **Go**: Version 1.21 or later
- **System**: Linux/macOS/Windows
- **Network**: UDP port 53 (or custom port)
- **Memory**: ~50MB base
- **Permissions**: Root/sudo for port 53

## Dependencies

- `github.com/miekg/dns`: High-performance DNS library

## License

MIT License - see project files for details. 