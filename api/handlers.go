package api

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"strings"
	"time"

	"github.com/n0needt0/bytefreezer-control/services"
	"github.com/n0needt0/bytefreezer-control/storage"
	"github.com/n0needt0/go-goodies/log"
	"github.com/swaggest/usecase"
	usecaseStatus "github.com/swaggest/usecase/status"
)

// GenerateShortID generates a short, URL-safe ID (12 characters)
func GenerateShortID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}

	encoded := base32.StdEncoding.EncodeToString(b)
	id := strings.ToLower(strings.TrimRight(encoded, "="))

	if len(id) > 12 {
		id = id[:12]
	}

	return id
}

// ValidateID validates that an ID follows the required format: lowercase alphanumeric + dash
// Must start with letter, 3-63 characters
func ValidateID(id string) error {
	if len(id) < 3 || len(id) > 63 {
		return fmt.Errorf("ID must be between 3 and 63 characters")
	}

	// Must start with a letter
	if id[0] < 'a' || id[0] > 'z' {
		return fmt.Errorf("ID must start with a lowercase letter")
	}

	// Check all characters
	for i, ch := range id {
		if !((ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-') {
			return fmt.Errorf("ID can only contain lowercase letters, numbers, and dashes")
		}

		// Cannot end with dash
		if i == len(id)-1 && ch == '-' {
			return fmt.Errorf("ID cannot end with a dash")
		}

		// Cannot have consecutive dashes
		if ch == '-' && i > 0 && id[i-1] == '-' {
			return fmt.Errorf("ID cannot have consecutive dashes")
		}
	}

	return nil
}

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
	Database  DatabaseConfigResponse  `json:"database"`
	Auth      AuthConfigResponse      `json:"auth"`
	RateLimit RateLimitConfigResponse `json:"rate_limit"`
}

