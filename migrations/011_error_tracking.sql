-- Migration: 008_error_tracking
-- Description: Create system_errors table for tracking and sampling errors across all ByteFreezer components
-- Author: Claude
-- Date: 2025-10-27

-- Create system_errors table with error deduplication and sampling
CREATE TABLE IF NOT EXISTS system_errors (
    id BIGSERIAL PRIMARY KEY,

    -- Error identification and deduplication
    error_hash VARCHAR(64) NOT NULL UNIQUE, -- SHA256 hash of (component + error_type + error_message_pattern)
    error_type VARCHAR(100) NOT NULL,       -- Error category (e.g., "pipeline_processing", "s3_upload", "validation")

    -- Component and context
    component VARCHAR(50) NOT NULL CHECK (component IN ('proxy', 'receiver', 'piper', 'packer', 'control', 'soc')),
    tenant_id VARCHAR(255),                 -- Optional: may be NULL for system-level errors
    dataset_id VARCHAR(255),                -- Optional: may be NULL for system-level errors

    -- Error details
    error_message TEXT NOT NULL,            -- Full error message
    error_sample JSONB,                     -- Sample error details (context, stack trace, etc.)

    -- Severity and status
    severity VARCHAR(20) NOT NULL DEFAULT 'error' CHECK (severity IN ('debug', 'info', 'warning', 'error', 'critical')),
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'resolved', 'ignored')),

    -- Occurrence tracking
    occurrence_count BIGINT NOT NULL DEFAULT 1,
    first_seen TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    last_seen TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Sampling metadata
    sample_rate FLOAT NOT NULL DEFAULT 1.0, -- Current sampling rate (1.0 = 100%, 0.1 = 10%)
    samples_collected INTEGER NOT NULL DEFAULT 1,
    samples_dropped INTEGER NOT NULL DEFAULT 0,

    -- Additional context
    metadata JSONB,                         -- Additional metadata (line numbers, file paths, etc.)

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMP WITH TIME ZONE,

    -- Foreign key constraint (optional, allows NULL for system errors)
    CONSTRAINT fk_error_tenant_dataset FOREIGN KEY (tenant_id, dataset_id)
        REFERENCES control_datasets(tenant_id, id) ON DELETE CASCADE
        DEFERRABLE INITIALLY DEFERRED
);

-- Create indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_system_errors_component
    ON system_errors(component);

CREATE INDEX IF NOT EXISTS idx_system_errors_tenant_dataset
    ON system_errors(tenant_id, dataset_id) WHERE tenant_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_system_errors_error_type
    ON system_errors(error_type);

CREATE INDEX IF NOT EXISTS idx_system_errors_severity
    ON system_errors(severity);

CREATE INDEX IF NOT EXISTS idx_system_errors_status
    ON system_errors(status);

CREATE INDEX IF NOT EXISTS idx_system_errors_last_seen
    ON system_errors(last_seen DESC);

CREATE INDEX IF NOT EXISTS idx_system_errors_occurrence_count
    ON system_errors(occurrence_count DESC);

-- Composite indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_system_errors_component_status_last_seen
    ON system_errors(component, status, last_seen DESC);

CREATE INDEX IF NOT EXISTS idx_system_errors_tenant_dataset_status
    ON system_errors(tenant_id, dataset_id, status, last_seen DESC)
    WHERE tenant_id IS NOT NULL;

-- GIN index for JSON metadata searching
CREATE INDEX IF NOT EXISTS idx_system_errors_metadata_gin
    ON system_errors USING GIN (metadata);

CREATE INDEX IF NOT EXISTS idx_system_errors_sample_gin
    ON system_errors USING GIN (error_sample);

-- Function to update the updated_at timestamp
CREATE OR REPLACE FUNCTION update_system_errors_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to automatically update updated_at
CREATE TRIGGER trigger_update_system_errors_updated_at
    BEFORE UPDATE ON system_errors
    FOR EACH ROW
    EXECUTE FUNCTION update_system_errors_updated_at();

