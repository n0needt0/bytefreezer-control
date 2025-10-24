package api

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
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

		// Log audit event
		api.logAuditEvent(ctx, account.ID, "account_created", "account", account.ID, map[string]interface{}{
			"account_name": account.Name,
			"email":        account.Email,
		})

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

		// Track changes for audit log
		changes := make(map[string]interface{})
		oldValues := make(map[string]interface{})

		// Update fields and track changes
		if input.Name != "" && input.Name != account.Name {
			oldValues["name"] = account.Name
			changes["name"] = input.Name
			account.Name = input.Name
		}
		if input.Email != "" && input.Email != account.Email {
			oldValues["email"] = account.Email
			changes["email"] = input.Email
			account.Email = input.Email
		}
		if input.Active != nil && *input.Active != account.Active {
			oldValues["active"] = account.Active
			changes["active"] = *input.Active
			account.Active = *input.Active
		}

		if err := api.Services.Storage.UpdateAccount(ctx, account); err != nil {
			return fmt.Errorf("failed to update account: %w", err)
		}

		// Log audit event with changes
		api.logAuditEvent(ctx, input.AccountID, "account_updated", "account", input.AccountID, map[string]interface{}{
			"account_name": account.Name,
			"changes":      changes,
			"old_values":   oldValues,
		})

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

		// Get account info before deletion for audit log
		account, err := api.Services.Storage.GetAccount(ctx, input.AccountID)
		if err != nil {
			return fmt.Errorf("failed to get account: %w", err)
		}

		// Log audit event BEFORE deletion
		api.logAuditEvent(ctx, input.AccountID, "account_deleted", "account", input.AccountID, map[string]interface{}{
			"account_name": account.Name,
			"email":        account.Email,
		})

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
		api.logAuditEvent(ctx, input.AccountID, "assume_account_admin", "account", input.AccountID, map[string]interface{}{
			"account_name": account.Name,
		})

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

		// Log audit event
		api.logAuditEvent(ctx, input.AccountID, "tenant_created", "tenant", tenant.ID, map[string]interface{}{
			"tenant_name": tenant.Name,
			"active":      tenant.Active,
		})

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

		// Track changes for audit log
		changes := make(map[string]interface{})
		oldValues := make(map[string]interface{})

		// Update fields and track changes
		if input.Name != "" && input.Name != tenant.Name {
			oldValues["name"] = tenant.Name
			changes["name"] = input.Name
			tenant.Name = input.Name
		}
		if input.Description != "" && input.Description != tenant.Description {
			oldValues["description"] = tenant.Description
			changes["description"] = input.Description
			tenant.Description = input.Description
		}
		if input.Active != nil && *input.Active != tenant.Active {
			oldValues["active"] = tenant.Active
			changes["active"] = *input.Active
			tenant.Active = *input.Active
		}
		if input.Config != nil {
			// Deep copy the old config to prevent reference sharing
			oldConfigBytes, err := json.Marshal(tenant.Config)
			if err == nil {
				var oldConfigCopy storage.TenantConfig
				if err := json.Unmarshal(oldConfigBytes, &oldConfigCopy); err == nil {
					// Store full config in audit log (backend)
					oldValues["config"] = oldConfigCopy
					changes["config"] = *input.Config
				}
			}
			tenant.Config = *input.Config
		}

		if err := api.Services.Storage.UpdateTenant(ctx, tenant); err != nil {
			return fmt.Errorf("failed to update tenant: %w", err)
		}

		// Log audit event with changes
		auditDetails := map[string]interface{}{
			"tenant_name": tenant.Name,
			"changes":     changes,
			"old_values":  oldValues,
		}
		api.logAuditEvent(ctx, input.AccountID, "tenant_updated", "tenant", input.TenantID, auditDetails)

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

		// Get tenant info before deletion for audit log
		tenant, err := api.Services.Storage.GetTenant(ctx, input.AccountID, input.TenantID)
		if err != nil {
			return fmt.Errorf("failed to get tenant: %w", err)
		}

		// Log audit event BEFORE deletion
		api.logAuditEvent(ctx, input.AccountID, "tenant_deleted", "tenant", input.TenantID, map[string]interface{}{
			"tenant_name": tenant.Name,
		})

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

		// Log audit event (use actual user info since login is a public endpoint)
		auditInfo := ExtractAuditInfo(ctx)
		if api.Services.AuditLog != nil {
			api.Services.AuditLog.LogAction(ctx, user.ID, user.Email, user.AccountID, "login", "user", user.ID, auditInfo.IPAddress, map[string]interface{}{})
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

		// Check if user exists in database with admin role (system_admin or account_admin)
		isAdmin := false
		if api.Services.Auth != nil {
			// Try to find user by email in database
			users, err := api.Services.Auth.ListUsers(ctx, "")
			if err == nil {
				for _, user := range users {
					if user.Email == input.Email && (user.Role == "system_admin" || user.Role == "account_admin") {
						isAdmin = true
						break
					}
				}
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
			log.Warnf("Password reset requested for non-admin or non-existent email: %s", input.Email)
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
			DisplayName: input.Name, // Default to same as name
			Description: input.Description,
			Active:      active,
			Status:      status,
			Config:      input.Config,
		}

		if err := api.Services.Storage.CreateDataset(ctx, dataset); err != nil {
			return fmt.Errorf("failed to create dataset: %w", err)
		}

		// Get tenant info for account ID
		tenant, err := api.Services.Storage.GetTenantByID(ctx, input.TenantID)
		accountID := ""
		if err == nil && tenant != nil {
			accountID = tenant.AccountID
		}

		// Log audit event
		api.logAuditEvent(ctx, accountID, "dataset_created", "dataset", dataset.ID, map[string]interface{}{
			"dataset_name": dataset.Name,
			"tenant_id":    dataset.TenantID,
			"active":       dataset.Active,
			"status":       dataset.Status,
		})

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
		Status      string                 `json:"status"`
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

		// Track changes for audit log
		changes := make(map[string]interface{})
		oldValues := make(map[string]interface{})

		// Update fields and track changes
		if input.Name != "" && input.Name != dataset.Name {
			oldValues["name"] = dataset.Name
			changes["name"] = input.Name
			dataset.Name = input.Name
		}
		if input.Description != "" && input.Description != dataset.Description {
			oldValues["description"] = dataset.Description
			changes["description"] = input.Description
			dataset.Description = input.Description
		}
		if input.Active != nil && *input.Active != dataset.Active {
			oldValues["active"] = dataset.Active
			changes["active"] = *input.Active
			dataset.Active = *input.Active
		}
		if input.Status != "" && input.Status != dataset.Status {
			oldValues["status"] = dataset.Status
			changes["status"] = input.Status
			dataset.Status = input.Status
		}
		if input.Config != nil {
			// Deep copy the old config to prevent reference sharing
			oldConfigBytes, err := json.Marshal(dataset.Config)
			if err == nil {
				var oldConfigCopy storage.DatasetConfig
				if err := json.Unmarshal(oldConfigBytes, &oldConfigCopy); err == nil {
					// Store full config in audit log (backend)
					oldValues["config"] = oldConfigCopy
					changes["config"] = *input.Config
				}
			}
			dataset.Config = *input.Config
		}

		if err := api.Services.Storage.UpdateDataset(ctx, dataset); err != nil {
			return fmt.Errorf("failed to update dataset: %w", err)
		}

		// Get tenant info for account ID
		tenant, err := api.Services.Storage.GetTenantByID(ctx, input.TenantID)
		accountID := ""
		if err == nil && tenant != nil {
			accountID = tenant.AccountID
		}

		// Log audit event with changes
		auditDetails := map[string]interface{}{
			"dataset_name": dataset.Name,
			"tenant_id":    input.TenantID,
			"changes":      changes,
			"old_values":   oldValues,
		}
		api.logAuditEvent(ctx, accountID, "dataset_updated", "dataset", input.DatasetID, auditDetails)

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
		TenantID       string `path:"tenantId" required:"true"`
		DatasetID      string `path:"datasetId" required:"true"`
		SkipS3Cleanup  bool   `query:"skip_s3_cleanup"`
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

		// Get dataset info for audit log before deletion
		dataset, err := api.Services.Storage.GetDataset(ctx, input.TenantID, input.DatasetID)
		if err != nil {
			return fmt.Errorf("failed to get dataset: %w", err)
		}

		// Get tenant info for account ID
		tenant, err := api.Services.Storage.GetTenantByID(ctx, input.TenantID)
		if err != nil {
			log.Warnf("Could not get tenant for audit log: %v", err)
		}

		accountID := ""
		if tenant != nil {
			accountID = tenant.AccountID
		}

		// Log audit event BEFORE deletion
		api.logAuditEvent(ctx, accountID, "delete_dataset", "dataset", input.DatasetID, map[string]interface{}{
			"dataset_name": dataset.Name,
			"tenant_id":    input.TenantID,
			"skip_s3_cleanup": input.SkipS3Cleanup,
		})

		// Perform deletion
		if err := api.Services.Storage.DeleteDataset(ctx, input.TenantID, input.DatasetID, input.SkipS3Cleanup); err != nil {
			// Log failure in audit log
			api.logAuditEvent(ctx, accountID, "delete_dataset_failed", "dataset", input.DatasetID, map[string]interface{}{
				"dataset_name": dataset.Name,
				"tenant_id":    input.TenantID,
				"error":        err.Error(),
			})
			return fmt.Errorf("failed to delete dataset: %w", err)
		}

		log.Infof("Dataset %s/%s (%s) deleted successfully", input.TenantID, input.DatasetID, dataset.Name)

		output.Success = true
		if input.SkipS3Cleanup {
			output.Message = fmt.Sprintf("Dataset %s configuration deleted successfully (S3 cleanup skipped)", input.DatasetID)
		} else {
			output.Message = fmt.Sprintf("Dataset %s deleted successfully", input.DatasetID)
		}
		return nil
	})

	u.SetTitle("Delete Dataset")
	u.SetDescription("Deletes a dataset with optional S3 cleanup skip")
	u.SetTags("datasets")
	u.SetExpectedErrors(usecaseStatus.NotFound)

	return u
}

