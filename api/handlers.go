package api

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"strings"
	"time"

	"github.com/n0needt0/bytefreezer-control/middleware"
	"github.com/n0needt0/bytefreezer-control/storage"
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

// CreateTenant creates a new tenant
func (api *API) CreateTenant() usecase.Interactor {
	type createTenantInput struct {
		AccountID   string `path:"accountId" required:"true"`
		ID          string `json:"id" required:"true"`
		Name        string `json:"name" required:"true"`
		Description string `json:"description"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input createTenantInput, output *storage.Tenant) error {
		api.Services.IncrementAPIRequests()
		api.Services.IncrementDatabaseQueries()

		if api.Services.Storage == nil {
			return fmt.Errorf("storage not initialized")
		}

		// Validate ID format
		if err := ValidateID(input.ID); err != nil {
			return fmt.Errorf("invalid tenant ID: %w", err)
		}

		tenant := &storage.Tenant{
			ID:          input.ID,
			AccountID:   input.AccountID,
			Name:        input.Name,
			Description: input.Description,
			Active:      true,
		}

		if err := api.Services.Storage.CreateTenant(ctx, tenant); err != nil {
			return fmt.Errorf("failed to create tenant: %w", err)
		}

		*output = *tenant
		return nil
	})

	u.SetTitle("Create Tenant")
	u.SetDescription("Creates a new tenant for an account")
	u.SetTags("tenants")
	u.SetExpectedErrors(usecaseStatus.InvalidArgument)

	return u
}

// UpdateTenant updates an existing tenant
func (api *API) UpdateTenant() usecase.Interactor {
	type updateTenantInput struct {
		AccountID   string `path:"accountId" required:"true"`
		TenantID    string `path:"tenantId" required:"true"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Active      *bool  `json:"active"`
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
		ID          string                `json:"id" required:"true"`
		Name        string                `json:"name" required:"true"`
		Description string                `json:"description"`
		Config      storage.DatasetConfig `json:"config"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input createDatasetInput, output *storage.Dataset) error {
		api.Services.IncrementAPIRequests()
		api.Services.IncrementDatabaseQueries()

		if api.Services.Storage == nil {
			return fmt.Errorf("storage not initialized")
		}

		// Validate ID format
		if err := ValidateID(input.ID); err != nil {
			return fmt.Errorf("invalid dataset ID: %w", err)
		}

		dataset := &storage.Dataset{
			ID:          input.ID,
			TenantID:    input.TenantID,
			Name:        input.Name,
			Description: input.Description,
			Active:      true,
			Status:      "active",
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

