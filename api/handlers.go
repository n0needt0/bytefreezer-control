package api

import (
	"context"
	"fmt"
	"time"

	"github.com/n0needt0/bytefreezer-control/middleware"
	"github.com/n0needt0/bytefreezer-control/services"
	"github.com/n0needt0/go-goodies/log"
	"github.com/swaggest/usecase"
	usecaseStatus "github.com/swaggest/usecase/status"
)

// HealthResponse represents health check response
type HealthResponse struct {
	Status    string    `json:"status"`
	Service   string    `json:"service"`
	Version   string    `json:"version"`
	Timestamp time.Time `json:"timestamp"`
	Uptime    string    `json:"uptime"`
}

// ConfigResponse represents configuration response
type ConfigResponse struct {
	App       AppConfigResponse       `json:"app"`
	Services  ServicesConfigResponse  `json:"services"`
	Database  DatabaseConfigResponse  `json:"database"`
	Auth      AuthConfigResponse      `json:"auth"`
	RateLimit RateLimitConfigResponse `json:"rate_limit"`
}

type AppConfigResponse struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type ServicesConfigResponse struct {
	Receiver ServiceEndpointResponse `json:"receiver"`
	Proxy    ServiceEndpointResponse `json:"proxy"`
	SOC      ServiceEndpointResponse `json:"soc"`
	Packer   ServiceEndpointResponse `json:"packer"`
}

type ServiceEndpointResponse struct {
	URL            string `json:"url"`
	HealthEndpoint string `json:"health_endpoint"`
	ConfigEndpoint string `json:"config_endpoint"`
	TimeoutSeconds int    `json:"timeout_seconds"`
}

type DatabaseConfigResponse struct {
	Enabled bool   `json:"enabled"`
	Type    string `json:"type"`
	Host    string `json:"host"`
	Port    int    `json:"port"`
}

type AuthConfigResponse struct {
	Enabled          bool `json:"enabled"`
	TokenExpiryHours int  `json:"token_expiry_hours"`
	AdminUsersCount  int  `json:"admin_users_count"`
}

type RateLimitConfigResponse struct {
	Enabled           bool `json:"enabled"`
	RequestsPerMinute int  `json:"requests_per_minute"`
	BurstSize         int  `json:"burst_size"`
}

// EcosystemHealthResponse represents ecosystem health response
type EcosystemHealthResponse struct {
	OverallStatus string                             `json:"overall_status"`
	Services      map[string]*services.ServiceStatus `json:"services"`
	LastCheck     time.Time                          `json:"last_check"`
	HealthyCount  int                                `json:"healthy_count"`
	TotalCount    int                                `json:"total_count"`
}

// StatsResponse represents statistics response
type StatsResponse struct {
	Uptime                string    `json:"uptime"`
	APIRequests           int64     `json:"api_requests"`
	ServicesMonitored     int64     `json:"services_monitored"`
	HealthChecksPerformed int64     `json:"health_checks_performed"`
	ConfigUpdates         int64     `json:"config_updates"`
	DatabaseQueries       int64     `json:"database_queries"`
	LastActivity          time.Time `json:"last_activity"`
}

// HealthCheck returns the health status of the control service
func (api *API) HealthCheck() usecase.Interactor {
	type healthInput struct{}

	u := usecase.NewInteractor(func(ctx context.Context, input healthInput, output *HealthResponse) error {
		stats := api.Services.GetStats()
		uptime := time.Since(stats.StartTime)

		output.Status = "ok"
		output.Service = api.Config.App.Name
		output.Version = api.Config.App.Version
		output.Timestamp = time.Now()
		output.Uptime = uptime.String()

		return nil
	})

	u.SetTitle("Health Check")
	u.SetDescription("Returns the health status of the control service")
	u.SetTags("health")

	return u
}

