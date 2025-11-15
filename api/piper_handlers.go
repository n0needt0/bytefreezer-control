package api

import (
	"context"
	"fmt"
	"time"

	"github.com/n0needt0/bytefreezer-control/storage"
	"github.com/swaggest/usecase"
	usecaseStatus "github.com/swaggest/usecase/status"
)

// ============================================================================
// PIPER FILE LOCK HANDLERS
// ============================================================================

// AcquireFileLock acquires a file lock for distributed processing
func (api *API) AcquireFileLock() usecase.Interactor {
	type acquireFileLockInput struct {
		TenantID            string `json:"tenant_id" required:"true"`
		DatasetID           string `json:"dataset_id" required:"true"`
		FileKey             string `json:"file_key" required:"true"`
		LockedBy            string `json:"locked_by" required:"true"`
		LockDurationSeconds int    `json:"lock_duration_seconds" required:"true"`
	}

	type acquireFileLockOutput struct {
		LockID  int64     `json:"lock_id"`
		TTL     time.Time `json:"ttl"`
		Success bool      `json:"success"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input acquireFileLockInput, output *acquireFileLockOutput) error {
		ttl := time.Now().Add(time.Duration(input.LockDurationSeconds) * time.Second)

		lock := &storage.PiperFileLock{
			TenantID:  input.TenantID,
			DatasetID: input.DatasetID,
			FileKey:   input.FileKey,
			LockedBy:  input.LockedBy,
			TTL:       ttl,
		}

		err := api.Services.Storage.AcquireFileLock(ctx, lock)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to acquire file lock: %w", err), usecaseStatus.Internal)
		}

		output.LockID = lock.LockID
		output.TTL = lock.TTL
		output.Success = true

		return nil
	})

	u.SetTitle("Acquire File Lock")
	u.SetDescription("Acquires a file lock for distributed piper processing")
	u.SetTags("piper", "locks")

	return u
}

// ReleaseFileLock releases a file lock
func (api *API) ReleaseFileLock() usecase.Interactor {
	type releaseFileLockInput struct {
		TenantID  string `json:"tenant_id" required:"true"`
		DatasetID string `json:"dataset_id" required:"true"`
		FileKey   string `json:"file_key" required:"true"`
		LockedBy  string `json:"locked_by" required:"true"`
	}

	type releaseFileLockOutput struct {
		Success bool `json:"success"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input releaseFileLockInput, output *releaseFileLockOutput) error {
		err := api.Services.Storage.ReleaseFileLock(ctx, input.TenantID, input.DatasetID, input.FileKey, input.LockedBy)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to release file lock: %w", err), usecaseStatus.Internal)
		}

		output.Success = true
		return nil
	})

	u.SetTitle("Release File Lock")
	u.SetDescription("Releases a file lock held by a piper instance")
	u.SetTags("piper", "locks")

	return u
}

// CheckFileLock checks if a file is locked
func (api *API) CheckFileLock() usecase.Interactor {
	type checkFileLockInput struct {
		TenantID  string `path:"tenant_id" required:"true"`
		DatasetID string `path:"dataset_id" required:"true"`
		FileKey   string `path:"file_key" required:"true"`
	}

	type checkFileLockOutput struct {
		IsLocked bool                  `json:"is_locked"`
		Lock     *storage.PiperFileLock `json:"lock,omitempty"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input checkFileLockInput, output *checkFileLockOutput) error {
		lock, err := api.Services.Storage.CheckFileLock(ctx, input.TenantID, input.DatasetID, input.FileKey)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to check file lock: %w", err), usecaseStatus.Internal)
		}

		if lock != nil {
			output.IsLocked = true
			output.Lock = lock
		} else {
			output.IsLocked = false
		}

		return nil
	})

	u.SetTitle("Check File Lock")
	u.SetDescription("Checks if a file is currently locked")
	u.SetTags("piper", "locks")

	return u
}

// CleanupExpiredFileLocks removes expired file locks
func (api *API) CleanupExpiredFileLocks() usecase.Interactor {
	type cleanupExpiredFileLocksInput struct{}

	type cleanupExpiredFileLocksOutput struct {
		DeletedCount int64 `json:"deleted_count"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input cleanupExpiredFileLocksInput, output *cleanupExpiredFileLocksOutput) error {
		count, err := api.Services.Storage.CleanupExpiredFileLocks(ctx)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to cleanup expired file locks: %w", err), usecaseStatus.Internal)
		}

		output.DeletedCount = count
		return nil
	})

	u.SetTitle("Cleanup Expired File Locks")
	u.SetDescription("Removes expired file locks from the database")
	u.SetTags("piper", "locks", "cleanup")

	return u
}

