package api

import (
	"context"
	"fmt"
	"time"

	"github.com/n0needt0/bytefreezer-control/storage"
	"github.com/n0needt0/go-goodies/log"
	"github.com/swaggest/usecase"
	usecaseStatus "github.com/swaggest/usecase/status"
)

// Transformation API request/response types

type FilterConfig struct {
	Type    string                 `json:"type"`
	Config  map[string]interface{} `json:"config"`
	Enabled bool                   `json:"enabled"`
}

type TransformationSample struct {
	LineNumber int                    `json:"line_number"`
	RawData    string                 `json:"raw_data"`
	ParsedData map[string]interface{} `json:"parsed_data"`
}

type TransformationJobResponse struct {
	JobID     string    `json:"job_id"`
	TenantID  string    `json:"tenant_id"`
	DatasetID string    `json:"dataset_id"`
	JobType   string    `json:"job_type"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	Message   string    `json:"message,omitempty"`
}

type TransformationJobStatusResponse struct {
	JobID       string      `json:"job_id"`
	TenantID    string      `json:"tenant_id"`
	DatasetID   string      `json:"dataset_id"`
	JobType     string      `json:"job_type"`
	Status      string      `json:"status"`
	ProcessorID string      `json:"processor_id,omitempty"`
	Result      interface{} `json:"result,omitempty"`
	ErrorMsg    string      `json:"error_message,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
	StartedAt   *time.Time  `json:"started_at,omitempty"`
	CompletedAt *time.Time  `json:"completed_at,omitempty"`
}

// CreateTransformationTest creates a test transformation job
func (api *API) CreateTransformationTest() usecase.Interactor {
	type Input struct {
		TenantID  string                 `path:"tenantId" required:"true"`
		DatasetID string                 `path:"datasetId" required:"true"`
		Filters   []FilterConfig         `json:"filters" required:"true"`
		Samples   []TransformationSample `json:"samples" required:"true"`
	}

	type Output struct {
		Response TransformationJobResponse `json:"response"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input Input, output *Output) error {
		// Create transformation job
		job := &storage.TransformationJob{
			TenantID:  input.TenantID,
			DatasetID: input.DatasetID,
			JobType:   storage.TransformationJobTypeTest,
			Status:    storage.JobStatusPending,
			Request: map[string]interface{}{
				"tenant_id":  input.TenantID,
				"dataset_id": input.DatasetID,
				"filters":    input.Filters,
				"samples":    input.Samples,
			},
			TTL: time.Now().Add(1 * time.Hour), // 1 hour TTL
		}

		if err := api.Services.Storage.CreateTransformationJob(ctx, job); err != nil {
			log.Errorf("Failed to create transformation test job: %v", err)
			return fmt.Errorf("failed to create transformation job: %w", err)
		}

		log.Infof("Created transformation test job %s for %s/%s", job.JobID, input.TenantID, input.DatasetID)

		output.Response = TransformationJobResponse{
			JobID:     job.JobID,
			TenantID:  job.TenantID,
			DatasetID: job.DatasetID,
			JobType:   string(job.JobType),
			Status:    string(job.Status),
			CreatedAt: job.CreatedAt,
			Message:   "Transformation test job created successfully",
		}

		return nil
	})

	u.SetExpectedErrors(usecaseStatus.InvalidArgument)
	return u
}

// CreateTransformationValidate creates a validate transformation job
func (api *API) CreateTransformationValidate() usecase.Interactor {
	type Input struct {
		TenantID  string         `path:"tenantId" required:"true"`
		DatasetID string         `path:"datasetId" required:"true"`
		Filters   []FilterConfig `json:"filters" required:"true"`
		Count     int            `json:"count" required:"true"`
	}

	type Output struct {
		Response TransformationJobResponse `json:"response"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input Input, output *Output) error {
		// Create transformation job
		job := &storage.TransformationJob{
			TenantID:  input.TenantID,
			DatasetID: input.DatasetID,
			JobType:   storage.TransformationJobTypeValidate,
			Status:    storage.JobStatusPending,
			Request: map[string]interface{}{
				"tenant_id":  input.TenantID,
				"dataset_id": input.DatasetID,
				"filters":    input.Filters,
				"count":      input.Count,
			},
			TTL: time.Now().Add(1 * time.Hour), // 1 hour TTL
		}

		if err := api.Services.Storage.CreateTransformationJob(ctx, job); err != nil {
			log.Errorf("Failed to create transformation validate job: %v", err)
			return fmt.Errorf("failed to create transformation job: %w", err)
		}

		log.Infof("Created transformation validate job %s for %s/%s", job.JobID, input.TenantID, input.DatasetID)

		output.Response = TransformationJobResponse{
			JobID:     job.JobID,
			TenantID:  job.TenantID,
			DatasetID: job.DatasetID,
			JobType:   string(job.JobType),
			Status:    string(job.Status),
			CreatedAt: job.CreatedAt,
			Message:   "Transformation validate job created successfully",
		}

		return nil
	})

	u.SetExpectedErrors(usecaseStatus.InvalidArgument)
	return u
}

