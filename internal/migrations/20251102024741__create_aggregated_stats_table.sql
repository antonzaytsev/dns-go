-- Create aggregated_stats table for caching dashboard statistics
CREATE TABLE IF NOT EXISTS aggregated_stats (
    id BIGSERIAL PRIMARY KEY,
    stats_type VARCHAR(50) NOT NULL UNIQUE,
    stats_data JSONB NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create index on stats_type for fast lookups
CREATE INDEX IF NOT EXISTS idx_aggregated_stats_type ON aggregated_stats(stats_type);

-- Create index on updated_at for tracking freshness
CREATE INDEX IF NOT EXISTS idx_aggregated_stats_updated_at ON aggregated_stats(updated_at);

