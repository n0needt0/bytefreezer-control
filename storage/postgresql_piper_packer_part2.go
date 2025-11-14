package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/lib/pq"
	"github.com/n0needt0/go-goodies/log"
)

// ============================================================================
// PACKER TENANT LOCK OPERATIONS
// ============================================================================

// AcquireTenantLock attempts to acquire a lock for a tenant
func (s *PostgreSQLStorage) AcquireTenantLock(ctx context.Context, lock *PackerTenantLock) error {
	// Set timestamps
	now := time.Now()
	lock.LockTimestamp = now
	lock.LastHeartbeat = now

	// Begin transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Clean up expired locks for this tenant
	cleanupQuery := `DELETE FROM packer_tenant_locks WHERE tenant_id = $1 AND ttl < NOW()`
	_, err = tx.ExecContext(ctx, cleanupQuery, lock.TenantID)
	if err != nil {
		return fmt.Errorf("failed to cleanup expired locks: %w", err)
	}

	// Try to insert the lock
	insertQuery := `INSERT INTO packer_tenant_locks
		(tenant_id, locked_by, lock_timestamp, last_heartbeat, ttl)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (tenant_id) DO NOTHING`

	result, err := tx.ExecContext(ctx, insertQuery,
		lock.TenantID, lock.LockedBy, lock.LockTimestamp, lock.LastHeartbeat, lock.TTL,
	)
	if err != nil {
		return fmt.Errorf("failed to insert tenant lock: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("tenant %s is already locked", lock.TenantID)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit lock transaction: %w", err)
	}

	log.Infof("Acquired tenant lock for %s by instance %s", lock.TenantID, lock.LockedBy)
	return nil
}