-- Function to upsert errors with sampling logic
CREATE OR REPLACE FUNCTION upsert_system_error(
    p_error_hash VARCHAR(64),
    p_error_type VARCHAR(100),
    p_component VARCHAR(50),
    p_tenant_id VARCHAR(255),
    p_dataset_id VARCHAR(255),
    p_error_message TEXT,
    p_error_sample JSONB,
    p_severity VARCHAR(20),
    p_metadata JSONB
)
RETURNS VOID AS $$
DECLARE
    v_occurrence_count BIGINT;
    v_sample_rate FLOAT;
    v_should_sample BOOLEAN;
BEGIN
    -- Check if error already exists
    SELECT occurrence_count, sample_rate INTO v_occurrence_count, v_sample_rate
    FROM system_errors
    WHERE error_hash = p_error_hash;

    IF FOUND THEN
        -- Calculate new sample rate based on occurrence count
        -- After 100 occurrences, sample 10%
        -- After 1000 occurrences, sample 1%
        -- After 10000 occurrences, sample 0.1%
        IF v_occurrence_count >= 10000 THEN
            v_sample_rate := 0.001;
        ELSIF v_occurrence_count >= 1000 THEN
            v_sample_rate := 0.01;
        ELSIF v_occurrence_count >= 100 THEN
            v_sample_rate := 0.1;
        ELSE
            v_sample_rate := 1.0;
        END IF;

        -- Determine if we should collect this sample
        v_should_sample := (random() <= v_sample_rate);

        -- Update existing error
        UPDATE system_errors
        SET
            occurrence_count = occurrence_count + 1,
            last_seen = NOW(),
            sample_rate = v_sample_rate,
            samples_collected = CASE WHEN v_should_sample THEN samples_collected + 1 ELSE samples_collected END,
            samples_dropped = CASE WHEN v_should_sample THEN samples_dropped ELSE samples_dropped + 1 END,
            error_sample = CASE WHEN v_should_sample THEN p_error_sample ELSE error_sample END,
            metadata = CASE WHEN v_should_sample AND p_metadata IS NOT NULL THEN p_metadata ELSE metadata END
        WHERE error_hash = p_error_hash;
    ELSE
        -- Insert new error
        INSERT INTO system_errors (
            error_hash,
            error_type,
            component,
            tenant_id,
            dataset_id,
            error_message,
            error_sample,
            severity,
            metadata,
            occurrence_count,
            sample_rate,
            samples_collected,
            samples_dropped
        ) VALUES (
            p_error_hash,
            p_error_type,
            p_component,
            p_tenant_id,
            p_dataset_id,
            p_error_message,
            p_error_sample,
            p_severity,
            p_metadata,
            1,
            1.0,
            1,
            0
        );
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Insert migration record
INSERT INTO control_migrations (version, name, applied_at)
VALUES (11, 'error_tracking', NOW())
ON CONFLICT (version) DO NOTHING;

-- Add comments for documentation
COMMENT ON TABLE system_errors IS 'Tracks and samples errors across all ByteFreezer components with automatic deduplication and adaptive sampling';
COMMENT ON COLUMN system_errors.error_hash IS 'SHA256 hash used for error deduplication';
COMMENT ON COLUMN system_errors.error_type IS 'Error category for classification (e.g., pipeline_processing, s3_upload, validation)';
COMMENT ON COLUMN system_errors.component IS 'Component that generated the error';
COMMENT ON COLUMN system_errors.occurrence_count IS 'Total number of times this error has occurred';
COMMENT ON COLUMN system_errors.sample_rate IS 'Current adaptive sampling rate (1.0 = 100%, 0.1 = 10%, 0.01 = 1%, 0.001 = 0.1%)';
COMMENT ON COLUMN system_errors.samples_collected IS 'Number of error samples collected';
COMMENT ON COLUMN system_errors.samples_dropped IS 'Number of error samples dropped due to sampling';
COMMENT ON COLUMN system_errors.error_sample IS 'Most recent error sample with full context';
COMMENT ON COLUMN system_errors.status IS 'Error status: active (ongoing), resolved (fixed), ignored (acknowledged but not fixed)';
COMMENT ON COLUMN system_errors.severity IS 'Error severity level: debug, info, warning, error, critical';
COMMENT ON FUNCTION upsert_system_error IS 'Upserts an error with adaptive sampling based on occurrence count';
