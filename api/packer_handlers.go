package api

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/bytedance/sonic"
	"github.com/n0needt0/bytefreezer-control/storage"
	"github.com/swaggest/usecase"
	usecaseStatus "github.com/swaggest/usecase/status"
)

// ============================================================================
// PACKER TENANT LOCK HANDLERS
// ============================================================================

// AcquireTenantLock acquires a tenant lock for packer operations
func (api *API) AcquireTenantLock() usecase.Interactor {
	type acquireTenantLockInput struct {
		TenantID            string `json:"tenant_id" required:"true"`
		LockedBy            string `json:"locked_by" required:"true"`
		LockDurationSeconds int    `json:"lock_duration_seconds" required:"true"`
	}

	type acquireTenantLockOutput struct {
		Lock storage.PackerTenantLock `json:"lock"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input acquireTenantLockInput, output *acquireTenantLockOutput) error {
		ttl := time.Now().Add(time.Duration(input.LockDurationSeconds) * time.Second)

		lock := &storage.PackerTenantLock{
			TenantID: input.TenantID,
			LockedBy: input.LockedBy,
			TTL:      ttl,
		}

		err := api.Services.Storage.AcquireTenantLock(ctx, lock)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to acquire tenant lock: %w", err), usecaseStatus.Internal)
		}

		output.Lock = *lock
		return nil
	})

	u.SetTitle("Acquire Tenant Lock")
	u.SetDescription("Acquires a tenant lock for distributed packer processing")
	u.SetTags("packer", "locks")

	return u
}

// ReleaseTenantLock releases a tenant lock
func (api *API) ReleaseTenantLock() usecase.Interactor {
	type releaseTenantLockInput struct {
		TenantID string `path:"tenant_id" required:"true"`
		LockedBy string `json:"locked_by" required:"true"`
	}

	type releaseTenantLockOutput struct {
		Success bool `json:"success"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input releaseTenantLockInput, output *releaseTenantLockOutput) error {
		// URL-decode tenant_id since it may contain slashes
		tenantID, err := url.QueryUnescape(input.TenantID)
		if err != nil {
			tenantID = input.TenantID // Use as-is if decode fails
		}

		err = api.Services.Storage.ReleaseTenantLock(ctx, tenantID, input.LockedBy)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to release tenant lock: %w", err), usecaseStatus.Internal)
		}

		output.Success = true
		return nil
	})

	u.SetTitle("Release Tenant Lock")
	u.SetDescription("Releases a tenant lock held by a packer instance")
	u.SetTags("packer", "locks")

	return u
}

// UpdateTenantLockHeartbeat updates a tenant lock heartbeat
func (api *API) UpdateTenantLockHeartbeat() usecase.Interactor {
	type updateTenantLockHeartbeatInput struct {
		TenantID string `path:"tenant_id" required:"true"`
		LockedBy string `json:"locked_by" required:"true"`
	}

	type updateTenantLockHeartbeatOutput struct {
		Success bool `json:"success"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input updateTenantLockHeartbeatInput, output *updateTenantLockHeartbeatOutput) error {
		// URL-decode tenant_id since it may contain slashes
		tenantID, err := url.QueryUnescape(input.TenantID)
		if err != nil {
			tenantID = input.TenantID // Use as-is if decode fails
		}

		err = api.Services.Storage.UpdateTenantLockHeartbeat(ctx, tenantID, input.LockedBy)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to update tenant lock heartbeat: %w", err), usecaseStatus.Internal)
		}

		output.Success = true
		return nil
	})

	u.SetTitle("Update Tenant Lock Heartbeat")
	u.SetDescription("Updates the heartbeat timestamp for an active tenant lock")
	u.SetTags("packer", "locks")

	return u
}

// CheckTenantLock checks if a tenant is locked
func (api *API) CheckTenantLock() usecase.Interactor {
	type checkTenantLockInput struct {
		TenantID string `path:"tenant_id" required:"true"`
	}

	type checkTenantLockOutput struct {
		IsLocked bool                       `json:"is_locked"`
		Lock     *storage.PackerTenantLock `json:"lock,omitempty"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input checkTenantLockInput, output *checkTenantLockOutput) error {
		// URL-decode tenant_id since it may contain slashes
		tenantID, err := url.QueryUnescape(input.TenantID)
		if err != nil {
			tenantID = input.TenantID // Use as-is if decode fails
		}

		lock, err := api.Services.Storage.CheckTenantLock(ctx, tenantID)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to check tenant lock: %w", err), usecaseStatus.Internal)
		}

		if lock != nil {
			output.IsLocked = true
			output.Lock = lock
		} else {
			output.IsLocked = false
		}

		return nil
	})

	u.SetTitle("Check Tenant Lock")
	u.SetDescription("Checks if a tenant is currently locked")
	u.SetTags("packer", "locks")

	return u
}