// ReleaseTenantLock releases a tenant lock
func (s *PostgreSQLStorage) ReleaseTenantLock(ctx context.Context, tenantID, lockedBy string) error {
	query := `DELETE FROM packer_tenant_locks WHERE tenant_id = $1 AND locked_by = $2`

	result, err := s.db.ExecContext(ctx, query, tenantID, lockedBy)
	if err != nil {
		return fmt.Errorf("failed to release tenant lock: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("cannot release lock for tenant %s: not owned by instance %s", tenantID, lockedBy)
	}

	log.Infof("Released tenant lock for %s by instance %s", tenantID, lockedBy)
	return nil
}

// UpdateTenantLockHeartbeat updates the last_heartbeat timestamp for a tenant lock
func (s *PostgreSQLStorage) UpdateTenantLockHeartbeat(ctx context.Context, tenantID, lockedBy string) error {
	query := `UPDATE packer_tenant_locks SET last_heartbeat = NOW()
		WHERE tenant_id = $1 AND locked_by = $2`

	result, err := s.db.ExecContext(ctx, query, tenantID, lockedBy)
	if err != nil {
		return fmt.Errorf("failed to update heartbeat for tenant %s: %w", tenantID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Warnf("Failed to get rows affected during heartbeat update: %v", err)
		return nil // Non-critical error
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no lock found to update heartbeat for tenant %s by instance %s", tenantID, lockedBy)
	}

	log.Debugf("Updated heartbeat for tenant %s by instance %s", tenantID, lockedBy)
	return nil
}

// CheckTenantLock checks if a tenant is locked
func (s *PostgreSQLStorage) CheckTenantLock(ctx context.Context, tenantID string) (*PackerTenantLock, error) {
	// Clean up expired locks first
	cleanupQuery := `DELETE FROM packer_tenant_locks WHERE tenant_id = $1 AND ttl < NOW()`
	_, err := s.db.ExecContext(ctx, cleanupQuery, tenantID)
	if err != nil {
		log.Warnf("Failed to cleanup expired locks for tenant %s: %v", tenantID, err)
	}

	query := `SELECT tenant_id, locked_by, lock_timestamp, last_heartbeat, ttl
		FROM packer_tenant_locks WHERE tenant_id = $1 AND ttl > NOW()`

	lock := &PackerTenantLock{}
	err = s.db.QueryRowContext(ctx, query, tenantID).Scan(
		&lock.TenantID, &lock.LockedBy, &lock.LockTimestamp, &lock.LastHeartbeat, &lock.TTL,
	)

	if err == sql.ErrNoRows {
		return nil, nil // No lock exists
	}
	if err != nil {
		return nil, fmt.Errorf("failed to check tenant lock: %w", err)
	}

	return lock, nil
}

// CleanupExpiredTenantLocks removes all expired tenant locks
func (s *PostgreSQLStorage) CleanupExpiredTenantLocks(ctx context.Context) (int64, error) {
	query := `DELETE FROM packer_tenant_locks WHERE ttl < NOW()`

	result, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired tenant locks: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected > 0 {
		log.Infof("Cleaned up %d expired tenant locks", rowsAffected)
	}

	return rowsAffected, nil
}

// ClearAllTenantLocks removes all tenant locks
func (s *PostgreSQLStorage) ClearAllTenantLocks(ctx context.Context) (int64, error) {
	query := `DELETE FROM packer_tenant_locks`

	result, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to clear all tenant locks: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	log.Infof("Cleared all %d tenant locks", rowsAffected)
	return rowsAffected, nil
}

// CleanupStaleTenantLocks removes tenant locks with stale heartbeats
func (s *PostgreSQLStorage) CleanupStaleTenantLocks(ctx context.Context, thresholdMinutes int) (int64, error) {
	query := `DELETE FROM packer_tenant_locks
		WHERE last_heartbeat < NOW() - INTERVAL '1 minute' * $1`

	result, err := s.db.ExecContext(ctx, query, thresholdMinutes)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup stale tenant locks: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected > 0 {
		log.Infof("Cleaned up %d stale tenant locks (heartbeat older than %d minutes)", rowsAffected, thresholdMinutes)
	}

	return rowsAffected, nil
}

// ============================================================================
// PACKER PARQUET METADATA OPERATIONS
// ============================================================================

// UpsertParquetFileMetadata inserts or updates Parquet file metadata
func (s *PostgreSQLStorage) UpsertParquetFileMetadata(ctx context.Context, metadata *PackerParquetFileMetadata) error {
	// Set TTL to 30 days from now
	metadata.TTL = time.Now().Add(30 * 24 * time.Hour)
	metadata.UpdatedAt = time.Now()

	if metadata.CreatedAt.IsZero() {
		metadata.CreatedAt = time.Now()
	}

	schemaJSON, err := sonic.Marshal(metadata.SchemaJSON)
	if err != nil {
		return fmt.Errorf("failed to marshal schema JSON: %w", err)
	}

	var columnStatsJSON []byte
	if metadata.ColumnStats != nil {
		columnStatsJSON, err = sonic.Marshal(metadata.ColumnStats)
		if err != nil {
			return fmt.Errorf("failed to marshal column stats: %w", err)
		}
	}

	query := `INSERT INTO packer_parquet_file_metadata
		(tenant_id, dataset_id, file_path, partition_path, file_size_bytes, row_count,
		 created_at, last_modified, schema_json, column_stats, file_checksum, instance_id, ttl, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (tenant_id, dataset_id, file_path) DO UPDATE SET
			file_size_bytes = EXCLUDED.file_size_bytes,
			row_count = EXCLUDED.row_count,
			last_modified = EXCLUDED.last_modified,
			schema_json = EXCLUDED.schema_json,
			column_stats = EXCLUDED.column_stats,
			file_checksum = EXCLUDED.file_checksum,
			instance_id = EXCLUDED.instance_id,
			ttl = EXCLUDED.ttl,
			updated_at = EXCLUDED.updated_at,
			metadata_version = packer_parquet_file_metadata.metadata_version + 1
		RETURNING id, metadata_version`

	err = s.db.QueryRowContext(ctx, query,
		metadata.TenantID, metadata.DatasetID, metadata.FilePath, metadata.PartitionPath,
		metadata.FileSizeBytes, metadata.RowCount, metadata.CreatedAt, metadata.LastModified,
		schemaJSON, columnStatsJSON, metadata.FileChecksum, metadata.InstanceID,
		metadata.TTL, metadata.UpdatedAt,
	).Scan(&metadata.ID, &metadata.MetadataVersion)

	if err != nil {
		return fmt.Errorf("failed to upsert parquet file metadata: %w", err)
	}

	log.Debugf("Upserted metadata for file %s (ID: %d, Version: %d)", metadata.FilePath, metadata.ID, metadata.MetadataVersion)
	return nil
}

// GetParquetFileMetadataByPartition retrieves file metadata for a partition
func (s *PostgreSQLStorage) GetParquetFileMetadataByPartition(ctx context.Context, tenantID, datasetID, partitionPath string) ([]*PackerParquetFileMetadata, error) {
	query := `SELECT id, tenant_id, dataset_id, file_path, partition_path,
		file_size_bytes, row_count, created_at, last_modified, schema_json,
		column_stats, file_checksum, instance_id, metadata_version, ttl, inserted_at, updated_at
		FROM packer_parquet_file_metadata
		WHERE tenant_id = $1 AND dataset_id = $2 AND partition_path = $3
		ORDER BY file_path`

	rows, err := s.db.QueryContext(ctx, query, tenantID, datasetID, partitionPath)
	if err != nil {
		return nil, fmt.Errorf("failed to query parquet file metadata: %w", err)
	}
	defer rows.Close()

	var files []*PackerParquetFileMetadata
	for rows.Next() {
		file := &PackerParquetFileMetadata{}
		var schemaJSON, columnStatsJSON []byte

		err := rows.Scan(
			&file.ID, &file.TenantID, &file.DatasetID, &file.FilePath, &file.PartitionPath,
			&file.FileSizeBytes, &file.RowCount, &file.CreatedAt, &file.LastModified, &schemaJSON,
			&columnStatsJSON, &file.FileChecksum, &file.InstanceID, &file.MetadataVersion,
			&file.TTL, &file.InsertedAt, &file.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan parquet file metadata: %w", err)
		}

		if len(schemaJSON) > 0 {
			if err := sonic.Unmarshal(schemaJSON, &file.SchemaJSON); err != nil {
				return nil, fmt.Errorf("failed to unmarshal schema JSON: %w", err)
			}
		}

		if len(columnStatsJSON) > 0 {
			if err := sonic.Unmarshal(columnStatsJSON, &file.ColumnStats); err != nil {
				return nil, fmt.Errorf("failed to unmarshal column stats: %w", err)
			}
		}

		files = append(files, file)
	}

	return files, rows.Err()
}

// GetAllParquetFileMetadata retrieves ALL file metadata for a tenant:dataset
func (s *PostgreSQLStorage) GetAllParquetFileMetadata(ctx context.Context, tenantID, datasetID string) ([]*PackerParquetFileMetadata, error) {
	query := `SELECT id, tenant_id, dataset_id, file_path, partition_path,
		file_size_bytes, row_count, created_at, last_modified, schema_json,
		column_stats, file_checksum, instance_id, metadata_version, ttl, inserted_at, updated_at
		FROM packer_parquet_file_metadata
		WHERE tenant_id = $1 AND dataset_id = $2
		ORDER BY partition_path, file_path`

	rows, err := s.db.QueryContext(ctx, query, tenantID, datasetID)
	if err != nil {
		return nil, fmt.Errorf("failed to query all parquet file metadata: %w", err)
	}
	defer rows.Close()

	var files []*PackerParquetFileMetadata
	for rows.Next() {
		file := &PackerParquetFileMetadata{}
		var schemaJSON, columnStatsJSON []byte

		err := rows.Scan(
			&file.ID, &file.TenantID, &file.DatasetID, &file.FilePath, &file.PartitionPath,
			&file.FileSizeBytes, &file.RowCount, &file.CreatedAt, &file.LastModified, &schemaJSON,
			&columnStatsJSON, &file.FileChecksum, &file.InstanceID, &file.MetadataVersion,
			&file.TTL, &file.InsertedAt, &file.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan parquet file metadata: %w", err)
		}

		if len(schemaJSON) > 0 {
			if err := sonic.Unmarshal(schemaJSON, &file.SchemaJSON); err != nil {
				return nil, fmt.Errorf("failed to unmarshal schema JSON: %w", err)
			}
		}

		if len(columnStatsJSON) > 0 {
			if err := sonic.Unmarshal(columnStatsJSON, &file.ColumnStats); err != nil {
				return nil, fmt.Errorf("failed to unmarshal column stats: %w", err)
			}
		}

		files = append(files, file)
	}

	return files, rows.Err()
}

// DeleteParquetFileMetadata removes metadata for a specific file
func (s *PostgreSQLStorage) DeleteParquetFileMetadata(ctx context.Context, tenantID, datasetID, filePath string) error {
	query := `DELETE FROM packer_parquet_file_metadata
		WHERE tenant_id = $1 AND dataset_id = $2 AND file_path = $3`

	_, err := s.db.ExecContext(ctx, query, tenantID, datasetID, filePath)
	if err != nil {
		return fmt.Errorf("failed to delete parquet file metadata: %w", err)
	}

	return nil
}

// CleanupOrphanedParquetMetadata removes metadata for files that no longer exist
func (s *PostgreSQLStorage) CleanupOrphanedParquetMetadata(ctx context.Context, tenantID, datasetID string, existingFiles []string) (int64, error) {
	if len(existingFiles) == 0 {
		return 0, nil
	}

	// Build a safe parameterized query
	inClausePlaceholders := make([]string, len(existingFiles))
	args := make([]interface{}, len(existingFiles)+2)

	args[0] = tenantID
	args[1] = datasetID

	for i, file := range existingFiles {
		inClausePlaceholders[i] = fmt.Sprintf("$%d", i+3)
		args[i+2] = file
	}

	inClause := strings.Join(inClausePlaceholders, ",")
	query := fmt.Sprintf(`DELETE FROM packer_parquet_file_metadata
		WHERE tenant_id = $1 AND dataset_id = $2 AND file_path NOT IN (%s)`, inClause)

	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup orphaned parquet metadata: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected, nil
}

// CleanupExpiredParquetMetadata removes expired metadata records
func (s *PostgreSQLStorage) CleanupExpiredParquetMetadata(ctx context.Context) (int64, int64, error) {
	// Cleanup file metadata
	fileQuery := `DELETE FROM packer_parquet_file_metadata WHERE ttl < NOW()`
	fileResult, err := s.db.ExecContext(ctx, fileQuery)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to cleanup expired file metadata: %w", err)
	}

	filesDeleted, err := fileResult.RowsAffected()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get file metadata rows affected: %w", err)
	}

	// Cleanup generation status
	statusQuery := `DELETE FROM packer_metadata_generation_status WHERE ttl < NOW()`
	statusResult, err := s.db.ExecContext(ctx, statusQuery)
	if err != nil {
		return filesDeleted, 0, fmt.Errorf("failed to cleanup expired generation status: %w", err)
	}

	statusDeleted, err := statusResult.RowsAffected()
	if err != nil {
		return filesDeleted, 0, fmt.Errorf("failed to get generation status rows affected: %w", err)
	}

	if filesDeleted > 0 || statusDeleted > 0 {
		log.Infof("Cleaned up %d expired file metadata records and %d expired generation status records", filesDeleted, statusDeleted)
	}

	return filesDeleted, statusDeleted, nil
}