// GetConfig returns the service configuration (sanitized)
func (api *API) GetConfig() usecase.Interactor {
	type configInput struct{}

	u := usecase.NewInteractor(func(ctx context.Context, input configInput, output *ConfigResponse) error {
		output.App = AppConfigResponse{
			Name:    api.Config.App.Name,
			Version: api.Config.App.Version,
		}

		output.Services = ServicesConfigResponse{
			Receiver: ServiceEndpointResponse{
				URL:            api.Config.Services.Receiver.URL,
				HealthEndpoint: api.Config.Services.Receiver.HealthEndpoint,
				ConfigEndpoint: api.Config.Services.Receiver.ConfigEndpoint,
				TimeoutSeconds: api.Config.Services.Receiver.TimeoutSeconds,
			},
			Proxy: ServiceEndpointResponse{
				URL:            api.Config.Services.Proxy.URL,
				HealthEndpoint: api.Config.Services.Proxy.HealthEndpoint,
				ConfigEndpoint: api.Config.Services.Proxy.ConfigEndpoint,
				TimeoutSeconds: api.Config.Services.Proxy.TimeoutSeconds,
			},
			SOC: ServiceEndpointResponse{
				URL:            api.Config.Services.SOC.URL,
				HealthEndpoint: api.Config.Services.SOC.HealthEndpoint,
				ConfigEndpoint: api.Config.Services.SOC.ConfigEndpoint,
				TimeoutSeconds: api.Config.Services.SOC.TimeoutSeconds,
			},
			Packer: ServiceEndpointResponse{
				URL:            api.Config.Services.Packer.URL,
				HealthEndpoint: api.Config.Services.Packer.HealthEndpoint,
				ConfigEndpoint: api.Config.Services.Packer.ConfigEndpoint,
				TimeoutSeconds: api.Config.Services.Packer.TimeoutSeconds,
			},
		}

		output.Database = DatabaseConfigResponse{
			Enabled: api.Config.Database.Enabled,
			Type:    api.Config.Database.Type,
			Host:    api.Config.Database.Host,
			Port:    api.Config.Database.Port,
		}

		output.Auth = AuthConfigResponse{
			Enabled:          api.Config.Auth.Enabled,
			TokenExpiryHours: api.Config.Auth.TokenExpiryHours,
			AdminUsersCount:  len(api.Config.Auth.AdminUsers),
		}

		output.RateLimit = RateLimitConfigResponse{
			Enabled:           api.Config.RateLimit.Enabled,
			RequestsPerMinute: api.Config.RateLimit.RequestsPerMinute,
			BurstSize:         api.Config.RateLimit.BurstSize,
		}

		return nil
	})

	u.SetTitle("Get Configuration")
	u.SetDescription("Returns the sanitized service configuration")
	u.SetTags("config")

	return u
}

// GetEcosystemHealth returns the health status of all ecosystem services
func (api *API) GetEcosystemHealth() usecase.Interactor {
	type ecosystemInput struct{}

	u := usecase.NewInteractor(func(ctx context.Context, input ecosystemInput, output *EcosystemHealthResponse) error {
		api.Services.IncrementAPIRequests()
		api.Services.IncrementHealthChecks()

		// Trigger fresh health check
		if err := api.Services.EcosystemMonitor.CheckAllServices(); err != nil {
			log.Warnf("Failed to check all services: %v", err)
		}

		statuses := api.Services.EcosystemMonitor.GetServiceStatuses()

		healthyCount := 0
		overallHealthy := true

		for _, status := range statuses {
			if status.Healthy {
				healthyCount++
			} else {
				overallHealthy = false
			}
		}

		output.Services = statuses
		output.LastCheck = time.Now()
		output.HealthyCount = healthyCount
		output.TotalCount = len(statuses)

		if overallHealthy && len(statuses) > 0 {
			output.OverallStatus = "healthy"
		} else if healthyCount > 0 {
			output.OverallStatus = "degraded"
		} else {
			output.OverallStatus = "unhealthy"
		}

		return nil
	})

	u.SetTitle("Get Ecosystem Health")
	u.SetDescription("Returns the health status of all ByteFreezer services")
	u.SetTags("ecosystem", "health")

	return u
}