// TestDataset tests dataset input and output configuration
func (api *API) TestDataset() usecase.Interactor {
	type testDatasetInput struct {
		TenantID  string `path:"tenantId" required:"true"`
		DatasetID string `path:"datasetId" required:"true"`
	}

	type testDatasetOutput struct {
		Success            bool      `json:"success"`
		InputTestStatus    string    `json:"input_test_status"`
		InputTestMessage   string    `json:"input_test_message,omitempty"`
		OutputTestStatus   string    `json:"output_test_status"`
		OutputTestMessage  string    `json:"output_test_message,omitempty"`
		LastTestedAt       time.Time `json:"last_tested_at"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input testDatasetInput, output *testDatasetOutput) error {
		api.Services.IncrementAPIRequests()
		api.Services.IncrementDatabaseQueries()

		if api.Services.Storage == nil {
			return fmt.Errorf("storage not initialized")
		}

		// Get dataset
		dataset, err := api.Services.Storage.GetDataset(ctx, input.TenantID, input.DatasetID)
		if err != nil {
			return fmt.Errorf("failed to get dataset: %w", err)
		}

		// Initialize test statuses
		inputStatus := "untested"
		inputMessage := ""
		outputStatus := "untested"
		outputMessage := ""
		now := time.Now()

		// Test Input: Check if any proxy configuration has a plugin for this dataset
		// AND verify the proxy is actually running via health service
		proxyConfigs, err := api.Services.Storage.ListProxyConfigs(ctx, input.TenantID)
		if err != nil {
			log.Warnf("Could not check proxy configs for dataset %s: %v", input.DatasetID, err)
			inputStatus = "untested"
			inputMessage = "Unable to check proxy configuration"
		} else {
			// Get all running proxies from health service
			runningProxies := make(map[string]bool)
			if api.Services.HealthService != nil {
				healthRecords, err := api.Services.HealthService.GetHealthRecordsByService("bytefreezer-proxy")
				if err == nil {
					for _, record := range healthRecords {
						if record.Status == "Healthy" || record.Status == "Active" {
							runningProxies[record.InstanceID] = true
						}
					}
				}
			}

			// Check if any plugin config references this dataset
			foundInProxy := false
			var proxyInstanceID string
			for _, proxyConfig := range proxyConfigs {
				for _, pluginConfig := range proxyConfig.PluginConfigs {
					// Check dataset_id in the nested config object
					var datasetID string
					if config, ok := pluginConfig["config"].(map[string]interface{}); ok {
						if id, ok := config["dataset_id"].(string); ok {
							datasetID = id
						}
					}

					if datasetID == input.DatasetID {
						foundInProxy = true
						proxyInstanceID = proxyConfig.InstanceID

						// Check if config was applied AND proxy is running
						if !proxyConfig.ConfigApplied {
							inputStatus = "untested"
							inputMessage = "Waiting for proxy to apply configuration"
						} else if !runningProxies[proxyInstanceID] {
							inputStatus = "degraded"
							inputMessage = fmt.Sprintf("Proxy %s is configured but not running/healthy", proxyInstanceID)
						} else {
							inputStatus = "active"
							inputMessage = fmt.Sprintf("Plugin configured and proxy %s is running", proxyInstanceID)
						}
						break
					}
				}
				if foundInProxy {
					break
				}
			}

			if !foundInProxy {
				inputStatus = "degraded"
				inputMessage = "No proxy configuration found for this dataset"
			}
		}

		// Test Output: Check S3/Minio configuration and actually write test data
		destType := dataset.Config.Destination.Type
		if (destType == "s3" || destType == "minio") && dataset.Config.Destination.Connection.Bucket != "" {
			conn := dataset.Config.Destination.Connection

			// Extract S3 credentials
			accessKey := conn.Credentials.AccessKey
			secretKey := conn.Credentials.SecretKey
			region := conn.Region
			endpoint := conn.Endpoint
			useSSL := conn.SSL
			bucket := conn.Bucket

			// Try to create S3 client
			s3Cleaner, err := storage.NewS3Cleaner(
				ctx,
				accessKey,
				secretKey,
				region,
				endpoint,
				useSSL,
			)

			if err != nil {
				outputStatus = "degraded"
				outputMessage = fmt.Sprintf("Failed to create S3 client: %v", err)
				log.Warnf("S3 client creation failed for dataset %s: %v", input.DatasetID, err)
			} else {
				// Actually test writing to the bucket
				err = s3Cleaner.TestWrite(ctx, bucket)
				if err != nil {
					outputStatus = "degraded"
					outputMessage = fmt.Sprintf("Failed to write test data to bucket: %v", err)
					log.Warnf("S3 write test failed for dataset %s bucket %s: %v", input.DatasetID, bucket, err)
				} else {
					outputStatus = "active"
					outputMessage = "S3 connection successful and write test passed"
					log.Infof("S3 write test passed for dataset %s bucket %s", input.DatasetID, bucket)
				}
			}
		} else {
			outputStatus = "degraded"
			outputMessage = "S3 output not configured"
		}

		// Update dataset with test results
		dataset.InputTestStatus = inputStatus
		dataset.InputTestMessage = inputMessage
		dataset.OutputTestStatus = outputStatus
		dataset.OutputTestMessage = outputMessage
		dataset.LastTestedAt = &now

		if err := api.Services.Storage.UpdateDataset(ctx, dataset); err != nil {
			return fmt.Errorf("failed to update dataset test status: %w", err)
		}

		// Get tenant info for account ID
		tenant, err := api.Services.Storage.GetTenantByID(ctx, input.TenantID)
		accountID := ""
		if err == nil && tenant != nil {
			accountID = tenant.AccountID
		}

		// Log audit event
		api.logAuditEvent(ctx, accountID, "dataset_tested", "dataset", input.DatasetID, map[string]interface{}{
			"dataset_name":        dataset.Name,
			"tenant_id":           input.TenantID,
			"input_test_status":   inputStatus,
			"output_test_status":  outputStatus,
		})

		// Populate output
		output.Success = true
		output.InputTestStatus = inputStatus
		output.InputTestMessage = inputMessage
		output.OutputTestStatus = outputStatus
		output.OutputTestMessage = outputMessage
		output.LastTestedAt = now

		log.Infof("Dataset %s/%s tested - Input: %s, Output: %s",
			input.TenantID, input.DatasetID, inputStatus, outputStatus)

		return nil
	})

	u.SetTitle("Test Dataset")
	u.SetDescription("Tests dataset input and output configuration")
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
		api.logAuditEvent(ctx, input.AccountID, "user_created", "user", user.ID, map[string]interface{}{
			"email": user.Email,
			"role":  user.Role,
		})

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

		// Get existing user to track changes
		oldUser, err := api.Services.Auth.GetUserByID(ctx, input.UserID)
		if err != nil {
			return fmt.Errorf("failed to get user: %w", err)
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

		// Track changes for audit log
		changes := make(map[string]interface{})
		oldValues := make(map[string]interface{})

		if input.FirstName != "" && input.FirstName != oldUser.FirstName {
			oldValues["first_name"] = oldUser.FirstName
			changes["first_name"] = input.FirstName
		}
		if input.LastName != "" && input.LastName != oldUser.LastName {
			oldValues["last_name"] = oldUser.LastName
			changes["last_name"] = input.LastName
		}
		if input.Role != "" && input.Role != oldUser.Role {
			oldValues["role"] = oldUser.Role
			changes["role"] = input.Role
		}
		if input.Active != nil && *input.Active != oldUser.Active {
			oldValues["active"] = oldUser.Active
			changes["active"] = *input.Active
		}

		// Log audit event with changes
		api.logAuditEvent(ctx, user.AccountID, "user_updated", "user", user.ID, map[string]interface{}{
			"email":      user.Email,
			"changes":    changes,
			"old_values": oldValues,
		})

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
		user, err := api.Services.Auth.GetUserByID(ctx, input.UserID)
		if err == nil {
			api.logAuditEvent(ctx, user.AccountID, "user_deleted", "user", user.ID, map[string]interface{}{
				"email": user.Email,
			})
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
		action := "user_deactivated"
		if input.Active {
			action = "user_activated"
		}
		api.logAuditEvent(ctx, user.AccountID, action, "user", user.ID, map[string]interface{}{
			"email": user.Email,
		})

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

// ProxyResponse represents a proxy instance response
type ProxyResponse struct {
	InstanceID     string                 `json:"instance_id"`
	InstanceAPI    string                 `json:"instance_api"`
	Status         string                 `json:"status"`
	LastSeen       string                 `json:"last_seen"`
	Configuration  map[string]interface{} `json:"configuration"`
	Metrics        map[string]interface{} `json:"metrics,omitempty"`
	ResponseTimeMs int                    `json:"response_time_ms,omitempty"`
}

// ProxiesListResponse represents the proxies list response
type ProxiesListResponse struct {
	Proxies []ProxyResponse `json:"proxies"`
	Count   int             `json:"count"`
}

// GetAccountProxies returns all proxy instances for a specific account
func (api *API) GetAccountProxies() usecase.Interactor {
	type Request struct {
		AccountID string `path:"account_id"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input Request, output *ProxiesListResponse) error {
		api.Services.IncrementAPIRequests()

		if api.Services.HealthService == nil {
			return fmt.Errorf("health service not available")
		}

		// Get proxies for this account
		records, err := api.Services.HealthService.GetProxiesByAccount(input.AccountID)
		if err != nil {
			return fmt.Errorf("failed to get proxy health records: %w", err)
		}

		// Convert health records to proxy responses
		proxies := make([]ProxyResponse, len(records))
		for i, record := range records {
			responseTimeMs := 0
			if record.ResponseTimeMs != nil {
				responseTimeMs = *record.ResponseTimeMs
			}

			// Enrich configuration with plugin data from proxy_instances table
			configuration := record.Configuration
			if configuration == nil {
				configuration = make(map[string]interface{})
			}

			// Get all tenants for this account
			tenantsResult, err := api.Services.Storage.ListTenants(ctx, input.AccountID, storage.ListOptions{})
			if err == nil && tenantsResult != nil {
				// Collect all plugin configs across all tenants for this proxy
				allPluginConfigs := []map[string]interface{}{}

				for _, tenant := range tenantsResult.Items {
					proxyConfigs, err := api.Services.Storage.ListProxyConfigs(ctx, tenant.ID)
					if err == nil {
						for _, pc := range proxyConfigs {
							if pc.InstanceID == record.InstanceID {
								// Add this proxy's plugin configs
								allPluginConfigs = append(allPluginConfigs, pc.PluginConfigs...)
							}
						}
					}
				}

				// Add plugins to configuration
				if len(allPluginConfigs) > 0 {
					configuration["plugins"] = map[string]interface{}{
						"total_plugins":  len(allPluginConfigs),
						"plugin_details": allPluginConfigs,
					}
				}
			}

			proxies[i] = ProxyResponse{
				InstanceID:     record.InstanceID,
				InstanceAPI:    record.InstanceAPI,
				Status:         record.Status,
				LastSeen:       record.LastSeen.Format("2006-01-02T15:04:05Z07:00"),
				Configuration:  configuration,
				Metrics:        record.Metrics,
				ResponseTimeMs: responseTimeMs,
			}
		}

		output.Proxies = proxies
		output.Count = len(proxies)

		log.Infof("Retrieved %d proxy instances for account %s", len(proxies), input.AccountID)

		return nil
	})

	u.SetTitle("Get Account Proxies")
	u.SetDescription("Returns all proxy instances registered to a specific account")
	u.SetTags("proxies", "health")

	return u
}

