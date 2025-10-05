package client

import "time"

// Account represents an account in the Control Service
type Account struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Email     string                 `json:"email"`
	Active    bool                   `json:"active"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	Config    map[string]interface{} `json:"config,omitempty"`
}

// Tenant represents a tenant within an account
type Tenant struct {
	ID          string                 `json:"id"`
	AccountID   string                 `json:"account_id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Active      bool                   `json:"active"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Config      map[string]interface{} `json:"config,omitempty"`
}

// Dataset represents a dataset within a tenant
type Dataset struct {
	ID          string                 `json:"id"`
	AccountID   string                 `json:"account_id"`
	TenantID    string                 `json:"tenant_id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Active      bool                   `json:"active"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Config      map[string]interface{} `json:"config,omitempty"`
}

// CreateAccountRequest is the request body for creating an account
type CreateAccountRequest struct {
	Name   string                 `json:"name"`
	Email  string                 `json:"email"`
	Config map[string]interface{} `json:"config,omitempty"`
}

// UpdateAccountRequest is the request body for updating an account
type UpdateAccountRequest struct {
	Name   string                 `json:"name,omitempty"`
	Email  string                 `json:"email,omitempty"`
	Active *bool                  `json:"active,omitempty"`
	Config map[string]interface{} `json:"config,omitempty"`
}

// CreateTenantRequest is the request body for creating a tenant
type CreateTenantRequest struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty"`
}

// UpdateTenantRequest is the request body for updating a tenant
type UpdateTenantRequest struct {
	Name        string                 `json:"name,omitempty"`
	Description string                 `json:"description,omitempty"`
	Active      *bool                  `json:"active,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty"`
}

// CreateDatasetRequest is the request body for creating a dataset
type CreateDatasetRequest struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty"`
}

// UpdateDatasetRequest is the request body for updating a dataset
type UpdateDatasetRequest struct {
	Name        string                 `json:"name,omitempty"`
	Description string                 `json:"description,omitempty"`
	Active      *bool                  `json:"active,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty"`
}
