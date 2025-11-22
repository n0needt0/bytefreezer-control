package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	"github.com/n0needt0/go-goodies/log"
)

// OperationTrackingService manages service operations and activity
type OperationTrackingService struct {
	db *sql.DB
}

// NewOperationTrackingService creates a new operation tracking service
func NewOperationTrackingService(db *sql.DB) *OperationTrackingService {
	return &OperationTrackingService{
		db: db,
	}
}

// ServiceOperation represents an operation being tracked
type ServiceOperation struct {
	ID            int64                  `json:"id"`
	ServiceType   string                 `json:"service_type"`
	InstanceID    string                 `json:"instance_id"`
	AccountID     string                 `json:"account_id,omitempty"`
	TenantID      string                 `json:"tenant_id,omitempty"`
	DatasetID     string                 `json:"dataset_id,omitempty"`
	OperationType string                 `json:"operation_type"`
	OperationID   string                 `json:"operation_id,omitempty"`
	Status        string                 `json:"status"`
	ProgressCurrent int64                `json:"progress_current,omitempty"`
	ProgressTotal   int64                `json:"progress_total,omitempty"`
	ProgressUnit    string               `json:"progress_unit,omitempty"`
	ProgressMessage string               `json:"progress_message,omitempty"`
	InputBytes      int64                `json:"input_bytes,omitempty"`
	OutputBytes     int64                `json:"output_bytes,omitempty"`
	RecordsProcessed int64               `json:"records_processed,omitempty"`
	ErrorCount      int                  `json:"error_count"`
	Details         map[string]interface{} `json:"details,omitempty"`
	StartedAt       time.Time            `json:"started_at"`
	UpdatedAt       time.Time            `json:"updated_at"`
	CompletedAt     *time.Time           `json:"completed_at,omitempty"`
	CreatedAt       time.Time            `json:"created_at"`
}

// OperationUpdate represents an update to an operation
type OperationUpdate struct {
	OperationID     string                 `json:"operation_id"`
	ServiceType     string                 `json:"service_type"`
	InstanceID      string                 `json:"instance_id"`
	AccountID       string                 `json:"account_id,omitempty"`
	TenantID        string                 `json:"tenant_id,omitempty"`
	DatasetID       string                 `json:"dataset_id,omitempty"`
	OperationType   string                 `json:"operation_type"`
	Status          string                 `json:"status"`
	ProgressCurrent int64                  `json:"progress_current,omitempty"`
	ProgressTotal   int64                  `json:"progress_total,omitempty"`
	ProgressUnit    string                 `json:"progress_unit,omitempty"`
	ProgressMessage string                 `json:"progress_message,omitempty"`
	InputBytes      int64                  `json:"input_bytes,omitempty"`
	OutputBytes     int64                  `json:"output_bytes,omitempty"`
	RecordsProcessed int64                 `json:"records_processed,omitempty"`
	ErrorCount      int                    `json:"error_count"`
	Details         map[string]interface{} `json:"details,omitempty"`
	StartedAt       *time.Time             `json:"started_at,omitempty"`
}

// UpsertOperation creates or updates an operation
func (s *OperationTrackingService) UpsertOperation(ctx context.Context, update *OperationUpdate) error {
	// Validate required fields
	if update.ServiceType == "" || update.InstanceID == "" || update.OperationType == "" {
		return fmt.Errorf("service_type, instance_id, and operation_type are required")
	}

	if update.Status == "" {
		update.Status = "in_progress"
	}

	// Marshal details to JSON
	var detailsJSON []byte
	var err error
	if update.Details != nil {
		detailsJSON, err = sonic.Marshal(update.Details)
		if err != nil {
			return fmt.Errorf("failed to marshal details: %w", err)
		}
	} else {
		detailsJSON = []byte("{}")
	}

	// Determine started_at
	startedAt := time.Now()
	if update.StartedAt != nil {
		startedAt = *update.StartedAt
	}

	// NULL handling for optional fields
	var accountID, tenantID, datasetID, operationID, progressUnit, progressMessage interface{}
	if update.AccountID != "" {
		accountID = update.AccountID
	}
	if update.TenantID != "" {
		tenantID = update.TenantID
	}
	if update.DatasetID != "" {
		datasetID = update.DatasetID
	}
	if update.OperationID != "" {
		operationID = update.OperationID
	}
	if update.ProgressUnit != "" {
		progressUnit = update.ProgressUnit
	}
	if update.ProgressMessage != "" {
		progressMessage = update.ProgressMessage
	}

	// Upsert query - insert or update based on operation_id (if provided) or create new
	query := `
		INSERT INTO service_operations (
			service_type, instance_id, account_id, tenant_id, dataset_id,
			operation_type, operation_id, status,
			progress_current, progress_total, progress_unit, progress_message,
			input_bytes, output_bytes, records_processed, error_count,
			details, started_at, updated_at, completed_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8,
			$9, $10, $11, $12,
			$13, $14, $15, $16,
			$17::jsonb, $18, NOW(),
			CASE WHEN $8 IN ('completed', 'failed') THEN NOW() ELSE NULL END
		)
		ON CONFLICT (operation_id)
		WHERE operation_id IS NOT NULL
		DO UPDATE SET
			status = EXCLUDED.status,
			progress_current = EXCLUDED.progress_current,
			progress_total = EXCLUDED.progress_total,
			progress_unit = EXCLUDED.progress_unit,
			progress_message = EXCLUDED.progress_message,
			input_bytes = EXCLUDED.input_bytes,
			output_bytes = EXCLUDED.output_bytes,
			records_processed = EXCLUDED.records_processed,
			error_count = EXCLUDED.error_count,
			details = EXCLUDED.details,
			updated_at = NOW(),
			completed_at = CASE WHEN EXCLUDED.status IN ('completed', 'failed') THEN NOW() ELSE NULL END
		RETURNING id`

	var id int64
	err = s.db.QueryRowContext(ctx, query,
		update.ServiceType, update.InstanceID, accountID, tenantID, datasetID,
		update.OperationType, operationID, update.Status,
		update.ProgressCurrent, update.ProgressTotal, progressUnit, progressMessage,
		update.InputBytes, update.OutputBytes, update.RecordsProcessed, update.ErrorCount,
		detailsJSON, startedAt,
	).Scan(&id)

	if err != nil {
		return fmt.Errorf("failed to upsert operation: %w", err)
	}

	log.Debugf("Upserted operation %s (ID: %d) for %s/%s", update.OperationType, id, update.ServiceType, update.InstanceID)
	return nil
}

