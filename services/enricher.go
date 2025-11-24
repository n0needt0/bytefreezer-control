package services

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"mime/multipart"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/n0needt0/bytefreezer-control/storage"
	"github.com/n0needt0/go-goodies/log"
)

type EnricherService struct {
	db              *sql.DB
	auditLogService *AuditLogService
}

func NewEnricherService(db *sql.DB, auditLogService *AuditLogService) *EnricherService {
	return &EnricherService{
		db:              db,
		auditLogService: auditLogService,
	}
}

// CreateEnricher creates a new enricher
func (s *EnricherService) CreateEnricher(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, req *storage.CreateEnricherRequest) (*storage.Enricher, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	enricher := &storage.Enricher{
		ID:           uuid.New(),
		TenantID:     tenantID,
		Name:         req.Name,
		Description:  req.Description,
		Columns:      req.Columns,
		IndexColumns: req.IndexColumns,
		Status:       storage.EnricherStatusPending,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		CreatedBy:    userID,
	}

	columnsJSON, err := sonic.Marshal(enricher.Columns)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal columns: %w", err)
	}

	indexColumnsJSON, err := sonic.Marshal(enricher.IndexColumns)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal index columns: %w", err)
	}

	query := `
		INSERT INTO enrichers (id, tenant_id, name, description, columns, index_columns,
		                       status, created_at, updated_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, tenant_id, name, description, status, row_count, file_size,
		          created_at, updated_at`

	err = s.db.QueryRowContext(ctx, query,
		enricher.ID, enricher.TenantID, enricher.Name, enricher.Description,
		columnsJSON, indexColumnsJSON, enricher.Status,
		enricher.CreatedAt, enricher.UpdatedAt, enricher.CreatedBy,
	).Scan(
		&enricher.ID, &enricher.TenantID, &enricher.Name, &enricher.Description,
		&enricher.Status, &enricher.RowCount, &enricher.FileSize,
		&enricher.CreatedAt, &enricher.UpdatedAt,
	)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return nil, storage.ErrAlreadyExists
		}
		return nil, fmt.Errorf("failed to create enricher: %w", err)
	}

	log.Infof("Created enricher %s for tenant %s", enricher.ID, tenantID)

	return enricher, nil
}

// GetEnricher retrieves an enricher by ID
func (s *EnricherService) GetEnricher(ctx context.Context, tenantID, enricherID uuid.UUID) (*storage.Enricher, error) {
	enricher := &storage.Enricher{}
	var columnsJSON, indexColumnsJSON []byte

	query := `
		SELECT id, tenant_id, name, description, columns, index_columns,
		       row_count, file_size, status, error_message,
		       created_at, updated_at, created_by
		FROM enrichers
		WHERE id = $1 AND tenant_id = $2`

	err := s.db.QueryRowContext(ctx, query, enricherID, tenantID).Scan(
		&enricher.ID, &enricher.TenantID, &enricher.Name, &enricher.Description,
		&columnsJSON, &indexColumnsJSON,
		&enricher.RowCount, &enricher.FileSize,
		&enricher.Status, &enricher.ErrorMessage,
		&enricher.CreatedAt, &enricher.UpdatedAt, &enricher.CreatedBy,
	)

	if err == sql.ErrNoRows {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get enricher: %w", err)
	}

	if err := sonic.Unmarshal(columnsJSON, &enricher.Columns); err != nil {
		return nil, fmt.Errorf("failed to unmarshal columns: %w", err)
	}

	if err := sonic.Unmarshal(indexColumnsJSON, &enricher.IndexColumns); err != nil {
		return nil, fmt.Errorf("failed to unmarshal index columns: %w", err)
	}

	return enricher, nil
}

