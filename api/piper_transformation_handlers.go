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

// CreatePiperTransformationJob creates a new transformation job
func (api *API) CreatePiperTransformationJob() usecase.Interactor {
	type createTransformationJobInput struct {
		JobID       string                              `json:"job_id" required:"true"`
		TenantID    string                              `json:"tenant_id" required:"true"`
		DatasetID   string                              `json:"dataset_id" required:"true"`
		JobType     storage.PiperTransformationJobType  `json:"job_type" required:"true"`
		Status      storage.PiperJobStatus              `json:"status" required:"true"`
		ProcessorID string                              `json:"processor_id,omitempty"`
		Priority    int                                 `json:"priority" default:"0"`
		Request     map[string]interface{}              `json:"request,omitempty"`
		TTLHours    int                                 `json:"ttl_hours" default:"24"`
	}

	type createTransformationJobOutput struct {
		JobID   string `json:"job_id"`
		Success bool   `json:"success"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input createTransformationJobInput, output *createTransformationJobOutput) error {
		now := time.Now()
		ttl := now.Add(time.Duration(input.TTLHours) * time.Hour)

		// Set priority to 10 for test jobs (instant feedback with 10 samples)
		priority := input.Priority
		if input.JobType == storage.PiperTransformationJobTypeTest {
			priority = 10
		}

		job := &storage.PiperTransformationJob{
			JobID:       input.JobID,
			TenantID:    input.TenantID,
			DatasetID:   input.DatasetID,
			JobType:     input.JobType,
			Status:      input.Status,
			ProcessorID: input.ProcessorID,
			Priority:    priority,
			Request:     input.Request,
			CreatedAt:   now,
			UpdatedAt:   now,
			TTL:         ttl,
		}

		err := api.Services.Storage.CreatePiperTransformationJob(ctx, job)
		if err != nil {
			log.Errorf("Failed to create piper transformation job: %v", err)
			return usecaseStatus.Wrap(fmt.Errorf("failed to create transformation job: %w", err), usecaseStatus.Internal)
		}

		output.JobID = job.JobID
		output.Success = true

		log.Infof("Created piper transformation job %s for %s/%s (type: %s)", job.JobID, input.TenantID, input.DatasetID, input.JobType)
		return nil
	})

	u.SetTitle("Create Piper Transformation Job")
	u.SetDescription("Creates a new transformation job in the piper queue")
	u.SetTags("piper", "transformation-jobs")

	return u
}

// ClaimPiperTransformationJob claims a pending transformation job
func (api *API) ClaimPiperTransformationJob() usecase.Interactor {
	type claimTransformationJobInput struct {
		ProcessorID string                               `json:"processor_id" required:"true"`
		JobTypes    []storage.PiperTransformationJobType `json:"job_types" required:"true"`
	}

	type claimTransformationJobOutput struct {
		Job *storage.PiperTransformationJob `json:"job,omitempty"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input claimTransformationJobInput, output *claimTransformationJobOutput) error {
		job, err := api.Services.Storage.ClaimPiperTransformationJob(ctx, input.ProcessorID, input.JobTypes)
		if err != nil {
			log.Errorf("Failed to claim piper transformation job: %v", err)
			return usecaseStatus.Wrap(fmt.Errorf("failed to claim transformation job: %w", err), usecaseStatus.Internal)
		}

		if job == nil {
			// No jobs available - this is not an error
			output.Job = nil
			return nil
		}

		output.Job = job
		log.Infof("Processor %s claimed transformation job %s", input.ProcessorID, job.JobID)
		return nil
	})

	u.SetTitle("Claim Piper Transformation Job")
	u.SetDescription("Claims a pending transformation job for processing")
	u.SetTags("piper", "transformation-jobs")

	return u
}

