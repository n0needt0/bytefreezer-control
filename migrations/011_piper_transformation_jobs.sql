-- Migration 011: Add Piper Transformation Jobs table
-- This enables async transformation job queue for testing, validating, and activating
-- data transformations/filters in the piper service

-- ============================================================================
-- PIPER TRANSFORMATION JOBS TABLE
-- ============================================================================

-- Transformation job queue for async processing
CREATE TABLE IF NOT EXISTS piper_transformation_jobs (
    job_id VARCHAR(255) PRIMARY KEY,
    tenant_id VARCHAR(255) NOT NULL,
    dataset_id VARCHAR(255) NOT NULL,
    job_type VARCHAR(50) NOT NULL,  -- 'test', 'validate', 'activate'
    status VARCHAR(50) NOT NULL,     -- 'pending', 'running', 'completed', 'failed'
    processor_id VARCHAR(255),
    request JSONB,                   -- Job-specific request data
    result JSONB,                    -- Job-specific result data
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    ttl TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_piper_transformation_jobs_tenant
    ON piper_transformation_jobs(tenant_id, dataset_id);
CREATE INDEX IF NOT EXISTS idx_piper_transformation_jobs_status
    ON piper_transformation_jobs(status);
CREATE INDEX IF NOT EXISTS idx_piper_transformation_jobs_type
    ON piper_transformation_jobs(job_type);
CREATE INDEX IF NOT EXISTS idx_piper_transformation_jobs_ttl
    ON piper_transformation_jobs(ttl);
CREATE INDEX IF NOT EXISTS idx_piper_transformation_jobs_created_at
    ON piper_transformation_jobs(created_at);

COMMENT ON TABLE piper_transformation_jobs IS
    'Async transformation job queue for testing, validating, and activating transformations';
