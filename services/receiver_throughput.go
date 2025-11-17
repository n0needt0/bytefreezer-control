package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// ReceiverThroughputService manages receiver throughput metrics
type ReceiverThroughputService struct {
	db *sql.DB
}

// ThroughputMetric represents a minute bucket of receiver metrics
type ThroughputMetric struct {
	ID              int64     `json:"id"`
	AccountID       string    `json:"account_id,omitempty"`
	TenantID        string    `json:"tenant_id"`
	DatasetID       string    `json:"dataset_id"`
	MinuteTimestamp time.Time `json:"minute"`
	RequestCount    int       `json:"request_count"`
	BytesReceived   int64     `json:"bytes_received"`
	LinesReceived   int64     `json:"lines_received"`
	BytesStored     int64     `json:"bytes_stored"`
	SuccessCount    int       `json:"success_count"`
	FailureCount    int       `json:"failure_count"`
}

// ThroughputSummary provides aggregated statistics
type ThroughputSummary struct {
	TimeRangeMinutes      int                `json:"time_range_minutes"`
	DataPoints            []*ThroughputMetric `json:"data_points"`
	AvgRequestsPerMinute  float64            `json:"avg_requests_per_minute"`
	AvgBytesPerMinute     float64            `json:"avg_bytes_per_minute"`
	AvgLinesPerMinute     float64            `json:"avg_lines_per_minute"`
	TotalRequests         int64              `json:"total_requests"`
	TotalBytes            int64              `json:"total_bytes"`
	TotalLines            int64              `json:"total_lines"`
	TotalSuccesses        int64              `json:"total_successes"`
	TotalFailures         int64              `json:"total_failures"`
	SuccessRate           float64            `json:"success_rate"`
}

// NewReceiverThroughputService creates a new service instance
func NewReceiverThroughputService(db *sql.DB) *ReceiverThroughputService {
	return &ReceiverThroughputService{db: db}
}

// RecordThroughput records or updates throughput for the current minute bucket
func (s *ReceiverThroughputService) RecordThroughput(ctx context.Context, accountID, tenantID, datasetID string,
	bytesReceived, linesReceived, bytesStored int64, success bool) error {

	// Truncate current time to minute boundary
	minuteTimestamp := time.Now().Truncate(time.Minute)

	successCount := 0
	failureCount := 0
	if success {
		successCount = 1
	} else {
		failureCount = 1
	}

	query := `
		INSERT INTO receiver_throughput (
			account_id, tenant_id, dataset_id, minute_timestamp,
			request_count, bytes_received, lines_received, bytes_stored,
			success_count, failure_count, updated_at
		) VALUES ($1, $2, $3, $4, 1, $5, $6, $7, $8, $9, NOW())
		ON CONFLICT (tenant_id, dataset_id, minute_timestamp)
		DO UPDATE SET
			request_count = receiver_throughput.request_count + 1,
			bytes_received = receiver_throughput.bytes_received + $5,
			lines_received = receiver_throughput.lines_received + $6,
			bytes_stored = receiver_throughput.bytes_stored + $7,
			success_count = receiver_throughput.success_count + $8,
			failure_count = receiver_throughput.failure_count + $9,
			updated_at = NOW()
	`

	_, err := s.db.ExecContext(ctx, query,
		accountID, tenantID, datasetID, minuteTimestamp,
		bytesReceived, linesReceived, bytesStored,
		successCount, failureCount,
	)

	return err
}