// GetActiveOperations retrieves currently in-progress operations
func (s *OperationTrackingService) GetActiveOperations(ctx context.Context, accountID string) ([]*ServiceOperation, error) {
	query := `
		SELECT id, service_type, instance_id,
		       COALESCE(account_id, '') as account_id,
		       COALESCE(tenant_id, '') as tenant_id,
		       COALESCE(dataset_id, '') as dataset_id,
		       operation_type,
		       COALESCE(operation_id, '') as operation_id,
		       status,
		       COALESCE(progress_current, 0) as progress_current,
		       COALESCE(progress_total, 0) as progress_total,
		       COALESCE(progress_unit, '') as progress_unit,
		       COALESCE(progress_message, '') as progress_message,
		       COALESCE(input_bytes, 0) as input_bytes,
		       COALESCE(output_bytes, 0) as output_bytes,
		       COALESCE(records_processed, 0) as records_processed,
		       error_count,
		       details,
		       started_at, updated_at, completed_at, created_at
		FROM service_operations
		WHERE status = 'in_progress'`

	args := []interface{}{}
	if accountID != "" {
		query += ` AND account_id = $1`
		args = append(args, accountID)
	}

	query += ` ORDER BY started_at DESC LIMIT 100`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query active operations: %w", err)
	}
	defer rows.Close()

	return s.scanOperations(rows)
}

// GetRecentOperations retrieves recently completed/failed operations
func (s *OperationTrackingService) GetRecentOperations(ctx context.Context, accountID string, minutes int) ([]*ServiceOperation, error) {
	if minutes <= 0 {
		minutes = 60 // Default to last hour
	}

	query := `
		SELECT id, service_type, instance_id,
		       COALESCE(account_id, '') as account_id,
		       COALESCE(tenant_id, '') as tenant_id,
		       COALESCE(dataset_id, '') as dataset_id,
		       operation_type,
		       COALESCE(operation_id, '') as operation_id,
		       status,
		       COALESCE(progress_current, 0) as progress_current,
		       COALESCE(progress_total, 0) as progress_total,
		       COALESCE(progress_unit, '') as progress_unit,
		       COALESCE(progress_message, '') as progress_message,
		       COALESCE(input_bytes, 0) as input_bytes,
		       COALESCE(output_bytes, 0) as output_bytes,
		       COALESCE(records_processed, 0) as records_processed,
		       error_count,
		       details,
		       started_at, updated_at, completed_at, created_at
		FROM service_operations
		WHERE status IN ('completed', 'failed')
		  AND completed_at > NOW() - $1 * INTERVAL '1 minute'`

	args := []interface{}{minutes}
	argIndex := 2

	if accountID != "" {
		query += fmt.Sprintf(` AND account_id = $%d`, argIndex)
		args = append(args, accountID)
	}

	query += ` ORDER BY completed_at DESC LIMIT 200`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent operations: %w", err)
	}
	defer rows.Close()

	return s.scanOperations(rows)
}

// scanOperations scans rows into ServiceOperation structs
func (s *OperationTrackingService) scanOperations(rows *sql.Rows) ([]*ServiceOperation, error) {
	var operations []*ServiceOperation

	for rows.Next() {
		var op ServiceOperation
		var detailsJSON []byte
		var completedAt sql.NullTime

		err := rows.Scan(
			&op.ID, &op.ServiceType, &op.InstanceID,
			&op.AccountID, &op.TenantID, &op.DatasetID,
			&op.OperationType, &op.OperationID, &op.Status,
			&op.ProgressCurrent, &op.ProgressTotal, &op.ProgressUnit, &op.ProgressMessage,
			&op.InputBytes, &op.OutputBytes, &op.RecordsProcessed, &op.ErrorCount,
			&detailsJSON,
			&op.StartedAt, &op.UpdatedAt, &completedAt, &op.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan operation: %w", err)
		}

		// Unmarshal details
		if len(detailsJSON) > 0 {
			if err := sonic.Unmarshal(detailsJSON, &op.Details); err != nil {
				log.Warnf("Failed to unmarshal operation details: %v", err)
			}
		}

		if completedAt.Valid {
			op.CompletedAt = &completedAt.Time
		}

		operations = append(operations, &op)
	}

	return operations, nil
}

// CleanupOldOperations removes old completed/failed operations
func (s *OperationTrackingService) CleanupOldOperations(ctx context.Context) (int, error) {
	var deletedCount int
	err := s.db.QueryRowContext(ctx, "SELECT cleanup_old_service_operations()").Scan(&deletedCount)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup old operations: %w", err)
	}

	if deletedCount > 0 {
		log.Infof("Cleaned up %d old service operations", deletedCount)
	}

	return deletedCount, nil
}
