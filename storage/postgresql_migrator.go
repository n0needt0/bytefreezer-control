package storage

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"
)

// PostgreSQLMigrator implements the Migrator interface for PostgreSQL
type PostgreSQLMigrator struct {
	db *sql.DB
}

// NewPostgreSQLMigrator creates a new PostgreSQL migrator
func NewPostgreSQLMigrator(db *sql.DB) *PostgreSQLMigrator {
	return &PostgreSQLMigrator{db: db}
}

// GetStatus returns the current migration status
func (m *PostgreSQLMigrator) GetStatus(ctx context.Context) (*MigrationStatus, error) {
	// Ensure migrations table exists
	if err := m.ensureMigrationsTable(ctx); err != nil {
		return nil, err
	}

	// Get all available migrations
	allMigrations := m.getAllMigrations()

	// Get applied migrations
	appliedMigrations, err := m.getAppliedMigrations(ctx)
	if err != nil {
		return nil, err
	}

	// Create maps for easy lookup
	appliedMap := make(map[int]Migration)
	for _, migration := range appliedMigrations {
		appliedMap[migration.Version] = migration
	}

	var applied, pending []Migration
	currentVersion := 0

	for _, migration := range allMigrations {
		if appliedMigration, exists := appliedMap[migration.Version]; exists {
			applied = append(applied, appliedMigration)
			if migration.Version > currentVersion {
				currentVersion = migration.Version
			}
		} else {
			pending = append(pending, migration)
		}
	}

	return &MigrationStatus{
		CurrentVersion: currentVersion,
		Migrations:     allMigrations,
		Applied:        applied,
		Pending:        pending,
	}, nil
}

// Migrate applies pending migrations
func (m *PostgreSQLMigrator) Migrate(ctx context.Context, opts MigrationOptions) error {
	if opts.FreshInstall {
		return m.freshInstall(ctx)
	}

	if opts.ResetData {
		return m.reset(ctx)
	}

	// Ensure migrations table exists
	if err := m.ensureMigrationsTable(ctx); err != nil {
		return err
	}

	status, err := m.GetStatus(ctx)
	if err != nil {
		return err
	}

	migrationsToApply := status.Pending
	if opts.TargetVersion > 0 {
		// Filter migrations up to target version
		var filtered []Migration
		for _, migration := range status.Pending {
			if migration.Version <= opts.TargetVersion {
				filtered = append(filtered, migration)
			}
		}
		migrationsToApply = filtered
	}

	// Sort migrations by version
	sort.Slice(migrationsToApply, func(i, j int) bool {
		return migrationsToApply[i].Version < migrationsToApply[j].Version
	})

	if opts.DryRun {
		fmt.Printf("Migrations to apply (DRY RUN):\n")
		for _, migration := range migrationsToApply {
			fmt.Printf("  - Version %d: %s\n", migration.Version, migration.Name)
		}
		return nil
	}

	// Apply migrations
	for _, migration := range migrationsToApply {
		if err := m.applyMigration(ctx, migration); err != nil {
			return fmt.Errorf("failed to apply migration %d: %w", migration.Version, err)
		}
	}

	return nil
}

// Rollback rolls back to a specific version
func (m *PostgreSQLMigrator) Rollback(ctx context.Context, targetVersion int) error {
	status, err := m.GetStatus(ctx)
	if err != nil {
		return err
	}

	if targetVersion >= status.CurrentVersion {
		return fmt.Errorf("target version %d must be less than current version %d", targetVersion, status.CurrentVersion)
	}

	// Find migrations to rollback (in reverse order)
	var migrationsToRollback []Migration
	for _, migration := range status.Applied {
		if migration.Version > targetVersion {
			migrationsToRollback = append(migrationsToRollback, migration)
		}
	}

	// Sort in descending order for rollback
	sort.Slice(migrationsToRollback, func(i, j int) bool {
		return migrationsToRollback[i].Version > migrationsToRollback[j].Version
	})

	// Apply rollbacks
	for _, migration := range migrationsToRollback {
		if err := m.rollbackMigration(ctx, migration); err != nil {
			return fmt.Errorf("failed to rollback migration %d: %w", migration.Version, err)
		}
	}

	return nil
}