type AppConfigResponse struct {
	Name    string `json:"name"`
	Version string `json:"version"`
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

// UpdateConfig updates the service configuration
func (api *API) UpdateConfig() usecase.Interactor {
	type updateConfigInput struct {
		Auth      *AuthConfigResponse      `json:"auth"`
		RateLimit *RateLimitConfigResponse `json:"rate_limit"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input updateConfigInput, output *ConfigResponse) error {
		// Update auth configuration if provided
		if input.Auth != nil {
			api.Config.Auth.Enabled = input.Auth.Enabled
			api.Config.Auth.TokenExpiryHours = input.Auth.TokenExpiryHours
		}

		// Update rate limit configuration if provided
		if input.RateLimit != nil {
			api.Config.RateLimit.Enabled = input.RateLimit.Enabled
			api.Config.RateLimit.RequestsPerMinute = input.RateLimit.RequestsPerMinute
			api.Config.RateLimit.BurstSize = input.RateLimit.BurstSize
		}

		// Return updated configuration
		output.App = AppConfigResponse{
			Name:    api.Config.App.Name,
			Version: api.Config.App.Version,
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

	u.SetTitle("Update Configuration")
	u.SetDescription("Updates the runtime service configuration")
	u.SetTags("config")

	return u
}

// StatsResponse represents statistics response
type StatsResponse struct {
	ServicesMonitored      int       `json:"services_monitored"`
	APIRequests            int64     `json:"api_requests"`
	HealthChecksPerformed  int64     `json:"health_checks_performed"`
	Uptime                 string    `json:"uptime"`
	LastActivity           time.Time `json:"last_activity"`
	DatabaseQueries        int64     `json:"database_queries"`
	StartTime              time.Time `json:"start_time"`
}

// GetStats returns system statistics
func (api *API) GetStats() usecase.Interactor {
	type statsInput struct{}

	u := usecase.NewInteractor(func(ctx context.Context, input statsInput, output *StatsResponse) error {
		api.Services.IncrementAPIRequests()

		stats := api.Services.GetStats()
		uptime := time.Since(stats.StartTime)

		output.ServicesMonitored = 1 // Just the control service for now
		output.APIRequests = stats.APIRequests
		output.HealthChecksPerformed = stats.APIRequests // Approximate health checks as API requests
		output.Uptime = uptime.String()
		output.LastActivity = stats.LastActivity
		output.DatabaseQueries = stats.DatabaseQueries
		output.StartTime = stats.StartTime

		return nil
	})

	u.SetTitle("Get Statistics")
	u.SetDescription("Returns system statistics and metrics")
	u.SetTags("stats")

	return u
}

// EcosystemHealthResponse represents ecosystem health response
type EcosystemHealthResponse struct {
	OverallStatus string                          `json:"overall_status"`
	Services      map[string]ServiceHealthStatus  `json:"services"`
	HealthyCount  int                             `json:"healthy_count"`
	TotalCount    int                             `json:"total_count"`
	LastCheck     time.Time                       `json:"last_check"`
}

// ServiceHealthStatus represents individual service health
type ServiceHealthStatus struct {
	Healthy      bool      `json:"healthy"`
	Version      string    `json:"version,omitempty"`
	ResponseTime string    `json:"response_time"`
	LastCheck    time.Time `json:"last_check"`
	Error        string    `json:"error,omitempty"`
}

// GetEcosystemHealth returns ecosystem health status
func (api *API) GetEcosystemHealth() usecase.Interactor {
	type healthInput struct{}

	u := usecase.NewInteractor(func(ctx context.Context, input healthInput, output *EcosystemHealthResponse) error {
		api.Services.IncrementAPIRequests()

		services := map[string]ServiceHealthStatus{}
		healthyCount := 0
		totalCount := 0

		// Check control service (self)
		controlStart := time.Now()
		services["control"] = ServiceHealthStatus{
			Healthy:      true,
			Version:      api.Config.App.Version,
			ResponseTime: time.Since(controlStart).String(),
			LastCheck:    time.Now(),
		}
		healthyCount++
		totalCount++

		// Check other services if they're configured
		// For now, let's add a basic check for receiver service
		receiverStart := time.Now()
		receiverHealthy := api.checkServiceHealth("http://192.168.86.103:8081/health")
		receiverStatus := ServiceHealthStatus{
			Healthy:      receiverHealthy,
			ResponseTime: time.Since(receiverStart).String(),
			LastCheck:    time.Now(),
		}
		if !receiverHealthy {
			receiverStatus.Error = "Service unreachable"
		}
		services["receiver"] = receiverStatus
		if receiverHealthy {
			healthyCount++
		}
		totalCount++

		// Determine overall status
		overallStatus := "healthy"
		if healthyCount == 0 {
			overallStatus = "unhealthy"
		} else if healthyCount < totalCount {
			overallStatus = "degraded"
		}

		output.OverallStatus = overallStatus
		output.Services = services
		output.HealthyCount = healthyCount
		output.TotalCount = totalCount
		output.LastCheck = time.Now()

		return nil
	})

	u.SetTitle("Get Ecosystem Health")
	u.SetDescription("Returns health status of all ByteFreezer services")
	u.SetTags("health")

	return u
}

// ServiceReport represents a service health report
type ServiceReport struct {
	ServiceName   string                 `json:"service_name"`
	ServiceType   string                 `json:"service_type"` // Alternative field name
	ServiceID     string                 `json:"service_id"`
	InstanceID    string                 `json:"instance_id"` // Alternative field name
	InstanceAPI   string                 `json:"instance_api"`
	Version       string                 `json:"version"`
	Timestamp     time.Time              `json:"timestamp"`
	Healthy       bool                   `json:"healthy"`
	Status        string                 `json:"status"` // Alternative: "Healthy" or "Unhealthy"
	Configuration map[string]interface{} `json:"configuration"`
	Metrics       map[string]interface{} `json:"metrics"`
}

// ServiceReportResponse represents the response to a service report
type ServiceReportResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// ReceiveServiceReport receives health reports from services
func (api *API) ReceiveServiceReport() usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input ServiceReport, output *ServiceReportResponse) error {
		api.Services.IncrementAPIRequests()

		// Handle both naming conventions
		serviceName := input.ServiceName
		if serviceName == "" && input.ServiceType != "" {
			serviceName = input.ServiceType
		}

		serviceID := input.ServiceID
		if serviceID == "" && input.InstanceID != "" {
			serviceID = input.InstanceID
		}

		// Handle both healthy (bool) and status (string) fields
		healthy := input.Healthy
		if !healthy && input.Status != "" {
			healthy = (input.Status == "Healthy")
		}

		// Legacy logging for backward compatibility
		log.Infof("Received health report from service %s (%s): healthy=%v",
			serviceName, serviceID, healthy)

		// Store in database if health service is available
		if api.Services.HealthService != nil {
			// Get current record to check if service is still starting
			records, err := api.Services.HealthService.GetHealthRecordsByService(serviceName)
			currentStatus := ""
			if err == nil {
				for _, record := range records {
					if record.InstanceID == serviceID {
						currentStatus = record.Status
						break
					}
				}
			}

			// Determine status from health report
			status := "Unhealthy"
			if healthy {
				status = "Healthy"
			}

			// If service is currently "Starting" and reports healthy, transition to "Healthy"
			// If service is currently "Starting" and reports unhealthy, keep it "Starting"
			// (the service might still be initializing)
			if currentStatus == "Starting" && !healthy {
				log.Debugf("Service %s:%s is still starting and not yet healthy - keeping status as Starting",
					serviceName, serviceID)
				status = "Starting"
			}

			// Try to update health status in database with configuration
			// Configuration is updated on every health report to keep it current
			err = api.Services.HealthService.UpdateServiceHealth(
				serviceName,          // service_type
				serviceID,            // instance_id (should be hostname)
				input.InstanceAPI,    // instance_api
				status,               // status
				input.Configuration,  // configuration (updated on every report)
				input.Metrics,        // metrics
				nil,                  // response_time_ms will be set by polling
			)

			if err != nil {
				log.Warnf("Failed to update health status in database for %s:%s: %v",
					serviceName, serviceID, err)
				// Don't fail the request, just log the error for backward compatibility
			}
		}

		output.Success = true
		output.Message = fmt.Sprintf("Health report received for service %s", serviceName)

		return nil
	})

	u.SetTitle("Receive Service Report")
	u.SetDescription("Receives health and configuration reports from ByteFreezer services")
	u.SetTags("health", "reporting")

	return u
}

// checkServiceHealth performs a basic HTTP health check
func (api *API) checkServiceHealth(url string) bool {
	// Simple HTTP GET with timeout
	// For a basic implementation, we'll just return true for now
	// In a real implementation, this would make an HTTP request to the service
	return false // Assume services are down for simplicity
}

// Account API handlers

// CreateAccount creates a new account
func (api *API) CreateAccount() usecase.Interactor {
	type createAccountInput struct {
		Name  string `json:"name" required:"true"`
		Email string `json:"email" required:"true"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input createAccountInput, output *storage.Account) error {
		api.Services.IncrementAPIRequests()
		api.Services.IncrementDatabaseQueries()

		if api.Services.Storage == nil {
			return fmt.Errorf("storage not initialized")
		}

		account := &storage.Account{
			Name:   input.Name,
			Email:  input.Email,
			Active: true,
		}

		if err := api.Services.Storage.CreateAccount(ctx, account); err != nil {
			return fmt.Errorf("failed to create account: %w", err)
		}

		*output = *account
		return nil
	})

	u.SetTitle("Create Account")
	u.SetDescription("Creates a new ByteFreezer account")
	u.SetTags("accounts")
	u.SetExpectedErrors(usecaseStatus.InvalidArgument)

	return u
}

