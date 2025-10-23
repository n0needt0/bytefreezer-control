package api

import (
	"context"
	"fmt"

	"github.com/n0needt0/bytefreezer-control/storage"
	"github.com/n0needt0/go-goodies/log"
	"github.com/swaggest/usecase"
	usecaseStatus "github.com/swaggest/usecase/status"
)

// ListProxyInstances returns all proxy instances across all tenants
func (api *API) ListProxyInstances() usecase.Interactor {
	type listProxyInput struct {
		TenantID string `query:"tenant_id" description:"Filter by tenant ID (optional)"`
	}

	type listProxyOutput struct {
		Proxies []storage.ProxyInstanceConfig `json:"proxies"`
		Count   int                           `json:"count"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input listProxyInput, output *listProxyOutput) error {
		var proxies []*storage.ProxyInstanceConfig
		var err error

		if input.TenantID != "" {
			// List proxies for specific tenant
			proxies, err = api.Services.Storage.ListProxyConfigs(ctx, input.TenantID)
			if err != nil {
				log.Errorf("Failed to list proxy configs for tenant %s: %v", input.TenantID, err)
				return usecaseStatus.Wrap(fmt.Errorf("failed to list proxy instances"), usecaseStatus.Internal)
			}
		} else {
			// List all proxies
			proxies, err = api.Services.Storage.ListAllProxyConfigs(ctx)
			if err != nil {
				log.Errorf("Failed to list all proxy configs: %v", err)
				return usecaseStatus.Wrap(fmt.Errorf("failed to list proxy instances"), usecaseStatus.Internal)
			}
		}

		// Convert to non-pointer slice for output
		output.Proxies = make([]storage.ProxyInstanceConfig, len(proxies))
		for i, proxy := range proxies {
			output.Proxies[i] = *proxy
		}
		output.Count = len(proxies)

		return nil
	})

	u.SetTitle("List Proxy Instances")
	u.SetDescription("Returns list of all proxy instances, optionally filtered by tenant")
	u.SetTags("proxy-config")

	return u
}

// GetProxyConfig returns configuration for a specific proxy instance
func (api *API) GetProxyConfig() usecase.Interactor {
	type getProxyConfigInput struct {
		InstanceID string `path:"instanceId" description:"Proxy instance ID (hostname)"`
		TenantID   string `query:"tenant_id" description:"Tenant ID" required:"true"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input getProxyConfigInput, output *storage.ProxyInstanceConfig) error {
		if input.TenantID == "" {
			return usecaseStatus.Wrap(fmt.Errorf("tenant_id query parameter is required"), usecaseStatus.InvalidArgument)
		}

		config, err := api.Services.Storage.GetProxyConfig(ctx, input.InstanceID, input.TenantID)
		if err != nil {
			log.Errorf("Failed to get proxy config for %s/%s: %v", input.TenantID, input.InstanceID, err)
			return usecaseStatus.Wrap(fmt.Errorf("proxy instance not found"), usecaseStatus.NotFound)
		}

		*output = *config
		return nil
	})

	u.SetTitle("Get Proxy Configuration")
	u.SetDescription("Returns configuration for a specific proxy instance")
	u.SetTags("proxy-config")

	return u
}

