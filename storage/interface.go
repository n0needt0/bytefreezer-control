package storage

import (
	"context"
	"fmt"
	"time"
)

// Account deployment types
const (
	DeploymentTypeManaged    = "managed"     // Fully managed by ByteFreezer (compute + storage)
	DeploymentTypeOnPrem     = "on_prem"     // Customer-hosted compute, central control plane
	DeploymentTypeAirGapped  = "air_gapped"  // Fully customer-managed, no external control
)

// Account represents a ByteFreezer account (top-level entity that owns tenants)
type Account struct {
	ID             string        `json:"id" db:"id"`
	Name           string        `json:"name" db:"name"`
	Email          string        `json:"email" db:"email"`
	Active         bool          `json:"active" db:"active"`
	DeploymentType string        `json:"deployment_type" db:"deployment_type"` // managed, on_prem, air_gapped
	CreatedAt      time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at" db:"updated_at"`
	Config         AccountConfig `json:"config" db:"config"`
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

	// Test status fields
	InputTestStatus   string     `json:"input_test_status" db:"input_test_status"`     // untested, testing, active, degraded
	InputTestMessage  string     `json:"input_test_message" db:"input_test_message"`   // why degraded
	OutputTestStatus  string     `json:"output_test_status" db:"output_test_status"`   // untested, testing, active, degraded
	OutputTestMessage string     `json:"output_test_message" db:"output_test_message"` // why degraded
	LastTestedAt      *time.Time `json:"last_tested_at" db:"last_tested_at"`           // when last tested
}

// DatasetMetrics represents processing metrics for a dataset (legacy, kept for compatibility)
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

// DatasetMetric represents a single metrics data point for time-series storage
type DatasetMetric struct {
	ID                   int64                  `json:"id" db:"id"`
	TenantID             string                 `json:"tenant_id" db:"tenant_id"`
	DatasetID            string                 `json:"dataset_id" db:"dataset_id"`
	RecordedAt           time.Time              `json:"recorded_at" db:"recorded_at"`
	Component            string                 `json:"component" db:"component"` // proxy, piper, packer, control

	// Metrics fields
	InputBytes           int64                  `json:"input_bytes" db:"input_bytes"`
	OutputBytes          int64                  `json:"output_bytes" db:"output_bytes"`
	LinesProcessed       int64                  `json:"lines_processed" db:"lines_processed"`
	ErrorCount           int64                  `json:"error_count" db:"error_count"`

	// Aggregation buckets
	HourBucket           *time.Time             `json:"hour_bucket,omitempty" db:"hour_bucket"`
	DayBucket            *time.Time             `json:"day_bucket,omitempty" db:"day_bucket"`

	// Additional data
	CustomMetrics        map[string]interface{} `json:"custom_metrics,omitempty" db:"custom_metrics"`
}

// MetricsQueryFilter provides filtering options for metrics queries
type MetricsQueryFilter struct {
	TenantID   string    // Required
	DatasetID  string    // Required
	Component  string    // Optional: filter by component (proxy, piper, packer, control)
	StartTime  time.Time // Required
	EndTime    time.Time // Required
	Limit      int       // Max records to return (default 1000)
}

// ComponentMetrics represents aggregated metrics for a specific component
type ComponentMetrics struct {
	Component      string    `json:"component"`
	InputBytes     int64     `json:"input_bytes"`
	OutputBytes    int64     `json:"output_bytes"`
	LinesProcessed int64     `json:"lines_processed"`
	ErrorCount     int64     `json:"error_count"`
	StartTime      time.Time `json:"start_time"`
	EndTime        time.Time `json:"end_time"`
}

// DatasetSample represents a data sample collected during pipeline processing
type DatasetSample struct {
	ID         int64                  `json:"id"`
	TenantID   string                 `json:"tenant_id"`
	DatasetID  string                 `json:"dataset_id"`
	SampleType string                 `json:"sample_type"` // "input" or "output"
	LineNumber int                    `json:"line_number"`
	SampleData map[string]interface{} `json:"sample_data"`
	BatchID    string                 `json:"batch_id"`
	CreatedAt  time.Time              `json:"created_at"`
}

