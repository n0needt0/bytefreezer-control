-- Migration 010: Add Piper and Packer state management tables
-- This enables piper and packer services to use Control API for state management
-- instead of direct database access, which is required for on-prem deployments

-- ============================================================================
-- PIPER TABLES
-- ============================================================================

-- File locks for distributed processing
CREATE TABLE IF NOT EXISTS piper_file_locks (
    lock_id SERIAL PRIMARY KEY,
    tenant_id VARCHAR(255) NOT NULL,
    dataset_id VARCHAR(255) NOT NULL,
    file_key TEXT NOT NULL,
    locked_by VARCHAR(255) NOT NULL,
    lock_timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    last_heartbeat TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    ttl TIMESTAMP WITH TIME ZONE NOT NULL,
    UNIQUE(tenant_id, dataset_id, file_key)
);

CREATE INDEX IF NOT EXISTS idx_piper_file_locks_tenant_dataset
    ON piper_file_locks(tenant_id, dataset_id);
CREATE INDEX IF NOT EXISTS idx_piper_file_locks_ttl
    ON piper_file_locks(ttl);
CREATE INDEX IF NOT EXISTS idx_piper_file_locks_heartbeat
    ON piper_file_locks(last_heartbeat);

COMMENT ON TABLE piper_file_locks IS
    'File-level locks for distributed piper processing to prevent duplicate work';

-- Job records for tracking processing jobs
CREATE TABLE IF NOT EXISTS piper_job_records (
    job_id VARCHAR(255) PRIMARY KEY,
    tenant_id VARCHAR(255) NOT NULL,
    dataset_id VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL,
    source_files JSONB,
    processor_type VARCHAR(100),
    processor_id VARCHAR(255),
    output_file TEXT,
    error_message TEXT,
    records_processed BIGINT DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_piper_job_records_tenant
    ON piper_job_records(tenant_id);
CREATE INDEX IF NOT EXISTS idx_piper_job_records_status
    ON piper_job_records(status);
CREATE INDEX IF NOT EXISTS idx_piper_job_records_created_at
    ON piper_job_records(created_at);

COMMENT ON TABLE piper_job_records IS
    'Job records for tracking piper processing jobs and their status';

-- Pipeline configuration cache
CREATE TABLE IF NOT EXISTS piper_pipeline_configurations (
    config_key VARCHAR(512) PRIMARY KEY,
    tenant_id VARCHAR(255) NOT NULL,
    dataset_id VARCHAR(255) NOT NULL,
    configuration JSONB NOT NULL,
    cached_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_piper_pipeline_configs_tenant
    ON piper_pipeline_configurations(tenant_id, dataset_id);
CREATE INDEX IF NOT EXISTS idx_piper_pipeline_configs_expires
    ON piper_pipeline_configurations(expires_at);

COMMENT ON TABLE piper_pipeline_configurations IS
    'Cached pipeline configurations to reduce control API calls';

-- Tenant cache
CREATE TABLE IF NOT EXISTS piper_tenants_cache (
    tenant_id VARCHAR(255) PRIMARY KEY,
    tenant_data JSONB NOT NULL,
    cached_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_piper_tenants_cache_expires
    ON piper_tenants_cache(expires_at);

COMMENT ON TABLE piper_tenants_cache IS
    'Cached tenant information to reduce control API calls';

-- ============================================================================
-- PACKER TABLES
-- ============================================================================

-- Tenant locks for packer operations
CREATE TABLE IF NOT EXISTS packer_tenant_locks (
    tenant_id VARCHAR(255) PRIMARY KEY,
    locked_by VARCHAR(255) NOT NULL,
    lock_timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    last_heartbeat TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    ttl TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_packer_tenant_locks_ttl
    ON packer_tenant_locks(ttl);
CREATE INDEX IF NOT EXISTS idx_packer_tenant_locks_heartbeat
    ON packer_tenant_locks(last_heartbeat);

COMMENT ON TABLE packer_tenant_locks IS
    'Tenant-level locks for distributed packer processing';

-- Parquet file metadata
CREATE TABLE IF NOT EXISTS packer_parquet_file_metadata (
    id SERIAL PRIMARY KEY,
    tenant_id VARCHAR(255) NOT NULL,
    dataset_id VARCHAR(255) NOT NULL,
    file_path VARCHAR(1000) NOT NULL,
    partition_path VARCHAR(500),
    file_size_bytes BIGINT NOT NULL,
    row_count BIGINT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    last_modified TIMESTAMP WITH TIME ZONE NOT NULL,
    schema_json JSONB NOT NULL,
    column_stats JSONB,
    file_checksum VARCHAR(64),
    instance_id VARCHAR(255),
    metadata_version INTEGER DEFAULT 1,
    ttl TIMESTAMP WITH TIME ZONE,
    inserted_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(tenant_id, dataset_id, file_path)
);

CREATE INDEX IF NOT EXISTS idx_packer_parquet_metadata_tenant_dataset
    ON packer_parquet_file_metadata(tenant_id, dataset_id);
CREATE INDEX IF NOT EXISTS idx_packer_parquet_metadata_partition
    ON packer_parquet_file_metadata(tenant_id, dataset_id, partition_path);
CREATE INDEX IF NOT EXISTS idx_packer_parquet_metadata_ttl
    ON packer_parquet_file_metadata(ttl);

COMMENT ON TABLE packer_parquet_file_metadata IS
    'Metadata for Parquet files generated by packer service';

-- Metadata generation status
CREATE TABLE IF NOT EXISTS packer_metadata_generation_status (
    tenant_id VARCHAR(255) NOT NULL,
    dataset_id VARCHAR(255) NOT NULL,
    partition_path VARCHAR(500) NOT NULL,
    last_generated_at TIMESTAMP WITH TIME ZONE,
    file_count INTEGER DEFAULT 0,
    total_rows BIGINT DEFAULT 0,
    total_size_bytes BIGINT DEFAULT 0,
    needs_regeneration BOOLEAN DEFAULT false,
    current_schema_hash VARCHAR(64),
    schema_version INTEGER DEFAULT 1,
    ttl TIMESTAMP WITH TIME ZONE,
    PRIMARY KEY (tenant_id, dataset_id, partition_path)
);

CREATE INDEX IF NOT EXISTS idx_packer_metadata_gen_status_tenant
    ON packer_metadata_generation_status(tenant_id, dataset_id);

COMMENT ON TABLE packer_metadata_generation_status IS
    'Tracks metadata generation status for partitions';

-- Metadata summary view
CREATE OR REPLACE VIEW packer_parquet_metadata_summary AS
SELECT
    tenant_id,
    dataset_id,
    partition_path,
    COUNT(*) as file_count,
    SUM(row_count) as total_rows,
    SUM(file_size_bytes) as total_size_bytes,
    MIN(created_at) as first_file_created,
    MAX(last_modified) as last_file_modified,
    MAX(updated_at) as metadata_last_updated
FROM packer_parquet_file_metadata
GROUP BY tenant_id, dataset_id, partition_path;

COMMENT ON VIEW packer_parquet_metadata_summary IS
    'Aggregated metadata summary view for partitions';
