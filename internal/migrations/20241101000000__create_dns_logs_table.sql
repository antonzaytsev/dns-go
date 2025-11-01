-- Migration: Create dns_logs table
-- Timestamp: 20241101000000
-- Description: Creates the main DNS logs table for storing DNS query/response data

CREATE TABLE IF NOT EXISTS dns_logs (
    id SERIAL PRIMARY KEY,
    uuid VARCHAR(255) UNIQUE NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    client_ip INET NOT NULL,
    query VARCHAR(255) NOT NULL,
    query_type VARCHAR(10) NOT NULL,
    query_id INTEGER,
    status VARCHAR(50) NOT NULL,
    cache_hit BOOLEAN DEFAULT FALSE,
    duration_ms DOUBLE PRECISION,
    response_upstream VARCHAR(255),
    response_rcode VARCHAR(10),
    response_answer_count INTEGER,
    response_rtt_ms DOUBLE PRECISION,
    upstreams JSONB,
    answers JSONB,
    ip_addresses INET[],
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_dns_logs_timestamp ON dns_logs(timestamp);
CREATE INDEX IF NOT EXISTS idx_dns_logs_timestamp_date ON dns_logs(DATE_TRUNC('day', timestamp));
CREATE INDEX IF NOT EXISTS idx_dns_logs_timestamp_hour ON dns_logs(DATE_TRUNC('hour', timestamp));
CREATE INDEX IF NOT EXISTS idx_dns_logs_timestamp_minute ON dns_logs(DATE_TRUNC('minute', timestamp));
CREATE INDEX IF NOT EXISTS idx_dns_logs_client_ip ON dns_logs(client_ip);
CREATE INDEX IF NOT EXISTS idx_dns_logs_timestamp_client ON dns_logs(timestamp, client_ip);
CREATE INDEX IF NOT EXISTS idx_dns_logs_status ON dns_logs(status);
CREATE INDEX IF NOT EXISTS idx_dns_logs_cache_hit ON dns_logs(cache_hit);
CREATE INDEX IF NOT EXISTS idx_dns_logs_query_type ON dns_logs(query_type);
CREATE INDEX IF NOT EXISTS idx_dns_logs_uuid ON dns_logs(uuid);
CREATE INDEX IF NOT EXISTS idx_dns_logs_query ON dns_logs(query);
CREATE INDEX IF NOT EXISTS idx_dns_logs_response_upstream ON dns_logs(response_upstream);