// ============================================================================
// PACKER METADATA GENERATION STATUS OPERATIONS
// ============================================================================

// UpdateMetadataGenerationStatus updates the metadata generation status for a partition
func (s *PostgreSQLStorage) UpdateMetadataGenerationStatus(ctx context.Context, status *PackerMetadataGenerationStatus) error {
	// Set TTL to 30 days from now
	status.TTL = time.Now().Add(30 * 24 * time.Hour)

	if status.LastGeneratedAt.IsZero() {
		status.LastGeneratedAt = time.Now()
	}

	query := `INSERT INTO packer_metadata_generation_status
		(tenant_id, dataset_id, partition_path, last_generated_at, file_count,
		 total_rows, total_size_bytes, needs_regeneration, current_schema_hash, schema_version, ttl)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (tenant_id, dataset_id, partition_path) DO UPDATE SET
			last_generated_at = EXCLUDED.last_generated_at,
			file_count = EXCLUDED.file_count,
			total_rows = EXCLUDED.total_rows,
			total_size_bytes = EXCLUDED.total_size_bytes,
			needs_regeneration = EXCLUDED.needs_regeneration,
			current_schema_hash = EXCLUDED.current_schema_hash,
			schema_version = EXCLUDED.schema_version,
			ttl = EXCLUDED.ttl`

	_, err := s.db.ExecContext(ctx, query,
		status.TenantID, status.DatasetID, status.PartitionPath, status.LastGeneratedAt,
		status.FileCount, status.TotalRows, status.TotalSizeBytes, status.NeedsRegeneration,
		status.CurrentSchemaHash, status.SchemaVersion, status.TTL,
	)

	if err != nil {
		return fmt.Errorf("failed to update metadata generation status: %w", err)
	}

	return nil
}

