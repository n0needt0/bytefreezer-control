-- Migration 013: Add transformation stats and preview tables
-- Description: Store transformation statistics and preview data submitted by piper
-- Date: 2025-11-17

-- Transformation statistics table
-- Stores current stats for each tenant/dataset transformation
-- Updated by piper whenever transformation stats change
CREATE TABLE IF NOT EXISTS piper_transformation_stats (
    tenant_id TEXT NOT NULL,
    dataset_id TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT false,
    filter_count INTEGER NOT NULL DEFAULT 0,
    total_processed BIGINT NOT NULL DEFAULT 0,
    success_count BIGINT NOT NULL DEFAULT 0,
    error_count BIGINT NOT NULL DEFAULT 0,
    skipped_count BIGINT NOT NULL DEFAULT 0,
    avg_rows_per_sec DOUBLE PRECISION NOT NULL DEFAULT 0,
    last_processed TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (tenant_id, dataset_id)
);

-- Index for querying stats
CREATE INDEX IF NOT EXISTS idx_transformation_stats_tenant ON piper_transformation_stats(tenant_id);
CREATE INDEX IF NOT EXISTS idx_transformation_stats_updated ON piper_transformation_stats(updated_at);

-- Transformation preview data table
-- Stores recent samples of transformed records
-- Updated by piper to show users what their transformations are producing
CREATE TABLE IF NOT EXISTS piper_transformation_preview (
    id BIGSERIAL PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    dataset_id TEXT NOT NULL,
    line_number INTEGER NOT NULL,
    original_data JSONB NOT NULL,
    transformed_data JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes for querying preview data
CREATE INDEX IF NOT EXISTS idx_transformation_preview_tenant_dataset ON piper_transformation_preview(tenant_id, dataset_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_transformation_preview_created ON piper_transformation_preview(created_at);

-- Add comment
COMMENT ON TABLE piper_transformation_stats IS 'Transformation statistics submitted by piper for each tenant/dataset';
COMMENT ON TABLE piper_transformation_preview IS 'Preview samples of transformed data submitted by piper';