// CleanupStaleFileLocks removes file locks with stale heartbeats
func (api *API) CleanupStaleFileLocks() usecase.Interactor {
	type cleanupStaleFileLocksInput struct {
		ThresholdMinutes int `query:"threshold_minutes" default:"5"`
	}

	type cleanupStaleFileLocksOutput struct {
		DeletedCount int64 `json:"deleted_count"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input cleanupStaleFileLocksInput, output *cleanupStaleFileLocksOutput) error {
		count, err := api.Services.Storage.CleanupStaleFileLocks(ctx, input.ThresholdMinutes)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to cleanup stale file locks: %w", err), usecaseStatus.Internal)
		}

		output.DeletedCount = count
		return nil
	})

	u.SetTitle("Cleanup Stale File Locks")
	u.SetDescription("Removes file locks with stale heartbeats")
	u.SetTags("piper", "locks", "cleanup")

	return u
}

// CleanupInstanceFileLocks removes all locks held by a specific instance
func (api *API) CleanupInstanceFileLocks() usecase.Interactor {
	type cleanupInstanceFileLocksInput struct {
		InstanceID string `json:"instance_id" required:"true"`
	}

	type cleanupInstanceFileLocksOutput struct {
		DeletedCount int64 `json:"deleted_count"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input cleanupInstanceFileLocksInput, output *cleanupInstanceFileLocksOutput) error {
		count, err := api.Services.Storage.CleanupInstanceFileLocks(ctx, input.InstanceID)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to cleanup instance file locks: %w", err), usecaseStatus.Internal)
		}

		output.DeletedCount = count
		return nil
	})

	u.SetTitle("Cleanup Instance File Locks")
	u.SetDescription("Removes all file locks held by a specific instance")
	u.SetTags("piper", "locks", "cleanup")

	return u
}

// ============================================================================
// PIPER JOB RECORD HANDLERS
// ============================================================================

// CreatePiperJob creates a new piper job record
func (api *API) CreatePiperJob() usecase.Interactor {
	type createPiperJobInput struct {
		JobID         string   `json:"job_id" required:"true"`
		TenantID      string   `json:"tenant_id" required:"true"`
		DatasetID     string   `json:"dataset_id" required:"true"`
		Status        string   `json:"status" required:"true"`
		SourceFiles   []string `json:"source_files"`
		ProcessorType string   `json:"processor_type"`
		ProcessorID   string   `json:"processor_id"`
	}

	type createPiperJobOutput struct {
		Success bool `json:"success"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input createPiperJobInput, output *createPiperJobOutput) error {
		job := &storage.PiperJobRecord{
			JobID:         input.JobID,
			TenantID:      input.TenantID,
			DatasetID:     input.DatasetID,
			Status:        input.Status,
			SourceFiles:   input.SourceFiles,
			ProcessorType: input.ProcessorType,
			ProcessorID:   input.ProcessorID,
		}

		err := api.Services.Storage.CreatePiperJob(ctx, job)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to create piper job: %w", err), usecaseStatus.Internal)
		}

		output.Success = true
		return nil
	})

	u.SetTitle("Create Piper Job")
	u.SetDescription("Creates a new piper job record for tracking")
	u.SetTags("piper", "jobs")

	return u
}

// UpdatePiperJobStatus updates a piper job's status
func (api *API) UpdatePiperJobStatus() usecase.Interactor {
	type updatePiperJobStatusInput struct {
		JobID            string `path:"job_id" required:"true"`
		Status           string `json:"status" required:"true"`
		ProcessorID      string `json:"processor_id"`
		OutputFile       string `json:"output_file"`
		ErrorMessage     string `json:"error_message"`
		RecordsProcessed int64  `json:"records_processed"`
	}

	type updatePiperJobStatusOutput struct {
		Success bool `json:"success"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input updatePiperJobStatusInput, output *updatePiperJobStatusOutput) error {
		err := api.Services.Storage.UpdatePiperJobStatus(ctx, input.JobID, input.Status, input.ProcessorID,
			input.OutputFile, input.ErrorMessage, input.RecordsProcessed)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to update piper job status: %w", err), usecaseStatus.Internal)
		}

		output.Success = true
		return nil
	})

	u.SetTitle("Update Piper Job Status")
	u.SetDescription("Updates the status and details of a piper job")
	u.SetTags("piper", "jobs")

	return u
}

