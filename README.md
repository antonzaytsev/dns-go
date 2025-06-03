# Simple DNS Proxy Server

A lightweight DNS proxy server written in Go that forwards queries to upstream DNS servers (Google DNS and Cloudflare DNS by default).

## Features

- **Simple and Fast**: Minimal overhead DNS proxy
- **Multiple Upstreams**: Supports multiple upstream DNS servers with failover
- **Configurable**: Command-line flags for easy configuration
- **Docker Ready**: Includes Docker and Docker Compose support
- **Logging**: Built-in query logging for monitoring

## Default Upstream Servers

- **Google DNS**: 8.8.8.8, 8.8.4.4
- **Cloudflare DNS**: 1.1.1.1, 1.0.0.1

## Usage

### Option 1: Run with Go (Development)

```bash
# Install dependencies
go mod tidy

# Run with default settings (listens on all interfaces, port 53)
sudo go run main.go

# Run with custom settings
go run main.go -listen 127.0.0.1 -port 5353 -upstreams "8.8.8.8:53,1.1.1.1:53"
```

### Option 2: Build and Run Binary

```bash
# Build the binary
go build -o dns-server main.go

# Run (requires sudo for port 53)
sudo ./dns-server

# Run on non-privileged port
./dns-server -port 5353
```

### Option 3: Docker Compose (Recommended)

```bash
# Build and start the DNS server
docker-compose up -d

# View logs
docker-compose logs -f

# Stop the server
docker-compose down
```

### Option 4: Docker (Manual)

```bash
# Build the image
docker build -t dns2-server .

# Run the container
docker run -d --name dns2 -p 53:53/udp dns2-server
```

## Configuration

### Command Line Flags

- `-listen`: Listen address (default: `0.0.0.0`)
- `-port`: Listen port (default: `53`)
- `-upstreams`: Comma-separated upstream DNS servers (default: `8.8.8.8:53,1.1.1.1:53`)

### Examples

```bash
# Listen only on localhost, port 5353
./dns-server -listen 127.0.0.1 -port 5353

# Use only Cloudflare DNS
./dns-server -upstreams "1.1.1.1:53,1.0.0.1:53"

# Use custom DNS servers
./dns-server -upstreams "9.9.9.9:53,149.112.112.112:53"
```

## Testing

Test your DNS server with `dig` or `nslookup`:

```bash
# Test with dig (default port 53)
dig @localhost google.com

# Test with custom port
dig @localhost -p 5353 google.com

# Test with nslookup
nslookup google.com localhost
```

## Docker Compose Configuration

The `docker-compose.yml` includes:
- Automatic restart policy
- UDP port 53 mapping
- Custom network for isolation
- Multiple upstream DNS servers for redundancy

## Requirements

- Go 1.21 or later
- Docker and Docker Compose (for containerized deployment)
- Root privileges (for binding to port 53)

## Dependencies

- `github.com/miekg/dns`: High-performance DNS library for Go

## License

This project is open source and available under the MIT License. 