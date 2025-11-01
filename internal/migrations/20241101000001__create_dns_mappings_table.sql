-- Migration: Create dns_mappings table
-- Timestamp: 20241101000001
-- Description: Creates the DNS mappings table for storing custom domain-to-IP mappings

CREATE TABLE IF NOT EXISTS dns_mappings (
    id SERIAL PRIMARY KEY,
    domain VARCHAR(255) UNIQUE NOT NULL,
    ip_address VARCHAR(45) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create index for efficient domain lookups
CREATE INDEX IF NOT EXISTS idx_dns_mappings_domain ON dns_mappings(domain);