// GetPiperJob retrieves a piper job by ID
func (api *API) GetPiperJob() usecase.Interactor {
	type getPiperJobInput struct {
		JobID string `path:"job_id" required:"true"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input getPiperJobInput, output *storage.PiperJobRecord) error {
		job, err := api.Services.Storage.GetPiperJob(ctx, input.JobID)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to get piper job: %w", err), usecaseStatus.Internal)
		}

		*output = *job
		return nil
	})

	u.SetTitle("Get Piper Job")
	u.SetDescription("Retrieves a piper job record by ID")
	u.SetTags("piper", "jobs")

	return u
}

// ListPiperJobsByStatus lists piper jobs by status
func (api *API) ListPiperJobsByStatus() usecase.Interactor {
	type listPiperJobsByStatusInput struct {
		Status string `query:"status" required:"true"`
	}

	type listPiperJobsByStatusOutput struct {
		Jobs []*storage.PiperJobRecord `json:"jobs"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input listPiperJobsByStatusInput, output *listPiperJobsByStatusOutput) error {
		jobs, err := api.Services.Storage.GetPiperJobsByStatus(ctx, input.Status)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to list piper jobs by status: %w", err), usecaseStatus.Internal)
		}

		output.Jobs = jobs
		return nil
	})

	u.SetTitle("List Piper Jobs by Status")
	u.SetDescription("Lists piper jobs filtered by status")
	u.SetTags("piper", "jobs")

	return u
}

// ListPiperJobsForTenant lists piper jobs for a tenant
func (api *API) ListPiperJobsForTenant() usecase.Interactor {
	type listPiperJobsForTenantInput struct {
		TenantID string `path:"tenant_id" required:"true"`
	}

	type listPiperJobsForTenantOutput struct {
		Jobs []*storage.PiperJobRecord `json:"jobs"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input listPiperJobsForTenantInput, output *listPiperJobsForTenantOutput) error {
		jobs, err := api.Services.Storage.GetPiperJobsForTenant(ctx, input.TenantID)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to list piper jobs for tenant: %w", err), usecaseStatus.Internal)
		}

		output.Jobs = jobs
		return nil
	})

	u.SetTitle("List Piper Jobs for Tenant")
	u.SetDescription("Lists all piper jobs for a specific tenant")
	u.SetTags("piper", "jobs")

	return u
}

// CleanupOldPiperJobs removes old completed/failed piper jobs
func (api *API) CleanupOldPiperJobs() usecase.Interactor {
	type cleanupOldPiperJobsInput struct {
		OlderThanDays int `query:"older_than_days" default:"30"`
	}

	type cleanupOldPiperJobsOutput struct {
		DeletedCount int64 `json:"deleted_count"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input cleanupOldPiperJobsInput, output *cleanupOldPiperJobsOutput) error {
		count, err := api.Services.Storage.CleanupOldPiperJobs(ctx, input.OlderThanDays)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to cleanup old piper jobs: %w", err), usecaseStatus.Internal)
		}

		output.DeletedCount = count
		return nil
	})

	u.SetTitle("Cleanup Old Piper Jobs")
	u.SetDescription("Removes old completed and failed piper jobs")
	u.SetTags("piper", "jobs", "cleanup")

	return u
}

// ============================================================================
// PIPER PIPELINE CONFIGURATION CACHE HANDLERS
// ============================================================================

