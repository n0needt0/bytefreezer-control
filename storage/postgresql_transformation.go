package storage

import (
	"context"
	"database/sql"
	"github.com/bytedance/sonic"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// CreateTransformationJob creates a new transformation job in the database
func (s *PostgreSQLStorage) CreateTransformationJob(ctx context.Context, job *TransformationJob) error {
	// Generate job ID if not provided
	if job.JobID == "" {
		job.JobID = uuid.New().String()
	}

	// Set timestamps
	now := time.Now()
	if job.CreatedAt.IsZero() {
		job.CreatedAt = now
	}
	if job.UpdatedAt.IsZero() {
		job.UpdatedAt = now
	}

	// Default status to pending
	if job.Status == "" {
		job.Status = JobStatusPending
	}

	// Default TTL to 24 hours if not set
	if job.TTL.IsZero() {
		job.TTL = now.Add(24 * time.Hour)
	}

	// Marshal request to JSON
	requestJSON, err := sonic.Marshal(job.Request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	query := `
		INSERT INTO transformation_jobs (
			job_id, tenant_id, dataset_id, job_type, status,
			processor_id, request, created_at, updated_at, ttl
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err = s.db.ExecContext(ctx, query,
		job.JobID,
		job.TenantID,
		job.DatasetID,
		job.JobType,
		job.Status,
		job.ProcessorID,
		requestJSON,
		job.CreatedAt,
		job.UpdatedAt,
		job.TTL,
	)

	if err != nil {
		return fmt.Errorf("failed to create transformation job: %w", err)
	}

	return nil
}

// GetTransformationJob retrieves a transformation job by ID
func (s *PostgreSQLStorage) GetTransformationJob(ctx context.Context, jobID string) (*TransformationJob, error) {
	query := `
		SELECT job_id, tenant_id, dataset_id, job_type, status,
		       processor_id, request, result, error_message,
		       created_at, updated_at, started_at, completed_at, ttl
		FROM transformation_jobs
		WHERE job_id = $1
	`

	var job TransformationJob
	var processorID sql.NullString
	var requestJSON, resultJSON []byte
	var errorMsg sql.NullString
	var startedAt, completedAt sql.NullTime

	err := s.db.QueryRowContext(ctx, query, jobID).Scan(
		&job.JobID,
		&job.TenantID,
		&job.DatasetID,
		&job.JobType,
		&job.Status,
		&processorID,
		&requestJSON,
		&resultJSON,
		&errorMsg,
		&job.CreatedAt,
		&job.UpdatedAt,
		&startedAt,
		&completedAt,
		&job.TTL,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get transformation job: %w", err)
	}

	// Unmarshal request and result
	if len(requestJSON) > 0 {
		if err := sonic.Unmarshal(requestJSON, &job.Request); err != nil {
			return nil, fmt.Errorf("failed to unmarshal request: %w", err)
		}
	}

	if len(resultJSON) > 0 {
		if err := sonic.Unmarshal(resultJSON, &job.Result); err != nil {
			return nil, fmt.Errorf("failed to unmarshal result: %w", err)
		}
	}

	// Handle nullable fields
	if processorID.Valid {
		job.ProcessorID = processorID.String
	}
	if errorMsg.Valid {
		job.ErrorMsg = errorMsg.String
	}
	if startedAt.Valid {
		t := startedAt.Time
		job.StartedAt = &t
	}
	if completedAt.Valid {
		t := completedAt.Time
		job.CompletedAt = &t
	}

	return &job, nil
}

// ListTransformationJobs retrieves all transformation jobs for a tenant/dataset
func (s *PostgreSQLStorage) ListTransformationJobs(ctx context.Context, tenantID, datasetID string) ([]*TransformationJob, error) {
	query := `
		SELECT job_id, tenant_id, dataset_id, job_type, status,
		       processor_id, request, result, error_message,
		       created_at, updated_at, started_at, completed_at, ttl
		FROM transformation_jobs
		WHERE tenant_id = $1 AND dataset_id = $2
		ORDER BY created_at DESC
		LIMIT 100
	`

	rows, err := s.db.QueryContext(ctx, query, tenantID, datasetID)
	if err != nil {
		return nil, fmt.Errorf("failed to list transformation jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*TransformationJob
	for rows.Next() {
		var job TransformationJob
		var processorID sql.NullString
		var requestJSON, resultJSON []byte
		var errorMsg sql.NullString
		var startedAt, completedAt sql.NullTime

		err := rows.Scan(
			&job.JobID,
			&job.TenantID,
			&job.DatasetID,
			&job.JobType,
			&job.Status,
			&processorID,
			&requestJSON,
			&resultJSON,
			&errorMsg,
			&job.CreatedAt,
			&job.UpdatedAt,
			&startedAt,
			&completedAt,
			&job.TTL,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transformation job: %w", err)
		}

		// Unmarshal request and result
		if len(requestJSON) > 0 {
			if err := sonic.Unmarshal(requestJSON, &job.Request); err != nil {
				return nil, fmt.Errorf("failed to unmarshal request: %w", err)
			}
		}

		if len(resultJSON) > 0 {
			if err := sonic.Unmarshal(resultJSON, &job.Result); err != nil {
				return nil, fmt.Errorf("failed to unmarshal result: %w", err)
			}
		}

		// Handle nullable fields
		if processorID.Valid {
			job.ProcessorID = processorID.String
		}
		if errorMsg.Valid {
			job.ErrorMsg = errorMsg.String
		}
		if startedAt.Valid {
			t := startedAt.Time
			job.StartedAt = &t
		}
		if completedAt.Valid {
			t := completedAt.Time
			job.CompletedAt = &t
		}

		jobs = append(jobs, &job)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transformation jobs: %w", err)
	}

	return jobs, nil
}
