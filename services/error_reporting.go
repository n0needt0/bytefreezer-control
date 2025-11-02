package services

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"github.com/bytedance/sonic"
	"fmt"
	"time"

	"github.com/n0needt0/go-goodies/log"
)

// ErrorReportingService handles error reporting and retrieval
type ErrorReportingService struct {
	db *sql.DB
}

// NewErrorReportingService creates a new error reporting service
func NewErrorReportingService(db *sql.DB) *ErrorReportingService {
	return &ErrorReportingService{
		db: db,
	}
}

// SystemError represents an error in the system_errors table
type SystemError struct {
	ID               int64                  `json:"id"`
	ErrorHash        string                 `json:"error_hash"`
	ErrorType        string                 `json:"error_type"`
	Component        string                 `json:"component"`
	AccountID        string                 `json:"account_id,omitempty"`
	TenantID         string                 `json:"tenant_id,omitempty"`
	DatasetID        string                 `json:"dataset_id,omitempty"`
	ErrorMessage     string                 `json:"error_message"`
	ErrorSample      map[string]interface{} `json:"error_sample,omitempty"`
	Severity         string                 `json:"severity"`
	Status           string                 `json:"status"`
	OccurrenceCount  int64                  `json:"occurrence_count"`
	FirstSeen        time.Time              `json:"first_seen"`
	LastSeen         time.Time              `json:"last_seen"`
	SampleRate       float64                `json:"sample_rate"`
	SamplesCollected int                    `json:"samples_collected"`
	SamplesDropped   int                    `json:"samples_dropped"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
	ResolvedAt       *time.Time             `json:"resolved_at,omitempty"`
}

// ErrorReport is the input structure for reporting errors
type ErrorReport struct {
	ErrorType    string                 `json:"error_type"`
	Component    string                 `json:"component"`
	TenantID     string                 `json:"tenant_id,omitempty"`
	DatasetID    string                 `json:"dataset_id,omitempty"`
	ErrorMessage string                 `json:"error_message"`
	ErrorSample  map[string]interface{} `json:"error_sample,omitempty"`
	Severity     string                 `json:"severity"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// ErrorListFilter contains filter options for listing errors
type ErrorListFilter struct {
	Component  string
	TenantID   string
	DatasetID  string
	Severity   string
	Status     string
	ErrorType  string
	AccountID  string // For account-based filtering
	Limit      int
	Offset     int
}

// ReportError reports an error using the upsert function with adaptive sampling
func (s *ErrorReportingService) ReportError(ctx context.Context, report ErrorReport) error {
	// Generate error hash from component + error_type + error_message + tenant_id + dataset_id
	hash := s.generateErrorHash(report.Component, report.ErrorType, report.ErrorMessage, report.TenantID, report.DatasetID)

	// Default severity if not provided
	severity := report.Severity
	if severity == "" {
		severity = "error"
	}

	// Validate component
	validComponents := map[string]bool{
		"proxy": true, "receiver": true, "piper": true, "packer": true, "control": true, "soc": true,
	}
	if !validComponents[report.Component] {
		return fmt.Errorf("invalid component: %s", report.Component)
	}

	// Validate severity
	validSeverities := map[string]bool{
		"debug": true, "info": true, "warning": true, "error": true, "critical": true,
	}
	if !validSeverities[severity] {
		return fmt.Errorf("invalid severity: %s", severity)
	}

	// Convert sample and metadata to JSONB
	var errorSampleJSON []byte
	var err error
	if report.ErrorSample != nil {
		errorSampleJSON, err = sonic.Marshal(report.ErrorSample)
		if err != nil {
			log.Warnf("Failed to marshal error sample: %v", err)
			errorSampleJSON = []byte("{}")
		}
	} else {
		errorSampleJSON = []byte("{}")
	}

	var metadataJSON []byte
	if report.Metadata != nil {
		metadataJSON, err = sonic.Marshal(report.Metadata)
		if err != nil {
			log.Warnf("Failed to marshal metadata: %v", err)
			metadataJSON = []byte("{}")
		}
	} else {
		metadataJSON = []byte("{}")
	}

	// Use NULL for tenant_id and dataset_id if empty
	var tenantID, datasetID interface{}
	if report.TenantID != "" {
		tenantID = report.TenantID
	} else {
		tenantID = nil
	}
	if report.DatasetID != "" {
		datasetID = report.DatasetID
	} else {
		datasetID = nil
	}

	// Call the upsert function
	query := `SELECT upsert_system_error($1, $2, $3, $4, $5, $6, $7::jsonb, $8, $9::jsonb)`
	_, err = s.db.ExecContext(ctx, query,
		hash,
		report.ErrorType,
		report.Component,
		tenantID,
		datasetID,
		report.ErrorMessage,
		errorSampleJSON,
		severity,
		metadataJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to report error: %w", err)
	}

	log.Debugf("Error reported: component=%s, type=%s, hash=%s", report.Component, report.ErrorType, hash)
	return nil
}

// ListErrors retrieves errors based on filter criteria
func (s *ErrorReportingService) ListErrors(ctx context.Context, filter ErrorListFilter) ([]SystemError, int, error) {
	// Build query with filters - always join with tenants to get account_id
	query := `
		SELECT e.id, e.error_hash, e.error_type, e.component,
		       COALESCE(t.account_id, '') as account_id,
		       COALESCE(e.tenant_id, '') as tenant_id,
		       COALESCE(e.dataset_id, '') as dataset_id,
		       e.error_message, e.error_sample, e.severity, e.status,
		       e.occurrence_count, e.first_seen, e.last_seen, e.sample_rate,
		       e.samples_collected, e.samples_dropped, e.metadata,
		       e.created_at, e.updated_at, e.resolved_at
		FROM system_errors e
		LEFT JOIN control_tenants t ON e.tenant_id = t.id`

	query += ` WHERE 1=1`

	args := []interface{}{}
	argIndex := 1

	// Add filters
	if filter.Component != "" {
		query += fmt.Sprintf(" AND e.component = $%d", argIndex)
		args = append(args, filter.Component)
		argIndex++
	}

	if filter.TenantID != "" {
		query += fmt.Sprintf(" AND e.tenant_id = $%d", argIndex)
		args = append(args, filter.TenantID)
		argIndex++
	}

	if filter.DatasetID != "" {
		query += fmt.Sprintf(" AND e.dataset_id = $%d", argIndex)
		args = append(args, filter.DatasetID)
		argIndex++
	}

	if filter.Severity != "" {
		query += fmt.Sprintf(" AND e.severity = $%d", argIndex)
		args = append(args, filter.Severity)
		argIndex++
	}

	if filter.Status != "" {
		query += fmt.Sprintf(" AND e.status = $%d", argIndex)
		args = append(args, filter.Status)
		argIndex++
	}

	if filter.ErrorType != "" {
		query += fmt.Sprintf(" AND e.error_type = $%d", argIndex)
		args = append(args, filter.ErrorType)
		argIndex++
	}

	// Account-based filtering
	// Non-admin users should only see errors for their account's tenants
	// System-level errors (tenant_id IS NULL) are only shown to system admins via the non-filtered endpoint
	if filter.AccountID != "" {
		query += fmt.Sprintf(" AND t.account_id = $%d", argIndex)
		args = append(args, filter.AccountID)
		argIndex++
	}

	// Order by last seen (most recent first)
	query += " ORDER BY e.last_seen DESC"

	// Set default limit
	if filter.Limit <= 0 {
		filter.Limit = 100
	}

	// Add pagination
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, filter.Limit, filter.Offset)

	// Execute query
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query errors: %w", err)
	}
	defer rows.Close()

	var errors []SystemError
	for rows.Next() {
		var e SystemError
		var errorSampleJSON, metadataJSON []byte
		var resolvedAt sql.NullTime

		err := rows.Scan(
			&e.ID, &e.ErrorHash, &e.ErrorType, &e.Component,
			&e.AccountID, &e.TenantID, &e.DatasetID, &e.ErrorMessage,
			&errorSampleJSON, &e.Severity, &e.Status,
			&e.OccurrenceCount, &e.FirstSeen, &e.LastSeen, &e.SampleRate,
			&e.SamplesCollected, &e.SamplesDropped, &metadataJSON,
			&e.CreatedAt, &e.UpdatedAt, &resolvedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan error: %w", err)
		}

		// Unmarshal JSON fields
		if len(errorSampleJSON) > 0 {
			if err := sonic.Unmarshal(errorSampleJSON, &e.ErrorSample); err != nil {
				log.Warnf("Failed to unmarshal error sample: %v", err)
			}
		}

		if len(metadataJSON) > 0 {
			if err := sonic.Unmarshal(metadataJSON, &e.Metadata); err != nil {
				log.Warnf("Failed to unmarshal metadata: %v", err)
			}
		}

		if resolvedAt.Valid {
			e.ResolvedAt = &resolvedAt.Time
		}

		errors = append(errors, e)
	}

	// Get total count
	countQuery := `SELECT COUNT(*) FROM system_errors e`
	if filter.AccountID != "" {
		countQuery += ` LEFT JOIN control_tenants t ON e.tenant_id = t.id`
	}
	countQuery += ` WHERE 1=1`

	countArgs := []interface{}{}
	countArgIndex := 1

	if filter.Component != "" {
		countQuery += fmt.Sprintf(" AND e.component = $%d", countArgIndex)
		countArgs = append(countArgs, filter.Component)
		countArgIndex++
	}
	if filter.TenantID != "" {
		countQuery += fmt.Sprintf(" AND e.tenant_id = $%d", countArgIndex)
		countArgs = append(countArgs, filter.TenantID)
		countArgIndex++
	}
	if filter.DatasetID != "" {
		countQuery += fmt.Sprintf(" AND e.dataset_id = $%d", countArgIndex)
		countArgs = append(countArgs, filter.DatasetID)
		countArgIndex++
	}
	if filter.Severity != "" {
		countQuery += fmt.Sprintf(" AND e.severity = $%d", countArgIndex)
		countArgs = append(countArgs, filter.Severity)
		countArgIndex++
	}
	if filter.Status != "" {
		countQuery += fmt.Sprintf(" AND e.status = $%d", countArgIndex)
		countArgs = append(countArgs, filter.Status)
		countArgIndex++
	}
	if filter.ErrorType != "" {
		countQuery += fmt.Sprintf(" AND e.error_type = $%d", countArgIndex)
		countArgs = append(countArgs, filter.ErrorType)
		countArgIndex++
	}
	if filter.AccountID != "" {
		countQuery += fmt.Sprintf(" AND t.account_id = $%d", countArgIndex)
		countArgs = append(countArgs, filter.AccountID)
		countArgIndex++
	}

	var total int
	err = s.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get error count: %w", err)
	}

	return errors, total, nil
}

