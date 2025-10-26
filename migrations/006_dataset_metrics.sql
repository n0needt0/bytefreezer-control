-- Migration: 006_dataset_metrics
-- Description: Create dataset_metrics table for tracking processing metrics across components
-- Author: Claude (fixing missing migration)
-- Date: 2025-10-26

-- Create dataset_metrics table
CREATE TABLE IF NOT EXISTS dataset_metrics (
    id BIGSERIAL PRIMARY KEY,
    tenant_id VARCHAR(255) NOT NULL,
    dataset_id VARCHAR(255) NOT NULL,
    recorded_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    component VARCHAR(50) NOT NULL CHECK (component IN ('proxy', 'receiver', 'piper', 'packer', 'control')),

    -- Metrics data
    input_bytes BIGINT NOT NULL DEFAULT 0 CHECK (input_bytes >= 0),
    output_bytes BIGINT NOT NULL DEFAULT 0 CHECK (output_bytes >= 0),
    lines_processed BIGINT NOT NULL DEFAULT 0 CHECK (lines_processed >= 0),
    error_count BIGINT NOT NULL DEFAULT 0 CHECK (error_count >= 0),

    -- Time buckets for aggregation
    hour_bucket TIMESTAMP WITH TIME ZONE NOT NULL,
    day_bucket TIMESTAMP WITH TIME ZONE NOT NULL,

    -- Custom metrics as JSONB
    custom_metrics JSONB,

    -- Indexes for efficient querying
    CONSTRAINT fk_tenant_dataset FOREIGN KEY (tenant_id, dataset_id)
        REFERENCES control_datasets(tenant_id, id) ON DELETE CASCADE
);

-- Create indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_dataset_metrics_tenant_dataset
    ON dataset_metrics(tenant_id, dataset_id);

CREATE INDEX IF NOT EXISTS idx_dataset_metrics_recorded_at
    ON dataset_metrics(recorded_at DESC);

CREATE INDEX IF NOT EXISTS idx_dataset_metrics_component
    ON dataset_metrics(component);

CREATE INDEX IF NOT EXISTS idx_dataset_metrics_hour_bucket
    ON dataset_metrics(hour_bucket);

CREATE INDEX IF NOT EXISTS idx_dataset_metrics_day_bucket
    ON dataset_metrics(day_bucket);

-- Composite index for common query patterns
CREATE INDEX IF NOT EXISTS idx_dataset_metrics_tenant_dataset_time
    ON dataset_metrics(tenant_id, dataset_id, recorded_at DESC);

CREATE INDEX IF NOT EXISTS idx_dataset_metrics_tenant_dataset_component_time
    ON dataset_metrics(tenant_id, dataset_id, component, recorded_at DESC);

-- Insert migration record
INSERT INTO control_migrations (version, name, applied_at)
VALUES (6, 'dataset_metrics', NOW())
ON CONFLICT (version) DO NOTHING;

-- Add comment for documentation
COMMENT ON TABLE dataset_metrics IS 'Stores processing metrics from all ByteFreezer components (proxy, receiver, piper, packer, control) for monitoring and analytics';
COMMENT ON COLUMN dataset_metrics.component IS 'Component that recorded the metric: proxy, receiver, piper, packer, or control';
COMMENT ON COLUMN dataset_metrics.input_bytes IS 'Number of input bytes processed';
COMMENT ON COLUMN dataset_metrics.output_bytes IS 'Number of output bytes after processing/compression';
COMMENT ON COLUMN dataset_metrics.lines_processed IS 'Number of log lines processed';
COMMENT ON COLUMN dataset_metrics.error_count IS 'Number of errors encountered during processing';
COMMENT ON COLUMN dataset_metrics.hour_bucket IS 'Hourly time bucket for aggregation queries';
COMMENT ON COLUMN dataset_metrics.day_bucket IS 'Daily time bucket for aggregation queries';
COMMENT ON COLUMN dataset_metrics.custom_metrics IS 'Additional custom metrics in JSON format';
