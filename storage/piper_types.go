package storage

import "time"

// PiperTransformationJobType represents the type of transformation operation
type PiperTransformationJobType string

const (
	PiperTransformationJobTypeTest     PiperTransformationJobType = "test"     // Test filters on sample data
	PiperTransformationJobTypeValidate PiperTransformationJobType = "validate" // Validate on fresh data
	PiperTransformationJobTypeActivate PiperTransformationJobType = "activate" // Activate/deactivate transformation
)

// PiperJobStatus represents the status of a job
type PiperJobStatus string

const (
	PiperJobStatusPending   PiperJobStatus = "pending"
	PiperJobStatusRunning   PiperJobStatus = "running"
	PiperJobStatusCompleted PiperJobStatus = "completed"
	PiperJobStatusFailed    PiperJobStatus = "failed"
)

// PiperTransformationJob represents an async transformation job managed by control service
type PiperTransformationJob struct {
	JobID       string                     `json:"job_id"`
	TenantID    string                     `json:"tenant_id"`
	DatasetID   string                     `json:"dataset_id"`
	JobType     PiperTransformationJobType `json:"job_type"`
	Status      PiperJobStatus             `json:"status"`
	ProcessorID string                     `json:"processor_id,omitempty"` // Instance that claimed the job
	Priority    int                        `json:"priority"`               // Job priority: 0 (normal), 10 (test with 10 samples)
	Request     map[string]interface{}     `json:"request,omitempty"`      // Job-specific request data
	Result      map[string]interface{}     `json:"result,omitempty"`       // Job-specific result data
	ErrorMsg    string                     `json:"error_message,omitempty"`
	CreatedAt   time.Time                  `json:"created_at"`
	UpdatedAt   time.Time                  `json:"updated_at"`
	StartedAt   *time.Time                 `json:"started_at,omitempty"`
	CompletedAt *time.Time                 `json:"completed_at,omitempty"`
	TTL         time.Time                  `json:"ttl"` // When this job expires
}

// TransformationHistory represents a saved version of transformation configuration
type TransformationHistory struct {
	ID         int64                  `json:"id" db:"id"`
	TenantID   string                 `json:"tenant_id" db:"tenant_id"`
	DatasetID  string                 `json:"dataset_id" db:"dataset_id"`
	Config     map[string]interface{} `json:"config" db:"config"` // Array of filter configurations
	Deployed   bool                   `json:"deployed" db:"deployed"`
	DeployedAt *time.Time             `json:"deployed_at" db:"deployed_at"`
	CreatedAt  time.Time              `json:"created_at" db:"created_at"`
	CreatedBy  string                 `json:"created_by" db:"created_by"`
	Label      string                 `json:"label" db:"label"`
}
