package storage

import (
	"context"
	"fmt"
	"time"
)

// Account represents a ByteFreezer account (top-level entity that owns tenants)
type Account struct {
	ID        string        `json:"id" db:"id"`
	Name      string        `json:"name" db:"name"`
	Email     string        `json:"email" db:"email"`
	Active    bool          `json:"active" db:"active"`
	CreatedAt time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt time.Time     `json:"updated_at" db:"updated_at"`
	Config    AccountConfig `json:"config" db:"config"`
}

// AccountConfig represents account-level configuration (shared by all tenants)
type AccountConfig struct {
	// Account limits
	Tier         string `json:"tier,omitempty"`          // free, pro, enterprise
	MaxTenants   int    `json:"max_tenants,omitempty"`   // Limit on number of tenants
	MaxDatasets  int    `json:"max_datasets,omitempty"`  // Total limit across all tenants

	// These are defined in config_types.go and imported here
	Organization   OrganizationConfig `json:"organization"`
	Subscription   SubscriptionConfig `json:"subscription"`
	Notifications  NotificationConfig `json:"notifications"`
	Security       SecurityConfig     `json:"security"`
	Billing        BillingConfig      `json:"billing"`

	CustomFields map[string]interface{} `json:"custom_fields,omitempty"` // Flexible custom configuration
}

// APIKey represents an API key for authentication
type APIKey struct {
	ID         string                 `json:"id" db:"id"`
	AccountID  string                 `json:"account_id" db:"account_id"`
	Name       string                 `json:"name" db:"name"`
	KeyHash    string                 `json:"-" db:"key_hash"` // Never expose in JSON
	Key        string                 `json:"key,omitempty"`   // Only returned on creation
	Permissions map[string]interface{} `json:"permissions" db:"permissions"`
	Active     bool                   `json:"active" db:"active"`
	LastUsedAt *time.Time             `json:"last_used_at" db:"last_used_at"`
	CreatedAt  time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at" db:"updated_at"`
}

// Tenant represents a ByteFreezer tenant (belongs to an account)
type Tenant struct {
	ID          string       `json:"id" db:"id"`
	AccountID   string       `json:"account_id" db:"account_id"`
	Name        string       `json:"name" db:"name"`
	DisplayName string       `json:"display_name" db:"display_name"`
	Description string       `json:"description" db:"description"`
	Active      bool         `json:"active" db:"active"`
	CreatedAt   time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at" db:"updated_at"`
	Config      TenantConfig `json:"config" db:"config"`
}

// Dataset represents a tenant's data processing pipeline with rich configuration
type Dataset struct {
	ID          string        `json:"id" db:"id"`
	TenantID    string        `json:"tenant_id" db:"tenant_id"`
	Name        string        `json:"name" db:"name"`
	DisplayName string        `json:"display_name" db:"display_name"`
	Description string        `json:"description" db:"description"`
	Active      bool          `json:"active" db:"active"`
	Status      string        `json:"status" db:"status"` // active, paused, error, processing
	CreatedAt   time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at" db:"updated_at"`
	Config      DatasetConfig `json:"config" db:"config"`
	
	// Runtime metrics
	RecordsProcessed   int64      `json:"records_processed" db:"records_processed"`
	LastProcessedAt    *time.Time `json:"last_processed_at" db:"last_processed_at"`
	ErrorCount         int        `json:"error_count" db:"error_count"`
	LastError          string     `json:"last_error" db:"last_error"`
	ProcessingMetrics  string     `json:"processing_metrics" db:"processing_metrics"` // JSON
}

// DatasetMetrics represents processing metrics for a dataset
type DatasetMetrics struct {
	TotalRecords      int64              `json:"total_records"`
	ProcessedRecords  int64              `json:"processed_records"`
	ErrorRecords      int64              `json:"error_records"`
	SkippedRecords    int64              `json:"skipped_records"`
	ProcessingRate    float64            `json:"processing_rate_per_sec"`
	AvgLatencyMS      float64            `json:"avg_latency_ms"`
	LastProcessedAt   time.Time          `json:"last_processed_at"`
	ErrorRate         float64            `json:"error_rate"`
	Throughput        map[string]int64   `json:"throughput"` // hourly/daily counts
	CustomMetrics     map[string]interface{} `json:"custom_metrics,omitempty"`
}

// ListOptions provides pagination and filtering options
type ListOptions struct {
	Limit  int    `json:"limit"`
	Cursor string `json:"cursor,omitempty"`
	Filter string `json:"filter,omitempty"`
}

// ListResult contains paginated results
type ListResult[T any] struct {
	Items      []T    `json:"items"`
	NextCursor string `json:"next_cursor,omitempty"`
	Total      int    `json:"total"`
}