// GetServiceStatuses returns the status of all services
func (api *API) GetServiceStatuses() usecase.Interactor {
	type servicesInput struct{}

	u := usecase.NewInteractor(func(ctx context.Context, input servicesInput, output *map[string]*services.ServiceStatus) error {
		api.Services.IncrementAPIRequests()

		statuses := api.Services.EcosystemMonitor.GetServiceStatuses()
		*output = statuses

		return nil
	})

	u.SetTitle("Get Service Statuses")
	u.SetDescription("Returns the status of all monitored services")
	u.SetTags("ecosystem", "services")

	return u
}

// GetServiceStatus returns the status of a specific service
func (api *API) GetServiceStatus() usecase.Interactor {
	type serviceInput struct {
		ServiceName string `path:"serviceName"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input serviceInput, output **services.ServiceStatus) error {
		api.Services.IncrementAPIRequests()
		api.Services.IncrementHealthChecks()

		serviceStatus, err := api.Services.EcosystemMonitor.CheckService(input.ServiceName)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to check service %s: %w", input.ServiceName, err), usecaseStatus.Internal)
		}

		*output = serviceStatus
		return nil
	})

	u.SetTitle("Get Service Status")
	u.SetDescription("Returns the status of a specific service")
	u.SetTags("ecosystem", "services")
	// u.SetExpectedErrors(usecaseStatus.NotFound, usecaseStatus.Internal)

	return u
}

// RestartService restarts a specific service (placeholder)
func (api *API) RestartService() usecase.Interactor {
	type restartInput struct {
		ServiceName string `path:"serviceName"`
	}

	type restartOutput struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input restartInput, output *restartOutput) error {
		api.Services.IncrementAPIRequests()

		// TODO: Implement actual service restart logic
		log.Infof("Restart requested for service: %s", input.ServiceName)

		output.Success = false
		output.Message = fmt.Sprintf("Service restart not implemented for %s", input.ServiceName)

		return fmt.Errorf("service restart not implemented")
	})

	u.SetTitle("Restart Service")
	u.SetDescription("Restarts a specific ByteFreezer service")
	u.SetTags("ecosystem", "control")
	// u.SetExpectedErrors(usecaseStatus.NotFound, usecaseStatus.NotImplemented501)

	return u
}

// GetTenants returns all tenants (placeholder)
func (api *API) GetTenants() usecase.Interactor {
	type tenantsInput struct{}

	u := usecase.NewInteractor(func(ctx context.Context, input tenantsInput, output *[]services.Tenant) error {
		api.Services.IncrementAPIRequests()
		api.Services.IncrementDatabaseQueries()

		// TODO: Implement actual database query
		*output = []services.Tenant{}

		return nil
	})

	u.SetTitle("Get Tenants")
	u.SetDescription("Returns all tenants")
	u.SetTags("tenants")

	return u
}

// GetTenant returns a specific tenant (placeholder)
func (api *API) GetTenant() usecase.Interactor {
	type tenantInput struct {
		TenantID string `path:"tenantId"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input tenantInput, output **services.Tenant) error {
		api.Services.IncrementAPIRequests()
		api.Services.IncrementDatabaseQueries()

		// TODO: Implement actual database query
		return fmt.Errorf("tenant not found")
	})

	u.SetTitle("Get Tenant")
	u.SetDescription("Returns a specific tenant by ID")
	u.SetTags("tenants")
	// u.SetExpectedErrors(usecaseStatus.NotFound)

	return u
}

// CreateTenant creates a new tenant (placeholder)
func (api *API) CreateTenant() usecase.Interactor {
	type createInput struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input createInput, output **services.Tenant) error {
		api.Services.IncrementAPIRequests()
		api.Services.IncrementDatabaseQueries()

		// TODO: Implement actual tenant creation
		return fmt.Errorf("tenant creation not implemented")
	})

	u.SetTitle("Create Tenant")
	u.SetDescription("Creates a new tenant")
	u.SetTags("tenants")
	// u.SetExpectedErrors(usecaseStatus.InvalidArgument, usecaseStatus.NotImplemented501)

	return u
}