// Reset drops all tables and recreates the schema
func (m *PostgreSQLMigrator) Reset(ctx context.Context) error {
	return m.reset(ctx)
}

// ensureMigrationsTable creates the migrations table if it doesn't exist
func (m *PostgreSQLMigrator) ensureMigrationsTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS control_migrations (
			version INTEGER PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			applied_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		)`

	_, err := m.db.ExecContext(ctx, query)
	return err
}

// getAppliedMigrations returns all applied migrations
func (m *PostgreSQLMigrator) getAppliedMigrations(ctx context.Context) ([]Migration, error) {
	query := `SELECT version, name, description, applied_at FROM control_migrations ORDER BY version`

	rows, err := m.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var migrations []Migration
	for rows.Next() {
		var migration Migration
		var description sql.NullString

		err := rows.Scan(&migration.Version, &migration.Name, &description, &migration.AppliedAt)
		if err != nil {
			return nil, err
		}

		if description.Valid {
			migration.Description = description.String
		}

		migrations = append(migrations, migration)
	}

	return migrations, nil
}

// getAllMigrations returns all available migrations
func (m *PostgreSQLMigrator) getAllMigrations() []Migration {
	return []Migration{
		{
			Version:     1,
			Name:        "initial_schema",
			Description: "Create accounts, tenants, datasets, and api_keys tables with JSONB support",
		},
		{
			Version:     2,
			Name:        "add_indexes",
			Description: "Add performance indexes on commonly queried fields",
		},
		{
			Version:     3,
			Name:        "add_gin_indexes",
			Description: "Add GIN indexes for JSONB config fields",
		},
		{
			Version:     4,
			Name:        "add_audit_log_resource_name",
			Description: "Add resource_name column to control_audit_log table",
		},
		{
			Version:     5,
			Name:        "add_audit_log_user_email",
			Description: "Add user_email column to control_audit_log table",
		},
		{
			Version:     6,
			Name:        "cleanup_token_refresh_audit_logs",
			Description: "Remove noisy token_refresh entries from audit logs",
		},
		{
			Version:     7,
			Name:        "add_dataset_test_status",
			Description: "Add test status fields for tracking input/output test results",
		},
		{
			Version:     8,
			Name:        "proxy_configuration",
			Description: "Add proxy instance configuration tracking tables and functions",
		},
	}
}

// applyMigration applies a single migration
func (m *PostgreSQLMigrator) applyMigration(ctx context.Context, migration Migration) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Apply the migration SQL
	migrationSQL := m.getMigrationSQL(migration.Version)
	if migrationSQL == "" {
		return fmt.Errorf("no SQL found for migration version %d", migration.Version)
	}

	// Execute migration in parts (split by semicolon for multiple statements)
	statements := strings.Split(migrationSQL, ";")
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		_, err := tx.ExecContext(ctx, stmt)
		if err != nil {
			return fmt.Errorf("failed to execute migration statement: %w", err)
		}
	}

	// Record migration as applied
	recordQuery := `INSERT INTO control_migrations (version, name, description, applied_at) VALUES ($1, $2, $3, $4)`
	_, err = tx.ExecContext(ctx, recordQuery, migration.Version, migration.Name, migration.Description, time.Now())
	if err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	return tx.Commit()
}

// rollbackMigration rolls back a single migration
func (m *PostgreSQLMigrator) rollbackMigration(ctx context.Context, migration Migration) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Apply the rollback SQL
	rollbackSQL := m.getRollbackSQL(migration.Version)
	if rollbackSQL == "" {
		return fmt.Errorf("no rollback SQL found for migration version %d", migration.Version)
	}

	// Execute rollback in parts
	statements := strings.Split(rollbackSQL, ";")
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		_, err := tx.ExecContext(ctx, stmt)
		if err != nil {
			return fmt.Errorf("failed to execute rollback statement: %w", err)
		}
	}

	// Remove migration record
	deleteQuery := `DELETE FROM control_migrations WHERE version = $1`
	_, err = tx.ExecContext(ctx, deleteQuery, migration.Version)
	if err != nil {
		return fmt.Errorf("failed to remove migration record: %w", err)
	}

	return tx.Commit()
}

// freshInstall creates the complete schema from scratch
func (m *PostgreSQLMigrator) freshInstall(ctx context.Context) error {
	// Drop existing tables
	if err := m.reset(ctx); err != nil {
		return err
	}

	// Apply all migrations in order
	return m.Migrate(ctx, MigrationOptions{})
}

// reset drops all tables
func (m *PostgreSQLMigrator) reset(ctx context.Context) error {
	dropQueries := []string{
		`DROP TABLE IF EXISTS control_api_keys CASCADE`,
		`DROP TABLE IF EXISTS control_datasets CASCADE`,
		`DROP TABLE IF EXISTS control_tenants CASCADE`,
		`DROP TABLE IF EXISTS control_accounts CASCADE`,
		`DROP TABLE IF EXISTS control_migrations CASCADE`,
		// Drop old tables without prefix for cleanup
		`DROP TABLE IF EXISTS api_keys CASCADE`,
		`DROP TABLE IF EXISTS datasets CASCADE`,
		`DROP TABLE IF EXISTS tenants CASCADE`,
		`DROP TABLE IF EXISTS accounts CASCADE`,
		`DROP TABLE IF EXISTS migrations CASCADE`,
	}

	for _, query := range dropQueries {
		_, err := m.db.ExecContext(ctx, query)
		if err != nil {
			return fmt.Errorf("failed to drop table: %w", err)
		}
	}

	return nil
}

// getMigrationSQL returns the SQL for a specific migration version
func (m *PostgreSQLMigrator) getMigrationSQL(version int) string {
	switch version {
	case 1:
		return `
			-- Create accounts table
			CREATE TABLE IF NOT EXISTS control_accounts (
				id VARCHAR(12) PRIMARY KEY,
				name VARCHAR(255) NOT NULL,
				email VARCHAR(255) UNIQUE NOT NULL,
				active BOOLEAN NOT NULL DEFAULT true,
				instance_id VARCHAR(100),
				created_at TIMESTAMP WITH TIME ZONE NOT NULL,
				updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
				config JSONB NOT NULL DEFAULT '{}'::jsonb
			);

			-- Create tenants table (belongs to accounts)
			CREATE TABLE IF NOT EXISTS control_tenants (
				id VARCHAR(12) PRIMARY KEY,
				account_id VARCHAR(12) NOT NULL REFERENCES control_accounts(id) ON DELETE CASCADE,
				name VARCHAR(255) NOT NULL,
				description TEXT,
				active BOOLEAN NOT NULL DEFAULT true,
				instance_id VARCHAR(100),
				created_at TIMESTAMP WITH TIME ZONE NOT NULL,
				updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
				config JSONB NOT NULL DEFAULT '{}'::jsonb,
				UNIQUE(account_id, name)
			);

			-- Create datasets table (belongs to tenants)
			CREATE TABLE IF NOT EXISTS control_datasets (
				id VARCHAR(12) PRIMARY KEY,
				tenant_id VARCHAR(12) NOT NULL REFERENCES control_tenants(id) ON DELETE CASCADE,
				name VARCHAR(255) NOT NULL,
				description TEXT,
				active BOOLEAN NOT NULL DEFAULT true,
				status VARCHAR(50) NOT NULL DEFAULT 'active',
				instance_id VARCHAR(100),
				created_at TIMESTAMP WITH TIME ZONE NOT NULL,
				updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
				config JSONB NOT NULL DEFAULT '{}'::jsonb,
				records_processed BIGINT NOT NULL DEFAULT 0,
				last_processed_at TIMESTAMP WITH TIME ZONE,
				error_count INTEGER NOT NULL DEFAULT 0,
				last_error TEXT,
				processing_metrics JSONB NOT NULL DEFAULT '{}'::jsonb,
				UNIQUE(tenant_id, name)
			);

			-- Create api_keys table for authentication
			CREATE TABLE IF NOT EXISTS control_api_keys (
				id VARCHAR(12) PRIMARY KEY,
				account_id VARCHAR(12) NOT NULL REFERENCES control_accounts(id) ON DELETE CASCADE,
				name VARCHAR(255) NOT NULL,
				key_hash VARCHAR(255) NOT NULL UNIQUE,
				permissions JSONB NOT NULL DEFAULT '{}'::jsonb,
				active BOOLEAN NOT NULL DEFAULT true,
				instance_id VARCHAR(100),
				last_used_at TIMESTAMP WITH TIME ZONE,
				created_at TIMESTAMP WITH TIME ZONE NOT NULL,
				updated_at TIMESTAMP WITH TIME ZONE NOT NULL
			);`

	case 2:
		return `
			-- Add indexes for performance
			CREATE INDEX idx_control_accounts_email ON control_accounts(email);
			CREATE INDEX idx_control_accounts_active ON control_accounts(active);
			CREATE INDEX idx_control_accounts_created_at ON control_accounts(created_at);
			CREATE INDEX idx_control_accounts_instance ON control_accounts(instance_id);

			CREATE INDEX idx_control_tenants_account_id ON control_tenants(account_id);
			CREATE INDEX idx_control_tenants_active ON control_tenants(active);
			CREATE INDEX idx_control_tenants_created_at ON control_tenants(created_at);
			CREATE INDEX idx_control_tenants_instance ON control_tenants(instance_id);

			CREATE INDEX idx_control_datasets_tenant_id ON control_datasets(tenant_id);
			CREATE INDEX idx_control_datasets_status ON control_datasets(status);
			CREATE INDEX idx_control_datasets_active ON control_datasets(active);
			CREATE INDEX idx_control_datasets_created_at ON control_datasets(created_at);
			CREATE INDEX idx_control_datasets_last_processed_at ON control_datasets(last_processed_at);
			CREATE INDEX idx_control_datasets_instance ON control_datasets(instance_id);

			CREATE INDEX idx_control_api_keys_account_id ON control_api_keys(account_id);
			CREATE INDEX idx_control_api_keys_key_hash ON control_api_keys(key_hash);
			CREATE INDEX idx_control_api_keys_active ON control_api_keys(active);
			CREATE INDEX idx_control_api_keys_instance ON control_api_keys(instance_id);`

	case 3:
		return `
			-- Add GIN indexes for JSONB config fields
			CREATE INDEX idx_control_accounts_config_gin ON control_accounts USING gin(config);
			CREATE INDEX idx_control_tenants_config_gin ON control_tenants USING gin(config);
			CREATE INDEX idx_control_datasets_config_gin ON control_datasets USING gin(config);
			CREATE INDEX idx_control_datasets_metrics_gin ON control_datasets USING gin(processing_metrics);
			CREATE INDEX idx_control_api_keys_permissions_gin ON control_api_keys USING gin(permissions);

			-- Add specific indexes for common JSONB queries
			CREATE INDEX idx_control_tenants_subscription_tier ON control_tenants USING gin((config->'subscription'->>'tier'));
			CREATE INDEX idx_control_tenants_org_size ON control_tenants USING gin((config->'organization'->>'size'));
			CREATE INDEX idx_control_datasets_source_type ON control_datasets USING gin((config->'source'->>'type'));`

	case 4:
		return `
			-- Add resource_name column to audit log
			ALTER TABLE control_audit_log ADD COLUMN IF NOT EXISTS resource_name VARCHAR(255)`

	case 5:
		return `
			-- Add user_email column to audit log
			ALTER TABLE control_audit_log ADD COLUMN IF NOT EXISTS user_email VARCHAR(255)`

	case 6:
		return `
			-- Delete all token_refresh audit log entries
			-- These are too noisy and were removed from logging in v2.2.1
			DELETE FROM control_audit_log WHERE action = 'token_refresh'`

	case 7:
		return `
			-- Add test status fields for dataset input/output testing
			ALTER TABLE control_datasets ADD COLUMN IF NOT EXISTS input_test_status VARCHAR(50) DEFAULT 'untested';
			ALTER TABLE control_datasets ADD COLUMN IF NOT EXISTS input_test_message TEXT;
			ALTER TABLE control_datasets ADD COLUMN IF NOT EXISTS output_test_status VARCHAR(50) DEFAULT 'untested';
			ALTER TABLE control_datasets ADD COLUMN IF NOT EXISTS output_test_message TEXT;
			ALTER TABLE control_datasets ADD COLUMN IF NOT EXISTS last_tested_at TIMESTAMP WITH TIME ZONE`

	case 8:
		return `
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
			RETURNS TRIGGER AS $$
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
			$$ LANGUAGE plpgsql;

			-- Trigger to automatically archive config changes
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
			RETURNS TABLE(id INTEGER, config_version INTEGER) AS $$
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
			$$ LANGUAGE plpgsql;

			-- Function to mark configuration as applied by proxy
			CREATE OR REPLACE FUNCTION mark_proxy_config_applied(
				p_instance_id VARCHAR(255),
				p_tenant_id VARCHAR(255),
				p_config_version INTEGER
			)
			RETURNS BOOLEAN AS $$
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
			$$ LANGUAGE plpgsql;

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
			) AS $$
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
			$$ LANGUAGE plpgsql;

			-- Function to cleanup old history records
			CREATE OR REPLACE FUNCTION cleanup_old_proxy_config_history()
			RETURNS INTEGER AS $$
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
			$$ LANGUAGE plpgsql;`

	default:
		return ""
	}
}

// getRollbackSQL returns the rollback SQL for a specific migration version
func (m *PostgreSQLMigrator) getRollbackSQL(version int) string {
	switch version {
	case 1:
		return `
			DROP TABLE IF EXISTS control_api_keys CASCADE;
			DROP TABLE IF EXISTS control_datasets CASCADE;
			DROP TABLE IF EXISTS control_tenants CASCADE;
			DROP TABLE IF EXISTS control_accounts CASCADE;`

	case 2:
		return `
			DROP INDEX IF EXISTS idx_control_api_keys_instance;
			DROP INDEX IF EXISTS idx_control_api_keys_active;
			DROP INDEX IF EXISTS idx_control_api_keys_key_hash;
			DROP INDEX IF EXISTS idx_control_api_keys_account_id;
			DROP INDEX IF EXISTS idx_control_datasets_instance;
			DROP INDEX IF EXISTS idx_control_datasets_last_processed_at;
			DROP INDEX IF EXISTS idx_control_datasets_created_at;
			DROP INDEX IF EXISTS idx_control_datasets_active;
			DROP INDEX IF EXISTS idx_control_datasets_status;
			DROP INDEX IF EXISTS idx_control_datasets_tenant_id;
			DROP INDEX IF EXISTS idx_control_tenants_instance;
			DROP INDEX IF EXISTS idx_control_tenants_created_at;
			DROP INDEX IF EXISTS idx_control_tenants_active;
			DROP INDEX IF EXISTS idx_control_tenants_account_id;
			DROP INDEX IF EXISTS idx_control_accounts_instance;
			DROP INDEX IF EXISTS idx_control_accounts_created_at;
			DROP INDEX IF EXISTS idx_control_accounts_active;
			DROP INDEX IF EXISTS idx_control_accounts_email;`

	case 3:
		return `
			DROP INDEX IF EXISTS idx_control_datasets_source_type;
			DROP INDEX IF EXISTS idx_control_tenants_org_size;
			DROP INDEX IF EXISTS idx_control_tenants_subscription_tier;
			DROP INDEX IF EXISTS idx_control_api_keys_permissions_gin;
			DROP INDEX IF EXISTS idx_control_datasets_metrics_gin;
			DROP INDEX IF EXISTS idx_control_datasets_config_gin;
			DROP INDEX IF EXISTS idx_control_tenants_config_gin;
			DROP INDEX IF EXISTS idx_control_accounts_config_gin;`

	case 4:
		return `
			-- Remove resource_name column from audit log
			ALTER TABLE control_audit_log DROP COLUMN IF EXISTS resource_name;`

	case 5:
		return `
			-- Remove user_email column from audit log
			ALTER TABLE control_audit_log DROP COLUMN IF EXISTS user_email;`

	case 6:
		return `
			-- No rollback for cleanup migration - deleted data cannot be restored`

	case 7:
		return `
			-- Remove test status fields from datasets table
			ALTER TABLE control_datasets DROP COLUMN IF EXISTS input_test_status;
			ALTER TABLE control_datasets DROP COLUMN IF EXISTS input_test_message;
			ALTER TABLE control_datasets DROP COLUMN IF EXISTS output_test_status;
			ALTER TABLE control_datasets DROP COLUMN IF EXISTS output_test_message;
			ALTER TABLE control_datasets DROP COLUMN IF EXISTS last_tested_at`

	default:
		return ""
	}
}