// CachePipelineConfiguration caches a pipeline configuration
func (api *API) CachePipelineConfiguration() usecase.Interactor {
	type cachePipelineConfigurationInput struct {
		TenantID      string                 `json:"tenant_id" required:"true"`
		DatasetID     string                 `json:"dataset_id" required:"true"`
		Configuration map[string]interface{} `json:"configuration" required:"true"`
		TTLHours      int                    `json:"ttl_hours" default:"24"`
	}

	type cachePipelineConfigurationOutput struct {
		Success bool `json:"success"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input cachePipelineConfigurationInput, output *cachePipelineConfigurationOutput) error {
		expiresAt := time.Now().Add(time.Duration(input.TTLHours) * time.Hour)

		cache := &storage.PiperPipelineCache{
			TenantID:      input.TenantID,
			DatasetID:     input.DatasetID,
			Configuration: input.Configuration,
			ExpiresAt:     expiresAt,
		}

		err := api.Services.Storage.CachePipelineConfiguration(ctx, cache)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to cache pipeline configuration: %w", err), usecaseStatus.Internal)
		}

		output.Success = true
		return nil
	})

	u.SetTitle("Cache Pipeline Configuration")
	u.SetDescription("Caches a pipeline configuration for faster access")
	u.SetTags("piper", "cache")

	return u
}

// GetCachedPipelineConfiguration retrieves a cached pipeline configuration
func (api *API) GetCachedPipelineConfiguration() usecase.Interactor {
	type getCachedPipelineConfigurationInput struct {
		TenantID  string `path:"tenant_id" required:"true"`
		DatasetID string `path:"dataset_id" required:"true"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input getCachedPipelineConfigurationInput, output *storage.PiperPipelineCache) error {
		cache, err := api.Services.Storage.GetCachedPipelineConfiguration(ctx, input.TenantID, input.DatasetID)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to get cached pipeline configuration: %w", err), usecaseStatus.Internal)
		}

		if cache == nil {
			return usecaseStatus.Wrap(fmt.Errorf("cached pipeline configuration not found"), usecaseStatus.NotFound)
		}

		*output = *cache
		return nil
	})

	u.SetTitle("Get Cached Pipeline Configuration")
	u.SetDescription("Retrieves a cached pipeline configuration")
	u.SetTags("piper", "cache")

	return u
}

// InvalidatePipelineConfiguration removes a cached pipeline configuration
func (api *API) InvalidatePipelineConfiguration() usecase.Interactor {
	type invalidatePipelineConfigurationInput struct {
		TenantID  string `path:"tenant_id" required:"true"`
		DatasetID string `path:"dataset_id" required:"true"`
	}

	type invalidatePipelineConfigurationOutput struct {
		Success bool `json:"success"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input invalidatePipelineConfigurationInput, output *invalidatePipelineConfigurationOutput) error {
		err := api.Services.Storage.InvalidatePipelineConfiguration(ctx, input.TenantID, input.DatasetID)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to invalidate pipeline configuration: %w", err), usecaseStatus.Internal)
		}

		output.Success = true
		return nil
	})

	u.SetTitle("Invalidate Pipeline Configuration")
	u.SetDescription("Removes a cached pipeline configuration")
	u.SetTags("piper", "cache")

	return u
}

// ListCachedPipelines lists all cached pipeline configurations
func (api *API) ListCachedPipelines() usecase.Interactor {
	type listCachedPipelinesInput struct{}

	type listCachedPipelinesOutput struct {
		Pipelines []*storage.PiperPipelineCache `json:"pipelines"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input listCachedPipelinesInput, output *listCachedPipelinesOutput) error {
		pipelines, err := api.Services.Storage.ListCachedPipelines(ctx)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to list cached pipelines: %w", err), usecaseStatus.Internal)
		}

		output.Pipelines = pipelines
		return nil
	})

	u.SetTitle("List Cached Pipelines")
	u.SetDescription("Lists all cached pipeline configurations")
	u.SetTags("piper", "cache")

	return u
}