// UpsertProxyConfig creates or updates proxy configuration
func (api *API) UpsertProxyConfig() usecase.Interactor {
	type upsertProxyConfigInput struct {
		InstanceID    string                   `path:"instanceId" description:"Proxy instance ID (hostname)"`
		TenantID      string                   `json:"tenant_id" required:"true" description:"Tenant ID"`
		InstanceAPI   string                   `json:"instance_api" required:"true" description:"Instance API endpoint (hostname:port)"`
		ConfigMode    string                   `json:"config_mode" default:"hybrid" description:"Configuration mode: local-only, control-only, or hybrid"`
		PluginConfigs []map[string]interface{} `json:"plugin_configs" description:"Array of plugin configurations"`
		ProxySettings storage.ProxySettings    `json:"proxy_settings" description:"Proxy-level settings"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input upsertProxyConfigInput, output *storage.ProxyInstanceConfig) error {
		// Validate config mode
		validModes := map[string]bool{
			"local-only":   true,
			"control-only": true,
			"hybrid":       true,
		}
		if !validModes[input.ConfigMode] {
			return usecaseStatus.Wrap(
				fmt.Errorf("invalid config_mode: must be 'local-only', 'control-only', or 'hybrid'"),
				usecaseStatus.InvalidArgument,
			)
		}

		// Create config object
		config := &storage.ProxyInstanceConfig{
			InstanceID:    input.InstanceID,
			TenantID:      input.TenantID,
			InstanceAPI:   input.InstanceAPI,
			ConfigMode:    input.ConfigMode,
			PluginConfigs: input.PluginConfigs,
			ProxySettings: input.ProxySettings,
		}

		// Set defaults for proxy settings if not provided
		if config.ProxySettings.Custom == nil {
			config.ProxySettings.Custom = make(map[string]interface{})
		}
		if config.PluginConfigs == nil {
			config.PluginConfigs = []map[string]interface{}{}
		}

		// Note: No validation of dataset_ids here because proxy generates plugin configs
		// dynamically from dataset configurations and reports back what it has applied.
		// This endpoint accepts whatever the proxy reports.

		// Upsert configuration
		err := api.Services.Storage.UpsertProxyConfig(ctx, config)
		if err != nil {
			log.Errorf("Failed to upsert proxy config for %s/%s: %v", input.TenantID, input.InstanceID, err)
			return usecaseStatus.Wrap(fmt.Errorf("failed to save proxy configuration"), usecaseStatus.Internal)
		}

		log.Infof("Proxy config upserted for %s/%s (version %d)", input.TenantID, input.InstanceID, config.ConfigVersion)

		*output = *config
		return nil
	})

	u.SetTitle("Upsert Proxy Configuration")
	u.SetDescription("Creates or updates proxy instance configuration")
	u.SetTags("proxy-config")

	return u
}

// MarkProxyConfigApplied marks a configuration version as applied by the proxy
func (api *API) MarkProxyConfigApplied() usecase.Interactor {
	type markAppliedInput struct {
		InstanceID    string `path:"instanceId" description:"Proxy instance ID (hostname)"`
		TenantID      string `json:"tenant_id" required:"true" description:"Tenant ID"`
		ConfigVersion int    `json:"config_version" required:"true" description:"Configuration version number"`
	}

	type markAppliedOutput struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input markAppliedInput, output *markAppliedOutput) error {
		err := api.Services.Storage.MarkProxyConfigApplied(ctx, input.InstanceID, input.TenantID, input.ConfigVersion)
		if err != nil {
			log.Errorf("Failed to mark proxy config applied for %s/%s v%d: %v",
				input.TenantID, input.InstanceID, input.ConfigVersion, err)
			return usecaseStatus.Wrap(fmt.Errorf("failed to mark configuration as applied"), usecaseStatus.Internal)
		}

		log.Infof("Proxy config marked as applied: %s/%s v%d", input.TenantID, input.InstanceID, input.ConfigVersion)

		output.Success = true
		output.Message = fmt.Sprintf("Configuration version %d marked as applied", input.ConfigVersion)
		return nil
	})

	u.SetTitle("Mark Proxy Config Applied")
	u.SetDescription("Marks a specific configuration version as successfully applied by the proxy")
	u.SetTags("proxy-config")

	return u
}

// GetProxyConfigHistory returns configuration history for a proxy instance
func (api *API) GetProxyConfigHistory() usecase.Interactor {
	type getHistoryInput struct {
		InstanceID string `path:"instanceId" description:"Proxy instance ID (hostname)"`
		TenantID   string `query:"tenant_id" required:"true" description:"Tenant ID"`
		Limit      int    `query:"limit" default:"10" description:"Maximum number of history records to return"`
	}

	type getHistoryOutput struct {
		History []storage.ProxyConfigHistory `json:"history"`
		Count   int                          `json:"count"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input getHistoryInput, output *getHistoryOutput) error {
		if input.TenantID == "" {
			return usecaseStatus.Wrap(fmt.Errorf("tenant_id query parameter is required"), usecaseStatus.InvalidArgument)
		}

		if input.Limit <= 0 {
			input.Limit = 10
		}

		history, err := api.Services.Storage.GetProxyConfigHistory(ctx, input.InstanceID, input.TenantID, input.Limit)
		if err != nil {
			log.Errorf("Failed to get proxy config history for %s/%s: %v", input.TenantID, input.InstanceID, err)
			return usecaseStatus.Wrap(fmt.Errorf("failed to retrieve configuration history"), usecaseStatus.Internal)
		}

		// Convert to non-pointer slice for output
		output.History = make([]storage.ProxyConfigHistory, len(history))
		for i, record := range history {
			output.History[i] = *record
		}
		output.Count = len(history)

		return nil
	})

	u.SetTitle("Get Proxy Config History")
	u.SetDescription("Returns configuration change history for a proxy instance")
	u.SetTags("proxy-config")

	return u
}