// CleanupExpiredTenantLocks removes expired tenant locks
func (api *API) CleanupExpiredTenantLocks() usecase.Interactor {
	type cleanupExpiredTenantLocksInput struct{}

	type cleanupExpiredTenantLocksOutput struct {
		DeletedCount int64 `json:"deleted_count"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input cleanupExpiredTenantLocksInput, output *cleanupExpiredTenantLocksOutput) error {
		count, err := api.Services.Storage.CleanupExpiredTenantLocks(ctx)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to cleanup expired tenant locks: %w", err), usecaseStatus.Internal)
		}

		output.DeletedCount = count
		return nil
	})

	u.SetTitle("Cleanup Expired Tenant Locks")
	u.SetDescription("Removes expired tenant locks from the database")
	u.SetTags("packer", "locks", "cleanup")

	return u
}

// ClearAllTenantLocks removes all tenant locks
func (api *API) ClearAllTenantLocks() usecase.Interactor {
	type clearAllTenantLocksInput struct{}

	type clearAllTenantLocksOutput struct {
		DeletedCount int64 `json:"deleted_count"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input clearAllTenantLocksInput, output *clearAllTenantLocksOutput) error {
		count, err := api.Services.Storage.ClearAllTenantLocks(ctx)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to clear all tenant locks: %w", err), usecaseStatus.Internal)
		}

		output.DeletedCount = count
		return nil
	})

	u.SetTitle("Clear All Tenant Locks")
	u.SetDescription("Removes all tenant locks (for clean restarts)")
	u.SetTags("packer", "locks", "cleanup")

	return u
}

// CleanupStaleTenantLocks removes stale tenant locks
func (api *API) CleanupStaleTenantLocks() usecase.Interactor {
	type cleanupStaleTenantLocksInput struct {
		ThresholdMinutes int `query:"threshold_minutes" default:"5"`
	}

	type cleanupStaleTenantLocksOutput struct {
		DeletedCount int64 `json:"deleted_count"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input cleanupStaleTenantLocksInput, output *cleanupStaleTenantLocksOutput) error {
		count, err := api.Services.Storage.CleanupStaleTenantLocks(ctx, input.ThresholdMinutes)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to cleanup stale tenant locks: %w", err), usecaseStatus.Internal)
		}

		output.DeletedCount = count
		return nil
	})

	u.SetTitle("Cleanup Stale Tenant Locks")
	u.SetDescription("Removes tenant locks with stale heartbeats")
	u.SetTags("packer", "locks", "cleanup")

	return u
}

// CleanupInstanceTenantLocks removes all locks held by a specific instance
func (api *API) CleanupInstanceTenantLocks() usecase.Interactor {
	type cleanupInstanceTenantLocksInput struct {
		InstanceID string `json:"instance_id" required:"true"`
	}

	type cleanupInstanceTenantLocksOutput struct {
		DeletedCount int64 `json:"deleted_count"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input cleanupInstanceTenantLocksInput, output *cleanupInstanceTenantLocksOutput) error {
		count, err := api.Services.Storage.CleanupInstanceTenantLocks(ctx, input.InstanceID)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to cleanup instance tenant locks: %w", err), usecaseStatus.Internal)
		}

		output.DeletedCount = count
		return nil
	})

	u.SetTitle("Cleanup Instance Tenant Locks")
	u.SetDescription("Removes all tenant locks held by a specific instance")
	u.SetTags("packer", "locks", "cleanup")

	return u
}

// ============================================================================
// PACKER PARQUET METADATA HANDLERS
// ============================================================================