// DatasetSchema represents inferred schema from dataset samples
type DatasetSchema struct {
	ID         int64     `json:"id"`
	TenantID   string    `json:"tenant_id"`
	DatasetID  string    `json:"dataset_id"`
	SchemaType string    `json:"schema_type"` // "input" or "output"
	SchemaData []byte    `json:"schema_data"` // JSON schema data
	UpdatedAt  time.Time `json:"updated_at"`
}

// AuditLog represents an audit log entry for tracking user actions
type AuditLog struct {
	ID           int64                  `json:"id" db:"id"`
	UserID       string                 `json:"user_id" db:"user_id"`
	UserEmail    string                 `json:"user_email" db:"user_email"`
	AccountID    string                 `json:"account_id" db:"account_id"`
	Action       string                 `json:"action" db:"action"`
	ResourceType string                 `json:"resource_type" db:"resource_type"`
	ResourceID   string                 `json:"resource_id" db:"resource_id"`
	ResourceName string                 `json:"resource_name" db:"resource_name"`
	Details      map[string]interface{} `json:"details" db:"details"`
	IPAddress    string                 `json:"ip_address" db:"ip_address"`
	UserAgent    string                 `json:"user_agent" db:"user_agent"`
	Status       string                 `json:"status" db:"status"`
	ErrorMessage string                 `json:"error_message" db:"error_message"`
	CreatedAt    time.Time              `json:"created_at" db:"created_at"`
}

