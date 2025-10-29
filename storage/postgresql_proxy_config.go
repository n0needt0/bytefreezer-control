package storage

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/lib/pq"
	"github.com/n0needt0/go-goodies/log"
)

// UpsertProxyConfig creates or updates a proxy instance configuration
func (p *PostgreSQLStorage) UpsertProxyConfig(ctx context.Context, config *ProxyInstanceConfig) error {
	// Validate required fields
	if config.InstanceID == "" {
		return fmt.Errorf("instance_id is required")
	}
	if config.TenantID == "" {
		return fmt.Errorf("tenant_id is required")
	}
	if config.InstanceAPI == "" {
		return fmt.Errorf("instance_api is required")
	}

	// Set defaults
	if config.ConfigMode == "" {
		config.ConfigMode = "hybrid"
	}
	if config.PluginConfigs == nil {
		config.PluginConfigs = []map[string]interface{}{}
	}
	if config.ProxySettings.Custom == nil {
		config.ProxySettings.Custom = make(map[string]interface{})
	}

	// Calculate config hash for change detection
	configHash, err := calculateConfigHash(config.PluginConfigs, config.ProxySettings)
	if err != nil {
		return fmt.Errorf("failed to calculate config hash: %w", err)
	}
	config.ConfigHash = configHash

	// Marshal JSONB fields
	pluginConfigsJSON, err := json.Marshal(config.PluginConfigs)
	if err != nil {
		return fmt.Errorf("failed to marshal plugin configs: %w", err)
	}

	proxySettingsJSON, err := json.Marshal(config.ProxySettings)
	if err != nil {
		return fmt.Errorf("failed to marshal proxy settings: %w", err)
	}

	// Use the database function to upsert
	query := `
		SELECT id, config_version FROM upsert_proxy_config(
			$1::VARCHAR(255),
			$2::VARCHAR(255),
			$3::VARCHAR(255),
			$4::VARCHAR(50),
			$5::JSONB,
			$6::JSONB,
			$7::VARCHAR(64)
		)`

	var id int
	var version int
	err = p.db.QueryRowContext(ctx, query,
		config.InstanceID,
		config.TenantID,
		config.InstanceAPI,
		config.ConfigMode,
		pluginConfigsJSON,
		proxySettingsJSON,
		configHash,
	).Scan(&id, &version)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" { // unique violation
			return fmt.Errorf("proxy instance already exists with different tenant")
		}
		return fmt.Errorf("failed to upsert proxy config: %w", err)
	}

	config.ConfigVersion = version
	return nil
}

// GetProxyConfig retrieves a proxy instance configuration
func (p *PostgreSQLStorage) GetProxyConfig(ctx context.Context, instanceID, tenantID string) (*ProxyInstanceConfig, error) {
	query := `
		SELECT
			instance_id, tenant_id, instance_api, config_mode,
			plugin_configs, proxy_settings, config_version, config_hash,
			config_applied, config_applied_at, active, created_at, updated_at
		FROM proxy_instances
		WHERE instance_id = $1 AND tenant_id = $2`

	var config ProxyInstanceConfig
	var pluginConfigsJSON []byte
	var proxySettingsJSON []byte

	err := p.db.QueryRowContext(ctx, query, instanceID, tenantID).Scan(
		&config.InstanceID,
		&config.TenantID,
		&config.InstanceAPI,
		&config.ConfigMode,
		&pluginConfigsJSON,
		&proxySettingsJSON,
		&config.ConfigVersion,
		&config.ConfigHash,
		&config.ConfigApplied,
		&config.ConfigAppliedAt,
		&config.Active,
		&config.CreatedAt,
		&config.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("proxy instance not found: %s/%s", tenantID, instanceID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get proxy config: %w", err)
	}

	// Unmarshal JSONB fields
	if err := json.Unmarshal(pluginConfigsJSON, &config.PluginConfigs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal plugin configs: %w", err)
	}

	if err := json.Unmarshal(proxySettingsJSON, &config.ProxySettings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal proxy settings: %w", err)
	}

	return &config, nil
}

// ListProxyConfigs retrieves all proxy configurations for a tenant
func (p *PostgreSQLStorage) ListProxyConfigs(ctx context.Context, tenantID string) ([]*ProxyInstanceConfig, error) {
	query := `
		SELECT
			instance_id, tenant_id, instance_api, config_mode,
			plugin_configs, proxy_settings, config_version, config_hash,
			config_applied, config_applied_at, active, created_at, updated_at
		FROM proxy_instances
		WHERE tenant_id = $1
		ORDER BY instance_id ASC`

	rows, err := p.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list proxy configs: %w", err)
	}
	defer rows.Close()

	configs := []*ProxyInstanceConfig{}
	for rows.Next() {
		var config ProxyInstanceConfig
		var pluginConfigsJSON []byte
		var proxySettingsJSON []byte

		err := rows.Scan(
			&config.InstanceID,
			&config.TenantID,
			&config.InstanceAPI,
			&config.ConfigMode,
			&pluginConfigsJSON,
			&proxySettingsJSON,
			&config.ConfigVersion,
			&config.ConfigHash,
			&config.ConfigApplied,
			&config.ConfigAppliedAt,
			&config.Active,
			&config.CreatedAt,
			&config.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan proxy config: %w", err)
		}

		// Unmarshal JSONB fields
		if err := json.Unmarshal(pluginConfigsJSON, &config.PluginConfigs); err != nil {
			return nil, fmt.Errorf("failed to unmarshal plugin configs: %w", err)
		}

		if err := json.Unmarshal(proxySettingsJSON, &config.ProxySettings); err != nil {
			return nil, fmt.Errorf("failed to unmarshal proxy settings: %w", err)
		}

		configs = append(configs, &config)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating proxy configs: %w", err)
	}

	return configs, nil
}

