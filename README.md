# DNS Proxy Server

A high-performance DNS proxy server written in Go with caching, concurrent upstream queries, and comprehensive monitoring.

## Quick Start

### Development Mode
```bash
# Run on localhost:5053 with debug logging
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
  -cache-size=50000 \
  -cache-ttl=5m \
  -max-concurrent=200
```

### Docker Deployment
```bash
# Quick start with Docker Compose
make docker-run

# Or manually
docker build -t dns-go .
docker run -d -p 53:53/udp -v $(pwd)/logs:/logs dns-go
```

## Key Features

- **🚀 High Performance**: DNS response caching (~95% fewer upstream queries)
- **⚡ Concurrent Queries**: Parallel upstream requests for faster failover
- **🏥 Health Monitoring**: Automatic upstream server health tracking with circuit breaker
- **📊 Rate Limiting**: Configurable concurrent request limiting
- **🔍 Dual Logging**: Clean JSON logs for analysis + human-readable logs for monitoring
- **🐳 Production Ready**: Docker support, graceful shutdown, comprehensive configuration
- **🛡️ Resilient**: Automatic failover and recovery mechanisms

## Configuration Options

```bash
Usage of ./dns-server:
  -cache-size int
        DNS cache size (default 10000)
  -cache-ttl duration
        DNS cache TTL (default 5m0s)
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
  -cache-size=100000 \
  -cache-ttl=10m \
  -max-concurrent=500 \
  -timeout=3s \
  -log-level=warn

# Development/debugging
./dns-server \
  -port=5353 \
  -log-level=debug \
  -cache-size=1000
```

### Testing
```bash
# Test basic functionality
dig @localhost google.com
dig @localhost -p 5353 cloudflare.com AAAA

# Performance test
time dig @localhost google.com  # Should be fast on second request (cache hit)
```

## Architecture

```
dns-go/
├── internal/
│   ├── cache/          # DNS response caching with TTL
│   ├── config/         # Configuration management
│   ├── logging/        # Structured logging
│   ├── types/          # Shared data structures
│   └── upstream/       # Upstream server management with health checks
├── main.go            # Main application
├── Makefile           # Development tasks
└── docker-compose.yml # Container deployment
```

## Performance Features

### DNS Caching
- **Thread-safe**: Concurrent request handling
- **TTL-aware**: Respects DNS record TTL values
- **Automatic cleanup**: Background cache maintenance
- **Configurable**: Adjustable cache size and default TTL

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
2025/06/10 16:14:13.345382 [INFO] DNS Proxy Server starting config=map[cache_size:10000 ...]
2025/06/10 16:14:13.345530 [INFO] Starting DNS server address=0.0.0.0 port=5053
2025/06/10 16:14:24.355900 REQ 1ebd3f6b from [::1]:62152: A google.com. -> success via 8.8.8.8:53 (39.51ms)
2025/06/10 16:14:33.673302 REQ d85e7c92 from [::1]:54813: A google.com. -> CACHE HIT (0.10ms)
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

# Cache hit rate
grep -c '"cache_hit":true' logs/dns-requests.log

# Slow queries (>100ms)
jq 'select(.total_duration_ms > 100)' logs/dns-requests.log

# Error analysis
jq 'select(.status != "success")' logs/dns-requests.log

# Request pattern analysis
grep "REQ.*CACHE HIT" logs/dns-server.log | wc -l

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
internal/cache/cache_test.go
internal/config/config_test.go

# Run specific package tests
go test -v ./internal/cache
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
      "-cache-size", "50000"
    ]
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
- **Memory**: ~50MB base + cache size
- **Permissions**: Root/sudo for port 53

## Dependencies

- `github.com/miekg/dns`: High-performance DNS library

## License

MIT License - see project files for details. 