// GetAccount returns a specific account
func (api *API) GetAccount() usecase.Interactor {
	type getAccountInput struct {
		AccountID string `path:"accountId" required:"true"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input getAccountInput, output *storage.Account) error {
		api.Services.IncrementAPIRequests()
		api.Services.IncrementDatabaseQueries()

		if api.Services.Storage == nil {
			return fmt.Errorf("storage not initialized")
		}

		account, err := api.Services.Storage.GetAccount(ctx, input.AccountID)
		if err != nil {
			return fmt.Errorf("failed to get account: %w", err)
		}

		*output = *account
		return nil
	})

	u.SetTitle("Get Account")
	u.SetDescription("Returns a specific account by ID")
	u.SetTags("accounts")
	u.SetExpectedErrors(usecaseStatus.NotFound)

	return u
}

// ListAccounts returns all accounts
func (api *API) ListAccounts() usecase.Interactor {
	type listAccountsInput struct {
		Limit int `query:"limit"`
	}

	type listAccountsOutput struct {
		Items []storage.Account `json:"items"`
		Total int               `json:"total"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input listAccountsInput, output *listAccountsOutput) error {
		api.Services.IncrementAPIRequests()
		api.Services.IncrementDatabaseQueries()

		if api.Services.Storage == nil {
			return fmt.Errorf("storage not initialized")
		}

		opts := storage.ListOptions{
			Limit: input.Limit,
		}

		result, err := api.Services.Storage.ListAccounts(ctx, opts)
		if err != nil {
			return fmt.Errorf("failed to list accounts: %w", err)
		}

		output.Items = result.Items
		output.Total = result.Total
		return nil
	})

	u.SetTitle("List Accounts")
	u.SetDescription("Returns all accounts with pagination")
	u.SetTags("accounts")

	return u
}

// UpdateAccount updates an existing account
func (api *API) UpdateAccount() usecase.Interactor {
	type updateAccountInput struct {
		AccountID string `path:"accountId" required:"true"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		Active    *bool  `json:"active"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input updateAccountInput, output *storage.Account) error {
		api.Services.IncrementAPIRequests()
		api.Services.IncrementDatabaseQueries()

		if api.Services.Storage == nil {
			return fmt.Errorf("storage not initialized")
		}

		// Get existing account
		account, err := api.Services.Storage.GetAccount(ctx, input.AccountID)
		if err != nil {
			return fmt.Errorf("failed to get account: %w", err)
		}

		// Update fields
		if input.Name != "" {
			account.Name = input.Name
		}
		if input.Email != "" {
			account.Email = input.Email
		}
		if input.Active != nil {
			account.Active = *input.Active
		}

		if err := api.Services.Storage.UpdateAccount(ctx, account); err != nil {
			return fmt.Errorf("failed to update account: %w", err)
		}

		*output = *account
		return nil
	})

	u.SetTitle("Update Account")
	u.SetDescription("Updates an existing account")
	u.SetTags("accounts")
	u.SetExpectedErrors(usecaseStatus.NotFound, usecaseStatus.InvalidArgument)

	return u
}

// DeleteAccount deletes an account
func (api *API) DeleteAccount() usecase.Interactor {
	type deleteAccountInput struct {
		AccountID string `path:"accountId" required:"true"`
	}

	type deleteAccountOutput struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input deleteAccountInput, output *deleteAccountOutput) error {
		api.Services.IncrementAPIRequests()
		api.Services.IncrementDatabaseQueries()

		if api.Services.Storage == nil {
			return fmt.Errorf("storage not initialized")
		}

		if err := api.Services.Storage.DeleteAccount(ctx, input.AccountID); err != nil {
			return fmt.Errorf("failed to delete account: %w", err)
		}

		output.Success = true
		output.Message = fmt.Sprintf("Account %s deleted successfully", input.AccountID)
		return nil
	})

	u.SetTitle("Delete Account")
	u.SetDescription("Deletes an account and all associated tenants/datasets")
	u.SetTags("accounts")
	u.SetExpectedErrors(usecaseStatus.NotFound)

	return u
}

// AssumeAccountAdmin allows system administrators to assume account admin role for any account
func (api *API) AssumeAccountAdmin() usecase.Interactor {
	type assumeAccountAdminInput struct {
		AccountID string `path:"accountId" required:"true"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input assumeAccountAdminInput, output *LoginResponse) error {
		api.Services.IncrementAPIRequests()
		api.Services.IncrementDatabaseQueries()

		// Check if Auth service is available
		if api.Services.Auth == nil {
			return fmt.Errorf("authentication service not available")
		}

		if api.Services.Storage == nil {
			return fmt.Errorf("storage not initialized")
		}

		// TODO: Verify that the current user is a system admin
		// This should be done via middleware that extracts the JWT token and validates it
		// For now, we'll proceed assuming the middleware has already validated this

		// Verify the account exists and is active
		account, err := api.Services.Storage.GetAccount(ctx, input.AccountID)
		if err != nil {
			return fmt.Errorf("account not found: %w", err)
		}

		if !account.Active {
			return fmt.Errorf("account is not active")
		}

		// Create a virtual user object for the account admin
		// This user represents the system admin acting as an account admin
		virtualUser := &services.User{
			ID:        fmt.Sprintf("sysadmin-as-%s", input.AccountID), // Virtual user ID
			AccountID: input.AccountID,
			Email:     account.Email,
			Role:      "account_admin", // Acting as account admin
			FirstName: "System",
			LastName:  "Administrator",
			Active:    true,
		}

		// Generate JWT tokens for the virtual user
		accessToken, refreshToken, expiresAt, err := api.Services.Auth.GenerateTokenPair(virtualUser)
		if err != nil {
			log.Errorf("Failed to generate tokens for account assumption: %v", err)
			return fmt.Errorf("failed to generate authentication tokens")
		}

		// Store session (ignore errors as session creation is not critical)
		_ = api.Services.Auth.CreateSession(ctx, virtualUser.ID, accessToken, refreshToken, "", "", expiresAt, expiresAt.Add(7*24*time.Hour))

		// Log audit event
		if api.Services.AuditLog != nil {
			api.Services.AuditLog.LogAction(ctx, virtualUser.ID, input.AccountID, "assume_account_admin", input.AccountID, "", map[string]interface{}{
				"account_name": account.Name,
			})
		}

		output.Token = accessToken
		output.RefreshToken = refreshToken
		output.ExpiresAt = expiresAt
		output.User = *virtualUser

		log.Infof("System admin assumed account admin role for account %s (%s)", account.Name, input.AccountID)
		return nil
	})

	u.SetTitle("Assume Account Admin")
	u.SetDescription("Allows system administrators to assume account admin role for any account")
	u.SetTags("accounts", "auth")
	u.SetExpectedErrors(usecaseStatus.NotFound, usecaseStatus.PermissionDenied)

	return u
}

// Tenant API handlers

// ListTenants returns all tenants for an account
func (api *API) ListTenants() usecase.Interactor {
	type listTenantsInput struct {
		AccountID string `path:"accountId" required:"true"`
		Limit     int    `query:"limit"`
	}

	type listTenantsOutput struct {
		Items []storage.Tenant `json:"items"`
		Total int              `json:"total"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input listTenantsInput, output *listTenantsOutput) error {
		api.Services.IncrementAPIRequests()
		api.Services.IncrementDatabaseQueries()

		if api.Services.Storage == nil {
			return fmt.Errorf("storage not initialized")
		}

		opts := storage.ListOptions{
			Limit: input.Limit,
		}

		result, err := api.Services.Storage.ListTenants(ctx, input.AccountID, opts)
		if err != nil {
			return fmt.Errorf("failed to list tenants: %w", err)
		}

		output.Items = result.Items
		output.Total = result.Total
		return nil
	})

	u.SetTitle("List Tenants")
	u.SetDescription("Returns all tenants for an account")
	u.SetTags("tenants")

	return u
}

