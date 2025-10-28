package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/n0needt0/bytefreezer-control/middleware"
	"github.com/n0needt0/bytefreezer-control/storage"
	"github.com/n0needt0/go-goodies/log"
	"github.com/swaggest/usecase"
)

// TrackError handles POST requests to track errors from services
func (api *API) TrackError() usecase.Interactor {
	type trackErrorInput struct {
		ErrorHash    string                 `json:"error_hash" minLength:"64" maxLength:"64" required:"true" description:"SHA256 hash for error deduplication"`
		ErrorType    string                 `json:"error_type" minLength:"1" maxLength:"100" required:"true" description:"Error category (pipeline_processing, s3_operation, validation, etc.)"`
		Component    string                 `json:"component" minLength:"1" maxLength:"50" required:"true" description:"Component that generated the error (piper, packer, receiver, proxy, control, soc)"`
		AccountID    string                 `json:"account_id,omitempty" maxLength:"255" description:"Optional account ID for access control"`
		TenantID     string                 `json:"tenant_id,omitempty" maxLength:"255" description:"Optional tenant ID"`
		DatasetID    string                 `json:"dataset_id,omitempty" maxLength:"255" description:"Optional dataset ID"`
		ErrorMessage string                 `json:"error_message" minLength:"1" required:"true" description:"Full error message (DO NOT include sensitive data like passwords, tokens, or PII)"`
		ErrorSample  map[string]interface{} `json:"error_sample,omitempty" description:"Sample error details (DO NOT include sensitive data)"`
		Severity     string                 `json:"severity" required:"true" description:"Severity level: debug, info, warning, error, critical"`
		Metadata     map[string]interface{} `json:"metadata,omitempty" description:"Additional metadata (DO NOT include sensitive data)"`
	}

	type trackErrorOutput struct {
		Success bool `json:"success"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input trackErrorInput, output *trackErrorOutput) error {
		api.Services.IncrementAPIRequests()

		// Validate component
		validComponents := map[string]bool{
			"proxy": true, "receiver": true, "piper": true, "packer": true, "control": true, "soc": true,
		}
		if !validComponents[input.Component] {
			return fmt.Errorf("invalid component: %s (must be one of: proxy, receiver, piper, packer, control, soc)", input.Component)
		}

		// Validate severity
		validSeverity := map[string]bool{
			"debug": true, "info": true, "warning": true, "error": true, "critical": true,
		}
		if !validSeverity[input.Severity] {
			return fmt.Errorf("invalid severity: %s (must be one of: debug, info, warning, error, critical)", input.Severity)
		}

		// Convert error sample to JSON
		var errorSampleJSON []byte
		var err error
		if input.ErrorSample != nil {
			errorSampleJSON, err = json.Marshal(input.ErrorSample)
			if err != nil {
				return fmt.Errorf("failed to marshal error_sample: %w", err)
			}
		}

		// Convert metadata to JSON
		var metadataJSON []byte
		if input.Metadata != nil {
			metadataJSON, err = json.Marshal(input.Metadata)
			if err != nil {
				return fmt.Errorf("failed to marshal metadata: %w", err)
			}
		}

		// Type-assert to PostgreSQLStorage to access DB
		pgStorage, ok := api.Services.Storage.(*storage.PostgreSQLStorage)
		if !ok {
			return fmt.Errorf("storage backend is not PostgreSQL")
		}

		// Call upsert_system_error PostgreSQL function
		_, err = pgStorage.DB().ExecContext(ctx,
			`SELECT upsert_system_error($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
			input.ErrorHash,
			input.ErrorType,
			input.Component,
			input.AccountID,
			input.TenantID,
			input.DatasetID,
			input.ErrorMessage,
			errorSampleJSON,
			input.Severity,
			metadataJSON,
		)
		if err != nil {
			log.Errorf("Failed to track error %s from %s: %v", input.ErrorHash, input.Component, err)
			return fmt.Errorf("failed to track error: %w", err)
		}

		log.Debugf("Successfully tracked error %s from %s (type: %s, severity: %s)",
			input.ErrorHash, input.Component, input.ErrorType, input.Severity)

		output.Success = true
		return nil
	})

	return u
}

