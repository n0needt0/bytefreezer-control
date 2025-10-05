package client

import (
	"context"
	"sync"
	"time"
)

// ConfigHelper provides configuration with automatic fallback
type ConfigHelper struct {
	client         *Client
	localConfig    map[string]interface{}
	defaultConfig  map[string]interface{}
	cacheDuration  time.Duration
	cachedTenant   *Tenant
	cacheExpiry    time.Time
	mu             sync.RWMutex
}

// NewConfigHelper creates a new configuration helper with fallback support
func NewConfigHelper(client *Client, localConfig, defaultConfig map[string]interface{}) *ConfigHelper {
	return &ConfigHelper{
		client:        client,
		localConfig:   localConfig,
		defaultConfig: defaultConfig,
		cacheDuration: 5 * time.Minute,
	}
}

// GetTenantConfig retrieves tenant configuration with fallback priority:
// 1. Control Service API (cached for cacheDuration)
// 2. Local configuration
// 3. Default configuration
func (h *ConfigHelper) GetTenantConfig(ctx context.Context, accountID, tenantID string) (map[string]interface{}, error) {
	h.mu.RLock()
	// Check cache first
	if h.cachedTenant != nil && time.Now().Before(h.cacheExpiry) {
		config := h.cachedTenant.Config
		h.mu.RUnlock()
		return config, nil
	}
	h.mu.RUnlock()

	// Try to fetch from API
	tenant, err := h.client.GetTenant(ctx, accountID, tenantID)
	if err == nil && tenant != nil {
		h.mu.Lock()
		h.cachedTenant = tenant
		h.cacheExpiry = time.Now().Add(h.cacheDuration)
		h.mu.Unlock()
		return tenant.Config, nil
	}

	// Fallback to local config
	if h.localConfig != nil {
		return h.localConfig, nil
	}

	// Final fallback to default config
	return h.defaultConfig, nil
}

// GetConfigString retrieves a string config value with fallback
func (h *ConfigHelper) GetConfigString(ctx context.Context, accountID, tenantID, key, defaultValue string) string {
	config, err := h.GetTenantConfig(ctx, accountID, tenantID)
	if err != nil || config == nil {
		return defaultValue
	}

	if val, ok := config[key].(string); ok {
		return val
	}

	return defaultValue
}

// GetConfigInt retrieves an int config value with fallback
func (h *ConfigHelper) GetConfigInt(ctx context.Context, accountID, tenantID, key string, defaultValue int) int {
	config, err := h.GetTenantConfig(ctx, accountID, tenantID)
	if err != nil || config == nil {
		return defaultValue
	}

	// Handle both int and float64 (JSON numbers are float64)
	switch val := config[key].(type) {
	case int:
		return val
	case float64:
		return int(val)
	}

	return defaultValue
}

// GetConfigBool retrieves a bool config value with fallback
func (h *ConfigHelper) GetConfigBool(ctx context.Context, accountID, tenantID, key string, defaultValue bool) bool {
	config, err := h.GetTenantConfig(ctx, accountID, tenantID)
	if err != nil || config == nil {
		return defaultValue
	}

	if val, ok := config[key].(bool); ok {
		return val
	}

	return defaultValue
}

// InvalidateCache forces a refresh on next config fetch
func (h *ConfigHelper) InvalidateCache() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cachedTenant = nil
	h.cacheExpiry = time.Time{}
}

// SetCacheDuration changes the cache duration
func (h *ConfigHelper) SetCacheDuration(d time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cacheDuration = d
}