// UpsertParquetFileMetadata upserts Parquet file metadata
func (api *API) UpsertParquetFileMetadata() usecase.Interactor {
	type upsertParquetFileMetadataInput struct {
		TenantID        string                 `json:"tenant_id" required:"true"`
		DatasetID       string                 `json:"dataset_id" required:"true"`
		FilePath        string                 `json:"file_path" required:"true"`
		PartitionPath   string                 `json:"partition_path"`
		FileSizeBytes   int64                  `json:"file_size_bytes" required:"true"`
		RowCount        int64                  `json:"row_count" required:"true"`
		CreatedAt       time.Time              `json:"created_at"`
		LastModified    time.Time              `json:"last_modified" required:"true"`
		SchemaJSON      map[string]interface{} `json:"schema_json" required:"true"`
		ColumnStats     interface{}            `json:"column_stats"` // Can be array or map
		FileChecksum    string                 `json:"file_checksum"`
		InstanceID      string                 `json:"instance_id"`
	}

	type upsertParquetFileMetadataOutput struct {
		ID      int64 `json:"id"`
		Version int   `json:"version"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input upsertParquetFileMetadataInput, output *upsertParquetFileMetadataOutput) error {
		metadata := &storage.PackerParquetFileMetadata{
			TenantID:       input.TenantID,
			DatasetID:      input.DatasetID,
			FilePath:       input.FilePath,
			PartitionPath:  input.PartitionPath,
			FileSizeBytes:  input.FileSizeBytes,
			RowCount:       input.RowCount,
			CreatedAt:      input.CreatedAt,
			LastModified:   input.LastModified,
			SchemaJSON:     input.SchemaJSON,
			ColumnStats:    input.ColumnStats,
			FileChecksum:   input.FileChecksum,
			InstanceID:     input.InstanceID,
		}

		err := api.Services.Storage.UpsertParquetFileMetadata(ctx, metadata)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to upsert parquet file metadata: %w", err), usecaseStatus.Internal)
		}

		output.ID = metadata.ID
		output.Version = metadata.MetadataVersion
		return nil
	})

	u.SetTitle("Upsert Parquet File Metadata")
	u.SetDescription("Inserts or updates Parquet file metadata")
	u.SetTags("packer", "metadata")

	return u
}

// GetParquetFileMetadataByPartition retrieves file metadata for a partition
func (api *API) GetParquetFileMetadataByPartition() usecase.Interactor {
	type getParquetFileMetadataByPartitionInput struct {
		TenantID      string `path:"tenant_id" required:"true"`
		DatasetID     string `path:"dataset_id" required:"true"`
		PartitionPath string `query:"partition" required:"true"`
	}

	type getParquetFileMetadataByPartitionOutput struct {
		Files []*storage.PackerParquetFileMetadata `json:"files"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input getParquetFileMetadataByPartitionInput, output *getParquetFileMetadataByPartitionOutput) error {
		files, err := api.Services.Storage.GetParquetFileMetadataByPartition(ctx, input.TenantID, input.DatasetID, input.PartitionPath)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to get parquet file metadata by partition: %w", err), usecaseStatus.Internal)
		}

		output.Files = files
		return nil
	})

	u.SetTitle("Get Parquet File Metadata by Partition")
	u.SetDescription("Retrieves all Parquet file metadata for a specific partition")
	u.SetTags("packer", "metadata")

	return u
}