// DeleteProxyInstance deletes a proxy instance configuration
func (api *API) DeleteProxyInstance() usecase.Interactor {
	type deleteProxyInput struct {
		InstanceID string `path:"instanceId" description:"Proxy instance ID (hostname)"`
		TenantID   string `query:"tenant_id" required:"true" description:"Tenant ID"`
	}

	type deleteProxyOutput struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input deleteProxyInput, output *deleteProxyOutput) error {
		if input.TenantID == "" {
			return usecaseStatus.Wrap(fmt.Errorf("tenant_id query parameter is required"), usecaseStatus.InvalidArgument)
		}

		err := api.Services.Storage.DeleteProxyConfig(ctx, input.InstanceID, input.TenantID)
		if err != nil {
			log.Errorf("Failed to delete proxy config for %s/%s: %v", input.TenantID, input.InstanceID, err)
			return usecaseStatus.Wrap(fmt.Errorf("failed to delete proxy instance"), usecaseStatus.NotFound)
		}

		log.Infof("Proxy instance deleted: %s/%s", input.TenantID, input.InstanceID)

		output.Success = true
		output.Message = fmt.Sprintf("Proxy instance %s deleted successfully", input.InstanceID)
		return nil
	})

	u.SetTitle("Delete Proxy Instance")
	u.SetDescription("Deletes a proxy instance and its configuration")
	u.SetTags("proxy-config")

	return u
}

// GetProxyConfiguration returns tenants and datasets for an account (for proxy polling)
func (api *API) GetProxyConfiguration() usecase.Interactor {
	type getProxyConfigInput struct {
		AccountID string `query:"account_id" required:"true" description:"Account ID"`
	}

	type tenantWithDatasets struct {
		Tenant   storage.Tenant    `json:"tenant"`
		Datasets []storage.Dataset `json:"datasets"`
	}

	type getProxyConfigOutput struct {
		Tenants []tenantWithDatasets `json:"tenants"`
		Count   int                  `json:"count"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input getProxyConfigInput, output *getProxyConfigOutput) error {
		// Query all tenants for the account
		tenantOpts := storage.ListOptions{Limit: 1000}
		tenantResult, err := api.Services.Storage.ListTenants(ctx, input.AccountID, tenantOpts)
		if err != nil {
			log.Errorf("Failed to list tenants for account %s: %v", input.AccountID, err)
			return usecaseStatus.Wrap(fmt.Errorf("failed to retrieve tenants"), usecaseStatus.Internal)
		}

		// For each tenant, query its datasets
		tenantsWithDatasets := make([]tenantWithDatasets, 0, len(tenantResult.Items))
		for _, tenant := range tenantResult.Items {
			datasetOpts := storage.ListOptions{Limit: 1000}
			datasetResult, err := api.Services.Storage.ListDatasets(ctx, tenant.ID, datasetOpts)
			if err != nil {
				log.Warnf("Failed to list datasets for tenant %s: %v", tenant.ID, err)
				continue
			}

			tenantsWithDatasets = append(tenantsWithDatasets, tenantWithDatasets{
				Tenant:   tenant,
				Datasets: datasetResult.Items,
			})
		}

		output.Tenants = tenantsWithDatasets
		output.Count = len(tenantsWithDatasets)

		log.Infof("Proxy configuration query for account %s: %d tenants, total datasets across all tenants",
			input.AccountID, len(tenantsWithDatasets))

		return nil
	})

	u.SetTitle("Get Proxy Configuration")
	u.SetDescription("Returns all tenants and datasets for an account (used by proxy for configuration polling)")
	u.SetTags("proxy-config")

	return u
}