// GetTenant returns a specific tenant
func (api *API) GetTenant() usecase.Interactor {
	type getTenantInput struct {
		AccountID string `path:"accountId" required:"true"`
		TenantID  string `path:"tenantId" required:"true"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input getTenantInput, output *storage.Tenant) error {
		api.Services.IncrementAPIRequests()
		api.Services.IncrementDatabaseQueries()

		if api.Services.Storage == nil {
			return fmt.Errorf("storage not initialized")
		}

		tenant, err := api.Services.Storage.GetTenant(ctx, input.AccountID, input.TenantID)
		if err != nil {
			return fmt.Errorf("failed to get tenant: %w", err)
		}

		*output = *tenant
		return nil
	})

	u.SetTitle("Get Tenant")
	u.SetDescription("Returns a specific tenant by ID")
	u.SetTags("tenants")
	u.SetExpectedErrors(usecaseStatus.NotFound)

	return u
}

// GetTenantDirect returns a tenant by ID only (for proxy validation)
func (api *API) GetTenantDirect() usecase.Interactor {
	type getTenantDirectInput struct {
		TenantID string `path:"tenantId" required:"true"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input getTenantDirectInput, output *storage.Tenant) error {
		api.Services.IncrementAPIRequests()
		api.Services.IncrementDatabaseQueries()

		if api.Services.Storage == nil {
			return fmt.Errorf("storage not initialized")
		}

		tenant, err := api.Services.Storage.GetTenantByID(ctx, input.TenantID)
		if err != nil {
			return fmt.Errorf("failed to get tenant: %w", err)
		}

		*output = *tenant
		return nil
	})

	u.SetTitle("Get Tenant Direct")
	u.SetDescription("Returns a specific tenant by ID (direct lookup for proxy validation)")
	u.SetTags("tenants")
	u.SetExpectedErrors(usecaseStatus.NotFound)

	return u
}

// CreateTenant creates a new tenant
func (api *API) CreateTenant() usecase.Interactor {
	type createTenantInput struct {
		AccountID   string              `path:"accountId" required:"true"`
		Name        string              `json:"name" required:"true"`
		Description string              `json:"description"`
		Config      storage.TenantConfig `json:"config"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input createTenantInput, output *storage.Tenant) error {
		api.Services.IncrementAPIRequests()
		api.Services.IncrementDatabaseQueries()

		if api.Services.Storage == nil {
			return fmt.Errorf("storage not initialized")
		}

		// Auto-generate unique tenant ID
		tenantID := storage.GenerateShortID()

		tenant := &storage.Tenant{
			ID:          tenantID,
			AccountID:   input.AccountID,
			Name:        input.Name,
			DisplayName: input.Name,
			Description: input.Description,
			Active:      true,
			Config:      input.Config,
		}

		if err := api.Services.Storage.CreateTenant(ctx, tenant); err != nil {
			return fmt.Errorf("failed to create tenant: %w", err)
		}

		*output = *tenant
		return nil
	})

	u.SetTitle("Create Tenant")
	u.SetDescription("Creates a new tenant for an account with auto-generated ID")
	u.SetTags("tenants")
	u.SetExpectedErrors(usecaseStatus.InvalidArgument)

	return u
}

// UpdateTenant updates an existing tenant
func (api *API) UpdateTenant() usecase.Interactor {
	type updateTenantInput struct {
		AccountID   string                `path:"accountId" required:"true"`
		TenantID    string                `path:"tenantId" required:"true"`
		Name        string                `json:"name"`
		Description string                `json:"description"`
		Active      *bool                 `json:"active"`
		Config      *storage.TenantConfig `json:"config"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input updateTenantInput, output *storage.Tenant) error {
		api.Services.IncrementAPIRequests()
		api.Services.IncrementDatabaseQueries()

		if api.Services.Storage == nil {
			return fmt.Errorf("storage not initialized")
		}

		// Get existing tenant
		tenant, err := api.Services.Storage.GetTenant(ctx, input.AccountID, input.TenantID)
		if err != nil {
			return fmt.Errorf("failed to get tenant: %w", err)
		}

		// Update fields
		if input.Name != "" {
			tenant.Name = input.Name
		}
		if input.Description != "" {
			tenant.Description = input.Description
		}
		if input.Active != nil {
			tenant.Active = *input.Active
		}
		if input.Config != nil {
			tenant.Config = *input.Config
		}

		if err := api.Services.Storage.UpdateTenant(ctx, tenant); err != nil {
			return fmt.Errorf("failed to update tenant: %w", err)
		}

		*output = *tenant
		return nil
	})

	u.SetTitle("Update Tenant")
	u.SetDescription("Updates an existing tenant")
	u.SetTags("tenants")
	u.SetExpectedErrors(usecaseStatus.NotFound, usecaseStatus.InvalidArgument)

	return u
}