// ListErrors handles GET requests to list errors with filters
func (api *API) ListErrors() usecase.Interactor {
	type listErrorsInput struct {
		Component string `query:"component" description:"Filter by component (optional)"`
		TenantID  string `query:"tenant_id" description:"Filter by tenant ID (optional)"`
		DatasetID string `query:"dataset_id" description:"Filter by dataset ID (optional)"`
		ErrorType string `query:"error_type" description:"Filter by error type (optional)"`
		Severity  string `query:"severity" description:"Filter by severity (optional)"`
		Status    string `query:"status" default:"active" description:"Filter by status: active, resolved, ignored (default: active)"`
		Limit     int    `query:"limit" minimum:"1" maximum:"1000" default:"100" description:"Maximum number of records to return"`
		Offset    int    `query:"offset" minimum:"0" default:"0" description:"Number of records to skip"`
	}

	type listErrorsOutput struct {
		Errors []*storage.SystemError `json:"errors"`
		Count  int                    `json:"count"`
		Total  int64                  `json:"total"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input listErrorsInput, output *listErrorsOutput) error {
		api.Services.IncrementAPIRequests()

		// Type-assert to PostgreSQLStorage to access DB
		pgStorage, ok := api.Services.Storage.(*storage.PostgreSQLStorage)
		if !ok {
			return fmt.Errorf("storage backend is not PostgreSQL")
		}

		// Get user from context for access control
		var userAccountID string
		var isSystemAdmin bool

		log.Debugf("[ACCESS CONTROL] Attempting to extract JWT claims from context")

		// Use the correct context key that middleware actually uses
		if claims, ok := ctx.Value(middleware.UserContextKey).(*middleware.UserClaims); ok {
			userAccountID = claims.AccountID
			isSystemAdmin = claims.IsSystemAdmin()
			log.Debugf("[ACCESS CONTROL] Successfully extracted claims: AccountID=%s, Role=%s, IsSystemAdmin=%v",
				claims.AccountID, claims.Role, isSystemAdmin)
		} else {
			log.Warnf("[ACCESS CONTROL] Failed to extract UserClaims from context - user will see all errors")
		}

		// Build WHERE clauses
		whereClauses := []string{"status = $1"}
		args := []interface{}{input.Status}
		argCount := 2

		// Access control: non-system-admins can only see their account's errors
		if !isSystemAdmin && userAccountID != "" {
			log.Debugf("[ACCESS CONTROL] Applying account filter: account_id = %s", userAccountID)
			whereClauses = append(whereClauses, fmt.Sprintf("account_id = $%d", argCount))
			args = append(args, userAccountID)
			argCount++
		} else {
			log.Debugf("[ACCESS CONTROL] No account filter applied (isSystemAdmin=%v, userAccountID=%s)", isSystemAdmin, userAccountID)
		}

		if input.Component != "" {
			whereClauses = append(whereClauses, fmt.Sprintf("component = $%d", argCount))
			args = append(args, input.Component)
			argCount++
		}
		if input.TenantID != "" {
			whereClauses = append(whereClauses, fmt.Sprintf("tenant_id = $%d", argCount))
			args = append(args, input.TenantID)
			argCount++
		}
		if input.DatasetID != "" {
			whereClauses = append(whereClauses, fmt.Sprintf("dataset_id = $%d", argCount))
			args = append(args, input.DatasetID)
			argCount++
		}
		if input.ErrorType != "" {
			whereClauses = append(whereClauses, fmt.Sprintf("error_type = $%d", argCount))
			args = append(args, input.ErrorType)
			argCount++
		}
		if input.Severity != "" {
			whereClauses = append(whereClauses, fmt.Sprintf("severity = $%d", argCount))
			args = append(args, input.Severity)
			argCount++
		}

		whereClause := ""
		if len(whereClauses) > 0 {
			whereClause = "WHERE " + whereClauses[0]
			for i := 1; i < len(whereClauses); i++ {
				whereClause += " AND " + whereClauses[i]
			}
		}

		// Get total count
		var total int64
		countQuery := fmt.Sprintf("SELECT COUNT(*) FROM system_errors %s", whereClause)
		err := pgStorage.DB().QueryRowContext(ctx, countQuery, args...).Scan(&total)
		if err != nil {
			log.Errorf("Failed to count errors: %v", err)
			return fmt.Errorf("failed to count errors: %w", err)
		}

		// Query errors with limit and offset
		query := fmt.Sprintf(`
			SELECT id, error_hash, error_type, component, account_id, tenant_id, dataset_id,
				   error_message, error_sample, severity, status, occurrence_count,
				   first_seen, last_seen, sample_rate, samples_collected, samples_dropped,
				   metadata, created_at, updated_at, resolved_at
			FROM system_errors
			%s
			ORDER BY last_seen DESC
			LIMIT $%d OFFSET $%d
		`, whereClause, argCount, argCount+1)
		args = append(args, input.Limit, input.Offset)

		rows, err := pgStorage.DB().QueryContext(ctx, query, args...)
		if err != nil {
			log.Errorf("Failed to query errors: %v", err)
			return fmt.Errorf("failed to query errors: %w", err)
		}
		defer rows.Close()

		errors := make([]*storage.SystemError, 0)
		for rows.Next() {
			e := &storage.SystemError{}
			var errorSampleJSON, metadataJSON []byte
			var accountID, tenantID, datasetID sql.NullString
			var resolvedAt sql.NullTime

			err := rows.Scan(
				&e.ID, &e.ErrorHash, &e.ErrorType, &e.Component, &accountID, &tenantID, &datasetID,
				&e.ErrorMessage, &errorSampleJSON, &e.Severity, &e.Status, &e.OccurrenceCount,
				&e.FirstSeen, &e.LastSeen, &e.SampleRate, &e.SamplesCollected, &e.SamplesDropped,
				&metadataJSON, &e.CreatedAt, &e.UpdatedAt, &resolvedAt,
			)
			if err != nil {
				log.Errorf("Failed to scan error row: %v", err)
				continue
			}

			if accountID.Valid {
				e.AccountID = accountID.String
			}
			if tenantID.Valid {
				e.TenantID = tenantID.String
			}
			if datasetID.Valid {
				e.DatasetID = datasetID.String
			}
			if resolvedAt.Valid {
				e.ResolvedAt = &resolvedAt.Time
			}

			// Parse JSONB fields
			if len(errorSampleJSON) > 0 {
				if err := json.Unmarshal(errorSampleJSON, &e.ErrorSample); err != nil {
					log.Warnf("Failed to unmarshal error_sample for error %d: %v", e.ID, err)
				}
			}
			if len(metadataJSON) > 0 {
				if err := json.Unmarshal(metadataJSON, &e.Metadata); err != nil {
					log.Warnf("Failed to unmarshal metadata for error %d: %v", e.ID, err)
				}
			}

			errors = append(errors, e)
		}

		log.Infof("Successfully queried errors: %d records (total: %d)", len(errors), total)

		output.Errors = errors
		output.Count = len(errors)
		output.Total = total

		return nil
	})

	return u
}

// GetError handles GET requests to get error details by ID
func (api *API) GetError() usecase.Interactor {
	type getErrorInput struct {
		ErrorID int64 `path:"errorId" minimum:"1"`
	}

	type getErrorOutput struct {
		Error *storage.SystemError `json:"error"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input getErrorInput, output *getErrorOutput) error {
		api.Services.IncrementAPIRequests()

		// Type-assert to PostgreSQLStorage to access DB
		pgStorage, ok := api.Services.Storage.(*storage.PostgreSQLStorage)
		if !ok {
			return fmt.Errorf("storage backend is not PostgreSQL")
		}

		query := `
			SELECT id, error_hash, error_type, component, account_id, tenant_id, dataset_id,
				   error_message, error_sample, severity, status, occurrence_count,
				   first_seen, last_seen, sample_rate, samples_collected, samples_dropped,
				   metadata, created_at, updated_at, resolved_at
			FROM system_errors
			WHERE id = $1
		`

		e := &storage.SystemError{}
		var errorSampleJSON, metadataJSON []byte
		var accountID, tenantID, datasetID sql.NullString
		var resolvedAt sql.NullTime

		err := pgStorage.DB().QueryRowContext(ctx, query, input.ErrorID).Scan(
			&e.ID, &e.ErrorHash, &e.ErrorType, &e.Component, &accountID, &tenantID, &datasetID,
			&e.ErrorMessage, &errorSampleJSON, &e.Severity, &e.Status, &e.OccurrenceCount,
			&e.FirstSeen, &e.LastSeen, &e.SampleRate, &e.SamplesCollected, &e.SamplesDropped,
			&metadataJSON, &e.CreatedAt, &e.UpdatedAt, &resolvedAt,
		)
		if err == sql.ErrNoRows {
			return fmt.Errorf("error not found with ID: %d", input.ErrorID)
		}
		if err != nil {
			log.Errorf("Failed to query error %d: %v", input.ErrorID, err)
			return fmt.Errorf("failed to query error: %w", err)
		}

		if accountID.Valid {
			e.AccountID = accountID.String
		}
		if tenantID.Valid {
			e.TenantID = tenantID.String
		}
		if datasetID.Valid {
			e.DatasetID = datasetID.String
		}
		if resolvedAt.Valid {
			e.ResolvedAt = &resolvedAt.Time
		}

		// Parse JSONB fields
		if len(errorSampleJSON) > 0 {
			if err := json.Unmarshal(errorSampleJSON, &e.ErrorSample); err != nil {
				log.Warnf("Failed to unmarshal error_sample for error %d: %v", e.ID, err)
			}
		}
		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &e.Metadata); err != nil {
				log.Warnf("Failed to unmarshal metadata for error %d: %v", e.ID, err)
			}
		}

		log.Infof("Successfully retrieved error %d", input.ErrorID)

		output.Error = e
		return nil
	})

	return u
}