// ListEnrichers lists enrichers for a tenant
func (s *EnricherService) ListEnrichers(ctx context.Context, tenantID uuid.UUID, page, pageSize int) (*storage.EnricherListResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	// Get total count
	var total int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM enrichers WHERE tenant_id = $1`, tenantID).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count enrichers: %w", err)
	}

	// Get enrichers
	query := `
		SELECT id, tenant_id, name, description, columns, index_columns,
		       row_count, file_size, status, created_at, updated_at
		FROM enrichers
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := s.db.QueryContext(ctx, query, tenantID, pageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list enrichers: %w", err)
	}
	defer rows.Close()

	enrichers := make([]storage.Enricher, 0)
	for rows.Next() {
		var enricher storage.Enricher
		var columnsJSON, indexColumnsJSON []byte

		err := rows.Scan(
			&enricher.ID, &enricher.TenantID, &enricher.Name, &enricher.Description,
			&columnsJSON, &indexColumnsJSON,
			&enricher.RowCount, &enricher.FileSize, &enricher.Status,
			&enricher.CreatedAt, &enricher.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan enricher: %w", err)
		}

		if err := sonic.Unmarshal(columnsJSON, &enricher.Columns); err != nil {
			log.Warnf("Failed to unmarshal columns for enricher %s: %v", enricher.ID, err)
			continue
		}

		if err := sonic.Unmarshal(indexColumnsJSON, &enricher.IndexColumns); err != nil {
			log.Warnf("Failed to unmarshal index columns for enricher %s: %v", enricher.ID, err)
			continue
		}

		enrichers = append(enrichers, enricher)
	}

	totalPages := (total + pageSize - 1) / pageSize

	return &storage.EnricherListResponse{
		Enrichers:  enrichers,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// UploadEnricherData uploads CSV data to an enricher
func (s *EnricherService) UploadEnricherData(ctx context.Context, tenantID, enricherID uuid.UUID, userID *uuid.UUID, file multipart.File, filename string) error {
	// Get enricher
	enricher, err := s.GetEnricher(ctx, tenantID, enricherID)
	if err != nil {
		return err
	}

	// Create version
	version := 1
	var maxVersion int
	err = s.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(version), 0) FROM enricher_versions WHERE enricher_id = $1`, enricherID).Scan(&maxVersion)
	if err == nil {
		version = maxVersion + 1
	}

	versionID := uuid.New()

	// Parse and validate CSV
	reader := csv.NewReader(file)
	reader.ReuseRecord = true

	// Read header
	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("failed to read CSV header: %w", err)
	}

	// Validate header matches enricher columns
	if !s.validateHeader(header, enricher.Columns) {
		return fmt.Errorf("CSV header does not match enricher columns")
	}

	// Process CSV and convert to binary format (in memory)
	var buf bytes.Buffer
	rowCount, err := s.processCSVToBinary(reader, header, &buf)
	if err != nil {
		return fmt.Errorf("failed to process CSV: %w", err)
	}

	binaryData := buf.Bytes()
	fileSize := int64(len(binaryData))

	// Validate limits
	if rowCount > storage.MaxEnricherRows {
		return fmt.Errorf("row count %d exceeds maximum %d", rowCount, storage.MaxEnricherRows)
	}

	if fileSize > storage.MaxEnricherFileSize {
		return fmt.Errorf("file size %d exceeds maximum %d", fileSize, storage.MaxEnricherFileSize)
	}

	// Update enricher with binary data
	query := `
		UPDATE enrichers
		SET row_count = $1, file_size = $2, file_data = $3, status = $4, updated_at = $5
		WHERE id = $6 AND tenant_id = $7`

	_, err = s.db.ExecContext(ctx, query, rowCount, fileSize, binaryData, storage.EnricherStatusActive, time.Now(), enricherID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to update enricher: %w", err)
	}

	// Create version record
	query = `
		INSERT INTO enricher_versions (id, enricher_id, version, original_filename, file_size, row_count, status, processed_at, created_at, uploaded_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	now := time.Now()
	_, err = s.db.ExecContext(ctx, query, versionID, enricherID, version, filename, fileSize, rowCount, storage.EnricherStatusActive, now, now, userID)
	if err != nil {
		return fmt.Errorf("failed to create version: %w", err)
	}

	log.Infof("Uploaded data to enricher %s: %d rows, %d bytes", enricherID, rowCount, fileSize)

	return nil
}

// DeleteEnricher deletes an enricher
func (s *EnricherService) DeleteEnricher(ctx context.Context, tenantID, enricherID uuid.UUID) error {
	// Delete from database (cascade will delete versions and audit logs, file_data cleared automatically)
	_, err := s.db.ExecContext(ctx, `DELETE FROM enrichers WHERE id = $1 AND tenant_id = $2`, enricherID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to delete enricher: %w", err)
	}

	log.Infof("Deleted enricher %s for tenant %s", enricherID, tenantID)

	return nil
}

// validateHeader checks if CSV header matches expected columns
func (s *EnricherService) validateHeader(header []string, expectedColumns []string) bool {
	if len(header) != len(expectedColumns) {
		return false
	}

	headerMap := make(map[string]bool)
	for _, col := range header {
		headerMap[strings.TrimSpace(col)] = true
	}

	for _, expectedCol := range expectedColumns {
		if !headerMap[expectedCol] {
			return false
		}
	}

	return true
}

// processCSVToBinary converts CSV to binary format (JSON lines)
func (s *EnricherService) processCSVToBinary(reader *csv.Reader, header []string, buf *bytes.Buffer) (int, error) {
	rowCount := 0
	encoder := sonic.ConfigDefault.NewEncoder(buf)

	// Write header as first line
	if err := encoder.Encode(header); err != nil {
		return 0, err
	}

	// Process rows
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}

		// Validate cell values
		for _, cell := range record {
			if len(cell) > storage.MaxCellValueLength {
				return 0, fmt.Errorf("cell value exceeds maximum length of %d", storage.MaxCellValueLength)
			}
		}

		if err := encoder.Encode(record); err != nil {
			return 0, err
		}

		rowCount++
	}

	return rowCount, nil
}