// DeleteTenant deletes a tenant
func (api *API) DeleteTenant() usecase.Interactor {
	type deleteTenantInput struct {
		AccountID string `path:"accountId" required:"true"`
		TenantID  string `path:"tenantId" required:"true"`
	}

	type deleteTenantOutput struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input deleteTenantInput, output *deleteTenantOutput) error {
		api.Services.IncrementAPIRequests()
		api.Services.IncrementDatabaseQueries()

		if api.Services.Storage == nil {
			return fmt.Errorf("storage not initialized")
		}

		if err := api.Services.Storage.DeleteTenant(ctx, input.AccountID, input.TenantID); err != nil {
			return fmt.Errorf("failed to delete tenant: %w", err)
		}

		output.Success = true
		output.Message = fmt.Sprintf("Tenant %s deleted successfully", input.TenantID)
		return nil
	})

	u.SetTitle("Delete Tenant")
	u.SetDescription("Deletes a tenant and all associated datasets")
	u.SetTags("tenants")
	u.SetExpectedErrors(usecaseStatus.NotFound)

	return u
}

// LoginRequest represents login request
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse represents login response
type LoginResponse struct {
	Token        string                `json:"token"`
	RefreshToken string                `json:"refresh_token"`
	ExpiresAt    time.Time             `json:"expires_at"`
	User         services.User         `json:"user"`
}

// PasswordResetRequest represents password reset request
type PasswordResetRequest struct {
	Email string `json:"email"`
}

// PasswordResetResponse represents password reset response
type PasswordResetResponse struct {
	Message string `json:"message"`
	Success bool   `json:"success"`
}

// Login authenticates a user and returns a JWT token
func (api *API) Login() usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input LoginRequest, output *LoginResponse) error {
		api.Services.IncrementAPIRequests()

		// Validate input
		if input.Email == "" || input.Password == "" {
			return fmt.Errorf("email and password are required")
		}

		// Check if Auth service is available
		if api.Services.Auth == nil {
			return fmt.Errorf("authentication service not available")
		}

		// Authenticate user
		user, err := api.Services.Auth.AuthenticateUser(ctx, input.Email, input.Password)
		if err != nil {
			log.Warnf("Authentication failed for %s: %v", input.Email, err)
			return fmt.Errorf("invalid email or password")
		}

		// Check if user is active
		if !user.Active {
			return fmt.Errorf("account is inactive")
		}

		// Generate JWT tokens
		accessToken, refreshToken, expiresAt, err := api.Services.Auth.GenerateTokenPair(user)
		if err != nil {
			log.Errorf("Failed to generate tokens for %s: %v", user.Email, err)
			return fmt.Errorf("failed to generate authentication tokens")
		}

		// Store session (ignore errors as session creation is not critical)
		_ = api.Services.Auth.CreateSession(ctx, user.ID, accessToken, refreshToken, "", "", expiresAt, expiresAt.Add(7*24*time.Hour))

		// Log audit event
		if api.Services.AuditLog != nil {
			api.Services.AuditLog.LogAction(ctx, user.ID, user.AccountID, "login", "", "", map[string]interface{}{})
		}

		output.Token = accessToken
		output.RefreshToken = refreshToken
		output.ExpiresAt = expiresAt
		output.User = *user

		log.Infof("User %s (%s) logged in successfully", user.Email, user.Role)
		return nil
	})

	u.SetTitle("User Login")
	u.SetDescription("Authenticates a user via email/password and returns JWT tokens")
	u.SetTags("auth")

	return u
}

// RefreshTokenRequest represents token refresh request
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// RefreshToken validates a refresh token and returns new access and refresh tokens
func (api *API) RefreshToken() usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input RefreshTokenRequest, output *LoginResponse) error {
		api.Services.IncrementAPIRequests()

		// Validate input
		if input.RefreshToken == "" {
			return fmt.Errorf("refresh token is required")
		}

		// Check if Auth service is available
		if api.Services.Auth == nil {
			return fmt.Errorf("authentication service not available")
		}

		// Refresh token and get new tokens
		user, newAccessToken, newRefreshToken, expiresAt, err := api.Services.Auth.RefreshToken(ctx, input.RefreshToken)
		if err != nil {
			log.Warnf("Token refresh failed: %v", err)
			return fmt.Errorf("invalid or expired refresh token")
		}

		// Store new session (ignore errors as session creation is not critical)
		_ = api.Services.Auth.CreateSession(ctx, user.ID, newAccessToken, newRefreshToken, "", "", expiresAt, expiresAt.Add(7*24*time.Hour))

		// Log audit event
		if api.Services.AuditLog != nil {
			api.Services.AuditLog.LogAction(ctx, user.ID, user.AccountID, "token_refresh", "", "", map[string]interface{}{})
		}

		output.Token = newAccessToken
		output.RefreshToken = newRefreshToken
		output.ExpiresAt = expiresAt
		output.User = *user

		log.Infof("Token refreshed for user %s", user.Email)
		return nil
	})

	u.SetTitle("Refresh Token")
	u.SetDescription("Validates a refresh token and returns new access and refresh tokens")
	u.SetTags("auth")

	return u
}

// RequestPasswordReset handles password reset requests
func (api *API) RequestPasswordReset() usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input PasswordResetRequest, output *PasswordResetResponse) error {
		api.Services.IncrementAPIRequests()

		// Check if email is in admin list
		isAdmin := false
		for _, adminUser := range api.Config.Auth.AdminUsers {
			if adminUser == input.Email {
				isAdmin = true
				break
			}
		}

		// Always return success to prevent email enumeration attacks
		output.Success = true
		if isAdmin {
			output.Message = "If your email is registered as an administrator, you will receive password reset instructions."
			// TODO: In a real implementation, this would:
			// 1. Generate a secure reset token
			// 2. Store it in the database with expiration
			// 3. Send an email with the reset link
			// For now, we'll just log the success
			log.Infof("Password reset requested for admin user: %s", input.Email)
		} else {
			output.Message = "If your email is registered as an administrator, you will receive password reset instructions."
			log.Warnf("Password reset requested for non-admin email: %s", input.Email)
		}

		return nil
	})

	u.SetTitle("Request Password Reset")
	u.SetDescription("Sends password reset instructions to admin users")
	u.SetTags("auth")

	return u
}