// AuditLogFilter provides filtering options for audit log queries
type AuditLogFilter struct {
	AccountID    string    // Required for account admins, optional for system admins
	UserID       string    // Filter by specific user
	Action       string    // Filter by action type
	ResourceType string    // Filter by resource type
	ResourceID   string    // Filter by specific resource
	StartDate    time.Time // Filter by date range
	EndDate      time.Time
	Limit        int       // Number of results to return (default 100)
	Offset       int       // Pagination offset
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

// ============================================================================
// PIPER DATA TYPES
// ============================================================================

// PiperFileLock represents a file lock for distributed processing
type PiperFileLock struct {
	LockID        int64     `json:"lock_id" db:"lock_id"`
	TenantID      string    `json:"tenant_id" db:"tenant_id"`
	DatasetID     string    `json:"dataset_id" db:"dataset_id"`
	FileKey       string    `json:"file_key" db:"file_key"`
	LockedBy      string    `json:"locked_by" db:"locked_by"`
	LockTimestamp time.Time `json:"lock_timestamp" db:"lock_timestamp"`
	LastHeartbeat time.Time `json:"last_heartbeat" db:"last_heartbeat"`
	TTL           time.Time `json:"ttl" db:"ttl"`
}

// PiperJobRecord represents a job record for tracking processing jobs
type PiperJobRecord struct {
	JobID            string    `json:"job_id" db:"job_id"`
	TenantID         string    `json:"tenant_id" db:"tenant_id"`
	DatasetID        string    `json:"dataset_id" db:"dataset_id"`
	Status           string    `json:"status" db:"status"`
	SourceFiles      []string  `json:"source_files" db:"source_files"`
	ProcessorType    string    `json:"processor_type" db:"processor_type"`
	ProcessorID      string    `json:"processor_id" db:"processor_id"`
	OutputFile       string    `json:"output_file" db:"output_file"`
	ErrorMessage     string    `json:"error_message" db:"error_message"`
	RecordsProcessed int64     `json:"records_processed" db:"records_processed"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

// PiperPipelineCache represents cached pipeline configuration
type PiperPipelineCache struct {
	ConfigKey     string                 `json:"config_key" db:"config_key"`
	TenantID      string                 `json:"tenant_id" db:"tenant_id"`
	DatasetID     string                 `json:"dataset_id" db:"dataset_id"`
	Configuration map[string]interface{} `json:"configuration" db:"configuration"`
	CachedAt      time.Time              `json:"cached_at" db:"cached_at"`
	ExpiresAt     time.Time              `json:"expires_at" db:"expires_at"`
}

// PiperTenantCache represents cached tenant information
type PiperTenantCache struct {
	TenantID   string                 `json:"tenant_id" db:"tenant_id"`
	TenantData map[string]interface{} `json:"tenant_data" db:"tenant_data"`
	CachedAt   time.Time              `json:"cached_at" db:"cached_at"`
	ExpiresAt  time.Time              `json:"expires_at" db:"expires_at"`
}

// ============================================================================
// PACKER DATA TYPES
// ============================================================================

// PackerTenantLock represents a tenant lock for packer operations
type PackerTenantLock struct {
	TenantID      string    `json:"tenant_id" db:"tenant_id"`
	LockedBy      string    `json:"locked_by" db:"locked_by"`
	LockTimestamp time.Time `json:"lock_timestamp" db:"lock_timestamp"`
	LastHeartbeat time.Time `json:"last_heartbeat" db:"last_heartbeat"`
	TTL           time.Time `json:"ttl" db:"ttl"`
}

// PackerParquetFileMetadata represents metadata for a Parquet file
type PackerParquetFileMetadata struct {
	ID              int64                  `json:"id" db:"id"`
	TenantID        string                 `json:"tenant_id" db:"tenant_id"`
	DatasetID       string                 `json:"dataset_id" db:"dataset_id"`
	FilePath        string                 `json:"file_path" db:"file_path"`
	PartitionPath   string                 `json:"partition_path" db:"partition_path"`
	FileSizeBytes   int64                  `json:"file_size_bytes" db:"file_size_bytes"`
	RowCount        int64                  `json:"row_count" db:"row_count"`
	CreatedAt       time.Time              `json:"created_at" db:"created_at"`
	LastModified    time.Time              `json:"last_modified" db:"last_modified"`
	SchemaJSON      map[string]interface{} `json:"schema_json" db:"schema_json"`
	ColumnStats     map[string]interface{} `json:"column_stats" db:"column_stats"`
	FileChecksum    string                 `json:"file_checksum" db:"file_checksum"`
	InstanceID      string                 `json:"instance_id" db:"instance_id"`
	MetadataVersion int                    `json:"metadata_version" db:"metadata_version"`
	TTL             time.Time              `json:"ttl" db:"ttl"`
	InsertedAt      time.Time              `json:"inserted_at" db:"inserted_at"`
	UpdatedAt       time.Time              `json:"updated_at" db:"updated_at"`
}

// PackerMetadataGenerationStatus tracks metadata generation for partitions
type PackerMetadataGenerationStatus struct {
	TenantID          string    `json:"tenant_id" db:"tenant_id"`
	DatasetID         string    `json:"dataset_id" db:"dataset_id"`
	PartitionPath     string    `json:"partition_path" db:"partition_path"`
	LastGeneratedAt   time.Time `json:"last_generated_at" db:"last_generated_at"`
	FileCount         int       `json:"file_count" db:"file_count"`
	TotalRows         int64     `json:"total_rows" db:"total_rows"`
	TotalSizeBytes    int64     `json:"total_size_bytes" db:"total_size_bytes"`
	NeedsRegeneration bool      `json:"needs_regeneration" db:"needs_regeneration"`
	CurrentSchemaHash string    `json:"current_schema_hash" db:"current_schema_hash"`
	SchemaVersion     int       `json:"schema_version" db:"schema_version"`
	TTL               time.Time `json:"ttl" db:"ttl"`
}

// PackerParquetMetadataSummary provides aggregated metadata information
type PackerParquetMetadataSummary struct {
	TenantID            string    `json:"tenant_id" db:"tenant_id"`
	DatasetID           string    `json:"dataset_id" db:"dataset_id"`
	PartitionPath       string    `json:"partition_path" db:"partition_path"`
	FileCount           int       `json:"file_count" db:"file_count"`
	TotalRows           int64     `json:"total_rows" db:"total_rows"`
	TotalSizeBytes      int64     `json:"total_size_bytes" db:"total_size_bytes"`
	FirstFileCreated    time.Time `json:"first_file_created" db:"first_file_created"`
	LastFileModified    time.Time `json:"last_file_modified" db:"last_file_modified"`
	MetadataLastUpdated time.Time `json:"metadata_last_updated" db:"metadata_last_updated"`
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
	ListAllTenants(ctx context.Context, opts ListOptions) (*ListResult[Tenant], error)         // List all tenants across all accounts
	ListTenantsForAccount(ctx context.Context, accountID string, opts ListOptions) (*ListResult[Tenant], error) // List tenants for a specific account

	// Advanced tenant queries
	FindTenantsBySubscriptionTier(ctx context.Context, tier string) ([]*Tenant, error)
	FindTenantsByOrganizationSize(ctx context.Context, size string) ([]*Tenant, error)
	GetTenantsWithNotificationEnabled(ctx context.Context, notificationType string) ([]*Tenant, error)

	// Dataset operations
	CreateDataset(ctx context.Context, dataset *Dataset) error
	GetDataset(ctx context.Context, tenantID, datasetID string) (*Dataset, error)
	UpdateDataset(ctx context.Context, dataset *Dataset) error
	DeleteDataset(ctx context.Context, tenantID, datasetID string, skipS3Cleanup bool) error
	ListDatasets(ctx context.Context, tenantID string, opts ListOptions) (*ListResult[Dataset], error)
	ListAllDatasets(ctx context.Context, opts ListOptions) (*ListResult[Dataset], error) // List all datasets across all tenants
	ListDatasetsForAccount(ctx context.Context, accountID string, opts ListOptions) (*ListResult[Dataset], error) // List datasets for a specific account
	
	// Advanced dataset queries
	FindDatasetsByStatus(ctx context.Context, tenantID, status string) ([]*Dataset, error)
	FindDatasetsBySourceType(ctx context.Context, tenantID, sourceType string) ([]*Dataset, error)

	// API Key operations
	CreateAPIKey(ctx context.Context, apiKey *APIKey) error
	GetAPIKey(ctx context.Context, id string) (*APIKey, error)
	GetAPIKeyByHash(ctx context.Context, keyHash string) (*APIKey, error)
	UpdateAPIKey(ctx context.Context, apiKey *APIKey) error
	DeleteAPIKey(ctx context.Context, id string) error
	ListAPIKeys(ctx context.Context, accountID string, opts ListOptions) (*ListResult[APIKey], error)
	UpdateAPIKeyLastUsed(ctx context.Context, id string) error

	// Audit Log operations
	ListAuditLogs(ctx context.Context, filter AuditLogFilter) (*ListResult[AuditLog], error)
	GetAuditLog(ctx context.Context, id int64) (*AuditLog, error)

	// Proxy Instance Configuration operations
	UpsertProxyConfig(ctx context.Context, config *ProxyInstanceConfig) error
	GetProxyConfig(ctx context.Context, instanceID, tenantID string) (*ProxyInstanceConfig, error)
	ListProxyConfigs(ctx context.Context, tenantID string) ([]*ProxyInstanceConfig, error)
	ListAllProxyConfigs(ctx context.Context) ([]*ProxyInstanceConfig, error)
	ListProxyConfigsForAccount(ctx context.Context, accountID string) ([]*ProxyInstanceConfig, error) // List proxy configs for a specific account
	DeleteProxyConfig(ctx context.Context, instanceID, tenantID string) error
	MarkProxyConfigApplied(ctx context.Context, instanceID, tenantID string, configVersion int) error
	GetProxyConfigHistory(ctx context.Context, instanceID, tenantID string, limit int) ([]*ProxyConfigHistory, error)

	// Dataset Metrics operations
	RecordDatasetMetric(ctx context.Context, metric *DatasetMetric) error
	QueryDatasetMetrics(ctx context.Context, filter MetricsQueryFilter) ([]*DatasetMetric, error)
	GetAggregatedMetrics(ctx context.Context, filter MetricsQueryFilter) ([]*ComponentMetrics, error)
	CleanupOldMetrics(ctx context.Context, olderThan time.Time) (int64, error)

	// Transformation Job operations
	CreateTransformationJob(ctx context.Context, job *TransformationJob) error
	GetTransformationJob(ctx context.Context, jobID string) (*TransformationJob, error)
	ListTransformationJobs(ctx context.Context, tenantID, datasetID string) ([]*TransformationJob, error)

	// Piper File Lock operations
	AcquireFileLock(ctx context.Context, lock *PiperFileLock) error
	ReleaseFileLock(ctx context.Context, tenantID, datasetID, fileKey, lockedBy string) error
	CheckFileLock(ctx context.Context, tenantID, datasetID, fileKey string) (*PiperFileLock, error)
	CleanupExpiredFileLocks(ctx context.Context) (int64, error)
	CleanupStaleFileLocks(ctx context.Context, thresholdMinutes int) (int64, error)
	CleanupInstanceFileLocks(ctx context.Context, instanceID string) (int64, error)

	// Piper Job Record operations
	CreatePiperJob(ctx context.Context, job *PiperJobRecord) error
	UpdatePiperJobStatus(ctx context.Context, jobID, status, processorID, outputFile, errorMessage string, recordsProcessed int64) error
	GetPiperJob(ctx context.Context, jobID string) (*PiperJobRecord, error)
	GetPiperJobsByStatus(ctx context.Context, status string) ([]*PiperJobRecord, error)
	GetPiperJobsForTenant(ctx context.Context, tenantID string) ([]*PiperJobRecord, error)
	CleanupOldPiperJobs(ctx context.Context, olderThanDays int) (int64, error)

	// Piper Pipeline Configuration Cache operations
	CachePipelineConfiguration(ctx context.Context, cache *PiperPipelineCache) error
	GetCachedPipelineConfiguration(ctx context.Context, tenantID, datasetID string) (*PiperPipelineCache, error)
	InvalidatePipelineConfiguration(ctx context.Context, tenantID, datasetID string) error
	ListCachedPipelines(ctx context.Context) ([]*PiperPipelineCache, error)
	CleanupExpiredPipelineCache(ctx context.Context) (int64, error)

	// Piper Tenant Cache operations
	CacheTenant(ctx context.Context, cache *PiperTenantCache) error
	GetCachedTenants(ctx context.Context) ([]*PiperTenantCache, error)
	InvalidateTenantCache(ctx context.Context) error
	CleanupExpiredTenantCache(ctx context.Context) (int64, error)

	// Packer Tenant Lock operations
	AcquireTenantLock(ctx context.Context, lock *PackerTenantLock) error
	ReleaseTenantLock(ctx context.Context, tenantID, lockedBy string) error
	UpdateTenantLockHeartbeat(ctx context.Context, tenantID, lockedBy string) error
	CheckTenantLock(ctx context.Context, tenantID string) (*PackerTenantLock, error)
	CleanupExpiredTenantLocks(ctx context.Context) (int64, error)
	ClearAllTenantLocks(ctx context.Context) (int64, error)
	CleanupStaleTenantLocks(ctx context.Context, thresholdMinutes int) (int64, error)
	CleanupInstanceTenantLocks(ctx context.Context, instanceID string) (int64, error)

	// Packer Parquet Metadata operations
	UpsertParquetFileMetadata(ctx context.Context, metadata *PackerParquetFileMetadata) error
	GetParquetFileMetadataByPartition(ctx context.Context, tenantID, datasetID, partitionPath string) ([]*PackerParquetFileMetadata, error)
	GetAllParquetFileMetadata(ctx context.Context, tenantID, datasetID string) ([]*PackerParquetFileMetadata, error)
	DeleteParquetFileMetadata(ctx context.Context, tenantID, datasetID, filePath string) error
	CleanupOrphanedParquetMetadata(ctx context.Context, tenantID, datasetID string, existingFiles []string) (int64, error)
	CleanupExpiredParquetMetadata(ctx context.Context) (int64, int64, error)

	// Packer Metadata Generation Status operations
	UpdateMetadataGenerationStatus(ctx context.Context, status *PackerMetadataGenerationStatus) error
	GetMetadataGenerationStatus(ctx context.Context, tenantID, datasetID, partitionPath string) (*PackerMetadataGenerationStatus, error)

	// Packer Metadata Summary operations
	GetParquetMetadataSummary(ctx context.Context, tenantID, datasetID, partitionPath string) (*PackerParquetMetadataSummary, error)

	// Piper Transformation Job operations
	CreatePiperTransformationJob(ctx context.Context, job *PiperTransformationJob) error
	ClaimPiperTransformationJob(ctx context.Context, processorID string, jobTypes []PiperTransformationJobType) (*PiperTransformationJob, error)
	UpdatePiperTransformationJob(ctx context.Context, job *PiperTransformationJob) error
	GetPiperTransformationJob(ctx context.Context, jobID string) (*PiperTransformationJob, error)
	ListPendingPiperTransformationJobs(ctx context.Context, limit int) ([]*PiperTransformationJob, error)
	CleanupExpiredPiperTransformationJobs(ctx context.Context) (int, error)

	// Dataset Schema and Sample operations
	UpsertDatasetSchema(ctx context.Context, tenantID, datasetID, schemaType string, schema interface{}) error
	GetDatasetSchema(ctx context.Context, tenantID, datasetID, schemaType string) ([]byte, error)
	UpsertDatasetSamples(ctx context.Context, tenantID, datasetID, sampleType string, samples []DatasetSample, keepCount int) error
	GetDatasetSamples(ctx context.Context, tenantID, datasetID, sampleType string, limit int) ([]DatasetSample, error)

	// Utility operations
	HealthCheck(ctx context.Context) error
	Migrate(ctx context.Context) error
	Close() error

	// Migration operations
	GetMigrator() Migrator
}

// Config represents PostgreSQL database configuration
type Config struct {
	Type           string   `yaml:"type" json:"type"`                       // postgresql (only supported type now)
	URI            string   `yaml:"uri" json:"uri"`                         // PostgreSQL connection URI
	Database       string   `yaml:"database" json:"database"`               // Database name
	TimeoutSeconds int      `yaml:"timeout_seconds" json:"timeout_seconds"` // Connection timeout
	SSLMode        string   `yaml:"ssl_mode" json:"ssl_mode"`               // PostgreSQL SSL mode (disable, require, verify-ca, verify-full)
	S3             S3Config `yaml:"s3" json:"s3"`                           // S3 configuration for dataset cleanup
}

// S3Config holds S3 credentials and settings for dataset cleanup operations
type S3Config struct {
	Enabled      bool   `yaml:"enabled" json:"enabled"`             // Enable S3 cleanup operations
	IntakeBucket string `yaml:"intake_bucket" json:"intake_bucket"` // Intake bucket name
	PiperBucket  string `yaml:"piper_bucket" json:"piper_bucket"`   // Piper bucket name
	Region       string `yaml:"region" json:"region"`               // AWS region (e.g., us-east-1)
	Endpoint     string `yaml:"endpoint" json:"endpoint"`           // Custom endpoint for MinIO/LocalStack
	AccessKey    string `yaml:"access_key" json:"access_key"`       // S3 access key
	SecretKey    string `yaml:"secret_key" json:"secret_key"`       // S3 secret key
	UseSSL       bool   `yaml:"use_ssl" json:"use_ssl"`             // Use SSL/TLS for S3 connections
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