// Audit Log API handlers

// AuditLogWithDiff extends AuditLog with computed diffs for config changes
type AuditLogWithDiff struct {
	storage.AuditLog
	ConfigDiff *ConfigDiff `json:"config_diff,omitempty"`
}

// ConfigDiff represents the computed differences in configuration
type ConfigDiff struct {
	OldValues map[string]interface{} `json:"old"`
	NewValues map[string]interface{} `json:"new"`
}

// ListAuditLogs returns audit logs filtered by account or system-wide
func (api *API) ListAuditLogs() usecase.Interactor {
	type listAuditLogsInput struct {
		AccountID    string `query:"account_id"`    // Filter by account (required for account admins, optional for system admins)
		UserID       string `query:"user_id"`       // Filter by specific user
		Action       string `query:"action"`        // Filter by action type
		ResourceType string `query:"resource_type"` // Filter by resource type
		ResourceID   string `query:"resource_id"`   // Filter by specific resource
		Limit        int    `query:"limit"`         // Number of results (default 100)
		Offset       int    `query:"offset"`        // Pagination offset
	}

	type listAuditLogsOutput struct {
		Items  []AuditLogWithDiff `json:"items"`
		Total  int                `json:"total"`
		Limit  int                `json:"limit"`
		Offset int                `json:"offset"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input listAuditLogsInput, output *listAuditLogsOutput) error {
		api.Services.IncrementAPIRequests()
		api.Services.IncrementDatabaseQueries()

		if api.Services.Storage == nil {
			return fmt.Errorf("storage not initialized")
		}

		// TODO: Extract user role from JWT token and enforce access control
		// - System admins: can view all logs (no account_id filter required)
		// - Account admins: can only view logs for their account (must match JWT account_id)
		// For now, we'll allow all requests but log a warning

		filter := storage.AuditLogFilter{
			AccountID:    input.AccountID,
			UserID:       input.UserID,
			Action:       input.Action,
			ResourceType: input.ResourceType,
			ResourceID:   input.ResourceID,
			Limit:        input.Limit,
			Offset:       input.Offset,
		}

		result, err := api.Services.Storage.ListAuditLogs(ctx, filter)
		if err != nil {
			return fmt.Errorf("failed to list audit logs: %w", err)
		}

		// Transform each audit log to include computed config diffs
		items := make([]AuditLogWithDiff, len(result.Items))
		for i, auditLog := range result.Items {
			item := AuditLogWithDiff{
				AuditLog: auditLog,
			}

			// Check if this audit log has config changes in Details
			if auditLog.Details != nil {
				oldValuesRaw, hasOldValues := auditLog.Details["old_values"]
				changesRaw, hasChanges := auditLog.Details["changes"]

				if hasOldValues && hasChanges {
					// Try to extract config from both old_values and changes
					oldValues, okOld := oldValuesRaw.(map[string]interface{})
					changes, okChanges := changesRaw.(map[string]interface{})

					if okOld && okChanges {
						oldConfig, hasOldConfig := oldValues["config"]
						newConfig, hasNewConfig := changes["config"]

						// If both configs exist, compute the diff
						if hasOldConfig && hasNewConfig {
							oldDiff, newDiff := computeConfigDiff(oldConfig, newConfig)

							// Only add ConfigDiff if there are actual differences
							if len(oldDiff) > 0 || len(newDiff) > 0 {
								item.ConfigDiff = &ConfigDiff{
									OldValues: oldDiff,
									NewValues: newDiff,
								}
							}
						}
					}
				}
			}

			items[i] = item
		}

		output.Items = items
		output.Total = result.Total
		output.Limit = input.Limit
		output.Offset = input.Offset

		return nil
	})

	u.SetTitle("List Audit Logs")
	u.SetDescription("Returns audit logs filtered by account or system-wide (account admins see their account only, system admins see all)")
	u.SetTags("audit")

	return u
}

// GetAuditLog returns a specific audit log entry by ID
func (api *API) GetAuditLog() usecase.Interactor {
	type getAuditLogInput struct {
		LogID int64 `path:"logId" required:"true"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input getAuditLogInput, output *AuditLogWithDiff) error {
		api.Services.IncrementAPIRequests()
		api.Services.IncrementDatabaseQueries()

		if api.Services.Storage == nil {
			return fmt.Errorf("storage not initialized")
		}

		// TODO: Verify that the user has permission to view this audit log
		// - System admins: can view all logs
		// - Account admins: can only view logs for their account
		// This should be done by extracting account_id from JWT and comparing with log.account_id

		auditLog, err := api.Services.Storage.GetAuditLog(ctx, input.LogID)
		if err != nil {
			return fmt.Errorf("failed to get audit log: %w", err)
		}

		// Populate output with audit log
		output.AuditLog = *auditLog

		// Check if this audit log has config changes and compute diff
		if auditLog.Details != nil {
			oldValuesRaw, hasOldValues := auditLog.Details["old_values"]
			changesRaw, hasChanges := auditLog.Details["changes"]

			if hasOldValues && hasChanges {
				oldValues, okOld := oldValuesRaw.(map[string]interface{})
				changes, okChanges := changesRaw.(map[string]interface{})

				if okOld && okChanges {
					oldConfig, hasOldConfig := oldValues["config"]
					newConfig, hasNewConfig := changes["config"]

					if hasOldConfig && hasNewConfig {
						oldDiff, newDiff := computeConfigDiff(oldConfig, newConfig)

						if len(oldDiff) > 0 || len(newDiff) > 0 {
							output.ConfigDiff = &ConfigDiff{
								OldValues: oldDiff,
								NewValues: newDiff,
							}
						}
					}
				}
			}
		}

		return nil
	})

	u.SetTitle("Get Audit Log")
	u.SetDescription("Returns a specific audit log entry by ID with computed config differences")
	u.SetTags("audit")
	u.SetExpectedErrors(usecaseStatus.NotFound)

	return u
}