// Dataset API handlers

// ListDatasets returns all datasets for a tenant
func (api *API) ListDatasets() usecase.Interactor {
	type listDatasetsInput struct {
		TenantID string `path:"tenantId" required:"true"`
		Limit    int    `query:"limit"`
	}

	type listDatasetsOutput struct {
		Items []storage.Dataset `json:"items"`
		Total int               `json:"total"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input listDatasetsInput, output *listDatasetsOutput) error {
		api.Services.IncrementAPIRequests()
		api.Services.IncrementDatabaseQueries()

		if api.Services.Storage == nil {
			return fmt.Errorf("storage not initialized")
		}

		opts := storage.ListOptions{
			Limit: input.Limit,
		}

		result, err := api.Services.Storage.ListDatasets(ctx, input.TenantID, opts)
		if err != nil {
			return fmt.Errorf("failed to list datasets: %w", err)
		}

		output.Items = result.Items
		output.Total = result.Total
		return nil
	})

	u.SetTitle("List Datasets")
	u.SetDescription("Returns all datasets for a tenant")
	u.SetTags("datasets")

	return u
}

// GetDataset returns a specific dataset
func (api *API) GetDataset() usecase.Interactor {
	type getDatasetInput struct {
		TenantID  string `path:"tenantId" required:"true"`
		DatasetID string `path:"datasetId" required:"true"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input getDatasetInput, output *storage.Dataset) error {
		api.Services.IncrementAPIRequests()
		api.Services.IncrementDatabaseQueries()

		if api.Services.Storage == nil {
			return fmt.Errorf("storage not initialized")
		}

		dataset, err := api.Services.Storage.GetDataset(ctx, input.TenantID, input.DatasetID)
		if err != nil {
			return fmt.Errorf("failed to get dataset: %w", err)
		}

		*output = *dataset
		return nil
	})

	u.SetTitle("Get Dataset")
	u.SetDescription("Returns a specific dataset by ID")
	u.SetTags("datasets")
	u.SetExpectedErrors(usecaseStatus.NotFound)

	return u
}

// CreateDataset creates a new dataset
func (api *API) CreateDataset() usecase.Interactor {
	type createDatasetInput struct {
		TenantID    string                `path:"tenantId" required:"true"`
		ID          string                `json:"id"`
		Name        string                `json:"name" required:"true"`
		Description string                `json:"description"`
		Active      *bool                 `json:"active"`
		Status      string                `json:"status"`
		Config      storage.DatasetConfig `json:"config"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input createDatasetInput, output *storage.Dataset) error {
		api.Services.IncrementAPIRequests()
		api.Services.IncrementDatabaseQueries()

		if api.Services.Storage == nil {
			return fmt.Errorf("storage not initialized")
		}

		// Auto-generate ID if not provided, otherwise validate it
		datasetID := input.ID
		if datasetID == "" {
			datasetID = storage.GenerateShortID()
		} else {
			// Validate ID format if provided by client
			if err := ValidateID(datasetID); err != nil {
				return fmt.Errorf("invalid dataset ID: %w", err)
			}
		}

		// Set default values for active and status if not provided
		active := true
		if input.Active != nil {
			active = *input.Active
		}

		status := "active"
		if input.Status != "" {
			status = input.Status
		}

		dataset := &storage.Dataset{
			ID:          datasetID,
			TenantID:    input.TenantID,
			Name:        input.Name,
			Description: input.Description,
			Active:      active,
			Status:      status,
			Config:      input.Config,
		}

		if err := api.Services.Storage.CreateDataset(ctx, dataset); err != nil {
			return fmt.Errorf("failed to create dataset: %w", err)
		}

		*output = *dataset
		return nil
	})

	u.SetTitle("Create Dataset")
	u.SetDescription("Creates a new dataset for a tenant")
	u.SetTags("datasets")
	u.SetExpectedErrors(usecaseStatus.InvalidArgument, usecaseStatus.AlreadyExists)

	return u
}

// UpdateDataset updates an existing dataset
func (api *API) UpdateDataset() usecase.Interactor {
	type updateDatasetInput struct {
		TenantID    string                 `path:"tenantId" required:"true"`
		DatasetID   string                 `path:"datasetId" required:"true"`
		Name        string                 `json:"name"`
		Description string                 `json:"description"`
		Active      *bool                  `json:"active"`
		Config      *storage.DatasetConfig `json:"config"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input updateDatasetInput, output *storage.Dataset) error {
		api.Services.IncrementAPIRequests()
		api.Services.IncrementDatabaseQueries()

		if api.Services.Storage == nil {
			return fmt.Errorf("storage not initialized")
		}

		// Get existing dataset
		dataset, err := api.Services.Storage.GetDataset(ctx, input.TenantID, input.DatasetID)
		if err != nil {
			return fmt.Errorf("dataset not found: %w", err)
		}

		// Update fields
		if input.Name != "" {
			dataset.Name = input.Name
		}
		if input.Description != "" {
			dataset.Description = input.Description
		}
		if input.Active != nil {
			dataset.Active = *input.Active
		}
		if input.Config != nil {
			dataset.Config = *input.Config
		}

		if err := api.Services.Storage.UpdateDataset(ctx, dataset); err != nil {
			return fmt.Errorf("failed to update dataset: %w", err)
		}

		*output = *dataset
		return nil
	})

	u.SetTitle("Update Dataset")
	u.SetDescription("Updates an existing dataset")
	u.SetTags("datasets")
	u.SetExpectedErrors(usecaseStatus.NotFound, usecaseStatus.InvalidArgument)

	return u
}

