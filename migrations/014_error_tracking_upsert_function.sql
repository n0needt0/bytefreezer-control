-- Migration: 014_error_tracking_upsert_function
-- Description: Add upsert function for system_errors with adaptive sampling
-- Author: Claude
-- Date: 2025-10-28

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
RETURNS VOID AS $func$
DECLARE
    v_occurrence_count BIGINT;
    v_sample_rate FLOAT;
    v_should_sample BOOLEAN;
BEGIN
    SELECT occurrence_count, sample_rate
    INTO v_occurrence_count, v_sample_rate
    FROM system_errors
    WHERE error_hash = p_error_hash;

    IF FOUND THEN
        -- Calculate adaptive sample rate based on occurrence count
        IF v_occurrence_count >= 10000 THEN
            v_sample_rate := 0.001;
        ELSIF v_occurrence_count >= 1000 THEN
            v_sample_rate := 0.01;
        ELSIF v_occurrence_count >= 100 THEN
            v_sample_rate := 0.1;
        ELSE
            v_sample_rate := 1.0;
        END IF;

        v_should_sample := (random() <= v_sample_rate);

        UPDATE system_errors SET
            occurrence_count = occurrence_count + 1,
            last_seen = NOW(),
            sample_rate = v_sample_rate,
            samples_collected = CASE WHEN v_should_sample THEN samples_collected + 1 ELSE samples_collected END,
            samples_dropped = CASE WHEN v_should_sample THEN samples_dropped ELSE samples_dropped + 1 END,
            error_sample = CASE WHEN v_should_sample THEN p_error_sample ELSE error_sample END,
            metadata = CASE WHEN v_should_sample AND p_metadata IS NOT NULL THEN p_metadata ELSE metadata END
        WHERE error_hash = p_error_hash;
    ELSE
        INSERT INTO system_errors (
            error_hash, error_type, component, tenant_id, dataset_id,
            error_message, error_sample, severity, metadata,
            occurrence_count, sample_rate, samples_collected, samples_dropped
        )
        VALUES (
            p_error_hash, p_error_type, p_component, p_tenant_id, p_dataset_id,
            p_error_message, p_error_sample, p_severity, p_metadata,
            1, 1.0, 1, 0
        );
    END IF;
END;
$func$ LANGUAGE plpgsql;

-- Record migration in control_migrations table
INSERT INTO control_migrations (version, name, applied_at)
VALUES (14, 'error_tracking_upsert_func', NOW())
ON CONFLICT (version) DO NOTHING;