// Storage defines the interface for all database backends
type Storage interface {
	// Account operations
	CreateAccount(ctx context.Context, account *Account) error
	GetAccount(ctx context.Context, id string) (*Account, error)
	GetAccountByEmail(ctx context.Context, email string) (*Account, error)
	UpdateAccount(ctx context.Context, account *Account) error
	DeleteAccount(ctx context.Context, id string) error
	ListAccounts(ctx context.Context, opts ListOptions) (*ListResult[Account], error)

	// Tenant operations (scoped to account)
	CreateTenant(ctx context.Context, tenant *Tenant) error
	GetTenant(ctx context.Context, accountID, tenantID string) (*Tenant, error)
	GetTenantByID(ctx context.Context, tenantID string) (*Tenant, error) // Direct lookup by ID (for proxy validation)
	UpdateTenant(ctx context.Context, tenant *Tenant) error
	DeleteTenant(ctx context.Context, accountID, tenantID string) error
	ListTenants(ctx context.Context, accountID string, opts ListOptions) (*ListResult[Tenant], error)
	ListAllTenants(ctx context.Context, opts ListOptions) (*ListResult[Tenant], error) // List all tenants across all accounts

	// Advanced tenant queries
	FindTenantsBySubscriptionTier(ctx context.Context, tier string) ([]*Tenant, error)
	FindTenantsByOrganizationSize(ctx context.Context, size string) ([]*Tenant, error)
	GetTenantsWithNotificationEnabled(ctx context.Context, notificationType string) ([]*Tenant, error)

	// Dataset operations
	CreateDataset(ctx context.Context, dataset *Dataset) error
	GetDataset(ctx context.Context, tenantID, datasetID string) (*Dataset, error)
	UpdateDataset(ctx context.Context, dataset *Dataset) error
	DeleteDataset(ctx context.Context, tenantID, datasetID string) error
	ListDatasets(ctx context.Context, tenantID string, opts ListOptions) (*ListResult[Dataset], error)
	ListAllDatasets(ctx context.Context, opts ListOptions) (*ListResult[Dataset], error) // List all datasets across all tenants
	
	// Advanced dataset queries
	FindDatasetsByStatus(ctx context.Context, tenantID, status string) ([]*Dataset, error)
	FindDatasetsBySourceType(ctx context.Context, tenantID, sourceType string) ([]*Dataset, error)
	GetDatasetMetrics(ctx context.Context, tenantID, datasetID string) (*DatasetMetrics, error)
	UpdateDatasetMetrics(ctx context.Context, tenantID, datasetID string, metrics *DatasetMetrics) error

	// API Key operations
	CreateAPIKey(ctx context.Context, apiKey *APIKey) error
	GetAPIKey(ctx context.Context, id string) (*APIKey, error)
	GetAPIKeyByHash(ctx context.Context, keyHash string) (*APIKey, error)
	UpdateAPIKey(ctx context.Context, apiKey *APIKey) error
	DeleteAPIKey(ctx context.Context, id string) error
	ListAPIKeys(ctx context.Context, accountID string, opts ListOptions) (*ListResult[APIKey], error)
	UpdateAPIKeyLastUsed(ctx context.Context, id string) error

	// Utility operations
	HealthCheck(ctx context.Context) error
	Migrate(ctx context.Context) error
	Close() error

	// Migration operations
	GetMigrator() Migrator
}

// Config represents PostgreSQL database configuration
type Config struct {
	Type           string `yaml:"type" json:"type"`                       // postgresql (only supported type now)
	URI            string `yaml:"uri" json:"uri"`                         // PostgreSQL connection URI
	Database       string `yaml:"database" json:"database"`               // Database name
	TimeoutSeconds int    `yaml:"timeout_seconds" json:"timeout_seconds"` // Connection timeout
	SSLMode        string `yaml:"ssl_mode" json:"ssl_mode"`               // PostgreSQL SSL mode (disable, require, verify-ca, verify-full)
}

// Migration represents a database migration
type Migration struct {
	Version     int       `json:"version"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	AppliedAt   time.Time `json:"applied_at"`
}

// MigrationStatus represents the current migration status
type MigrationStatus struct {
	CurrentVersion int         `json:"current_version"`
	Migrations     []Migration `json:"migrations"`
	Applied        []Migration `json:"applied"`
	Pending        []Migration `json:"pending"`
}

// MigrationOptions provides options for migrations
type MigrationOptions struct {
	FreshInstall  bool `json:"fresh_install"`
	ResetData     bool `json:"reset_data"`
	TargetVersion int  `json:"target_version"`
	DryRun        bool `json:"dry_run"`
	Force         bool `json:"force"`
}

// Migrator interface for database migrations
type Migrator interface {
	GetStatus(ctx context.Context) (*MigrationStatus, error)
	Migrate(ctx context.Context, opts MigrationOptions) error
	Rollback(ctx context.Context, targetVersion int) error
	Reset(ctx context.Context) error
}

// NewStorage creates a new PostgreSQL storage backend
func NewStorage(config Config) (Storage, error) {
	// Only PostgreSQL is supported now
	if config.Type != "" && config.Type != "postgresql" {
		return nil, fmt.Errorf("only PostgreSQL storage is supported, got: %s", config.Type)
	}
	
	return NewPostgreSQLStorage(config)
}