// ListAllProxyConfigs retrieves all proxy configurations across all tenants
func (p *PostgreSQLStorage) ListAllProxyConfigs(ctx context.Context) ([]*ProxyInstanceConfig, error) {
	query := `
		SELECT
			instance_id, tenant_id, instance_api, config_mode,
			plugin_configs, proxy_settings, config_version, config_hash,
			config_applied, config_applied_at, active, created_at, updated_at
		FROM proxy_instances
		WHERE active = true
		ORDER BY tenant_id ASC, instance_id ASC`

	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list all proxy configs: %w", err)
	}
	defer rows.Close()

	configs := []*ProxyInstanceConfig{}
	for rows.Next() {
		var config ProxyInstanceConfig
		var pluginConfigsJSON []byte
		var proxySettingsJSON []byte

		err := rows.Scan(
			&config.InstanceID,
			&config.TenantID,
			&config.InstanceAPI,
			&config.ConfigMode,
			&pluginConfigsJSON,
			&proxySettingsJSON,
			&config.ConfigVersion,
			&config.ConfigHash,
			&config.ConfigApplied,
			&config.ConfigAppliedAt,
			&config.Active,
			&config.CreatedAt,
			&config.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan proxy config: %w", err)
		}

		// Unmarshal JSONB fields
		if err := json.Unmarshal(pluginConfigsJSON, &config.PluginConfigs); err != nil {
			return nil, fmt.Errorf("failed to unmarshal plugin configs: %w", err)
		}

		if err := json.Unmarshal(proxySettingsJSON, &config.ProxySettings); err != nil {
			return nil, fmt.Errorf("failed to unmarshal proxy settings: %w", err)
		}

		configs = append(configs, &config)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating all proxy configs: %w", err)
	}

	return configs, nil
}

// ListProxyConfigsForAccount retrieves proxy configurations for a specific account (account-filtered)
func (p *PostgreSQLStorage) ListProxyConfigsForAccount(ctx context.Context, accountID string) ([]*ProxyInstanceConfig, error) {
	log.Debugf("Listing proxy configs for account_id=%s", accountID)

	query := `
		SELECT
			pi.instance_id, pi.tenant_id, pi.instance_api, pi.config_mode,
			pi.plugin_configs, pi.proxy_settings, pi.config_version, pi.config_hash,
			pi.config_applied, pi.config_applied_at, pi.active, pi.created_at, pi.updated_at
		FROM proxy_instances pi
		INNER JOIN control_tenants t ON pi.tenant_id = t.id
		WHERE t.account_id = $1 AND pi.active = true
		ORDER BY pi.tenant_id ASC, pi.instance_id ASC`

	rows, err := p.db.QueryContext(ctx, query, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to list proxy configs for account %s: %w", accountID, err)
	}
	defer rows.Close()

	configs := []*ProxyInstanceConfig{}
	for rows.Next() {
		var config ProxyInstanceConfig
		var pluginConfigsJSON []byte
		var proxySettingsJSON []byte

		err := rows.Scan(
			&config.InstanceID,
			&config.TenantID,
			&config.InstanceAPI,
			&config.ConfigMode,
			&pluginConfigsJSON,
			&proxySettingsJSON,
			&config.ConfigVersion,
			&config.ConfigHash,
			&config.ConfigApplied,
			&config.ConfigAppliedAt,
			&config.Active,
			&config.CreatedAt,
			&config.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan proxy config: %w", err)
		}

		// Unmarshal JSONB fields
		if err := json.Unmarshal(pluginConfigsJSON, &config.PluginConfigs); err != nil {
			return nil, fmt.Errorf("failed to unmarshal plugin configs: %w", err)
		}

		if err := json.Unmarshal(proxySettingsJSON, &config.ProxySettings); err != nil {
			return nil, fmt.Errorf("failed to unmarshal proxy settings: %w", err)
		}

		configs = append(configs, &config)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating proxy configs for account: %w", err)
	}

	log.Debugf("Found %d proxy configs for account_id=%s", len(configs), accountID)

	return configs, nil
}

