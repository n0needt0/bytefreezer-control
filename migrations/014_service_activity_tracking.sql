-- Migration: Add service activity tracking
-- Purpose: Track real-time operations and recent activity across all services
-- Retention: 7 days for completed/failed operations

-- Table for tracking service operations (current and historical)
CREATE TABLE IF NOT EXISTS service_operations (
    id BIGSERIAL PRIMARY KEY,

    -- Service identification
    service_type VARCHAR(50) NOT NULL,  -- 'receiver', 'piper', 'packer'
    instance_id VARCHAR(100) NOT NULL,

    -- Scoping
    account_id VARCHAR(50),
    tenant_id VARCHAR(50),
    dataset_id VARCHAR(50),

    -- Operation details
    operation_type VARCHAR(50) NOT NULL, -- 'receiving', 'processing', 'converting', 'uploading', 'metadata_generation'
    operation_id VARCHAR(200),           -- Unique ID for this specific operation (job_id, etc)
    status VARCHAR(20) NOT NULL,         -- 'in_progress', 'completed', 'failed'

    -- Progress tracking
    progress_current BIGINT,
    progress_total BIGINT,
    progress_unit VARCHAR(20),           -- 'files', 'bytes', 'records', 'percent'
    progress_message TEXT,               -- Human-readable progress description

    -- Metrics
    input_bytes BIGINT,
    output_bytes BIGINT,
    records_processed BIGINT,
    error_count INTEGER DEFAULT 0,

    -- Additional details (JSON for flexibility)
    details JSONB,

    -- Timestamps
    started_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP DEFAULT NOW(),
    completed_at TIMESTAMP,

    created_at TIMESTAMP DEFAULT NOW()
);

-- Indexes for efficient queries
CREATE INDEX idx_service_operations_service ON service_operations(service_type, instance_id);
CREATE INDEX idx_service_operations_status ON service_operations(status, updated_at DESC);
CREATE INDEX idx_service_operations_tenant ON service_operations(tenant_id, dataset_id);
CREATE INDEX idx_service_operations_updated ON service_operations(updated_at DESC);
CREATE INDEX idx_service_operations_account ON service_operations(account_id, updated_at DESC);
CREATE INDEX idx_service_operations_operation_id ON service_operations(operation_id) WHERE operation_id IS NOT NULL;

-- Table for aggregated service metrics (last N minutes)
CREATE TABLE IF NOT EXISTS service_metrics_rollup (
    id BIGSERIAL PRIMARY KEY,

    service_type VARCHAR(50) NOT NULL,
    instance_id VARCHAR(100) NOT NULL,
    account_id VARCHAR(50),
    tenant_id VARCHAR(50),
    dataset_id VARCHAR(50),

    -- Time bucket (15-minute intervals)
    time_bucket TIMESTAMP NOT NULL,

    -- Aggregated metrics
    operations_started INTEGER DEFAULT 0,
    operations_completed INTEGER DEFAULT 0,
    operations_failed INTEGER DEFAULT 0,

    total_input_bytes BIGINT DEFAULT 0,
    total_output_bytes BIGINT DEFAULT 0,
    total_records BIGINT DEFAULT 0,
    total_errors INTEGER DEFAULT 0,

    avg_duration_seconds NUMERIC(10,2),

    created_at TIMESTAMP DEFAULT NOW(),

    UNIQUE(service_type, instance_id, account_id, tenant_id, dataset_id, time_bucket)
);

CREATE INDEX idx_service_metrics_rollup_time ON service_metrics_rollup(time_bucket DESC);
CREATE INDEX idx_service_metrics_rollup_service ON service_metrics_rollup(service_type, time_bucket DESC);

-- Function to cleanup old completed operations (7 days retention)
CREATE OR REPLACE FUNCTION cleanup_old_service_operations() RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM service_operations
    WHERE status IN ('completed', 'failed')
      AND completed_at < NOW() - INTERVAL '7 days';

    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Function to cleanup old metrics rollup (30 days retention)
CREATE OR REPLACE FUNCTION cleanup_old_service_metrics() RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM service_metrics_rollup
    WHERE time_bucket < NOW() - INTERVAL '30 days';

    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Comments
COMMENT ON TABLE service_operations IS 'Tracks current and recent operations across all services for activity monitoring';
COMMENT ON TABLE service_metrics_rollup IS 'Aggregated metrics in 15-minute buckets for historical analysis';
COMMENT ON COLUMN service_operations.operation_id IS 'Unique identifier for the operation (e.g., job_id, request_id)';
COMMENT ON COLUMN service_operations.progress_message IS 'Human-readable progress like "Converting 500/1000 files"';