// CleanupExpiredPipelineCache removes expired pipeline cache entries
func (api *API) CleanupExpiredPipelineCache() usecase.Interactor {
	type cleanupExpiredPipelineCacheInput struct{}

	type cleanupExpiredPipelineCacheOutput struct {
		DeletedCount int64 `json:"deleted_count"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input cleanupExpiredPipelineCacheInput, output *cleanupExpiredPipelineCacheOutput) error {
		count, err := api.Services.Storage.CleanupExpiredPipelineCache(ctx)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to cleanup expired pipeline cache: %w", err), usecaseStatus.Internal)
		}

		output.DeletedCount = count
		return nil
	})

	u.SetTitle("Cleanup Expired Pipeline Cache")
	u.SetDescription("Removes expired pipeline cache entries")
	u.SetTags("piper", "cache", "cleanup")

	return u
}

// ============================================================================
// PIPER TENANT CACHE HANDLERS
// ============================================================================

// CacheTenant caches tenant information
func (api *API) CacheTenant() usecase.Interactor {
	type cacheTenantInput struct {
		TenantID   string                 `json:"tenant_id" required:"true"`
		TenantData map[string]interface{} `json:"tenant_data" required:"true"`
		TTLHours   int                    `json:"ttl_hours" default:"24"`
	}

	type cacheTenantOutput struct {
		Success bool `json:"success"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input cacheTenantInput, output *cacheTenantOutput) error {
		expiresAt := time.Now().Add(time.Duration(input.TTLHours) * time.Hour)

		cache := &storage.PiperTenantCache{
			TenantID:   input.TenantID,
			TenantData: input.TenantData,
			ExpiresAt:  expiresAt,
		}

		err := api.Services.Storage.CacheTenant(ctx, cache)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to cache tenant: %w", err), usecaseStatus.Internal)
		}

		output.Success = true
		return nil
	})

	u.SetTitle("Cache Tenant")
	u.SetDescription("Caches tenant information for faster access")
	u.SetTags("piper", "cache")

	return u
}

// GetCachedTenants retrieves all cached tenants
func (api *API) GetCachedTenants() usecase.Interactor {
	type getCachedTenantsInput struct{}

	type getCachedTenantsOutput struct {
		Tenants []*storage.PiperTenantCache `json:"tenants"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input getCachedTenantsInput, output *getCachedTenantsOutput) error {
		tenants, err := api.Services.Storage.GetCachedTenants(ctx)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to get cached tenants: %w", err), usecaseStatus.Internal)
		}

		output.Tenants = tenants
		return nil
	})

	u.SetTitle("Get Cached Tenants")
	u.SetDescription("Retrieves all cached tenant information")
	u.SetTags("piper", "cache")

	return u
}

// InvalidateTenantCache removes all cached tenant information
func (api *API) InvalidateTenantCache() usecase.Interactor {
	type invalidateTenantCacheInput struct{}

	type invalidateTenantCacheOutput struct {
		Success bool `json:"success"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input invalidateTenantCacheInput, output *invalidateTenantCacheOutput) error {
		err := api.Services.Storage.InvalidateTenantCache(ctx)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to invalidate tenant cache: %w", err), usecaseStatus.Internal)
		}

		output.Success = true
		return nil
	})

	u.SetTitle("Invalidate Tenant Cache")
	u.SetDescription("Removes all cached tenant information")
	u.SetTags("piper", "cache")

	return u
}

// CleanupExpiredTenantCache removes expired tenant cache entries
func (api *API) CleanupExpiredTenantCache() usecase.Interactor {
	type cleanupExpiredTenantCacheInput struct{}

	type cleanupExpiredTenantCacheOutput struct{
		DeletedCount int64 `json:"deleted_count"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input cleanupExpiredTenantCacheInput, output *cleanupExpiredTenantCacheOutput) error {
		count, err := api.Services.Storage.CleanupExpiredTenantCache(ctx)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to cleanup expired tenant cache: %w", err), usecaseStatus.Internal)
		}

		output.DeletedCount = count
		return nil
	})

	u.SetTitle("Cleanup Expired Tenant Cache")
	u.SetDescription("Removes expired tenant cache entries")
	u.SetTags("piper", "cache", "cleanup")

	return u
}
