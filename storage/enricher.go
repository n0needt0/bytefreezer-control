package storage

import (
	"time"

	"github.com/google/uuid"
)

// Enricher limits
const (
	MaxEnricherColumns      = 20
	MaxEnricherIndexColumns = 3
	MaxColumnNameLength     = 64
	MaxCellValueLength      = 1024
	MaxEnricherRows         = 1000000
	MaxEnricherFileSize     = 100 * 1024 * 1024 // 100MB
	MaxEnricherNameLength   = 128
)

// EnricherStatus represents the status of an enricher
type EnricherStatus string

const (
	EnricherStatusPending    EnricherStatus = "pending"
	EnricherStatusProcessing EnricherStatus = "processing"
	EnricherStatusActive     EnricherStatus = "active"
	EnricherStatusError      EnricherStatus = "error"
	EnricherStatusDisabled   EnricherStatus = "disabled"
)

// Enricher represents a customer-uploaded enrichment lookup table
type Enricher struct {
	ID          uuid.UUID      `json:"id" db:"id"`
	TenantID    uuid.UUID      `json:"tenant_id" db:"tenant_id"`
	Name        string         `json:"name" db:"name"`
	Description string         `json:"description" db:"description"`

	// Column configuration
	Columns      []string `json:"columns" db:"columns"`       // Array of column names (max 20)
	IndexColumns []string `json:"index_columns" db:"index_columns"` // Array of 1-3 column names for lookup

	// Data info
	RowCount int    `json:"row_count" db:"row_count"`
	FileSize int64  `json:"file_size" db:"file_size"`
	FileData []byte `json:"-" db:"file_data"` // Binary enrichment data (JSON lines format, max 100MB)

	// Status
	Status       EnricherStatus `json:"status" db:"status"`
	ErrorMessage string         `json:"error_message,omitempty" db:"error_message"`

	// Metadata
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	CreatedBy *uuid.UUID `json:"created_by,omitempty" db:"created_by"`
}

// EnricherVersion represents a version/upload of enricher data
type EnricherVersion struct {
	ID          uuid.UUID      `json:"id" db:"id"`
	EnricherID  uuid.UUID      `json:"enricher_id" db:"enricher_id"`
	Version     int            `json:"version" db:"version"`

	// Upload info
	OriginalFilename string `json:"original_filename" db:"original_filename"`
	FileSize         int64  `json:"file_size" db:"file_size"`
	RowCount         int    `json:"row_count" db:"row_count"`

	// Processing info
	Status       EnricherStatus `json:"status" db:"status"`
	ErrorMessage string         `json:"error_message,omitempty" db:"error_message"`
	ProcessedAt  *time.Time     `json:"processed_at,omitempty" db:"processed_at"`

	// Metadata
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	UploadedBy *uuid.UUID `json:"uploaded_by,omitempty" db:"uploaded_by"`
}

// EnricherAuditLog represents an audit log entry for enricher operations
type EnricherAuditLog struct {
	ID         uuid.UUID              `json:"id" db:"id"`
	TenantID   uuid.UUID              `json:"tenant_id" db:"tenant_id"`
	EnricherID *uuid.UUID             `json:"enricher_id,omitempty" db:"enricher_id"`

	// Action info
	Action     string                 `json:"action" db:"action"` // create, upload, update, delete, enable, disable
	ActorID    *uuid.UUID             `json:"actor_id,omitempty" db:"actor_id"`
	ActorEmail string                 `json:"actor_email,omitempty" db:"actor_email"`

	// Details
	Details    map[string]interface{} `json:"details,omitempty" db:"details"`

	// Metadata
	CreatedAt  time.Time              `json:"created_at" db:"created_at"`
}

// CreateEnricherRequest represents a request to create a new enricher
type CreateEnricherRequest struct {
	Name         string   `json:"name" binding:"required,max=128"`
	Description  string   `json:"description" binding:"max=1000"`
	Columns      []string `json:"columns" binding:"required,min=1,max=20"`
	IndexColumns []string `json:"index_columns" binding:"required,min=1,max=3"`
}

// UpdateEnricherRequest represents a request to update an enricher
type UpdateEnricherRequest struct {
	Name        *string `json:"name" binding:"omitempty,max=128"`
	Description *string `json:"description" binding:"omitempty,max=1000"`
}

// EnricherUploadRequest represents a request to upload data to an enricher
type EnricherUploadRequest struct {
	// File will be uploaded via multipart form
	// CSV format expected with headers matching Columns
}

// EnricherListResponse represents a paginated list of enrichers
type EnricherListResponse struct {
	Enrichers  []Enricher `json:"enrichers"`
	Total      int        `json:"total"`
	Page       int        `json:"page"`
	PageSize   int        `json:"page_size"`
	TotalPages int        `json:"total_pages"`
}

// EnricherStatsResponse represents statistics about an enricher
type EnricherStatsResponse struct {
	EnricherID    uuid.UUID `json:"enricher_id"`
	RowCount      int       `json:"row_count"`
	FileSize      int64     `json:"file_size"`
	VersionCount  int       `json:"version_count"`
	LastUploadAt  *time.Time `json:"last_upload_at,omitempty"`
	UsageCount    int64      `json:"usage_count"` // Number of times used in transformations
}

// Validate validates the create enricher request
func (r *CreateEnricherRequest) Validate() error {
	if len(r.Name) == 0 || len(r.Name) > MaxEnricherNameLength {
		return ErrInvalidInput
	}

	if len(r.Columns) == 0 || len(r.Columns) > MaxEnricherColumns {
		return ErrInvalidInput
	}

	if len(r.IndexColumns) == 0 || len(r.IndexColumns) > MaxEnricherIndexColumns {
		return ErrInvalidInput
	}

	// Validate column names
	for _, col := range r.Columns {
		if len(col) == 0 || len(col) > MaxColumnNameLength {
			return ErrInvalidInput
		}
	}

	// Validate index columns are subset of columns
	colMap := make(map[string]bool)
	for _, col := range r.Columns {
		colMap[col] = true
	}

	for _, idxCol := range r.IndexColumns {
		if !colMap[idxCol] {
			return ErrInvalidInput
		}
	}

	return nil
}