// UpdateErrorStatus handles PATCH requests to update error status
func (api *API) UpdateErrorStatus() usecase.Interactor {
	type updateErrorStatusInput struct {
		ErrorID int64  `path:"errorId" minimum:"1"`
		Status  string `json:"status" required:"true" description:"New status: active, resolved, ignored"`
	}

	type updateErrorStatusOutput struct {
		Success bool `json:"success"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input updateErrorStatusInput, output *updateErrorStatusOutput) error {
		api.Services.IncrementAPIRequests()

		// Type-assert to PostgreSQLStorage to access DB
		pgStorage, ok := api.Services.Storage.(*storage.PostgreSQLStorage)
		if !ok {
			return fmt.Errorf("storage backend is not PostgreSQL")
		}

		// Validate status
		validStatus := map[string]bool{
			"active": true, "resolved": true, "ignored": true,
		}
		if !validStatus[input.Status] {
			return fmt.Errorf("invalid status: %s (must be one of: active, resolved, ignored)", input.Status)
		}

		// Update status
		query := `
			UPDATE system_errors
			SET status = $1, resolved_at = CASE WHEN $1 = 'resolved' THEN NOW() ELSE NULL END
			WHERE id = $2
		`

		result, err := pgStorage.DB().ExecContext(ctx, query, input.Status, input.ErrorID)
		if err != nil {
			log.Errorf("Failed to update error status for error %d: %v", input.ErrorID, err)
			return fmt.Errorf("failed to update error status: %w", err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get rows affected: %w", err)
		}
		if rowsAffected == 0 {
			return fmt.Errorf("error not found with ID: %d", input.ErrorID)
		}

		log.Infof("Successfully updated error %d status to %s", input.ErrorID, input.Status)

		output.Success = true
		return nil
	})

	return u
}

// GetErrorStats handles GET requests for error statistics
func (api *API) GetErrorStats() usecase.Interactor {
	type getErrorStatsInput struct {
		Range string `query:"range" default:"24h" description:"Time range (1h, 6h, 24h, 7d, 30d)"`
	}

	type componentStats struct {
		Component       string `json:"component"`
		ErrorCount      int64  `json:"error_count"`
		TotalOccurences int64  `json:"total_occurrences"`
	}

	type severityStats struct {
		Severity        string `json:"severity"`
		ErrorCount      int64  `json:"error_count"`
		TotalOccurences int64  `json:"total_occurrences"`
	}

	type getErrorStatsOutput struct {
		TotalErrors         int64            `json:"total_errors"`
		TotalOccurrences    int64            `json:"total_occurrences"`
		ActiveErrors        int64            `json:"active_errors"`
		ResolvedErrors      int64            `json:"resolved_errors"`
		ByComponent         []componentStats `json:"by_component"`
		BySeverity          []severityStats  `json:"by_severity"`
		MostFrequentErrors  []string         `json:"most_frequent_error_types"`
		Range               string           `json:"range"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input getErrorStatsInput, output *getErrorStatsOutput) error {
		api.Services.IncrementAPIRequests()

		// Type-assert to PostgreSQLStorage to access DB
		pgStorage, ok := api.Services.Storage.(*storage.PostgreSQLStorage)
		if !ok {
			return fmt.Errorf("storage backend is not PostgreSQL")
		}

		// Parse time range
		startTime, _, err := parseTimeRange(input.Range)
		if err != nil {
			return fmt.Errorf("invalid time range: %w", err)
		}

		// Get total errors
		err = pgStorage.DB().QueryRowContext(ctx,
			"SELECT COUNT(*), COALESCE(SUM(occurrence_count), 0) FROM system_errors WHERE last_seen >= $1",
			startTime,
		).Scan(&output.TotalErrors, &output.TotalOccurrences)
		if err != nil {
			return fmt.Errorf("failed to get total errors: %w", err)
		}

		// Get active/resolved counts
		err = pgStorage.DB().QueryRowContext(ctx,
			"SELECT COUNT(*) FROM system_errors WHERE status = 'active' AND last_seen >= $1",
			startTime,
		).Scan(&output.ActiveErrors)
		if err != nil {
			return fmt.Errorf("failed to get active errors count: %w", err)
		}

		err = pgStorage.DB().QueryRowContext(ctx,
			"SELECT COUNT(*) FROM system_errors WHERE status = 'resolved' AND last_seen >= $1",
			startTime,
		).Scan(&output.ResolvedErrors)
		if err != nil {
			return fmt.Errorf("failed to get resolved errors count: %w", err)
		}

		// Get stats by component
		rows, err := pgStorage.DB().QueryContext(ctx, `
			SELECT component, COUNT(*), COALESCE(SUM(occurrence_count), 0)
			FROM system_errors
			WHERE last_seen >= $1
			GROUP BY component
			ORDER BY SUM(occurrence_count) DESC
		`, startTime)
		if err != nil {
			return fmt.Errorf("failed to get component stats: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var cs componentStats
			if err := rows.Scan(&cs.Component, &cs.ErrorCount, &cs.TotalOccurences); err != nil {
				log.Warnf("Failed to scan component stats: %v", err)
				continue
			}
			output.ByComponent = append(output.ByComponent, cs)
		}

		// Get stats by severity
		rows, err = pgStorage.DB().QueryContext(ctx, `
			SELECT severity, COUNT(*), COALESCE(SUM(occurrence_count), 0)
			FROM system_errors
			WHERE last_seen >= $1
			GROUP BY severity
			ORDER BY CASE severity
				WHEN 'critical' THEN 1
				WHEN 'error' THEN 2
				WHEN 'warning' THEN 3
				WHEN 'info' THEN 4
				WHEN 'debug' THEN 5
			END
		`, startTime)
		if err != nil {
			return fmt.Errorf("failed to get severity stats: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var ss severityStats
			if err := rows.Scan(&ss.Severity, &ss.ErrorCount, &ss.TotalOccurences); err != nil {
				log.Warnf("Failed to scan severity stats: %v", err)
				continue
			}
			output.BySeverity = append(output.BySeverity, ss)
		}

		// Get most frequent error types
		rows, err = pgStorage.DB().QueryContext(ctx, `
			SELECT error_type
			FROM system_errors
			WHERE last_seen >= $1
			ORDER BY occurrence_count DESC
			LIMIT 10
		`, startTime)
		if err != nil {
			return fmt.Errorf("failed to get frequent errors: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var errorType string
			if err := rows.Scan(&errorType); err != nil {
				log.Warnf("Failed to scan error type: %v", err)
				continue
			}
			output.MostFrequentErrors = append(output.MostFrequentErrors, errorType)
		}

		output.Range = input.Range

		log.Infof("Successfully retrieved error stats (range: %s, total: %d)", input.Range, output.TotalErrors)

		return nil
	})

	return u
}

// GetRecentErrors handles GET requests for recent errors
func (api *API) GetRecentErrors() usecase.Interactor {
	type getRecentErrorsInput struct {
		Limit int `query:"limit" minimum:"1" maximum:"100" default:"20" description:"Maximum number of recent errors to return"`
	}

	type getRecentErrorsOutput struct {
		Errors []*storage.SystemError `json:"errors"`
		Count  int                    `json:"count"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input getRecentErrorsInput, output *getRecentErrorsOutput) error {
		api.Services.IncrementAPIRequests()

		// Type-assert to PostgreSQLStorage to access DB
		pgStorage, ok := api.Services.Storage.(*storage.PostgreSQLStorage)
		if !ok {
			return fmt.Errorf("storage backend is not PostgreSQL")
		}

		query := `
			SELECT id, error_hash, error_type, component, account_id, tenant_id, dataset_id,
				   error_message, error_sample, severity, status, occurrence_count,
				   first_seen, last_seen, sample_rate, samples_collected, samples_dropped,
				   metadata, created_at, updated_at, resolved_at
			FROM system_errors
			WHERE status = 'active'
			ORDER BY last_seen DESC
			LIMIT $1
		`

		rows, err := pgStorage.DB().QueryContext(ctx, query, input.Limit)
		if err != nil {
			log.Errorf("Failed to query recent errors: %v", err)
			return fmt.Errorf("failed to query recent errors: %w", err)
		}
		defer rows.Close()

		errors := make([]*storage.SystemError, 0)
		for rows.Next() {
			e := &storage.SystemError{}
			var errorSampleJSON, metadataJSON []byte
			var accountID, tenantID, datasetID sql.NullString
			var resolvedAt sql.NullTime

			err := rows.Scan(
				&e.ID, &e.ErrorHash, &e.ErrorType, &e.Component, &accountID, &tenantID, &datasetID,
				&e.ErrorMessage, &errorSampleJSON, &e.Severity, &e.Status, &e.OccurrenceCount,
				&e.FirstSeen, &e.LastSeen, &e.SampleRate, &e.SamplesCollected, &e.SamplesDropped,
				&metadataJSON, &e.CreatedAt, &e.UpdatedAt, &resolvedAt,
			)
			if err != nil {
				log.Errorf("Failed to scan error row: %v", err)
				continue
			}

			if accountID.Valid {
				e.AccountID = accountID.String
			}
			if tenantID.Valid {
				e.TenantID = tenantID.String
			}
			if datasetID.Valid {
				e.DatasetID = datasetID.String
			}
			if resolvedAt.Valid {
				e.ResolvedAt = &resolvedAt.Time
			}

			// Parse JSONB fields
			if len(errorSampleJSON) > 0 {
				if err := json.Unmarshal(errorSampleJSON, &e.ErrorSample); err != nil {
					log.Warnf("Failed to unmarshal error_sample for error %d: %v", e.ID, err)
				}
			}
			if len(metadataJSON) > 0 {
				if err := json.Unmarshal(metadataJSON, &e.Metadata); err != nil {
					log.Warnf("Failed to unmarshal metadata for error %d: %v", e.ID, err)
				}
			}

			errors = append(errors, e)
		}

		log.Infof("Successfully queried recent errors: %d records", len(errors))

		output.Errors = errors
		output.Count = len(errors)

		return nil
	})

	return u
}

// GetDatasetErrors handles GET requests for errors specific to a dataset
func (api *API) GetDatasetErrors() usecase.Interactor {
	type getDatasetErrorsInput struct {
		TenantID  string `path:"tenantId" minLength:"1"`
		DatasetID string `path:"datasetId" minLength:"1"`
		Status    string `query:"status" default:"active" description:"Filter by status: active, resolved, ignored (default: active)"`
		Limit     int    `query:"limit" minimum:"1" maximum:"1000" default:"100" description:"Maximum number of records to return"`
	}

	type getDatasetErrorsOutput struct {
		Errors []*storage.SystemError `json:"errors"`
		Count  int                    `json:"count"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input getDatasetErrorsInput, output *getDatasetErrorsOutput) error {
		api.Services.IncrementAPIRequests()

		// Type-assert to PostgreSQLStorage to access DB
		pgStorage, ok := api.Services.Storage.(*storage.PostgreSQLStorage)
		if !ok {
			return fmt.Errorf("storage backend is not PostgreSQL")
		}

		query := `
			SELECT id, error_hash, error_type, component, account_id, tenant_id, dataset_id,
				   error_message, error_sample, severity, status, occurrence_count,
				   first_seen, last_seen, sample_rate, samples_collected, samples_dropped,
				   metadata, created_at, updated_at, resolved_at
			FROM system_errors
			WHERE tenant_id = $1 AND dataset_id = $2 AND status = $3
			ORDER BY last_seen DESC
			LIMIT $4
		`

		rows, err := pgStorage.DB().QueryContext(ctx, query, input.TenantID, input.DatasetID, input.Status, input.Limit)
		if err != nil {
			log.Errorf("Failed to query dataset errors for %s/%s: %v", input.TenantID, input.DatasetID, err)
			return fmt.Errorf("failed to query dataset errors: %w", err)
		}
		defer rows.Close()

		errors := make([]*storage.SystemError, 0)
		for rows.Next() {
			e := &storage.SystemError{}
			var errorSampleJSON, metadataJSON []byte
			var accountID, tenantID, datasetID sql.NullString
			var resolvedAt sql.NullTime

			err := rows.Scan(
				&e.ID, &e.ErrorHash, &e.ErrorType, &e.Component, &accountID, &tenantID, &datasetID,
				&e.ErrorMessage, &errorSampleJSON, &e.Severity, &e.Status, &e.OccurrenceCount,
				&e.FirstSeen, &e.LastSeen, &e.SampleRate, &e.SamplesCollected, &e.SamplesDropped,
				&metadataJSON, &e.CreatedAt, &e.UpdatedAt, &resolvedAt,
			)
			if err != nil {
				log.Errorf("Failed to scan error row: %v", err)
				continue
			}

			if accountID.Valid {
				e.AccountID = accountID.String
			}
			if tenantID.Valid {
				e.TenantID = tenantID.String
			}
			if datasetID.Valid {
				e.DatasetID = datasetID.String
			}
			if resolvedAt.Valid {
				e.ResolvedAt = &resolvedAt.Time
			}

			// Parse JSONB fields
			if len(errorSampleJSON) > 0 {
				if err := json.Unmarshal(errorSampleJSON, &e.ErrorSample); err != nil {
					log.Warnf("Failed to unmarshal error_sample for error %d: %v", e.ID, err)
				}
			}
			if len(metadataJSON) > 0 {
				if err := json.Unmarshal(metadataJSON, &e.Metadata); err != nil {
					log.Warnf("Failed to unmarshal metadata for error %d: %v", e.ID, err)
				}
			}

			errors = append(errors, e)
		}

		log.Infof("Successfully queried dataset errors for %s/%s: %d records", input.TenantID, input.DatasetID, len(errors))

		output.Errors = errors
		output.Count = len(errors)

		return nil
	})

	return u
}

