-- Migration 017: Add transformation configuration history
-- Description: Store history of transformation configurations (deployed and saved)
-- Date: 2025-11-21

-- Transformation configuration history table
-- Stores historical configurations for transformations
-- Automatically saves when deploying, keeps last 10 versions per dataset
CREATE TABLE IF NOT EXISTS control_transformation_history (
    id BIGSERIAL PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    dataset_id TEXT NOT NULL,
    config JSONB NOT NULL,
    deployed BOOLEAN DEFAULT FALSE,
    deployed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by TEXT,
    label TEXT,
    CONSTRAINT fk_transformation_history_dataset
        FOREIGN KEY (dataset_id)
        REFERENCES control_datasets(id)
        ON DELETE CASCADE
);

-- Indexes for querying history
CREATE INDEX IF NOT EXISTS idx_transformation_history_tenant_dataset
    ON control_transformation_history(tenant_id, dataset_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_transformation_history_deployed
    ON control_transformation_history(tenant_id, dataset_id, deployed, created_at DESC);

-- Add comment
COMMENT ON TABLE control_transformation_history IS 'Historical transformation configurations for version control and rollback';
COMMENT ON COLUMN control_transformation_history.config IS 'Array of filter configurations in JSONB format';
COMMENT ON COLUMN control_transformation_history.deployed IS 'True if this config was deployed to piper';
COMMENT ON COLUMN control_transformation_history.deployed_at IS 'Timestamp when config was deployed (NULL if not deployed)';
