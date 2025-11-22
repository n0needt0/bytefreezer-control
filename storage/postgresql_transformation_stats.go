package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/n0needt0/go-goodies/log"
)

// UpsertActiveTransformation creates or updates the active transformation configuration
func (s *PostgreSQLStorage) UpsertActiveTransformation(ctx context.Context, transformation *ActiveTransformation) error {
	if transformation == nil {
		return fmt.Errorf("transformation cannot be nil")
	}

	if transformation.TenantID == "" || transformation.DatasetID == "" {
		return fmt.Errorf("tenant_id and dataset_id are required")
	}

	// Marshal filters to JSONB
	filtersJSON, err := json.Marshal(transformation.Filters)
	if err != nil {
		return fmt.Errorf("failed to marshal filters: %w", err)
	}

	query := `
		INSERT INTO control_active_transformations (
			tenant_id, dataset_id, enabled, filters, version, activated_at
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (tenant_id, dataset_id)
		DO UPDATE SET
			enabled = EXCLUDED.enabled,
			filters = EXCLUDED.filters,
			version = EXCLUDED.version,
			activated_at = EXCLUDED.activated_at,
			updated_at = NOW()
		RETURNING updated_at`

	err = s.db.QueryRowContext(ctx, query,
		transformation.TenantID,
		transformation.DatasetID,
		transformation.Enabled,
		filtersJSON,
		transformation.Version,
		transformation.ActivatedAt,
	).Scan(&transformation.UpdatedAt)

	if err != nil {
		log.Errorf("Failed to upsert active transformation for %s/%s: %v", transformation.TenantID, transformation.DatasetID, err)
		return fmt.Errorf("failed to upsert active transformation: %w", err)
	}

	log.Infof("Upserted active transformation for %s/%s (enabled: %v, filters: %d)",
		transformation.TenantID, transformation.DatasetID, transformation.Enabled, len(transformation.Filters))

	return nil
}

// GetActiveTransformation retrieves the active transformation configuration
func (s *PostgreSQLStorage) GetActiveTransformation(ctx context.Context, tenantID, datasetID string) (*ActiveTransformation, error)  {
	if tenantID == "" || datasetID == "" {
		return nil, fmt.Errorf("tenant_id and dataset_id are required")
	}

	query := `
		SELECT tenant_id, dataset_id, enabled, filters, version, activated_at, updated_at
		FROM control_active_transformations
		WHERE tenant_id = $1 AND dataset_id = $2`

	var transformation ActiveTransformation
	var filtersJSON []byte

	err := s.db.QueryRowContext(ctx, query, tenantID, datasetID).Scan(
		&transformation.TenantID,
		&transformation.DatasetID,
		&transformation.Enabled,
		&filtersJSON,
		&transformation.Version,
		&transformation.ActivatedAt,
		&transformation.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // No active transformation
	}

	if err != nil {
		log.Errorf("Failed to get active transformation for %s/%s: %v", tenantID, datasetID, err)
		return nil, fmt.Errorf("failed to get active transformation: %w", err)
	}

	// Unmarshal filters
	if err := json.Unmarshal(filtersJSON, &transformation.Filters); err != nil {
		log.Errorf("Failed to unmarshal filters for %s/%s: %v", tenantID, datasetID, err)
		return nil, fmt.Errorf("failed to unmarshal filters: %w", err)
	}

	return &transformation, nil
}

// SaveTransformationStats saves transformation statistics (replaces old stats for same tenant/dataset)
func (s *PostgreSQLStorage) SaveTransformationStats(ctx context.Context, stats *TransformationStats) error {
	if stats == nil {
		return fmt.Errorf("stats cannot be nil")
	}

	if stats.TenantID == "" || stats.DatasetID == "" {
		return fmt.Errorf("tenant_id and dataset_id are required")
	}

	query := `
		INSERT INTO control_transformation_stats (
			tenant_id, dataset_id, total_processed, success_count, error_count,
			skipped_count, avg_rows_per_sec, last_error, last_processed, reported_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
		RETURNING id, reported_at`

	err := s.db.QueryRowContext(ctx, query,
		stats.TenantID,
		stats.DatasetID,
		stats.TotalProcessed,
		stats.SuccessCount,
		stats.ErrorCount,
		stats.SkippedCount,
		stats.AvgRowsPerSec,
		stats.LastError,
		stats.LastProcessed,
	).Scan(&stats.ID, &stats.ReportedAt)

	if err != nil {
		log.Errorf("Failed to save transformation stats for %s/%s: %v", stats.TenantID, stats.DatasetID, err)
		return fmt.Errorf("failed to save transformation stats: %w", err)
	}

	log.Debugf("Saved transformation stats for %s/%s (processed: %d, success: %d, error: %d)",
		stats.TenantID, stats.DatasetID, stats.TotalProcessed, stats.SuccessCount, stats.ErrorCount)

	return nil
}

// GetLatestTransformationStats retrieves the most recent statistics for a dataset
func (s *PostgreSQLStorage) GetLatestTransformationStats(ctx context.Context, tenantID, datasetID string) (*TransformationStats, error) {
	if tenantID == "" || datasetID == "" {
		return nil, fmt.Errorf("tenant_id and dataset_id are required")
	}

	query := `
		SELECT id, tenant_id, dataset_id, total_processed, success_count, error_count,
		       skipped_count, avg_rows_per_sec, last_error, last_processed, reported_at
		FROM control_transformation_stats
		WHERE tenant_id = $1 AND dataset_id = $2
		ORDER BY reported_at DESC
		LIMIT 1`

	var stats TransformationStats

	err := s.db.QueryRowContext(ctx, query, tenantID, datasetID).Scan(
		&stats.ID,
		&stats.TenantID,
		&stats.DatasetID,
		&stats.TotalProcessed,
		&stats.SuccessCount,
		&stats.ErrorCount,
		&stats.SkippedCount,
		&stats.AvgRowsPerSec,
		&stats.LastError,
		&stats.LastProcessed,
		&stats.ReportedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // No stats yet
	}

	if err != nil {
		log.Errorf("Failed to get latest transformation stats for %s/%s: %v", tenantID, datasetID, err)
		return nil, fmt.Errorf("failed to get latest transformation stats: %w", err)
	}

	return &stats, nil
}

// CleanupOldTransformationStats removes old stats entries, keeping only the most recent N entries
func (s *PostgreSQLStorage) CleanupOldTransformationStats(ctx context.Context, tenantID, datasetID string, keepCount int) error {
	if tenantID == "" || datasetID == "" {
		return fmt.Errorf("tenant_id and dataset_id are required")
	}

	if keepCount <= 0 {
		keepCount = 100 // Default keep last 100 stats entries
	}

	query := `
		DELETE FROM control_transformation_stats
		WHERE id IN (
			SELECT id FROM control_transformation_stats
			WHERE tenant_id = $1 AND dataset_id = $2
			ORDER BY reported_at DESC
			OFFSET $3
		)`

	result, err := s.db.ExecContext(ctx, query, tenantID, datasetID, keepCount)
	if err != nil {
		log.Errorf("Failed to cleanup old transformation stats for %s/%s: %v", tenantID, datasetID, err)
		return fmt.Errorf("failed to cleanup old transformation stats: %w", err)
	}

	rowsDeleted, _ := result.RowsAffected()
	if rowsDeleted > 0 {
		log.Debugf("Cleaned up %d old transformation stats entries for %s/%s (kept %d)",
			rowsDeleted, tenantID, datasetID, keepCount)
	}

	return nil
}