// GetMetadataGenerationStatus retrieves the metadata generation status for a partition
func (s *PostgreSQLStorage) GetMetadataGenerationStatus(ctx context.Context, tenantID, datasetID, partitionPath string) (*PackerMetadataGenerationStatus, error) {
	query := `SELECT tenant_id, dataset_id, partition_path, last_generated_at,
		file_count, total_rows, total_size_bytes, needs_regeneration,
		current_schema_hash, schema_version, ttl
		FROM packer_metadata_generation_status
		WHERE tenant_id = $1 AND dataset_id = $2 AND partition_path = $3`

	status := &PackerMetadataGenerationStatus{}
	err := s.db.QueryRowContext(ctx, query, tenantID, datasetID, partitionPath).Scan(
		&status.TenantID, &status.DatasetID, &status.PartitionPath, &status.LastGeneratedAt,
		&status.FileCount, &status.TotalRows, &status.TotalSizeBytes, &status.NeedsRegeneration,
		&status.CurrentSchemaHash, &status.SchemaVersion, &status.TTL,
	)

	if err == sql.ErrNoRows {
		// Return default status if not found
		return &PackerMetadataGenerationStatus{
			TenantID:      tenantID,
			DatasetID:     datasetID,
			PartitionPath: partitionPath,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata generation status: %w", err)
	}

	return status, nil
}

// ============================================================================
// PACKER METADATA SUMMARY OPERATIONS
// ============================================================================

// GetParquetMetadataSummary retrieves aggregated metadata for a partition
func (s *PostgreSQLStorage) GetParquetMetadataSummary(ctx context.Context, tenantID, datasetID, partitionPath string) (*PackerParquetMetadataSummary, error) {
	query := `SELECT tenant_id, dataset_id, partition_path, file_count, total_rows, total_size_bytes,
		first_file_created, last_file_modified, metadata_last_updated
		FROM packer_parquet_metadata_summary
		WHERE tenant_id = $1 AND dataset_id = $2 AND partition_path = $3`

	summary := &PackerParquetMetadataSummary{}
	err := s.db.QueryRowContext(ctx, query, tenantID, datasetID, partitionPath).Scan(
		&summary.TenantID, &summary.DatasetID, &summary.PartitionPath,
		&summary.FileCount, &summary.TotalRows, &summary.TotalSizeBytes,
		&summary.FirstFileCreated, &summary.LastFileModified, &summary.MetadataLastUpdated,
	)

	if err == sql.ErrNoRows {
		return nil, nil // No data found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get parquet metadata summary: %w", err)
	}

	return summary, nil
}

// ============================================================================
// PIPER TRANSFORMATION JOB OPERATIONS
// ============================================================================

// CreatePiperTransformationJob creates a new transformation job
func (s *PostgreSQLStorage) CreatePiperTransformationJob(ctx context.Context, job *PiperTransformationJob) error {
	query := `INSERT INTO piper_transformation_jobs
		(job_id, tenant_id, dataset_id, job_type, status, processor_id, request, result,
		error_message, created_at, updated_at, started_at, completed_at, ttl)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`

	requestJSON, err := sonic.Marshal(job.Request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resultJSON, err := sonic.Marshal(job.Result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	_, err = s.db.ExecContext(ctx, query,
		job.JobID, job.TenantID, job.DatasetID, job.JobType, job.Status,
		job.ProcessorID, requestJSON, resultJSON, job.ErrorMsg,
		job.CreatedAt, job.UpdatedAt, job.StartedAt, job.CompletedAt, job.TTL)

	if err != nil {
		return fmt.Errorf("failed to create transformation job: %w", err)
	}

	log.Infof("Created transformation job %s for %s/%s (type: %s)", job.JobID, job.TenantID, job.DatasetID, job.JobType)
	return nil
}

// ClaimPiperTransformationJob claims a pending transformation job
func (s *PostgreSQLStorage) ClaimPiperTransformationJob(ctx context.Context, processorID string, jobTypes []PiperTransformationJobType) (*PiperTransformationJob, error) {
	// Clean up expired jobs first
	_, err := s.CleanupExpiredPiperTransformationJobs(ctx)
	if err != nil {
		log.Warnf("Failed to cleanup expired transformation jobs: %v", err)
	}

	// Convert job types to strings for query
	jobTypeStrings := make([]string, len(jobTypes))
	for i, jt := range jobTypes {
		jobTypeStrings[i] = string(jt)
	}

	// Use a transaction to atomically claim the job
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Find oldest pending job matching the job types
	query := `SELECT job_id, tenant_id, dataset_id, job_type, status, processor_id, request, result,
		error_message, created_at, updated_at, started_at, completed_at, ttl
		FROM piper_transformation_jobs
		WHERE status = 'pending' AND job_type = ANY($1) AND ttl > NOW()
		ORDER BY created_at ASC
		LIMIT 1
		FOR UPDATE SKIP LOCKED`

	job := &PiperTransformationJob{}
	var requestJSON, resultJSON []byte
	var processorIDNullable sql.NullString
	var errorMsgNullable sql.NullString
	var startedAtNullable sql.NullTime
	var completedAtNullable sql.NullTime

	err = tx.QueryRowContext(ctx, query, pq.Array(jobTypeStrings)).Scan(
		&job.JobID, &job.TenantID, &job.DatasetID, &job.JobType, &job.Status,
		&processorIDNullable, &requestJSON, &resultJSON, &errorMsgNullable,
		&job.CreatedAt, &job.UpdatedAt, &startedAtNullable, &completedAtNullable, &job.TTL)

	if err == sql.ErrNoRows {
		// No jobs available
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find pending job: %w", err)
	}

	// Unmarshal JSON fields
	if len(requestJSON) > 0 && string(requestJSON) != "null" {
		if err := sonic.Unmarshal(requestJSON, &job.Request); err != nil {
			return nil, fmt.Errorf("failed to unmarshal request: %w", err)
		}
	}
	if len(resultJSON) > 0 && string(resultJSON) != "null" {
		if err := sonic.Unmarshal(resultJSON, &job.Result); err != nil {
			return nil, fmt.Errorf("failed to unmarshal result: %w", err)
		}
	}

	if processorIDNullable.Valid {
		job.ProcessorID = processorIDNullable.String
	}
	if errorMsgNullable.Valid {
		job.ErrorMsg = errorMsgNullable.String
	}
	if startedAtNullable.Valid {
		job.StartedAt = &startedAtNullable.Time
	}
	if completedAtNullable.Valid {
		job.CompletedAt = &completedAtNullable.Time
	}

	// Update the job to running status
	now := time.Now()
	updateQuery := `UPDATE piper_transformation_jobs
		SET status = 'running', processor_id = $1, started_at = $2, updated_at = $3
		WHERE job_id = $4`

	_, err = tx.ExecContext(ctx, updateQuery, processorID, now, now, job.JobID)
	if err != nil {
		return nil, fmt.Errorf("failed to claim job: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Update the job object
	job.Status = PiperJobStatusRunning
	job.ProcessorID = processorID
	job.StartedAt = &now
	job.UpdatedAt = now

	log.Infof("Claimed transformation job %s by processor %s (type: %s)", job.JobID, processorID, job.JobType)
	return job, nil
}

// UpdatePiperTransformationJob updates an existing transformation job
func (s *PostgreSQLStorage) UpdatePiperTransformationJob(ctx context.Context, job *PiperTransformationJob) error {
	requestJSON, err := sonic.Marshal(job.Request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resultJSON, err := sonic.Marshal(job.Result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	query := `UPDATE piper_transformation_jobs
		SET status = $1, processor_id = $2, request = $3, result = $4,
		error_message = $5, updated_at = $6, started_at = $7, completed_at = $8
		WHERE job_id = $9`

	result, err := s.db.ExecContext(ctx, query,
		job.Status, job.ProcessorID, requestJSON, resultJSON,
		job.ErrorMsg, job.UpdatedAt, job.StartedAt, job.CompletedAt, job.JobID)

	if err != nil {
		return fmt.Errorf("failed to update transformation job: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("transformation job %s not found", job.JobID)
	}

	log.Debugf("Updated transformation job %s to status %s", job.JobID, job.Status)
	return nil
}

// GetPiperTransformationJob retrieves a transformation job by ID
func (s *PostgreSQLStorage) GetPiperTransformationJob(ctx context.Context, jobID string) (*PiperTransformationJob, error) {
	query := `SELECT job_id, tenant_id, dataset_id, job_type, status, processor_id, request, result,
		error_message, created_at, updated_at, started_at, completed_at, ttl
		FROM piper_transformation_jobs
		WHERE job_id = $1`

	job := &PiperTransformationJob{}
	var requestJSON, resultJSON []byte
	var processorIDNullable sql.NullString
	var errorMsgNullable sql.NullString
	var startedAtNullable sql.NullTime
	var completedAtNullable sql.NullTime

	err := s.db.QueryRowContext(ctx, query, jobID).Scan(
		&job.JobID, &job.TenantID, &job.DatasetID, &job.JobType, &job.Status,
		&processorIDNullable, &requestJSON, &resultJSON, &errorMsgNullable,
		&job.CreatedAt, &job.UpdatedAt, &startedAtNullable, &completedAtNullable, &job.TTL)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("transformation job %s not found", jobID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get transformation job: %w", err)
	}

	// Unmarshal JSON fields
	if len(requestJSON) > 0 && string(requestJSON) != "null" {
		if err := sonic.Unmarshal(requestJSON, &job.Request); err != nil {
			return nil, fmt.Errorf("failed to unmarshal request: %w", err)
		}
	}
	if len(resultJSON) > 0 && string(resultJSON) != "null" {
		if err := sonic.Unmarshal(resultJSON, &job.Result); err != nil {
			return nil, fmt.Errorf("failed to unmarshal result: %w", err)
		}
	}

	if processorIDNullable.Valid {
		job.ProcessorID = processorIDNullable.String
	}
	if errorMsgNullable.Valid {
		job.ErrorMsg = errorMsgNullable.String
	}
	if startedAtNullable.Valid {
		job.StartedAt = &startedAtNullable.Time
	}
	if completedAtNullable.Valid {
		job.CompletedAt = &completedAtNullable.Time
	}

	return job, nil
}

// ListPendingPiperTransformationJobs lists pending transformation jobs
func (s *PostgreSQLStorage) ListPendingPiperTransformationJobs(ctx context.Context, limit int) ([]*PiperTransformationJob, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `SELECT job_id, tenant_id, dataset_id, job_type, status, processor_id, request, result,
		error_message, created_at, updated_at, started_at, completed_at, ttl
		FROM piper_transformation_jobs
		WHERE status = 'pending' AND ttl > NOW()
		ORDER BY created_at ASC
		LIMIT $1`

	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list pending transformation jobs: %w", err)
	}
	defer rows.Close()

	jobs := []*PiperTransformationJob{}
	for rows.Next() {
		job := &PiperTransformationJob{}
		var requestJSON, resultJSON []byte
		var processorIDNullable sql.NullString
		var errorMsgNullable sql.NullString
		var startedAtNullable sql.NullTime
		var completedAtNullable sql.NullTime

		err := rows.Scan(
			&job.JobID, &job.TenantID, &job.DatasetID, &job.JobType, &job.Status,
			&processorIDNullable, &requestJSON, &resultJSON, &errorMsgNullable,
			&job.CreatedAt, &job.UpdatedAt, &startedAtNullable, &completedAtNullable, &job.TTL)

		if err != nil {
			return nil, fmt.Errorf("failed to scan transformation job: %w", err)
		}

		// Unmarshal JSON fields
		if len(requestJSON) > 0 && string(requestJSON) != "null" {
			if err := sonic.Unmarshal(requestJSON, &job.Request); err != nil {
				return nil, fmt.Errorf("failed to unmarshal request: %w", err)
			}
		}
		if len(resultJSON) > 0 && string(resultJSON) != "null" {
			if err := sonic.Unmarshal(resultJSON, &job.Result); err != nil {
				return nil, fmt.Errorf("failed to unmarshal result: %w", err)
			}
		}

		if processorIDNullable.Valid {
			job.ProcessorID = processorIDNullable.String
		}
		if errorMsgNullable.Valid {
			job.ErrorMsg = errorMsgNullable.String
		}
		if startedAtNullable.Valid {
			job.StartedAt = &startedAtNullable.Time
		}
		if completedAtNullable.Valid {
			job.CompletedAt = &completedAtNullable.Time
		}

		jobs = append(jobs, job)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transformation jobs: %w", err)
	}

	return jobs, nil
}

// CleanupExpiredPiperTransformationJobs removes expired transformation jobs
func (s *PostgreSQLStorage) CleanupExpiredPiperTransformationJobs(ctx context.Context) (int, error) {
	query := `DELETE FROM piper_transformation_jobs WHERE ttl < NOW()`

	result, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired transformation jobs: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected > 0 {
		log.Infof("Cleaned up %d expired transformation jobs", rowsAffected)
	}

	return int(rowsAffected), nil
}
