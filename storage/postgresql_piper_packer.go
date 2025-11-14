package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	"github.com/n0needt0/go-goodies/log"
)

// ============================================================================
// PIPER FILE LOCK OPERATIONS
// ============================================================================

// AcquireFileLock attempts to acquire a lock for a file
func (s *PostgreSQLStorage) AcquireFileLock(ctx context.Context, lock *PiperFileLock) error {
	// Set timestamps
	now := time.Now()
	lock.LockTimestamp = now
	lock.LastHeartbeat = now

	// First try to clean up expired locks for this file
	cleanupQuery := `DELETE FROM piper_file_locks
		WHERE tenant_id = $1 AND dataset_id = $2 AND file_key = $3 AND ttl < NOW()`
	_, err := s.db.ExecContext(ctx, cleanupQuery, lock.TenantID, lock.DatasetID, lock.FileKey)
	if err != nil {
		log.Warnf("Failed to cleanup expired file locks: %v", err)
	}

	// Try to insert the lock
	query := `INSERT INTO piper_file_locks
		(tenant_id, dataset_id, file_key, locked_by, lock_timestamp, last_heartbeat, ttl)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (tenant_id, dataset_id, file_key) DO NOTHING
		RETURNING lock_id`

	err = s.db.QueryRowContext(ctx, query,
		lock.TenantID, lock.DatasetID, lock.FileKey, lock.LockedBy,
		lock.LockTimestamp, lock.LastHeartbeat, lock.TTL,
	).Scan(&lock.LockID)

	if err == sql.ErrNoRows {
		return fmt.Errorf("file already locked")
	}
	if err != nil {
		return fmt.Errorf("failed to acquire file lock: %w", err)
	}

	log.Debugf("Acquired file lock for %s/%s/%s by %s", lock.TenantID, lock.DatasetID, lock.FileKey, lock.LockedBy)
	return nil
}

// ReleaseFileLock releases a file lock
func (s *PostgreSQLStorage) ReleaseFileLock(ctx context.Context, tenantID, datasetID, fileKey, lockedBy string) error {
	query := `DELETE FROM piper_file_locks
		WHERE tenant_id = $1 AND dataset_id = $2 AND file_key = $3 AND locked_by = $4`

	result, err := s.db.ExecContext(ctx, query, tenantID, datasetID, fileKey, lockedBy)
	if err != nil {
		return fmt.Errorf("failed to release file lock: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no lock found to release or not owned by %s", lockedBy)
	}

	log.Debugf("Released file lock for %s/%s/%s by %s", tenantID, datasetID, fileKey, lockedBy)
	return nil
}

// CheckFileLock checks if a file is locked
func (s *PostgreSQLStorage) CheckFileLock(ctx context.Context, tenantID, datasetID, fileKey string) (*PiperFileLock, error) {
	// Clean up expired locks first
	cleanupQuery := `DELETE FROM piper_file_locks
		WHERE tenant_id = $1 AND dataset_id = $2 AND file_key = $3 AND ttl < NOW()`
	_, err := s.db.ExecContext(ctx, cleanupQuery, tenantID, datasetID, fileKey)
	if err != nil {
		log.Warnf("Failed to cleanup expired file locks: %v", err)
	}

	query := `SELECT lock_id, tenant_id, dataset_id, file_key, locked_by,
		lock_timestamp, last_heartbeat, ttl
		FROM piper_file_locks
		WHERE tenant_id = $1 AND dataset_id = $2 AND file_key = $3 AND ttl > NOW()`

	lock := &PiperFileLock{}
	err = s.db.QueryRowContext(ctx, query, tenantID, datasetID, fileKey).Scan(
		&lock.LockID, &lock.TenantID, &lock.DatasetID, &lock.FileKey, &lock.LockedBy,
		&lock.LockTimestamp, &lock.LastHeartbeat, &lock.TTL,
	)

	if err == sql.ErrNoRows {
		return nil, nil // No lock exists
	}
	if err != nil {
		return nil, fmt.Errorf("failed to check file lock: %w", err)
	}

	return lock, nil
}

