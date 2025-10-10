-- Health monitoring system tables
-- Migration 003: Health monitoring system

-- Table for current health status of services
CREATE TABLE IF NOT EXISTS health_current (
    id SERIAL PRIMARY KEY,
    service_type VARCHAR(100) NOT NULL,        -- e.g., 'bytefreezer-receiver'
    instance_id VARCHAR(255) NOT NULL,         -- hostname
    instance_api VARCHAR(255) NOT NULL,        -- hostname:port
    status VARCHAR(50) NOT NULL DEFAULT 'Unhealthy',  -- 'Healthy', 'Unhealthy', 'Unknown'
    configuration JSONB,                        -- sanitized configuration
    metrics JSONB,                             -- current metrics
    response_time_ms INTEGER,                   -- last health check response time
    last_seen TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE (service_type, instance_id)
);

-- Table for historical health data
CREATE TABLE IF NOT EXISTS health_history (
    id SERIAL PRIMARY KEY,
    service_type VARCHAR(100) NOT NULL,
    instance_id VARCHAR(255) NOT NULL,
    instance_api VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL,
    configuration JSONB,
    metrics JSONB,
    response_time_ms INTEGER,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_health_current_service_type ON health_current(service_type);
CREATE INDEX IF NOT EXISTS idx_health_current_status ON health_current(status);
CREATE INDEX IF NOT EXISTS idx_health_current_last_seen ON health_current(last_seen);
CREATE INDEX IF NOT EXISTS idx_health_current_service_instance ON health_current(service_type, instance_id);

CREATE INDEX IF NOT EXISTS idx_health_history_service_type ON health_history(service_type);
CREATE INDEX IF NOT EXISTS idx_health_history_timestamp ON health_history(timestamp);
CREATE INDEX IF NOT EXISTS idx_health_history_service_instance ON health_history(service_type, instance_id);

-- Function to automatically move stale records to history
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
        service_type, instance_id, instance_api, status,
        configuration, metrics, response_time_ms, timestamp
    )
    SELECT
        service_type, instance_id, instance_api, status,
        configuration, metrics, response_time_ms, last_seen
    FROM moved_records;

    GET DIAGNOSTICS moved_count = ROW_COUNT;
    RETURN moved_count;
END;
$$ LANGUAGE plpgsql;

-- Function to cleanup old history records
CREATE OR REPLACE FUNCTION cleanup_old_health_history()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER := 0;
BEGIN
    -- Delete records older than 30 days
    DELETE FROM health_history
    WHERE timestamp < NOW() - INTERVAL '30 days';

    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Function to upsert health data
CREATE OR REPLACE FUNCTION upsert_health_current(
    p_service_type VARCHAR(100),
    p_instance_id VARCHAR(255),
    p_instance_api VARCHAR(255),
    p_status VARCHAR(50),
    p_configuration JSONB DEFAULT NULL,
    p_metrics JSONB DEFAULT NULL,
    p_response_time_ms INTEGER DEFAULT NULL
)
RETURNS VOID AS $$
BEGIN
    INSERT INTO health_current (
        service_type, instance_id, instance_api, status,
        configuration, metrics, response_time_ms, last_seen, updated_at
    )
    VALUES (
        p_service_type, p_instance_id, p_instance_api, p_status,
        p_configuration, p_metrics, p_response_time_ms, NOW(), NOW()
    )
    ON CONFLICT (service_type, instance_id)
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