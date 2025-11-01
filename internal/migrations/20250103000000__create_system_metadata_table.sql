-- Migration: Create system_metadata table
-- Timestamp: 20250103000000
-- Description: Creates the system_metadata table for storing system-level information like DNS server start time

CREATE TABLE IF NOT EXISTS system_metadata (
    id SERIAL PRIMARY KEY,
    metadata_key VARCHAR(100) UNIQUE NOT NULL,
    metadata_value TEXT NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create index for efficient key lookups
CREATE INDEX IF NOT EXISTS idx_system_metadata_key ON system_metadata(metadata_key);