// GetAllParquetFileMetadata retrieves all file metadata for a tenant:dataset
func (api *API) GetAllParquetFileMetadata() usecase.Interactor {
	type getAllParquetFileMetadataInput struct {
		TenantID  string `path:"tenant_id" required:"true"`
		DatasetID string `path:"dataset_id" required:"true"`
	}

	type parquetFileMetadataResponse struct {
		ID              int64     `json:"id"`
		TenantID        string    `json:"tenant_id"`
		DatasetID       string    `json:"dataset_id"`
		FilePath        string    `json:"file_path"`
		PartitionPath   string    `json:"partition_path"`
		FileSizeBytes   int64     `json:"file_size_bytes"`
		RowCount        int64     `json:"row_count"`
		CreatedAt       time.Time `json:"created_at"`
		LastModified    time.Time `json:"last_modified"`
		SchemaJSON      string    `json:"schema_json"`      // JSON string, not object
		ColumnStats     string    `json:"column_stats"`     // JSON string, not object
		FileChecksum    string    `json:"file_checksum"`
		InstanceID      string    `json:"instance_id"`
		MetadataVersion int       `json:"metadata_version"`
	}

	type getAllParquetFileMetadataOutput struct {
		Files []*parquetFileMetadataResponse `json:"files"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input getAllParquetFileMetadataInput, output *getAllParquetFileMetadataOutput) error {
		files, err := api.Services.Storage.GetAllParquetFileMetadata(ctx, input.TenantID, input.DatasetID)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to get all parquet file metadata: %w", err), usecaseStatus.Internal)
		}

		// Convert to response format with JSON strings
		output.Files = make([]*parquetFileMetadataResponse, len(files))
		for i, file := range files {
			// Marshal schema_json to JSON string
			schemaJSON, err := sonic.Marshal(file.SchemaJSON)
			if err != nil {
				return usecaseStatus.Wrap(fmt.Errorf("failed to marshal schema_json: %w", err), usecaseStatus.Internal)
			}

			// Marshal column_stats to JSON string
			columnStats, err := sonic.Marshal(file.ColumnStats)
			if err != nil {
				return usecaseStatus.Wrap(fmt.Errorf("failed to marshal column_stats: %w", err), usecaseStatus.Internal)
			}

			output.Files[i] = &parquetFileMetadataResponse{
				ID:              file.ID,
				TenantID:        file.TenantID,
				DatasetID:       file.DatasetID,
				FilePath:        file.FilePath,
				PartitionPath:   file.PartitionPath,
				FileSizeBytes:   file.FileSizeBytes,
				RowCount:        file.RowCount,
				CreatedAt:       file.CreatedAt,
				LastModified:    file.LastModified,
				SchemaJSON:      string(schemaJSON),
				ColumnStats:     string(columnStats),
				FileChecksum:    file.FileChecksum,
				InstanceID:      file.InstanceID,
				MetadataVersion: file.MetadataVersion,
			}
		}

		return nil
	})

	u.SetTitle("Get All Parquet File Metadata")
	u.SetDescription("Retrieves all Parquet file metadata for a tenant:dataset")
	u.SetTags("packer", "metadata")

	return u
}

// DeleteParquetFileMetadata deletes metadata for a specific file
func (api *API) DeleteParquetFileMetadata() usecase.Interactor {
	type deleteParquetFileMetadataInput struct {
		TenantID  string `path:"tenant_id" required:"true"`
		DatasetID string `path:"dataset_id" required:"true"`
		FilePath  string `json:"file_path" required:"true"`
	}

	type deleteParquetFileMetadataOutput struct {
		Success bool `json:"success"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input deleteParquetFileMetadataInput, output *deleteParquetFileMetadataOutput) error {
		err := api.Services.Storage.DeleteParquetFileMetadata(ctx, input.TenantID, input.DatasetID, input.FilePath)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to delete parquet file metadata: %w", err), usecaseStatus.Internal)
		}

		output.Success = true
		return nil
	})

	u.SetTitle("Delete Parquet File Metadata")
	u.SetDescription("Deletes metadata for a specific Parquet file")
	u.SetTags("packer", "metadata")

	return u
}

// CleanupOrphanedParquetMetadata removes metadata for files that no longer exist
func (api *API) CleanupOrphanedParquetMetadata() usecase.Interactor {
	type cleanupOrphanedParquetMetadataInput struct {
		TenantID      string   `json:"tenant_id" required:"true"`
		DatasetID     string   `json:"dataset_id" required:"true"`
		ExistingFiles []string `json:"existing_files" required:"true"`
	}

	type cleanupOrphanedParquetMetadataOutput struct {
		DeletedCount int64 `json:"deleted_count"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input cleanupOrphanedParquetMetadataInput, output *cleanupOrphanedParquetMetadataOutput) error {
		count, err := api.Services.Storage.CleanupOrphanedParquetMetadata(ctx, input.TenantID, input.DatasetID, input.ExistingFiles)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to cleanup orphaned parquet metadata: %w", err), usecaseStatus.Internal)
		}

		output.DeletedCount = count
		return nil
	})

	u.SetTitle("Cleanup Orphaned Parquet Metadata")
	u.SetDescription("Removes metadata for Parquet files that no longer exist in S3")
	u.SetTags("packer", "metadata", "cleanup")

	return u
}

// CleanupExpiredParquetMetadata removes expired metadata records
func (api *API) CleanupExpiredParquetMetadata() usecase.Interactor {
	type cleanupExpiredParquetMetadataInput struct{}

	type cleanupExpiredParquetMetadataOutput struct {
		FilesDeleted  int64 `json:"files_deleted"`
		StatusDeleted int64 `json:"status_deleted"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input cleanupExpiredParquetMetadataInput, output *cleanupExpiredParquetMetadataOutput) error {
		filesDeleted, statusDeleted, err := api.Services.Storage.CleanupExpiredParquetMetadata(ctx)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to cleanup expired parquet metadata: %w", err), usecaseStatus.Internal)
		}

		output.FilesDeleted = filesDeleted
		output.StatusDeleted = statusDeleted
		return nil
	})

	u.SetTitle("Cleanup Expired Parquet Metadata")
	u.SetDescription("Removes expired Parquet metadata records")
	u.SetTags("packer", "metadata", "cleanup")

	return u
}