// UpdatePiperTransformationJob updates an existing transformation job
func (api *API) UpdatePiperTransformationJob() usecase.Interactor {
	type updateTransformationJobInput struct {
		JobID       string                             `path:"job_id" required:"true"`
		Status      storage.PiperJobStatus             `json:"status" required:"true"`
		ProcessorID string                             `json:"processor_id,omitempty"`
		Result      map[string]interface{}             `json:"result,omitempty"`
		ErrorMsg    string                             `json:"error_message,omitempty"`
	}

	type updateTransformationJobOutput struct {
		Success bool `json:"success"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input updateTransformationJobInput, output *updateTransformationJobOutput) error {
		// Get existing job
		job, err := api.Services.Storage.GetPiperTransformationJob(ctx, input.JobID)
		if err != nil {
			log.Errorf("Failed to get piper transformation job %s: %v", input.JobID, err)
			return usecaseStatus.Wrap(fmt.Errorf("failed to get transformation job: %w", err), usecaseStatus.NotFound)
		}

		// Update fields
		job.Status = input.Status
		job.UpdatedAt = time.Now()

		if input.ProcessorID != "" {
			job.ProcessorID = input.ProcessorID
		}

		if input.Result != nil {
			job.Result = input.Result
		}

		if input.ErrorMsg != "" {
			job.ErrorMsg = input.ErrorMsg
		}

		// Set timestamps based on status
		now := time.Now()
		switch input.Status {
		case storage.PiperJobStatusRunning:
			if job.StartedAt == nil {
				job.StartedAt = &now
			}
		case storage.PiperJobStatusCompleted, storage.PiperJobStatusFailed:
			if job.CompletedAt == nil {
				job.CompletedAt = &now
			}
		}

		err = api.Services.Storage.UpdatePiperTransformationJob(ctx, job)
		if err != nil {
			log.Errorf("Failed to update piper transformation job %s: %v", input.JobID, err)
			return usecaseStatus.Wrap(fmt.Errorf("failed to update transformation job: %w", err), usecaseStatus.Internal)
		}

		output.Success = true
		log.Infof("Updated piper transformation job %s to status %s", input.JobID, input.Status)
		return nil
	})

	u.SetTitle("Update Piper Transformation Job")
	u.SetDescription("Updates a transformation job's status and results")
	u.SetTags("piper", "transformation-jobs")

	return u
}

// GetPiperTransformationJob retrieves a specific transformation job
func (api *API) GetPiperTransformationJob() usecase.Interactor {
	type getTransformationJobInput struct {
		JobID string `path:"job_id" required:"true"`
	}

	type getTransformationJobOutput struct {
		Job *storage.PiperTransformationJob `json:"job"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input getTransformationJobInput, output *getTransformationJobOutput) error {
		job, err := api.Services.Storage.GetPiperTransformationJob(ctx, input.JobID)
		if err != nil {
			log.Errorf("Failed to get piper transformation job %s: %v", input.JobID, err)
			return usecaseStatus.Wrap(fmt.Errorf("failed to get transformation job: %w", err), usecaseStatus.NotFound)
		}

		output.Job = job
		return nil
	})

	u.SetTitle("Get Piper Transformation Job")
	u.SetDescription("Retrieves a transformation job by ID")
	u.SetTags("piper", "transformation-jobs")

	return u
}

// ListPendingPiperTransformationJobs lists pending transformation jobs
func (api *API) ListPendingPiperTransformationJobs() usecase.Interactor {
	type listPendingTransformationJobsInput struct {
		Limit int `query:"limit" default:"100"`
	}

	type listPendingTransformationJobsOutput struct {
		Jobs  []*storage.PiperTransformationJob `json:"jobs"`
		Count int                               `json:"count"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input listPendingTransformationJobsInput, output *listPendingTransformationJobsOutput) error {
		jobs, err := api.Services.Storage.ListPendingPiperTransformationJobs(ctx, input.Limit)
		if err != nil {
			log.Errorf("Failed to list pending piper transformation jobs: %v", err)
			return usecaseStatus.Wrap(fmt.Errorf("failed to list pending transformation jobs: %w", err), usecaseStatus.Internal)
		}

		output.Jobs = jobs
		output.Count = len(jobs)
		return nil
	})

	u.SetTitle("List Pending Piper Transformation Jobs")
	u.SetDescription("Lists all pending transformation jobs")
	u.SetTags("piper", "transformation-jobs")

	return u
}

// CleanupExpiredPiperTransformationJobs removes expired transformation jobs
func (api *API) CleanupExpiredPiperTransformationJobs() usecase.Interactor {
	type cleanupExpiredTransformationJobsOutput struct {
		DeletedCount int  `json:"deleted_count"`
		Success      bool `json:"success"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input struct{}, output *cleanupExpiredTransformationJobsOutput) error {
		count, err := api.Services.Storage.CleanupExpiredPiperTransformationJobs(ctx)
		if err != nil {
			log.Warnf("Failed to cleanup expired piper transformation jobs: %v", err)
			// Don't fail the request, just log the warning
		}

		output.DeletedCount = count
		output.Success = true
		log.Infof("Cleaned up %d expired piper transformation jobs", count)
		return nil
	})

	u.SetTitle("Cleanup Expired Piper Transformation Jobs")
	u.SetDescription("Removes transformation jobs that have exceeded their TTL")
	u.SetTags("piper", "transformation-jobs")

	return u
}
