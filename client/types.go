package client

import "time"

// ====================================================================================
// PIPER TYPES
// ====================================================================================

// PiperFileLock represents a file lock in the piper system
type PiperFileLock struct {
	LockID        int64     `json:"lock_id"`
	TenantID      string    `json:"tenant_id"`
	DatasetID     string    `json:"dataset_id"`
	FileKey       string    `json:"file_key"`
	LockedBy      string    `json:"locked_by"`
	LockTimestamp time.Time `json:"lock_timestamp"`
	LastHeartbeat time.Time `json:"last_heartbeat"`
	TTL           time.Time `json:"ttl"`
}

// PiperJobRecord represents a processing job record
type PiperJobRecord struct {
	JobID             string                 `json:"job_id"`
	TenantID          string                 `json:"tenant_id"`
	DatasetID         string                 `json:"dataset_id"`
	Status            string                 `json:"status"`
	SourceFiles       []string               `json:"source_files"`
	ProcessorType     string                 `json:"processor_type"`
	ProcessorID       string                 `json:"processor_id"`
	OutputFile        string                 `json:"output_file"`
	ErrorMessage      string                 `json:"error_message"`
	RecordsProcessed  int64                  `json:"records_processed"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
}

// PiperPipelineConfiguration represents a cached pipeline configuration
type PiperPipelineConfiguration struct {
	ConfigKey     string                 `json:"config_key"`
	TenantID      string                 `json:"tenant_id"`
	DatasetID     string                 `json:"dataset_id"`
	Configuration map[string]interface{} `json:"configuration"`
	CachedAt      time.Time              `json:"cached_at"`
	ExpiresAt     time.Time              `json:"expires_at"`
}

// PiperTenantCache represents a cached tenant
type PiperTenantCache struct {
	TenantID   string                 `json:"tenant_id"`
	TenantData map[string]interface{} `json:"tenant_data"`
	CachedAt   time.Time              `json:"cached_at"`
	ExpiresAt  time.Time              `json:"expires_at"`
}

// ====================================================================================
// PACKER TYPES
// ====================================================================================

// PackerTenantLock represents a tenant lock in the packer system
type PackerTenantLock struct {
	TenantID      string    `json:"tenant_id"`
	LockedBy      string    `json:"locked_by"`
	LockTimestamp time.Time `json:"lock_timestamp"`
	LastHeartbeat time.Time `json:"last_heartbeat"`
	TTL           time.Time `json:"ttl"`
}

// PackerParquetFileMetadata represents metadata for a parquet file
type PackerParquetFileMetadata struct {
	ID              int64                  `json:"id"`
	TenantID        string                 `json:"tenant_id"`
	DatasetID       string                 `json:"dataset_id"`
	FilePath        string                 `json:"file_path"`
	PartitionPath   string                 `json:"partition_path"`
	FileSizeBytes   int64                  `json:"file_size_bytes"`
	RowCount        int64                  `json:"row_count"`
	CreatedAt       time.Time              `json:"created_at"`
	LastModified    time.Time              `json:"last_modified"`
	SchemaJSON      map[string]interface{} `json:"schema_json"`
	ColumnStats     map[string]interface{} `json:"column_stats"`
	FileChecksum    string                 `json:"file_checksum"`
	InstanceID      string                 `json:"instance_id"`
	MetadataVersion int                    `json:"metadata_version"`
	TTL             time.Time              `json:"ttl"`
	InsertedAt      time.Time              `json:"inserted_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// PackerMetadataGenerationStatus represents metadata generation status
type PackerMetadataGenerationStatus struct {
	TenantID          string    `json:"tenant_id"`
	DatasetID         string    `json:"dataset_id"`
	PartitionPath     string    `json:"partition_path"`
	LastGeneratedAt   time.Time `json:"last_generated_at"`
	FileCount         int       `json:"file_count"`
	TotalRows         int64     `json:"total_rows"`
	TotalSizeBytes    int64     `json:"total_size_bytes"`
	NeedsRegeneration bool      `json:"needs_regeneration"`
	CurrentSchemaHash string    `json:"current_schema_hash"`
	SchemaVersion     int       `json:"schema_version"`
	TTL               time.Time `json:"ttl"`
}

// PackerParquetMetadataSummary represents aggregated metadata for a partition
type PackerParquetMetadataSummary struct {
	TenantID            string    `json:"tenant_id"`
	DatasetID           string    `json:"dataset_id"`
	PartitionPath       string    `json:"partition_path"`
	FileCount           int       `json:"file_count"`
	TotalRows           int64     `json:"total_rows"`
	TotalSizeBytes      int64     `json:"total_size_bytes"`
	FirstFileCreated    time.Time `json:"first_file_created"`
	LastFileModified    time.Time `json:"last_file_modified"`
	MetadataLastUpdated time.Time `json:"metadata_last_updated"`
}

// ====================================================================================
// REQUEST/RESPONSE TYPES
// ====================================================================================

// Standard response wrappers
type ItemsResponse struct {
	Items interface{} `json:"items"`
	Total int         `json:"total"`
}

type MessageResponse struct {
	Message string `json:"message"`
}

type CountResponse struct {
	Count int `json:"count"`
}
