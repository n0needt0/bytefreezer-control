-- Migration 018: Transformation statistics and active configurations
-- Create table for storing active transformation configurations
CREATE TABLE IF NOT EXISTS control_active_transformations (
    tenant_id TEXT NOT NULL,
    dataset_id TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    filters JSONB NOT NULL DEFAULT '[]'::jsonb,
    version TEXT NOT NULL,
    activated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, dataset_id),
    CONSTRAINT fk_active_transformation_dataset
        FOREIGN KEY (dataset_id)
        REFERENCES control_datasets(id)
        ON DELETE CASCADE
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_active_transformations_tenant ON control_active_transformations(tenant_id);
CREATE INDEX IF NOT EXISTS idx_active_transformations_dataset ON control_active_transformations(dataset_id);
CREATE INDEX IF NOT EXISTS idx_active_transformations_enabled ON control_active_transformations(enabled);

-- Create table for storing transformation statistics
CREATE TABLE IF NOT EXISTS control_transformation_stats (
    id BIGSERIAL PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    dataset_id TEXT NOT NULL,
    total_processed BIGINT NOT NULL DEFAULT 0,
    success_count BIGINT NOT NULL DEFAULT 0,
    error_count BIGINT NOT NULL DEFAULT 0,
    skipped_count BIGINT NOT NULL DEFAULT 0,
    avg_rows_per_sec DOUBLE PRECISION NOT NULL DEFAULT 0,
    last_error TEXT,
    last_processed TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    reported_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_transformation_stats_dataset
        FOREIGN KEY (dataset_id)
        REFERENCES control_datasets(id)
        ON DELETE CASCADE
);

-- Create indexes for stats
CREATE INDEX IF NOT EXISTS idx_transformation_stats_tenant_dataset ON control_transformation_stats(tenant_id, dataset_id);
CREATE INDEX IF NOT EXISTS idx_transformation_stats_reported_at ON control_transformation_stats(reported_at DESC);

-- Create trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_active_transformation_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_active_transformation_timestamp
    BEFORE UPDATE ON control_active_transformations
    FOR EACH ROW
    EXECUTE FUNCTION update_active_transformation_timestamp();
