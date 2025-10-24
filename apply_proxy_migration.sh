#!/bin/bash
# Apply proxy configuration migration manually

set -e

DB_HOST="${DB_HOST:-192.168.86.137}"
DB_USER="${DB_USER:-bytefreezer}"
DB_NAME="${DB_NAME:-bytefreezer}"
DB_PASSWORD="${DB_PASSWORD:-bytefreezer123}"

echo "Applying proxy configuration migration..."
echo "Database: $DB_HOST/$DB_NAME"

PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -U $DB_USER -d $DB_NAME << 'EOF'

-- Check if migration already applied
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM control_migrations WHERE version = 8) THEN
        RAISE NOTICE 'Migration 8 already applied, skipping';
    ELSE
        -- Proxy configuration management system
        -- Migration 008: Proxy instance configuration storage and management

        -- Table for proxy instance configurations
        CREATE TABLE IF NOT EXISTS proxy_instances (
            id SERIAL PRIMARY KEY,
            instance_id VARCHAR(255) NOT NULL,
            tenant_id VARCHAR(255) NOT NULL,
            instance_api VARCHAR(255) NOT NULL,
            config_mode VARCHAR(50) NOT NULL DEFAULT 'hybrid',
            plugin_configs JSONB DEFAULT '[]'::jsonb,
            proxy_settings JSONB DEFAULT '{}'::jsonb,
            config_version INTEGER NOT NULL DEFAULT 1,
            config_applied BOOLEAN DEFAULT false,
            config_applied_at TIMESTAMP WITH TIME ZONE,
            config_hash VARCHAR(64),
            active BOOLEAN NOT NULL DEFAULT true,
            created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
            updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
            UNIQUE (instance_id, tenant_id)
        );

        -- Table for proxy configuration history (audit trail)
        CREATE TABLE IF NOT EXISTS proxy_config_history (
            id SERIAL PRIMARY KEY,
            instance_id VARCHAR(255) NOT NULL,
            tenant_id VARCHAR(255) NOT NULL,
            config_version INTEGER NOT NULL,
            plugin_configs JSONB,
            proxy_settings JSONB,
            config_hash VARCHAR(64),
            changed_by VARCHAR(255),
            change_reason TEXT,
            timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
        );

        -- Indexes for performance
        CREATE INDEX IF NOT EXISTS idx_proxy_instances_instance_id ON proxy_instances(instance_id);
        CREATE INDEX IF NOT EXISTS idx_proxy_instances_tenant_id ON proxy_instances(tenant_id);
        CREATE INDEX IF NOT EXISTS idx_proxy_instances_active ON proxy_instances(active);
        CREATE INDEX IF NOT EXISTS idx_proxy_instances_config_applied ON proxy_instances(config_applied);
        CREATE INDEX IF NOT EXISTS idx_proxy_instances_tenant_instance ON proxy_instances(tenant_id, instance_id);
        CREATE INDEX IF NOT EXISTS idx_proxy_config_history_instance_id ON proxy_config_history(instance_id);
        CREATE INDEX IF NOT EXISTS idx_proxy_config_history_timestamp ON proxy_config_history(timestamp);
        CREATE INDEX IF NOT EXISTS idx_proxy_config_history_tenant_instance ON proxy_config_history(tenant_id, instance_id);

        -- Function to automatically create history entry on config update
        CREATE OR REPLACE FUNCTION archive_proxy_config_on_update()
        RETURNS TRIGGER AS $BODY$
        BEGIN
            IF OLD.plugin_configs IS DISTINCT FROM NEW.plugin_configs
               OR OLD.proxy_settings IS DISTINCT FROM NEW.proxy_settings THEN
                INSERT INTO proxy_config_history (
                    instance_id, tenant_id, config_version,
                    plugin_configs, proxy_settings, config_hash, timestamp
                )
                VALUES (
                    OLD.instance_id, OLD.tenant_id, OLD.config_version,
                    OLD.plugin_configs, OLD.proxy_settings, OLD.config_hash, NOW()
                );
            END IF;
            RETURN NEW;
        END;
        $BODY$ LANGUAGE plpgsql;

        -- Trigger to automatically archive config changes
        DROP TRIGGER IF EXISTS trigger_archive_proxy_config ON proxy_instances;
        CREATE TRIGGER trigger_archive_proxy_config
            BEFORE UPDATE ON proxy_instances
            FOR EACH ROW
            EXECUTE FUNCTION archive_proxy_config_on_update();

        -- Function to upsert proxy instance configuration
        CREATE OR REPLACE FUNCTION upsert_proxy_config(
            p_instance_id VARCHAR(255),
            p_tenant_id VARCHAR(255),
            p_instance_api VARCHAR(255),
            p_config_mode VARCHAR(50) DEFAULT 'hybrid',
            p_plugin_configs JSONB DEFAULT '[]'::jsonb,
            p_proxy_settings JSONB DEFAULT '{}'::jsonb,
            p_config_hash VARCHAR(64) DEFAULT NULL
        )
        RETURNS TABLE(id INTEGER, config_version INTEGER) AS $BODY$
        DECLARE
            v_id INTEGER;
            v_version INTEGER;
        BEGIN
            INSERT INTO proxy_instances (
                instance_id, tenant_id, instance_api, config_mode,
                plugin_configs, proxy_settings, config_hash, config_version, updated_at
            )
            VALUES (
                p_instance_id, p_tenant_id, p_instance_api, p_config_mode,
                p_plugin_configs, p_proxy_settings, p_config_hash, 1, NOW()
            )
            ON CONFLICT (instance_id, tenant_id)
            DO UPDATE SET
                instance_api = EXCLUDED.instance_api,
                config_mode = EXCLUDED.config_mode,
                plugin_configs = EXCLUDED.plugin_configs,
                proxy_settings = EXCLUDED.proxy_settings,
                config_hash = EXCLUDED.config_hash,
                config_version = proxy_instances.config_version + 1,
                config_applied = false,
                updated_at = NOW()
            RETURNING proxy_instances.id, proxy_instances.config_version INTO v_id, v_version;

            RETURN QUERY SELECT v_id, v_version;
        END;
        $BODY$ LANGUAGE plpgsql;

        -- Function to mark configuration as applied by proxy
        CREATE OR REPLACE FUNCTION mark_proxy_config_applied(
            p_instance_id VARCHAR(255),
            p_tenant_id VARCHAR(255),
            p_config_version INTEGER
        )
        RETURNS BOOLEAN AS $BODY$
        DECLARE
            v_updated INTEGER;
        BEGIN
            UPDATE proxy_instances
            SET
                config_applied = true,
                config_applied_at = NOW(),
                updated_at = NOW()
            WHERE
                instance_id = p_instance_id
                AND tenant_id = p_tenant_id
                AND config_version = p_config_version;

            GET DIAGNOSTICS v_updated = ROW_COUNT;
            RETURN v_updated > 0;
        END;
        $BODY$ LANGUAGE plpgsql;

        -- Function to get proxy configuration for polling
        CREATE OR REPLACE FUNCTION get_proxy_config(
            p_instance_id VARCHAR(255),
            p_tenant_id VARCHAR(255)
        )
        RETURNS TABLE(
            instance_id VARCHAR(255),
            tenant_id VARCHAR(255),
            config_mode VARCHAR(50),
            plugin_configs JSONB,
            proxy_settings JSONB,
            config_version INTEGER,
            config_hash VARCHAR(64),
            updated_at TIMESTAMP WITH TIME ZONE
        ) AS $BODY$
        BEGIN
            RETURN QUERY
            SELECT
                pi.instance_id,
                pi.tenant_id,
                pi.config_mode,
                pi.plugin_configs,
                pi.proxy_settings,
                pi.config_version,
                pi.config_hash,
                pi.updated_at
            FROM proxy_instances pi
            WHERE
                pi.instance_id = p_instance_id
                AND pi.tenant_id = p_tenant_id
                AND pi.active = true;
        END;
        $BODY$ LANGUAGE plpgsql;

        -- Function to cleanup old history records
        CREATE OR REPLACE FUNCTION cleanup_old_proxy_config_history()
        RETURNS INTEGER AS $BODY$
        DECLARE
            deleted_count INTEGER := 0;
        BEGIN
            DELETE FROM proxy_config_history
            WHERE id IN (
                SELECT id FROM (
                    SELECT
                        id,
                        ROW_NUMBER() OVER (
                            PARTITION BY instance_id, tenant_id
                            ORDER BY timestamp DESC
                        ) AS rn
                    FROM proxy_config_history
                    WHERE timestamp < NOW() - INTERVAL '90 days'
                ) sub
                WHERE rn > 10
            );

            GET DIAGNOSTICS deleted_count = ROW_COUNT;
            RETURN deleted_count;
        END;
        $BODY$ LANGUAGE plpgsql;

        -- Record migration as applied
        INSERT INTO control_migrations (version, name, description, applied_at)
        VALUES (8, 'proxy_configuration', 'Add proxy instance configuration tracking tables and functions', NOW());

        RAISE NOTICE 'Migration 8 applied successfully';
    END IF;
END $$;

-- Verify the functions exist
SELECT 'upsert_proxy_config' as function, proname
FROM pg_proc
WHERE proname = 'upsert_proxy_config';

SELECT 'Tables created:' as status;
SELECT tablename FROM pg_tables WHERE tablename IN ('proxy_instances', 'proxy_config_history');

EOF

echo "Migration completed!"
