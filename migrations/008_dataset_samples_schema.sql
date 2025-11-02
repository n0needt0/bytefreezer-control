-- Migration: 008_dataset_samples_schema
-- Description: Create tables for storing dataset data samples and inferred schema
-- Author: Claude
-- Date: 2025-11-02

-- Create dataset_samples table for storing data samples collected during pipeline processing
CREATE TABLE IF NOT EXISTS dataset_samples (
    id BIGSERIAL PRIMARY KEY,
    tenant_id VARCHAR(255) NOT NULL,
    dataset_id VARCHAR(255) NOT NULL,
    sample_type VARCHAR(10) NOT NULL CHECK (sample_type IN ('input', 'output')),
    line_number INT NOT NULL,
    sample_data JSONB NOT NULL,
    batch_id VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Foreign key constraint
    CONSTRAINT fk_samples_dataset FOREIGN KEY (dataset_id)
        REFERENCES control_datasets(id) ON DELETE CASCADE
);

-- Create indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_dataset_samples_tenant_dataset
    ON dataset_samples(tenant_id, dataset_id);

CREATE INDEX IF NOT EXISTS idx_dataset_samples_type
    ON dataset_samples(tenant_id, dataset_id, sample_type);

CREATE INDEX IF NOT EXISTS idx_dataset_samples_created_at
    ON dataset_samples(created_at DESC);

-- Composite index for common query patterns (latest samples by type)
CREATE INDEX IF NOT EXISTS idx_dataset_samples_tenant_dataset_type_created
    ON dataset_samples(tenant_id, dataset_id, sample_type, created_at DESC);

-- Create dataset_schema table for storing inferred schema (computed from samples)
CREATE TABLE IF NOT EXISTS dataset_schema (
    id BIGSERIAL PRIMARY KEY,
    tenant_id VARCHAR(255) NOT NULL,
    dataset_id VARCHAR(255) NOT NULL,
    schema_type VARCHAR(10) NOT NULL CHECK (schema_type IN ('input', 'output')),
    schema_fields JSONB NOT NULL,
    sample_count INT NOT NULL DEFAULT 0,
    last_updated TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Foreign key constraint
    CONSTRAINT fk_schema_dataset FOREIGN KEY (dataset_id)
        REFERENCES control_datasets(id) ON DELETE CASCADE,

    -- Unique constraint: one schema per tenant/dataset/type
    CONSTRAINT uk_schema_tenant_dataset_type UNIQUE (tenant_id, dataset_id, schema_type)
);

-- Create indexes for dataset_schema
CREATE INDEX IF NOT EXISTS idx_dataset_schema_tenant_dataset
    ON dataset_schema(tenant_id, dataset_id);

CREATE INDEX IF NOT EXISTS idx_dataset_schema_type
    ON dataset_schema(tenant_id, dataset_id, schema_type);

-- Insert migration record
INSERT INTO control_migrations (version, name, applied_at)
VALUES (8, 'dataset_samples_schema', NOW())
ON CONFLICT (version) DO NOTHING;

-- Add comments for documentation
COMMENT ON TABLE dataset_samples IS 'Stores data samples collected during pipeline processing (input and output). Keeps latest 10 samples per type for schema inference and user preview.';
COMMENT ON COLUMN dataset_samples.sample_type IS 'Type of sample: input (before transformations) or output (after transformations)';
COMMENT ON COLUMN dataset_samples.line_number IS 'Line number in the batch where sample was extracted';
COMMENT ON COLUMN dataset_samples.sample_data IS 'Actual data sample as JSON object';
COMMENT ON COLUMN dataset_samples.batch_id IS 'Batch identifier that generated this sample';

COMMENT ON TABLE dataset_schema IS 'Stores inferred schema from dataset samples. Schema is computed when transform configuration is edited.';
COMMENT ON COLUMN dataset_schema.schema_type IS 'Type of schema: input (before transformations) or output (after transformations)';
COMMENT ON COLUMN dataset_schema.schema_fields IS 'Schema fields as JSON array with field name, type, count, nullable, and sample value';
COMMENT ON COLUMN dataset_schema.sample_count IS 'Number of samples used to infer this schema';