// GetThroughput retrieves throughput metrics for a time range
func (s *ReceiverThroughputService) GetThroughput(ctx context.Context, accountID, tenantID, datasetID string, minutes int) (*ThroughputSummary, error) {
	query := `
		SELECT
			id, account_id, tenant_id, dataset_id, minute_timestamp,
			request_count, bytes_received, lines_received, bytes_stored,
			success_count, failure_count
		FROM receiver_throughput
		WHERE tenant_id = $1
			AND dataset_id = $2
			AND minute_timestamp >= NOW() - INTERVAL '1 minute' * $3
	`

	// Add account filtering for non-system users
	if accountID != "" {
		query += " AND account_id = $4"
	}

	query += " ORDER BY minute_timestamp DESC"

	var rows *sql.Rows
	var err error

	if accountID != "" {
		rows, err = s.db.QueryContext(ctx, query, tenantID, datasetID, minutes, accountID)
	} else {
		rows, err = s.db.QueryContext(ctx, query, tenantID, datasetID, minutes)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to query throughput: %w", err)
	}
	defer rows.Close()

	var dataPoints []*ThroughputMetric
	var totalRequests, totalBytes, totalLines int64
	var totalSuccesses, totalFailures int64

	for rows.Next() {
		metric := &ThroughputMetric{}
		var accountIDNull sql.NullString

		err := rows.Scan(
			&metric.ID,
			&accountIDNull,
			&metric.TenantID,
			&metric.DatasetID,
			&metric.MinuteTimestamp,
			&metric.RequestCount,
			&metric.BytesReceived,
			&metric.LinesReceived,
			&metric.BytesStored,
			&metric.SuccessCount,
			&metric.FailureCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan throughput: %w", err)
		}

		if accountIDNull.Valid {
			metric.AccountID = accountIDNull.String
		}

		dataPoints = append(dataPoints, metric)

		// Accumulate totals
		totalRequests += int64(metric.RequestCount)
		totalBytes += metric.BytesReceived
		totalLines += metric.LinesReceived
		totalSuccesses += int64(metric.SuccessCount)
		totalFailures += int64(metric.FailureCount)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	// Calculate averages
	numDataPoints := len(dataPoints)
	avgRequestsPerMinute := 0.0
	avgBytesPerMinute := 0.0
	avgLinesPerMinute := 0.0
	successRate := 0.0

	if numDataPoints > 0 {
		avgRequestsPerMinute = float64(totalRequests) / float64(numDataPoints)
		avgBytesPerMinute = float64(totalBytes) / float64(numDataPoints)
		avgLinesPerMinute = float64(totalLines) / float64(numDataPoints)
	}

	if totalRequests > 0 {
		successRate = float64(totalSuccesses) / float64(totalRequests)
	}

	summary := &ThroughputSummary{
		TimeRangeMinutes:     minutes,
		DataPoints:           dataPoints,
		AvgRequestsPerMinute: avgRequestsPerMinute,
		AvgBytesPerMinute:    avgBytesPerMinute,
		AvgLinesPerMinute:    avgLinesPerMinute,
		TotalRequests:        totalRequests,
		TotalBytes:           totalBytes,
		TotalLines:           totalLines,
		TotalSuccesses:       totalSuccesses,
		TotalFailures:        totalFailures,
		SuccessRate:          successRate,
	}

	return summary, nil
}

// GetThroughputByAccount retrieves throughput for all datasets in an account
func (s *ReceiverThroughputService) GetThroughputByAccount(ctx context.Context, accountID string, minutes int) ([]*ThroughputMetric, error) {
	query := `
		SELECT
			id, account_id, tenant_id, dataset_id, minute_timestamp,
			request_count, bytes_received, lines_received, bytes_stored,
			success_count, failure_count
		FROM receiver_throughput
		WHERE account_id = $1
			AND minute_timestamp >= NOW() - INTERVAL '1 minute' * $2
		ORDER BY minute_timestamp DESC, tenant_id, dataset_id
	`

	rows, err := s.db.QueryContext(ctx, query, accountID, minutes)
	if err != nil {
		return nil, fmt.Errorf("failed to query account throughput: %w", err)
	}
	defer rows.Close()

	var metrics []*ThroughputMetric

	for rows.Next() {
		metric := &ThroughputMetric{}
		var accountIDNull sql.NullString

		err := rows.Scan(
			&metric.ID,
			&accountIDNull,
			&metric.TenantID,
			&metric.DatasetID,
			&metric.MinuteTimestamp,
			&metric.RequestCount,
			&metric.BytesReceived,
			&metric.LinesReceived,
			&metric.BytesStored,
			&metric.SuccessCount,
			&metric.FailureCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan metric: %w", err)
		}

		if accountIDNull.Valid {
			metric.AccountID = accountIDNull.String
		}

		metrics = append(metrics, metric)
	}

	return metrics, rows.Err()
}