// computeConfigDiff compares two config objects and returns only the differences
func computeConfigDiff(oldConfig, newConfig interface{}) (oldDiff, newDiff map[string]interface{}) {
	oldDiff = make(map[string]interface{})
	newDiff = make(map[string]interface{})

	// Marshal both configs to JSON for comparison
	oldBytes, err1 := json.Marshal(oldConfig)
	newBytes, err2 := json.Marshal(newConfig)
	if err1 != nil || err2 != nil {
		// If marshaling fails, return empty diffs
		return
	}

	// Unmarshal to generic maps for comparison
	var oldMap, newMap map[string]interface{}
	json.Unmarshal(oldBytes, &oldMap)
	json.Unmarshal(newBytes, &newMap)

	// Compare the two maps recursively
	compareMap("", oldMap, newMap, oldDiff, newDiff)

	return oldDiff, newDiff
}

// compareMap recursively compares two maps and stores differences with dotted path notation
func compareMap(prefix string, oldMap, newMap map[string]interface{}, oldDiff, newDiff map[string]interface{}) {
	// Check all keys in both maps
	allKeys := make(map[string]bool)
	for k := range oldMap {
		allKeys[k] = true
	}
	for k := range newMap {
		allKeys[k] = true
	}

	for key := range allKeys {
		path := key
		if prefix != "" {
			path = prefix + "." + key
		}

		oldVal, oldExists := oldMap[key]
		newVal, newExists := newMap[key]

		// Key removed
		if oldExists && !newExists {
			oldDiff[path] = oldVal
			newDiff[path] = nil
			continue
		}

		// Key added
		if !oldExists && newExists {
			oldDiff[path] = nil
			newDiff[path] = newVal
			continue
		}

		// Both exist - check if they're maps (nested objects)
		oldMapVal, oldIsMap := oldVal.(map[string]interface{})
		newMapVal, newIsMap := newVal.(map[string]interface{})

		if oldIsMap && newIsMap {
			// Recursively compare nested maps
			compareMap(path, oldMapVal, newMapVal, oldDiff, newDiff)
		} else {
			// Compare values directly
			oldJSON, _ := json.Marshal(oldVal)
			newJSON, _ := json.Marshal(newVal)
			if string(oldJSON) != string(newJSON) {
				oldDiff[path] = oldVal
				newDiff[path] = newVal
			}
		}
	}
}

// logAuditEvent is a helper that extracts user info from context and logs an audit event
// This ensures audit logs always capture the user email and IP at the time of the event
func (api *API) logAuditEvent(ctx context.Context, accountID, action, resourceType, resourceID string, details map[string]interface{}) {
	if api.Services.AuditLog == nil {
		return
	}

	// Extract user info and IP from context (set by middleware)
	auditInfo := ExtractAuditInfo(ctx)

	// Log the action with the captured user info and IP address
	api.Services.AuditLog.LogAction(ctx, auditInfo.UserID, auditInfo.UserEmail, accountID, action, resourceType, resourceID, auditInfo.IPAddress, details)
}
