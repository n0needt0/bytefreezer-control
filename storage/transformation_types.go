package storage

import "time"

// TransformationJobType represents the type of transformation operation
type TransformationJobType string

const (
	TransformationJobTypeTest     TransformationJobType = "test"     // Test filters on sample data
	TransformationJobTypeValidate TransformationJobType = "validate" // Validate on fresh data
	TransformationJobTypeActivate TransformationJobType = "activate" // Activate/deactivate transformation
)

// JobStatus represents the status of a processing job
type JobStatus string

const (
	JobStatusQueued     JobStatus = "queued"
	JobStatusPending    JobStatus = "pending"
	JobStatusProcessing JobStatus = "processing"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
	JobStatusRetrying   JobStatus = "retrying"
	JobStatusCancelled  JobStatus = "cancelled"
)

// TransformationJob represents an async transformation job
type TransformationJob struct {
	JobID       string                `json:"job_id"`
	TenantID    string                `json:"tenant_id"`
	DatasetID   string                `json:"dataset_id"`
	JobType     TransformationJobType `json:"job_type"`
	Status      JobStatus             `json:"status"`
	ProcessorID string                `json:"processor_id,omitempty"` // Instance that claimed the job
	Request     interface{}           `json:"request"`                // Job-specific request data
	Result      interface{}           `json:"result,omitempty"`       // Job-specific result data
	ErrorMsg    string                `json:"error_message,omitempty"`
	CreatedAt   time.Time             `json:"created_at"`
	UpdatedAt   time.Time             `json:"updated_at"`
	StartedAt   *time.Time            `json:"started_at,omitempty"`
	CompletedAt *time.Time            `json:"completed_at,omitempty"`
	TTL         time.Time             `json:"ttl"` // When this job expires
}
