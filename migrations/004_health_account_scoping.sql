-- Health monitoring account scoping
-- Migration 004: Add account_id to health tables for multi-tenant isolation

-- Add account_id column to health_current
ALTER TABLE health_current ADD COLUMN account_id VARCHAR(50);

-- Add account_id column to health_history
ALTER TABLE health_history ADD COLUMN account_id VARCHAR(50);

-- Add indexes for account filtering
CREATE INDEX IF NOT EXISTS idx_health_current_account ON health_current(account_id);
CREATE INDEX IF NOT EXISTS idx_health_history_account ON health_history(account_id);

-- Update unique constraint to support multiple accounts with same instance_id
-- This allows different accounts to have proxies with the same hostname
ALTER TABLE health_current DROP CONSTRAINT IF EXISTS health_current_service_type_instance_id_key;

-- Create new unique constraint that includes account_id
-- NULL account_id represents system services (packer, piper, receiver)
CREATE UNIQUE INDEX health_current_unique_idx ON health_current (
    service_type,
    instance_id,
    COALESCE(account_id, '')
);

-- Update upsert function to include account_id
CREATE OR REPLACE FUNCTION upsert_health_current(
    p_service_type VARCHAR(100),
    p_instance_id VARCHAR(255),
    p_instance_api VARCHAR(255),
    p_status VARCHAR(50),
    p_account_id VARCHAR(50) DEFAULT NULL,
    p_configuration JSONB DEFAULT NULL,
    p_metrics JSONB DEFAULT NULL,
    p_response_time_ms INTEGER DEFAULT NULL
)
RETURNS VOID AS $$
BEGIN
    INSERT INTO health_current (
        service_type, instance_id, instance_api, status, account_id,
        configuration, metrics, response_time_ms, last_seen, updated_at
    )
    VALUES (
        p_service_type, p_instance_id, p_instance_api, p_status, p_account_id,
        p_configuration, p_metrics, p_response_time_ms, NOW(), NOW()
    )
    ON CONFLICT ON CONSTRAINT health_current_unique_idx
    DO UPDATE SET
        instance_api = EXCLUDED.instance_api,
        status = EXCLUDED.status,
        configuration = COALESCE(EXCLUDED.configuration, health_current.configuration),
        metrics = COALESCE(EXCLUDED.metrics, health_current.metrics),
        response_time_ms = COALESCE(EXCLUDED.response_time_ms, health_current.response_time_ms),
        last_seen = NOW(),
        updated_at = NOW();
END;
$$ LANGUAGE plpgsql;

-- Update move_stale_health_to_history function to include account_id
CREATE OR REPLACE FUNCTION move_stale_health_to_history()
RETURNS INTEGER AS $$
DECLARE
    moved_count INTEGER := 0;
BEGIN
    -- Move records older than 24 hours to history
    WITH moved_records AS (
        DELETE FROM health_current
        WHERE last_seen < NOW() - INTERVAL '24 hours'
        RETURNING *
    )
    INSERT INTO health_history (
        service_type, instance_id, instance_api, status, account_id,
        configuration, metrics, response_time_ms, timestamp
    )
    SELECT
        service_type, instance_id, instance_api, status, account_id,
        configuration, metrics, response_time_ms, last_seen
    FROM moved_records;

    GET DIAGNOSTICS moved_count = ROW_COUNT;
    RETURN moved_count;
END;
$$ LANGUAGE plpgsql;

COMMENT ON COLUMN health_current.account_id IS 'Account ID for account-scoped services (proxy). NULL for system services (packer, piper, receiver).';
COMMENT ON COLUMN health_history.account_id IS 'Account ID for account-scoped services (proxy). NULL for system services (packer, piper, receiver).';
