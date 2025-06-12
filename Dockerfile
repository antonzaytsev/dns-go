# Build stage
FROM golang:1.21-alpine AS builder

# Install git for version information
RUN apk add --no-cache git

WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build args for version information
ARG VERSION=unknown
ARG GIT_COMMIT=unknown
ARG BUILD_DATE=unknown

# Build the application with version information
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-X 'dns-go/pkg/version.Version=${VERSION}' \
              -X 'dns-go/pkg/version.GitCommit=${GIT_COMMIT}' \
              -X 'dns-go/pkg/version.BuildDate=${BUILD_DATE}' \
              -w -s" \
    -a -installsuffix cgo \
    -o dns-server \
    ./cmd/dns-server

# Production stage
FROM alpine:latest AS production

# Install ca-certificates and timezone data
RUN apk --no-cache add ca-certificates tzdata \
    && addgroup -g 1001 -S dns \
    && adduser -u 1001 -S dns -G dns

# Create logs directory
RUN mkdir -p /logs && chown dns:dns /logs

WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/dns-server .

# Change ownership to non-root user
RUN chown dns:dns /app/dns-server

# Switch to non-root user
USER dns

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD ./dns-server -help > /dev/null || exit 1

# Expose DNS port
EXPOSE 53/udp

# Default command
CMD ["./dns-server"] 