// GetErrorStats returns statistics about errors
func (s *ErrorReportingService) GetErrorStats(ctx context.Context, accountID string) (map[string]interface{}, error) {
	query := `
		SELECT
			COUNT(*) as total_errors,
			COUNT(DISTINCT component) as affected_components,
			COALESCE(SUM(occurrence_count), 0) as total_occurrences,
			COUNT(CASE WHEN severity = 'critical' THEN 1 END) as critical_count,
			COUNT(CASE WHEN severity = 'error' THEN 1 END) as error_count,
			COUNT(CASE WHEN severity = 'warning' THEN 1 END) as warning_count,
			COUNT(CASE WHEN status = 'active' THEN 1 END) as active_count,
			COUNT(CASE WHEN status = 'resolved' THEN 1 END) as resolved_count
		FROM system_errors e`

	args := []interface{}{}
	if accountID != "" {
		query += `
		LEFT JOIN control_tenants t ON e.tenant_id = t.id
		WHERE t.account_id = $1`
		args = append(args, accountID)
	}

	var stats struct {
		TotalErrors        int
		AffectedComponents int
		TotalOccurrences   int64
		CriticalCount      int
		ErrorCount         int
		WarningCount       int
		ActiveCount        int
		ResolvedCount      int
	}

	err := s.db.QueryRowContext(ctx, query, args...).Scan(
		&stats.TotalErrors,
		&stats.AffectedComponents,
		&stats.TotalOccurrences,
		&stats.CriticalCount,
		&stats.ErrorCount,
		&stats.WarningCount,
		&stats.ActiveCount,
		&stats.ResolvedCount,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get error stats: %w", err)
	}

	return map[string]interface{}{
		"total_errors":        stats.TotalErrors,
		"affected_components": stats.AffectedComponents,
		"total_occurrences":   stats.TotalOccurrences,
		"critical_count":      stats.CriticalCount,
		"error_count":         stats.ErrorCount,
		"warning_count":       stats.WarningCount,
		"active_count":        stats.ActiveCount,
		"resolved_count":      stats.ResolvedCount,
	}, nil
}

// generateErrorHash creates a consistent hash for error deduplication
func (s *ErrorReportingService) generateErrorHash(component, errorType, errorMessage, tenantID, datasetID string) string {
	// Create a consistent string for hashing
	hashInput := fmt.Sprintf("%s|%s|%s|%s|%s", component, errorType, errorMessage, tenantID, datasetID)
	hash := sha256.Sum256([]byte(hashInput))
	return hex.EncodeToString(hash[:])
}
