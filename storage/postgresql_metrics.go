package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// RecordDatasetMetric inserts a new metrics data point into the database
func (s *PostgreSQLStorage) RecordDatasetMetric(ctx context.Context, metric *DatasetMetric) error {
	// Set recorded_at if not provided
	if metric.RecordedAt.IsZero() {
		metric.RecordedAt = time.Now()
	}

	// Calculate time buckets for aggregation
	hourBucket := metric.RecordedAt.Truncate(time.Hour)
	dayBucket := time.Date(metric.RecordedAt.Year(), metric.RecordedAt.Month(), metric.RecordedAt.Day(), 0, 0, 0, 0, metric.RecordedAt.Location())

	metric.HourBucket = &hourBucket
	metric.DayBucket = &dayBucket

	// Serialize custom_metrics to JSONB
	customMetricsJSON, err := json.Marshal(metric.CustomMetrics)
	if err != nil {
		return fmt.Errorf("failed to marshal custom_metrics: %w", err)
	}

	query := `
		INSERT INTO dataset_metrics (
			tenant_id, dataset_id, recorded_at, component,
			input_bytes, output_bytes, lines_processed, error_count,
			hour_bucket, day_bucket, custom_metrics
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id`

	err = s.db.QueryRowContext(ctx, query,
		metric.TenantID,
		metric.DatasetID,
		metric.RecordedAt,
		metric.Component,
		metric.InputBytes,
		metric.OutputBytes,
		metric.LinesProcessed,
		metric.ErrorCount,
		metric.HourBucket,
		metric.DayBucket,
		customMetricsJSON,
	).Scan(&metric.ID)

	if err != nil {
		return fmt.Errorf("failed to insert dataset metric: %w", err)
	}

	return nil
}

// QueryDatasetMetrics retrieves metrics based on filter criteria
func (s *PostgreSQLStorage) QueryDatasetMetrics(ctx context.Context, filter MetricsQueryFilter) ([]*DatasetMetric, error) {
	// Validate required fields
	if filter.TenantID == "" || filter.DatasetID == "" {
		return nil, fmt.Errorf("tenant_id and dataset_id are required")
	}
	if filter.StartTime.IsZero() || filter.EndTime.IsZero() {
		return nil, fmt.Errorf("start_time and end_time are required")
	}

	// Set default limit
	if filter.Limit == 0 {
		filter.Limit = 1000
	}

	query := `
		SELECT
			id, tenant_id, dataset_id, recorded_at, component,
			input_bytes, output_bytes, lines_processed, error_count,
			hour_bucket, day_bucket, custom_metrics
		FROM dataset_metrics
		WHERE tenant_id = $1
			AND dataset_id = $2
			AND recorded_at >= $3
			AND recorded_at <= $4`

	args := []interface{}{filter.TenantID, filter.DatasetID, filter.StartTime, filter.EndTime}

	// Add optional component filter
	if filter.Component != "" {
		query += " AND component = $5"
		args = append(args, filter.Component)
	}

	query += " ORDER BY recorded_at DESC LIMIT $" + fmt.Sprintf("%d", len(args)+1)
	args = append(args, filter.Limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query metrics: %w", err)
	}
	defer rows.Close()

	var metrics []*DatasetMetric
	for rows.Next() {
		var metric DatasetMetric
		var customMetricsJSON []byte

		err := rows.Scan(
			&metric.ID,
			&metric.TenantID,
			&metric.DatasetID,
			&metric.RecordedAt,
			&metric.Component,
			&metric.InputBytes,
			&metric.OutputBytes,
			&metric.LinesProcessed,
			&metric.ErrorCount,
			&metric.HourBucket,
			&metric.DayBucket,
			&customMetricsJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan metric row: %w", err)
		}

		// Deserialize custom_metrics
		if len(customMetricsJSON) > 0 {
			if err := json.Unmarshal(customMetricsJSON, &metric.CustomMetrics); err != nil {
				return nil, fmt.Errorf("failed to unmarshal custom_metrics: %w", err)
			}
		}

		metrics = append(metrics, &metric)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating metrics rows: %w", err)
	}

	return metrics, nil
}

// GetAggregatedMetrics returns aggregated metrics grouped by component
func (s *PostgreSQLStorage) GetAggregatedMetrics(ctx context.Context, filter MetricsQueryFilter) ([]*ComponentMetrics, error) {
	// Validate required fields
	if filter.TenantID == "" || filter.DatasetID == "" {
		return nil, fmt.Errorf("tenant_id and dataset_id are required")
	}
	if filter.StartTime.IsZero() || filter.EndTime.IsZero() {
		return nil, fmt.Errorf("start_time and end_time are required")
	}

	query := `
		SELECT
			component,
			SUM(input_bytes) as total_input_bytes,
			SUM(output_bytes) as total_output_bytes,
			SUM(lines_processed) as total_lines_processed,
			SUM(error_count) as total_error_count,
			MIN(recorded_at) as start_time,
			MAX(recorded_at) as end_time
		FROM dataset_metrics
		WHERE tenant_id = $1
			AND dataset_id = $2
			AND recorded_at >= $3
			AND recorded_at <= $4`

	args := []interface{}{filter.TenantID, filter.DatasetID, filter.StartTime, filter.EndTime}

	// Add optional component filter
	if filter.Component != "" {
		query += " AND component = $5"
		args = append(args, filter.Component)
	}

	query += " GROUP BY component ORDER BY component"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query aggregated metrics: %w", err)
	}
	defer rows.Close()

	var results []*ComponentMetrics
	for rows.Next() {
		var cm ComponentMetrics
		var startTime, endTime sql.NullTime

		err := rows.Scan(
			&cm.Component,
			&cm.InputBytes,
			&cm.OutputBytes,
			&cm.LinesProcessed,
			&cm.ErrorCount,
			&startTime,
			&endTime,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan aggregated metric row: %w", err)
		}

		if startTime.Valid {
			cm.StartTime = startTime.Time
		}
		if endTime.Valid {
			cm.EndTime = endTime.Time
		}

		results = append(results, &cm)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating aggregated metrics rows: %w", err)
	}

	return results, nil
}

// CleanupOldMetrics deletes metrics older than the specified time
func (s *PostgreSQLStorage) CleanupOldMetrics(ctx context.Context, olderThan time.Time) (int64, error) {
	query := `DELETE FROM dataset_metrics WHERE recorded_at < $1`

	result, err := s.db.ExecContext(ctx, query, olderThan)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup old metrics: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected, nil
}