// DeleteDataset deletes a dataset
func (api *API) DeleteDataset() usecase.Interactor {
	type deleteDatasetInput struct {
		TenantID  string `path:"tenantId" required:"true"`
		DatasetID string `path:"datasetId" required:"true"`
	}

	type deleteDatasetOutput struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input deleteDatasetInput, output *deleteDatasetOutput) error {
		api.Services.IncrementAPIRequests()
		api.Services.IncrementDatabaseQueries()

		if api.Services.Storage == nil {
			return fmt.Errorf("storage not initialized")
		}

		if err := api.Services.Storage.DeleteDataset(ctx, input.TenantID, input.DatasetID); err != nil {
			return fmt.Errorf("failed to delete dataset: %w", err)
		}

		output.Success = true
		output.Message = fmt.Sprintf("Dataset %s deleted successfully", input.DatasetID)
		return nil
	})

	u.SetTitle("Delete Dataset")
	u.SetDescription("Deletes a dataset")
	u.SetTags("datasets")
	u.SetExpectedErrors(usecaseStatus.NotFound)

	return u
}

// User Management API handlers

// ListUsers returns all users, optionally filtered by account ID
func (api *API) ListUsers() usecase.Interactor {
	type listUsersInput struct {
		AccountID string `query:"account_id"`
	}

	type listUsersOutput struct {
		Items []services.User `json:"items"`
		Total int             `json:"total"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input listUsersInput, output *listUsersOutput) error {
		api.Services.IncrementAPIRequests()
		api.Services.IncrementDatabaseQueries()

		if api.Services.Auth == nil {
			return fmt.Errorf("authentication service not available")
		}

		users, err := api.Services.Auth.ListUsers(ctx, input.AccountID)
		if err != nil {
			return fmt.Errorf("failed to list users: %w", err)
		}

		output.Items = users
		output.Total = len(users)
		return nil
	})

	u.SetTitle("List Users")
	u.SetDescription("Returns all users, optionally filtered by account ID")
	u.SetTags("users")

	return u
}

// GetUser returns a specific user by ID
func (api *API) GetUser() usecase.Interactor {
	type getUserInput struct {
		UserID string `path:"userId" required:"true"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input getUserInput, output *services.User) error {
		api.Services.IncrementAPIRequests()
		api.Services.IncrementDatabaseQueries()

		if api.Services.Auth == nil {
			return fmt.Errorf("authentication service not available")
		}

		user, err := api.Services.Auth.GetUserByID(ctx, input.UserID)
		if err != nil {
			return fmt.Errorf("failed to get user: %w", err)
		}

		*output = *user
		return nil
	})

	u.SetTitle("Get User")
	u.SetDescription("Returns a specific user by ID")
	u.SetTags("users")
	u.SetExpectedErrors(usecaseStatus.NotFound)

	return u
}

// CreateUser creates a new user
func (api *API) CreateUser() usecase.Interactor {
	type createUserInput struct {
		AccountID string `json:"account_id" required:"true"`
		Email     string `json:"email" required:"true"`
		Password  string `json:"password" required:"true"`
		FirstName string `json:"first_name" required:"true"`
		LastName  string `json:"last_name" required:"true"`
		Role      string `json:"role" required:"true"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input createUserInput, output *services.User) error {
		api.Services.IncrementAPIRequests()
		api.Services.IncrementDatabaseQueries()

		if api.Services.Auth == nil {
			return fmt.Errorf("authentication service not available")
		}

		// Validate role
		validRoles := map[string]bool{
			"system_admin":     true,
			"account_admin":    true,
			"account_readonly": true,
		}
		if !validRoles[input.Role] {
			return fmt.Errorf("invalid role: %s", input.Role)
		}

		user, err := api.Services.Auth.CreateUser(
			ctx,
			input.AccountID,
			input.Email,
			input.Password,
			input.Role,
			input.FirstName,
			input.LastName,
		)
		if err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}

		// Log audit event
		if api.Services.AuditLog != nil {
			api.Services.AuditLog.LogAction(ctx, user.ID, input.AccountID, "user_created", user.ID, "", map[string]interface{}{
				"email": user.Email,
				"role":  user.Role,
			})
		}

		*output = *user
		return nil
	})

	u.SetTitle("Create User")
	u.SetDescription("Creates a new user")
	u.SetTags("users")
	u.SetExpectedErrors(usecaseStatus.InvalidArgument, usecaseStatus.AlreadyExists)

	return u
}

// UpdateUser updates an existing user
func (api *API) UpdateUser() usecase.Interactor {
	type updateUserInput struct {
		UserID    string `path:"userId" required:"true"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Role      string `json:"role"`
		Active    *bool  `json:"active"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input updateUserInput, output *services.User) error {
		api.Services.IncrementAPIRequests()
		api.Services.IncrementDatabaseQueries()

		if api.Services.Auth == nil {
			return fmt.Errorf("authentication service not available")
		}

		// Validate role if provided
		if input.Role != "" {
			validRoles := map[string]bool{
				"system_admin":     true,
				"account_admin":    true,
				"account_readonly": true,
			}
			if !validRoles[input.Role] {
				return fmt.Errorf("invalid role: %s", input.Role)
			}
		}

		user, err := api.Services.Auth.UpdateUser(
			ctx,
			input.UserID,
			input.FirstName,
			input.LastName,
			input.Role,
			input.Active,
		)
		if err != nil {
			return fmt.Errorf("failed to update user: %w", err)
		}

		// Log audit event
		if api.Services.AuditLog != nil {
			api.Services.AuditLog.LogAction(ctx, user.ID, user.AccountID, "user_updated", user.ID, "", map[string]interface{}{
				"email": user.Email,
			})
		}

		*output = *user
		return nil
	})

	u.SetTitle("Update User")
	u.SetDescription("Updates an existing user")
	u.SetTags("users")
	u.SetExpectedErrors(usecaseStatus.NotFound, usecaseStatus.InvalidArgument)

	return u
}

// DeleteUser deletes a user
func (api *API) DeleteUser() usecase.Interactor {
	type deleteUserInput struct {
		UserID string `path:"userId" required:"true"`
	}

	type deleteUserOutput struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input deleteUserInput, output *deleteUserOutput) error {
		api.Services.IncrementAPIRequests()
		api.Services.IncrementDatabaseQueries()

		if api.Services.Auth == nil {
			return fmt.Errorf("authentication service not available")
		}

		// Log audit event before deletion
		if api.Services.AuditLog != nil {
			user, err := api.Services.Auth.GetUserByID(ctx, input.UserID)
			if err == nil {
				api.Services.AuditLog.LogAction(ctx, user.ID, user.AccountID, "user_deleted", user.ID, "", map[string]interface{}{
					"email": user.Email,
				})
			}
		}

		if err := api.Services.Auth.DeleteUser(ctx, input.UserID); err != nil {
			return fmt.Errorf("failed to delete user: %w", err)
		}

		output.Success = true
		output.Message = fmt.Sprintf("User %s deleted successfully", input.UserID)
		return nil
	})

	u.SetTitle("Delete User")
	u.SetDescription("Deletes a user")
	u.SetTags("users")
	u.SetExpectedErrors(usecaseStatus.NotFound)

	return u
}

// ToggleUserActive activates or deactivates a user
func (api *API) ToggleUserActive() usecase.Interactor {
	type toggleUserActiveInput struct {
		UserID string `path:"userId" required:"true"`
		Active bool   `json:"active" required:"true"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input toggleUserActiveInput, output *services.User) error {
		api.Services.IncrementAPIRequests()
		api.Services.IncrementDatabaseQueries()

		if api.Services.Auth == nil {
			return fmt.Errorf("authentication service not available")
		}

		user, err := api.Services.Auth.ToggleUserActive(ctx, input.UserID, input.Active)
		if err != nil {
			return fmt.Errorf("failed to toggle user status: %w", err)
		}

		// Log audit event
		if api.Services.AuditLog != nil {
			action := "user_deactivated"
			if input.Active {
				action = "user_activated"
			}
			api.Services.AuditLog.LogAction(ctx, user.ID, user.AccountID, action, user.ID, "", map[string]interface{}{
				"email": user.Email,
			})
		}

		*output = *user
		return nil
	})

	u.SetTitle("Toggle User Active Status")
	u.SetDescription("Activates or deactivates a user")
	u.SetTags("users")
	u.SetExpectedErrors(usecaseStatus.NotFound)

	return u
}