// CleanupExpiredFileLocks removes all expired file locks
func (s *PostgreSQLStorage) CleanupExpiredFileLocks(ctx context.Context) (int64, error) {
	query := `DELETE FROM piper_file_locks WHERE ttl < NOW()`

	result, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired file locks: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected > 0 {
		log.Infof("Cleaned up %d expired file locks", rowsAffected)
	}

	return rowsAffected, nil
}

// CleanupStaleFileLocks removes file locks with stale heartbeats
func (s *PostgreSQLStorage) CleanupStaleFileLocks(ctx context.Context, thresholdMinutes int) (int64, error) {
	query := `DELETE FROM piper_file_locks
		WHERE last_heartbeat < NOW() - INTERVAL '1 minute' * $1`

	result, err := s.db.ExecContext(ctx, query, thresholdMinutes)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup stale file locks: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected > 0 {
		log.Infof("Cleaned up %d stale file locks", rowsAffected)
	}

	return rowsAffected, nil
}

// ============================================================================
// PIPER JOB RECORD OPERATIONS
// ============================================================================

// CreatePiperJob creates a new piper job record
func (s *PostgreSQLStorage) CreatePiperJob(ctx context.Context, job *PiperJobRecord) error {
	job.CreatedAt = time.Now()
	job.UpdatedAt = job.CreatedAt

	query := `INSERT INTO piper_job_records
		(job_id, tenant_id, dataset_id, status, source_files, processor_type, processor_id,
		 output_file, error_message, records_processed, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	sourceFilesJSON, err := sonic.Marshal(job.SourceFiles)
	if err != nil {
		return fmt.Errorf("failed to marshal source files: %w", err)
	}

	_, err = s.db.ExecContext(ctx, query,
		job.JobID, job.TenantID, job.DatasetID, job.Status, sourceFilesJSON,
		job.ProcessorType, job.ProcessorID, job.OutputFile, job.ErrorMessage,
		job.RecordsProcessed, job.CreatedAt, job.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create piper job: %w", err)
	}

	log.Debugf("Created piper job %s for tenant %s", job.JobID, job.TenantID)
	return nil
}

// UpdatePiperJobStatus updates a piper job's status
func (s *PostgreSQLStorage) UpdatePiperJobStatus(ctx context.Context, jobID, status, processorID, outputFile, errorMessage string, recordsProcessed int64) error {
	// NOTE: processor_id is NOT updated here - it's set during job creation and should not change
	query := `UPDATE piper_job_records
		SET status = $1, output_file = $2, error_message = $3,
		    records_processed = $4, updated_at = NOW()
		WHERE job_id = $5`

	result, err := s.db.ExecContext(ctx, query, status, outputFile, errorMessage, recordsProcessed, jobID)
	if err != nil {
		return fmt.Errorf("failed to update piper job status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("job not found: %s", jobID)
	}

	log.Debugf("Updated piper job %s to status %s", jobID, status)
	return nil
}

// GetPiperJob retrieves a piper job by ID
func (s *PostgreSQLStorage) GetPiperJob(ctx context.Context, jobID string) (*PiperJobRecord, error) {
	query := `SELECT job_id, tenant_id, dataset_id, status, source_files, processor_type,
		processor_id, output_file, error_message, records_processed, created_at, updated_at
		FROM piper_job_records WHERE job_id = $1`

	job := &PiperJobRecord{}
	var sourceFilesJSON []byte

	err := s.db.QueryRowContext(ctx, query, jobID).Scan(
		&job.JobID, &job.TenantID, &job.DatasetID, &job.Status, &sourceFilesJSON,
		&job.ProcessorType, &job.ProcessorID, &job.OutputFile, &job.ErrorMessage,
		&job.RecordsProcessed, &job.CreatedAt, &job.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("job not found: %s", jobID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get piper job: %w", err)
	}

	if len(sourceFilesJSON) > 0 {
		if err := sonic.Unmarshal(sourceFilesJSON, &job.SourceFiles); err != nil {
			return nil, fmt.Errorf("failed to unmarshal source files: %w", err)
		}
	}

	return job, nil
}

// GetPiperJobsByStatus retrieves piper jobs by status
func (s *PostgreSQLStorage) GetPiperJobsByStatus(ctx context.Context, status string) ([]*PiperJobRecord, error) {
	query := `SELECT job_id, tenant_id, dataset_id, status, source_files, processor_type,
		processor_id, output_file, error_message, records_processed, created_at, updated_at
		FROM piper_job_records WHERE status = $1 ORDER BY created_at ASC`

	rows, err := s.db.QueryContext(ctx, query, status)
	if err != nil {
		return nil, fmt.Errorf("failed to query piper jobs by status: %w", err)
	}
	defer rows.Close()

	var jobs []*PiperJobRecord
	for rows.Next() {
		job := &PiperJobRecord{}
		var sourceFilesJSON []byte

		err := rows.Scan(
			&job.JobID, &job.TenantID, &job.DatasetID, &job.Status, &sourceFilesJSON,
			&job.ProcessorType, &job.ProcessorID, &job.OutputFile, &job.ErrorMessage,
			&job.RecordsProcessed, &job.CreatedAt, &job.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan piper job: %w", err)
		}

		if len(sourceFilesJSON) > 0 {
			if err := sonic.Unmarshal(sourceFilesJSON, &job.SourceFiles); err != nil {
				return nil, fmt.Errorf("failed to unmarshal source files: %w", err)
			}
		}

		jobs = append(jobs, job)
	}

	return jobs, rows.Err()
}

// GetPiperJobsForTenant retrieves piper jobs for a tenant
func (s *PostgreSQLStorage) GetPiperJobsForTenant(ctx context.Context, tenantID string) ([]*PiperJobRecord, error) {
	query := `SELECT job_id, tenant_id, dataset_id, status, source_files, processor_type,
		processor_id, output_file, error_message, records_processed, created_at, updated_at
		FROM piper_job_records WHERE tenant_id = $1 ORDER BY created_at DESC`

	rows, err := s.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query piper jobs for tenant: %w", err)
	}
	defer rows.Close()

	var jobs []*PiperJobRecord
	for rows.Next() {
		job := &PiperJobRecord{}
		var sourceFilesJSON []byte

		err := rows.Scan(
			&job.JobID, &job.TenantID, &job.DatasetID, &job.Status, &sourceFilesJSON,
			&job.ProcessorType, &job.ProcessorID, &job.OutputFile, &job.ErrorMessage,
			&job.RecordsProcessed, &job.CreatedAt, &job.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan piper job: %w", err)
		}

		if len(sourceFilesJSON) > 0 {
			if err := sonic.Unmarshal(sourceFilesJSON, &job.SourceFiles); err != nil {
				return nil, fmt.Errorf("failed to unmarshal source files: %w", err)
			}
		}

		jobs = append(jobs, job)
	}

	return jobs, rows.Err()
}

// CleanupOldPiperJobs removes old completed/failed piper jobs
func (s *PostgreSQLStorage) CleanupOldPiperJobs(ctx context.Context, olderThanDays int) (int64, error) {
	query := `DELETE FROM piper_job_records
		WHERE status IN ('completed', 'failed')
		AND updated_at < NOW() - INTERVAL '1 day' * $1`

	result, err := s.db.ExecContext(ctx, query, olderThanDays)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup old piper jobs: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected > 0 {
		log.Infof("Cleaned up %d old piper jobs", rowsAffected)
	}

	return rowsAffected, nil
}

// ============================================================================
// PIPER PIPELINE CONFIGURATION CACHE OPERATIONS
// ============================================================================

// CachePipelineConfiguration caches a pipeline configuration
func (s *PostgreSQLStorage) CachePipelineConfiguration(ctx context.Context, cache *PiperPipelineCache) error {
	cache.CachedAt = time.Now()
	cache.ConfigKey = fmt.Sprintf("%s:%s", cache.TenantID, cache.DatasetID)

	configJSON, err := sonic.Marshal(cache.Configuration)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	query := `INSERT INTO piper_pipeline_configurations
		(config_key, tenant_id, dataset_id, configuration, cached_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (config_key) DO UPDATE SET
			configuration = EXCLUDED.configuration,
			cached_at = EXCLUDED.cached_at,
			expires_at = EXCLUDED.expires_at`

	_, err = s.db.ExecContext(ctx, query,
		cache.ConfigKey, cache.TenantID, cache.DatasetID, configJSON,
		cache.CachedAt, cache.ExpiresAt,
	)

	if err != nil {
		return fmt.Errorf("failed to cache pipeline configuration: %w", err)
	}

	log.Debugf("Cached pipeline configuration for %s/%s", cache.TenantID, cache.DatasetID)
	return nil
}

// GetCachedPipelineConfiguration retrieves a cached pipeline configuration
func (s *PostgreSQLStorage) GetCachedPipelineConfiguration(ctx context.Context, tenantID, datasetID string) (*PiperPipelineCache, error) {
	configKey := fmt.Sprintf("%s:%s", tenantID, datasetID)

	// Clean up expired cache first
	cleanupQuery := `DELETE FROM piper_pipeline_configurations WHERE config_key = $1 AND expires_at < NOW()`
	_, err := s.db.ExecContext(ctx, cleanupQuery, configKey)
	if err != nil {
		log.Warnf("Failed to cleanup expired pipeline cache: %v", err)
	}

	query := `SELECT config_key, tenant_id, dataset_id, configuration, cached_at, expires_at
		FROM piper_pipeline_configurations WHERE config_key = $1 AND expires_at > NOW()`

	cache := &PiperPipelineCache{}
	var configJSON []byte

	err = s.db.QueryRowContext(ctx, query, configKey).Scan(
		&cache.ConfigKey, &cache.TenantID, &cache.DatasetID, &configJSON,
		&cache.CachedAt, &cache.ExpiresAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // No cached config exists
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get cached pipeline configuration: %w", err)
	}

	if len(configJSON) > 0 {
		if err := sonic.Unmarshal(configJSON, &cache.Configuration); err != nil {
			return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
		}
	}

	return cache, nil
}

// InvalidatePipelineConfiguration removes a cached pipeline configuration
func (s *PostgreSQLStorage) InvalidatePipelineConfiguration(ctx context.Context, tenantID, datasetID string) error {
	configKey := fmt.Sprintf("%s:%s", tenantID, datasetID)

	query := `DELETE FROM piper_pipeline_configurations WHERE config_key = $1`

	_, err := s.db.ExecContext(ctx, query, configKey)
	if err != nil {
		return fmt.Errorf("failed to invalidate pipeline configuration: %w", err)
	}

	log.Debugf("Invalidated pipeline configuration for %s/%s", tenantID, datasetID)
	return nil
}

// ListCachedPipelines retrieves all cached pipeline configurations
func (s *PostgreSQLStorage) ListCachedPipelines(ctx context.Context) ([]*PiperPipelineCache, error) {
	query := `SELECT config_key, tenant_id, dataset_id, configuration, cached_at, expires_at
		FROM piper_pipeline_configurations WHERE expires_at > NOW() ORDER BY cached_at DESC`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list cached pipelines: %w", err)
	}
	defer rows.Close()

	var caches []*PiperPipelineCache
	for rows.Next() {
		cache := &PiperPipelineCache{}
		var configJSON []byte

		err := rows.Scan(
			&cache.ConfigKey, &cache.TenantID, &cache.DatasetID, &configJSON,
			&cache.CachedAt, &cache.ExpiresAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan cached pipeline: %w", err)
		}

		if len(configJSON) > 0 {
			if err := sonic.Unmarshal(configJSON, &cache.Configuration); err != nil {
				return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
			}
		}

		caches = append(caches, cache)
	}

	return caches, rows.Err()
}

// CleanupExpiredPipelineCache removes expired pipeline cache entries
func (s *PostgreSQLStorage) CleanupExpiredPipelineCache(ctx context.Context) (int64, error) {
	query := `DELETE FROM piper_pipeline_configurations WHERE expires_at < NOW()`

	result, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired pipeline cache: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected > 0 {
		log.Infof("Cleaned up %d expired pipeline cache entries", rowsAffected)
	}

	return rowsAffected, nil
}

// ============================================================================
// PIPER TENANT CACHE OPERATIONS
// ============================================================================

// CacheTenant caches tenant information
func (s *PostgreSQLStorage) CacheTenant(ctx context.Context, cache *PiperTenantCache) error {
	cache.CachedAt = time.Now()

	tenantDataJSON, err := sonic.Marshal(cache.TenantData)
	if err != nil {
		return fmt.Errorf("failed to marshal tenant data: %w", err)
	}

	query := `INSERT INTO piper_tenants_cache
		(tenant_id, tenant_data, cached_at, expires_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (tenant_id) DO UPDATE SET
			tenant_data = EXCLUDED.tenant_data,
			cached_at = EXCLUDED.cached_at,
			expires_at = EXCLUDED.expires_at`

	_, err = s.db.ExecContext(ctx, query,
		cache.TenantID, tenantDataJSON, cache.CachedAt, cache.ExpiresAt,
	)

	if err != nil {
		return fmt.Errorf("failed to cache tenant: %w", err)
	}

	log.Debugf("Cached tenant %s", cache.TenantID)
	return nil
}

// GetCachedTenants retrieves all cached tenants
func (s *PostgreSQLStorage) GetCachedTenants(ctx context.Context) ([]*PiperTenantCache, error) {
	// Clean up expired cache first
	cleanupQuery := `DELETE FROM piper_tenants_cache WHERE expires_at < NOW()`
	_, err := s.db.ExecContext(ctx, cleanupQuery)
	if err != nil {
		log.Warnf("Failed to cleanup expired tenant cache: %v", err)
	}

	query := `SELECT tenant_id, tenant_data, cached_at, expires_at
		FROM piper_tenants_cache WHERE expires_at > NOW() ORDER BY cached_at DESC`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get cached tenants: %w", err)
	}
	defer rows.Close()

	var caches []*PiperTenantCache
	for rows.Next() {
		cache := &PiperTenantCache{}
		var tenantDataJSON []byte

		err := rows.Scan(
			&cache.TenantID, &tenantDataJSON, &cache.CachedAt, &cache.ExpiresAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan cached tenant: %w", err)
		}

		if len(tenantDataJSON) > 0 {
			if err := sonic.Unmarshal(tenantDataJSON, &cache.TenantData); err != nil {
				return nil, fmt.Errorf("failed to unmarshal tenant data: %w", err)
			}
		}

		caches = append(caches, cache)
	}

	return caches, rows.Err()
}

// InvalidateTenantCache removes all cached tenant information
func (s *PostgreSQLStorage) InvalidateTenantCache(ctx context.Context) error {
	query := `DELETE FROM piper_tenants_cache`

	result, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to invalidate tenant cache: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	log.Infof("Invalidated %d cached tenants", rowsAffected)
	return nil
}

// CleanupExpiredTenantCache removes expired tenant cache entries
func (s *PostgreSQLStorage) CleanupExpiredTenantCache(ctx context.Context) (int64, error) {
	query := `DELETE FROM piper_tenants_cache WHERE expires_at < NOW()`

	result, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired tenant cache: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected > 0 {
		log.Infof("Cleaned up %d expired tenant cache entries", rowsAffected)
	}

	return rowsAffected, nil
}