// UpdateTenant updates an existing tenant (placeholder)
func (api *API) UpdateTenant() usecase.Interactor {
	type updateInput struct {
		TenantID string `path:"tenantId"`
		Name     string `json:"name"`
		Email    string `json:"email"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input updateInput, output **services.Tenant) error {
		api.Services.IncrementAPIRequests()
		api.Services.IncrementDatabaseQueries()

		// TODO: Implement actual tenant update
		return fmt.Errorf("tenant update not implemented")
	})

	u.SetTitle("Update Tenant")
	u.SetDescription("Updates an existing tenant")
	u.SetTags("tenants")
	// u.SetExpectedErrors(usecaseStatus.NotFound, usecaseStatus.InvalidArgument, usecaseStatus.NotImplemented501)

	return u
}

// DeleteTenant deletes a tenant (placeholder)
func (api *API) DeleteTenant() usecase.Interactor {
	type deleteInput struct {
		TenantID string `path:"tenantId"`
	}

	type deleteOutput struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input deleteInput, output *deleteOutput) error {
		api.Services.IncrementAPIRequests()
		api.Services.IncrementDatabaseQueries()

		// TODO: Implement actual tenant deletion
		output.Success = false
		output.Message = "Tenant deletion not implemented"

		return fmt.Errorf("tenant deletion not implemented")
	})

	u.SetTitle("Delete Tenant")
	u.SetDescription("Deletes a tenant")
	u.SetTags("tenants")
	// u.SetExpectedErrors(usecaseStatus.NotFound, usecaseStatus.NotImplemented501)

	return u
}

// LoginRequest represents login request
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents login response
type LoginResponse struct {
	Token   string `json:"token"`
	Expires string `json:"expires"`
}

// Login authenticates a user and returns a JWT token
func (api *API) Login() usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input LoginRequest, output *LoginResponse) error {
		api.Services.IncrementAPIRequests()

		// Check if user is in admin list
		isAdmin := false
		for _, adminUser := range api.Config.Auth.AdminUsers {
			if adminUser == input.Username {
				isAdmin = true
				break
			}
		}

		// TODO: Implement actual password verification
		// For now, just check if user exists in admin list
		if !isAdmin {
			return fmt.Errorf("invalid credentials")
		}

		// Generate JWT token
		token, err := middleware.GenerateToken(input.Username, isAdmin, api.Config.Auth)
		if err != nil {
			return fmt.Errorf("failed to generate token: %w", err)
		}

		expiresAt := time.Now().Add(time.Duration(api.Config.Auth.TokenExpiryHours) * time.Hour)

		output.Token = token
		output.Expires = expiresAt.Format(time.RFC3339)

		return nil
	})

	u.SetTitle("User Login")
	u.SetDescription("Authenticates a user and returns a JWT token")
	u.SetTags("auth")
	// u.SetExpectedErrors(usecaseStatus.Unauthorized401, usecaseStatus.Internal)

	return u
}

// GetStats returns service statistics
func (api *API) GetStats() usecase.Interactor {
	type statsInput struct{}

	u := usecase.NewInteractor(func(ctx context.Context, input statsInput, output *StatsResponse) error {
		stats := api.Services.GetStats()
		uptime := time.Since(stats.StartTime)

		output.Uptime = uptime.String()
		output.APIRequests = stats.APIRequests
		output.ServicesMonitored = stats.ServicesMonitored
		output.HealthChecksPerformed = stats.HealthChecksPerformed
		output.ConfigUpdates = stats.ConfigUpdates
		output.DatabaseQueries = stats.DatabaseQueries
		output.LastActivity = stats.LastActivity

		return nil
	})

	u.SetTitle("Get Statistics")
	u.SetDescription("Returns service statistics and metrics")
	u.SetTags("stats")

	return u
}