// Flat List API handlers (for UI convenience)

// ListAllTenants returns all tenants across all accounts
func (api *API) ListAllTenants() usecase.Interactor {
	type listAllTenantsInput struct {
		Limit int `query:"limit"`
	}

	type listAllTenantsOutput struct {
		Items []storage.Tenant `json:"items"`
		Total int              `json:"total"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input listAllTenantsInput, output *listAllTenantsOutput) error {
		api.Services.IncrementAPIRequests()
		api.Services.IncrementDatabaseQueries()

		if api.Services.Storage == nil {
			return fmt.Errorf("storage not initialized")
		}

		opts := storage.ListOptions{
			Limit: input.Limit,
		}

		result, err := api.Services.Storage.ListAllTenants(ctx, opts)
		if err != nil {
			return fmt.Errorf("failed to list all tenants: %w", err)
		}

		output.Items = result.Items
		output.Total = result.Total
		return nil
	})

	u.SetTitle("List All Tenants")
	u.SetDescription("Returns all tenants across all accounts (flat list)")
	u.SetTags("tenants")

	return u
}

// ListAllDatasets returns all datasets across all tenants
func (api *API) ListAllDatasets() usecase.Interactor {
	type listAllDatasetsInput struct {
		Search   string `query:"search"`
		TenantID string `query:"tenant_id"`
		Active   string `query:"active"`
		Limit    int    `query:"limit"`
	}

	type listAllDatasetsOutput struct {
		Items []storage.Dataset `json:"items"`
		Total int               `json:"total"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input listAllDatasetsInput, output *listAllDatasetsOutput) error {
		api.Services.IncrementAPIRequests()
		api.Services.IncrementDatabaseQueries()

		if api.Services.Storage == nil {
			return fmt.Errorf("storage not initialized")
		}

		opts := storage.ListOptions{
			Limit:  input.Limit,
			Filter: input.Search,
		}

		result, err := api.Services.Storage.ListAllDatasets(ctx, opts)
		if err != nil {
			return fmt.Errorf("failed to list all datasets: %w", err)
		}

		// Apply client-side filtering for tenant_id and active since we don't have DB support yet
		var filtered []storage.Dataset
		for _, dataset := range result.Items {
			// Filter by tenant_id if specified
			if input.TenantID != "" && input.TenantID != "all" && dataset.TenantID != input.TenantID {
				continue
			}

			// Filter by active status if specified
			if input.Active != "" && input.Active != "all" {
				isActive := dataset.Active
				if input.Active == "active" && !isActive {
					continue
				}
				if input.Active == "inactive" && isActive {
					continue
				}
			}

			filtered = append(filtered, dataset)
		}

		output.Items = filtered
		output.Total = len(filtered)
		return nil
	})

	u.SetTitle("List All Datasets")
	u.SetDescription("Returns all datasets across all tenants (flat list with filtering)")
	u.SetTags("datasets")

	return u
}

// Plugin Schema API handlers

// PluginSchemasResponse represents plugin schemas response
type PluginSchemasResponse struct {
	Plugins []map[string]interface{} `json:"plugins"`
	Count   int                      `json:"count"`
}

// GetPluginSchemas returns plugin schemas from all registered proxy instances
func (api *API) GetPluginSchemas() usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input struct{}, output *PluginSchemasResponse) error {
		api.Services.IncrementAPIRequests()

		if api.Services.HealthService == nil {
			return fmt.Errorf("health service not available")
		}

		schemas, err := api.Services.HealthService.GetPluginSchemas()
		if err != nil {
			return fmt.Errorf("failed to get plugin schemas: %w", err)
		}

		output.Plugins = schemas
		output.Count = len(schemas)

		log.Infof("Retrieved %d plugin schemas for client", len(schemas))

		return nil
	})

	u.SetTitle("Get Plugin Schemas")
	u.SetDescription("Returns plugin configuration schemas from all registered proxy instances")
	u.SetTags("plugins", "schemas")

	return u
}