// ============================================================================
// PACKER METADATA GENERATION STATUS HANDLERS
// ============================================================================

// UpdateMetadataGenerationStatus updates metadata generation status
func (api *API) UpdateMetadataGenerationStatus() usecase.Interactor {
	type updateMetadataGenerationStatusInput struct {
		TenantID          string    `json:"tenant_id" required:"true"`
		DatasetID         string    `json:"dataset_id" required:"true"`
		PartitionPath     string    `json:"partition_path" required:"true"`
		LastGeneratedAt   time.Time `json:"last_generated_at"`
		FileCount         int       `json:"file_count"`
		TotalRows         int64     `json:"total_rows"`
		TotalSizeBytes    int64     `json:"total_size_bytes"`
		NeedsRegeneration bool      `json:"needs_regeneration"`
		CurrentSchemaHash string    `json:"current_schema_hash"`
		SchemaVersion     int       `json:"schema_version"`
	}

	type updateMetadataGenerationStatusOutput struct {
		Success bool `json:"success"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input updateMetadataGenerationStatusInput, output *updateMetadataGenerationStatusOutput) error {
		status := &storage.PackerMetadataGenerationStatus{
			TenantID:          input.TenantID,
			DatasetID:         input.DatasetID,
			PartitionPath:     input.PartitionPath,
			LastGeneratedAt:   input.LastGeneratedAt,
			FileCount:         input.FileCount,
			TotalRows:         input.TotalRows,
			TotalSizeBytes:    input.TotalSizeBytes,
			NeedsRegeneration: input.NeedsRegeneration,
			CurrentSchemaHash: input.CurrentSchemaHash,
			SchemaVersion:     input.SchemaVersion,
		}

		err := api.Services.Storage.UpdateMetadataGenerationStatus(ctx, status)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to update metadata generation status: %w", err), usecaseStatus.Internal)
		}

		output.Success = true
		return nil
	})

	u.SetTitle("Update Metadata Generation Status")
	u.SetDescription("Updates the metadata generation status for a partition")
	u.SetTags("packer", "metadata")

	return u
}

// GetMetadataGenerationStatus retrieves metadata generation status
func (api *API) GetMetadataGenerationStatus() usecase.Interactor {
	type getMetadataGenerationStatusInput struct {
		TenantID      string `path:"tenant_id" required:"true"`
		DatasetID     string `path:"dataset_id" required:"true"`
		PartitionPath string `path:"partition_path" required:"true"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input getMetadataGenerationStatusInput, output *storage.PackerMetadataGenerationStatus) error {
		status, err := api.Services.Storage.GetMetadataGenerationStatus(ctx, input.TenantID, input.DatasetID, input.PartitionPath)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to get metadata generation status: %w", err), usecaseStatus.Internal)
		}

		*output = *status
		return nil
	})

	u.SetTitle("Get Metadata Generation Status")
	u.SetDescription("Retrieves the metadata generation status for a partition")
	u.SetTags("packer", "metadata")

	return u
}

// ============================================================================
// PACKER METADATA SUMMARY HANDLERS
// ============================================================================

// GetParquetMetadataSummary retrieves aggregated metadata for a partition
func (api *API) GetParquetMetadataSummary() usecase.Interactor {
	type getParquetMetadataSummaryInput struct {
		TenantID      string `path:"tenant_id" required:"true"`
		DatasetID     string `path:"dataset_id" required:"true"`
		PartitionPath string `path:"partition_path" required:"true"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input getParquetMetadataSummaryInput, output *storage.PackerParquetMetadataSummary) error {
		summary, err := api.Services.Storage.GetParquetMetadataSummary(ctx, input.TenantID, input.DatasetID, input.PartitionPath)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to get parquet metadata summary: %w", err), usecaseStatus.Internal)
		}

		if summary == nil {
			return usecaseStatus.Wrap(fmt.Errorf("parquet metadata summary not found"), usecaseStatus.NotFound)
		}

		*output = *summary
		return nil
	})

	u.SetTitle("Get Parquet Metadata Summary")
	u.SetDescription("Retrieves aggregated Parquet metadata for a partition")
	u.SetTags("packer", "metadata")

	return u
}