// CreateTransformationActivate creates an activate transformation job
func (api *API) CreateTransformationActivate() usecase.Interactor {
	type Input struct {
		TenantID  string         `path:"tenantId" required:"true"`
		DatasetID string         `path:"datasetId" required:"true"`
		Filters   []FilterConfig `json:"filters" required:"true"`
		Enabled   bool           `json:"enabled"`
	}

	type Output struct {
		Response TransformationJobResponse `json:"response"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input Input, output *Output) error {
		// Create transformation job
		job := &storage.TransformationJob{
			TenantID:  input.TenantID,
			DatasetID: input.DatasetID,
			JobType:   storage.TransformationJobTypeActivate,
			Status:    storage.JobStatusPending,
			Request: map[string]interface{}{
				"tenant_id":  input.TenantID,
				"dataset_id": input.DatasetID,
				"filters":    input.Filters,
				"enabled":    input.Enabled,
			},
			TTL: time.Now().Add(1 * time.Hour), // 1 hour TTL
		}

		if err := api.Services.Storage.CreateTransformationJob(ctx, job); err != nil {
			log.Errorf("Failed to create transformation activate job: %v", err)
			return fmt.Errorf("failed to create transformation job: %w", err)
		}

		log.Infof("Created transformation activate job %s for %s/%s", job.JobID, input.TenantID, input.DatasetID)

		output.Response = TransformationJobResponse{
			JobID:     job.JobID,
			TenantID:  job.TenantID,
			DatasetID: job.DatasetID,
			JobType:   string(job.JobType),
			Status:    string(job.Status),
			CreatedAt: job.CreatedAt,
			Message:   "Transformation activate job created successfully",
		}

		return nil
	})

	u.SetExpectedErrors(usecaseStatus.InvalidArgument)
	return u
}

// GetTransformationJobStatus retrieves transformation job status and results
func (api *API) GetTransformationJobStatus() usecase.Interactor {
	type Input struct {
		JobID string `path:"jobId" required:"true"`
	}

	type Output struct {
		Response TransformationJobStatusResponse `json:"response"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input Input, output *Output) error {
		job, err := api.Services.Storage.GetTransformationJob(ctx, input.JobID)
		if err != nil {
			log.Errorf("Failed to get transformation job %s: %v", input.JobID, err)
			return fmt.Errorf("failed to get transformation job: %w", err)
		}

		if job == nil {
			return usecaseStatus.NotFound
		}

		output.Response = TransformationJobStatusResponse{
			JobID:       job.JobID,
			TenantID:    job.TenantID,
			DatasetID:   job.DatasetID,
			JobType:     string(job.JobType),
			Status:      string(job.Status),
			ProcessorID: job.ProcessorID,
			Result:      job.Result,
			ErrorMsg:    job.ErrorMsg,
			CreatedAt:   job.CreatedAt,
			UpdatedAt:   job.UpdatedAt,
			StartedAt:   job.StartedAt,
			CompletedAt: job.CompletedAt,
		}

		return nil
	})

	u.SetExpectedErrors(usecaseStatus.NotFound)
	return u
}

// ListTransformationJobs retrieves all transformation jobs for a dataset
func (api *API) ListTransformationJobs() usecase.Interactor {
	type Input struct {
		TenantID  string `path:"tenantId" required:"true"`
		DatasetID string `path:"datasetId" required:"true"`
	}

	type Output struct {
		Jobs []TransformationJobStatusResponse `json:"jobs"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input Input, output *Output) error {
		jobs, err := api.Services.Storage.ListTransformationJobs(ctx, input.TenantID, input.DatasetID)
		if err != nil {
			log.Errorf("Failed to list transformation jobs for %s/%s: %v", input.TenantID, input.DatasetID, err)
			return fmt.Errorf("failed to list transformation jobs: %w", err)
		}

		// Convert to response format
		output.Jobs = make([]TransformationJobStatusResponse, 0, len(jobs))
		for _, job := range jobs {
			output.Jobs = append(output.Jobs, TransformationJobStatusResponse{
				JobID:       job.JobID,
				TenantID:    job.TenantID,
				DatasetID:   job.DatasetID,
				JobType:     string(job.JobType),
				Status:      string(job.Status),
				ProcessorID: job.ProcessorID,
				Result:      job.Result,
				ErrorMsg:    job.ErrorMsg,
				CreatedAt:   job.CreatedAt,
				UpdatedAt:   job.UpdatedAt,
				StartedAt:   job.StartedAt,
				CompletedAt: job.CompletedAt,
			})
		}

		return nil
	})

	return u
}
