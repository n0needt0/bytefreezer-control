-- Migration: Receiver Throughput Tracking
-- Tracks aggregated receiver metrics per minute per tenant/dataset
-- Provides bytes/min, lines/min, requests/min visibility

CREATE TABLE IF NOT EXISTS receiver_throughput (
    id BIGSERIAL PRIMARY KEY,
    account_id VARCHAR(50),
    tenant_id VARCHAR(50) NOT NULL,
    dataset_id VARCHAR(50) NOT NULL,
    minute_timestamp TIMESTAMP NOT NULL,

    -- Aggregated metrics for this minute bucket
    request_count INT NOT NULL DEFAULT 0,
    bytes_received BIGINT NOT NULL DEFAULT 0,
    lines_received BIGINT NOT NULL DEFAULT 0,
    bytes_stored BIGINT NOT NULL DEFAULT 0,

    -- Success/failure tracking
    success_count INT NOT NULL DEFAULT 0,
    failure_count INT NOT NULL DEFAULT 0,

    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    -- Unique constraint for upserts
    UNIQUE(tenant_id, dataset_id, minute_timestamp)
);

-- Indexes for efficient queries
CREATE INDEX idx_receiver_throughput_tenant_dataset_time
    ON receiver_throughput(tenant_id, dataset_id, minute_timestamp DESC);

CREATE INDEX idx_receiver_throughput_account
    ON receiver_throughput(account_id);

CREATE INDEX idx_receiver_throughput_time
    ON receiver_throughput(minute_timestamp);

-- Cleanup function for 7-day retention
CREATE OR REPLACE FUNCTION cleanup_old_receiver_throughput() RETURNS void AS $$
BEGIN
    DELETE FROM receiver_throughput WHERE minute_timestamp < NOW() - INTERVAL '7 days';
    RAISE NOTICE 'Cleaned up receiver_throughput older than 7 days';
END;
$$ LANGUAGE plpgsql;

-- Add comment for documentation
COMMENT ON TABLE receiver_throughput IS 'Aggregated receiver webhook metrics per minute per tenant/dataset. Retention: 7 days.';