// DeleteProxyConfig deletes a proxy instance configuration
func (p *PostgreSQLStorage) DeleteProxyConfig(ctx context.Context, instanceID, tenantID string) error {
	query := `DELETE FROM proxy_instances WHERE instance_id = $1 AND tenant_id = $2`

	result, err := p.db.ExecContext(ctx, query, instanceID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to delete proxy config: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("proxy instance not found: %s/%s", tenantID, instanceID)
	}

	return nil
}

// MarkProxyConfigApplied marks a proxy configuration as applied
func (p *PostgreSQLStorage) MarkProxyConfigApplied(ctx context.Context, instanceID, tenantID string, configVersion int) error {
	query := `
		SELECT mark_proxy_config_applied(
			$1::VARCHAR(255),
			$2::VARCHAR(255),
			$3::INTEGER
		)`

	var success bool
	err := p.db.QueryRowContext(ctx, query, instanceID, tenantID, configVersion).Scan(&success)
	if err != nil {
		return fmt.Errorf("failed to mark proxy config applied: %w", err)
	}

	if !success {
		return fmt.Errorf("proxy instance not found or config version mismatch: %s/%s v%d", tenantID, instanceID, configVersion)
	}

	return nil
}

// GetProxyConfigHistory retrieves configuration history for a proxy instance
func (p *PostgreSQLStorage) GetProxyConfigHistory(ctx context.Context, instanceID, tenantID string, limit int) ([]*ProxyConfigHistory, error) {
	if limit <= 0 {
		limit = 10 // Default to last 10 versions
	}

	query := `
		SELECT
			id, instance_id, tenant_id, config_version,
			plugin_configs, proxy_settings, config_hash,
			changed_by, change_reason, timestamp
		FROM proxy_config_history
		WHERE instance_id = $1 AND tenant_id = $2
		ORDER BY timestamp DESC
		LIMIT $3`

	rows, err := p.db.QueryContext(ctx, query, instanceID, tenantID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get proxy config history: %w", err)
	}
	defer rows.Close()

	history := []*ProxyConfigHistory{}
	for rows.Next() {
		var record ProxyConfigHistory
		var pluginConfigsJSON []byte
		var proxySettingsJSON []byte
		var changedBy, changeReason sql.NullString

		err := rows.Scan(
			&record.ID,
			&record.InstanceID,
			&record.TenantID,
			&record.ConfigVersion,
			&pluginConfigsJSON,
			&proxySettingsJSON,
			&record.ConfigHash,
			&changedBy,
			&changeReason,
			&record.Timestamp,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan proxy config history: %w", err)
		}

		// Unmarshal JSONB fields
		if err := json.Unmarshal(pluginConfigsJSON, &record.PluginConfigs); err != nil {
			return nil, fmt.Errorf("failed to unmarshal plugin configs: %w", err)
		}

		if err := json.Unmarshal(proxySettingsJSON, &record.ProxySettings); err != nil {
			return nil, fmt.Errorf("failed to unmarshal proxy settings: %w", err)
		}

		if changedBy.Valid {
			record.ChangedBy = changedBy.String
		}
		if changeReason.Valid {
			record.ChangeReason = changeReason.String
		}

		history = append(history, &record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating proxy config history: %w", err)
	}

	return history, nil
}

// calculateConfigHash computes a SHA256 hash of the configuration for change detection
func calculateConfigHash(pluginConfigs []map[string]interface{}, proxySettings ProxySettings) (string, error) {
	// Combine both configs into a single struct for hashing
	combined := struct {
		PluginConfigs []map[string]interface{} `json:"plugin_configs"`
		ProxySettings ProxySettings            `json:"proxy_settings"`
	}{
		PluginConfigs: pluginConfigs,
		ProxySettings: proxySettings,
	}

	data, err := json.Marshal(combined)
	if err != nil {
		return "", fmt.Errorf("failed to marshal config for hashing: %w", err)
	}

	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}
