package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/n0needt0/go-goodies/log"
)

// UpsertPiperFilter inserts or updates a piper filter in the catalog
func (s *PostgreSQLStorage) UpsertPiperFilter(ctx context.Context, filter *PiperFilter) error {
	if filter == nil {
		return fmt.Errorf("filter cannot be nil")
	}

	if filter.FilterType == "" {
		return fmt.Errorf("filter_type cannot be empty")
	}

	// Marshal JSONB fields
	parametersJSON, err := json.Marshal(filter.Parameters)
	if err != nil {
		return fmt.Errorf("failed to marshal parameters: %w", err)
	}

	examplesJSON, err := json.Marshal(filter.Examples)
	if err != nil {
		return fmt.Errorf("failed to marshal examples: %w", err)
	}

	query := `
		INSERT INTO control_piper_filter_catalog (
			filter_type, display_name, category, purpose, parameters, examples, version, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (filter_type) DO UPDATE SET
			display_name = EXCLUDED.display_name,
			category = EXCLUDED.category,
			purpose = EXCLUDED.purpose,
			parameters = EXCLUDED.parameters,
			examples = EXCLUDED.examples,
			version = EXCLUDED.version,
			updated_at = NOW()
		RETURNING updated_at, created_at`

	now := time.Now()
	err = s.db.QueryRowContext(ctx, query,
		filter.FilterType,
		filter.DisplayName,
		filter.Category,
		filter.Purpose,
		parametersJSON,
		examplesJSON,
		filter.Version,
		now,
		now,
	).Scan(&filter.UpdatedAt, &filter.CreatedAt)

	if err != nil {
		log.Errorf("Failed to upsert piper filter %s: %v", filter.FilterType, err)
		return fmt.Errorf("failed to upsert piper filter: %w", err)
	}

	log.Infof("Upserted piper filter: %s (category: %s)", filter.FilterType, filter.Category)
	return nil
}

// GetPiperFilter retrieves a specific piper filter by type
func (s *PostgreSQLStorage) GetPiperFilter(ctx context.Context, filterType string) (*PiperFilter, error)  {
	if filterType == "" {
		return nil, fmt.Errorf("filter_type cannot be empty")
	}

	query := `
		SELECT
			filter_type, display_name, category, purpose,
			parameters, examples, version, updated_at, created_at
		FROM control_piper_filter_catalog
		WHERE filter_type = $1`

	var filter PiperFilter
	var parametersJSON, examplesJSON []byte

	err := s.db.QueryRowContext(ctx, query, filterType).Scan(
		&filter.FilterType,
		&filter.DisplayName,
		&filter.Category,
		&filter.Purpose,
		&parametersJSON,
		&examplesJSON,
		&filter.Version,
		&filter.UpdatedAt,
		&filter.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("filter not found: %s", filterType)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get piper filter: %w", err)
	}

	// Unmarshal JSONB fields
	if err := json.Unmarshal(parametersJSON, &filter.Parameters); err != nil {
		return nil, fmt.Errorf("failed to unmarshal parameters: %w", err)
	}
	if err := json.Unmarshal(examplesJSON, &filter.Examples); err != nil {
		return nil, fmt.Errorf("failed to unmarshal examples: %w", err)
	}

	return &filter, nil
}

// ListPiperFilters retrieves all piper filters
func (s *PostgreSQLStorage) ListPiperFilters(ctx context.Context) ([]*PiperFilter, error) {
	query := `
		SELECT
			filter_type, display_name, category, purpose,
			parameters, examples, version, updated_at, created_at
		FROM control_piper_filter_catalog
		ORDER BY category, filter_type`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list piper filters: %w", err)
	}
	defer rows.Close()

	var filters []*PiperFilter
	for rows.Next() {
		var filter PiperFilter
		var parametersJSON, examplesJSON []byte

		err := rows.Scan(
			&filter.FilterType,
			&filter.DisplayName,
			&filter.Category,
			&filter.Purpose,
			&parametersJSON,
			&examplesJSON,
			&filter.Version,
			&filter.UpdatedAt,
			&filter.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan piper filter: %w", err)
		}

		// Unmarshal JSONB fields
		if err := json.Unmarshal(parametersJSON, &filter.Parameters); err != nil {
			return nil, fmt.Errorf("failed to unmarshal parameters: %w", err)
		}
		if err := json.Unmarshal(examplesJSON, &filter.Examples); err != nil {
			return nil, fmt.Errorf("failed to unmarshal examples: %w", err)
		}

		filters = append(filters, &filter)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating piper filters: %w", err)
	}

	return filters, nil
}

// ListPiperFiltersByCategory retrieves piper filters for a specific category
func (s *PostgreSQLStorage) ListPiperFiltersByCategory(ctx context.Context, category string) ([]*PiperFilter, error) {
	if category == "" {
		return nil, fmt.Errorf("category cannot be empty")
	}

	query := `
		SELECT
			filter_type, display_name, category, purpose,
			parameters, examples, version, updated_at, created_at
		FROM control_piper_filter_catalog
		WHERE category = $1
		ORDER BY filter_type`

	rows, err := s.db.QueryContext(ctx, query, category)
	if err != nil {
		return nil, fmt.Errorf("failed to list piper filters by category: %w", err)
	}
	defer rows.Close()

	var filters []*PiperFilter
	for rows.Next() {
		var filter PiperFilter
		var parametersJSON, examplesJSON []byte

		err := rows.Scan(
			&filter.FilterType,
			&filter.DisplayName,
			&filter.Category,
			&filter.Purpose,
			&parametersJSON,
			&examplesJSON,
			&filter.Version,
			&filter.UpdatedAt,
			&filter.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan piper filter: %w", err)
		}

		// Unmarshal JSONB fields
		if err := json.Unmarshal(parametersJSON, &filter.Parameters); err != nil {
			return nil, fmt.Errorf("failed to unmarshal parameters: %w", err)
		}
		if err := json.Unmarshal(examplesJSON, &filter.Examples); err != nil {
			return nil, fmt.Errorf("failed to unmarshal examples: %w", err)
		}

		filters = append(filters, &filter)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating piper filters: %w", err)
	}

	return filters, nil
}

// DeletePiperFilter deletes a piper filter from the catalog
func (s *PostgreSQLStorage) DeletePiperFilter(ctx context.Context, filterType string) error {
	if filterType == "" {
		return fmt.Errorf("filter_type cannot be empty")
	}

	query := `DELETE FROM control_piper_filter_catalog WHERE filter_type = $1`

	result, err := s.db.ExecContext(ctx, query, filterType)
	if err != nil {
		return fmt.Errorf("failed to delete piper filter: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("filter not found: %s", filterType)
	}

	log.Infof("Deleted piper filter: %s", filterType)
	return nil
}

// GetPiperFilterCatalog retrieves the filter catalog in various formats
// format can be: "ui", "ai", or empty for full catalog
func (s *PostgreSQLStorage) GetPiperFilterCatalog(ctx context.Context, format string) (interface{}, error) {
	filters, err := s.ListPiperFilters(ctx)
	if err != nil {
		return nil, err
	}

	switch format {
	case "ui":
		// UI-optimized format: grouped by category
		grouped := make(map[string][]*PiperFilter)
		for _, filter := range filters {
			grouped[filter.Category] = append(grouped[filter.Category], filter)
		}
		return map[string]interface{}{
			"filters_by_category": grouped,
			"total_filters":       len(filters),
			"categories":          getCategories(filters),
		}, nil

	case "ai":
		// AI-optimized format: flattened with examples
		var aiFilters []map[string]interface{}
		for _, filter := range filters {
			aiFilters = append(aiFilters, map[string]interface{}{
				"type":        filter.FilterType,
				"name":        filter.DisplayName,
				"category":    filter.Category,
				"purpose":     filter.Purpose,
				"parameters":  filter.Parameters,
				"examples":    filter.Examples,
			})
		}
		return map[string]interface{}{
			"filters": aiFilters,
			"version": getLatestVersion(filters),
		}, nil

	default:
		// Full catalog
		return filters, nil
	}
}

// getCategories extracts unique categories from filters
func getCategories(filters []*PiperFilter) []string {
	categoryMap := make(map[string]bool)
	for _, filter := range filters {
		categoryMap[filter.Category] = true
	}

	categories := make([]string, 0, len(categoryMap))
	for category := range categoryMap {
		categories = append(categories, category)
	}

	return categories
}

// getLatestVersion finds the latest version string from filters
func getLatestVersion(filters []*PiperFilter) string {
	if len(filters) == 0 {
		return ""
	}
	// Return the first non-empty version (they should all be the same)
	for _, filter := range filters {
		if filter.Version != "" {
			return filter.Version
		}
	}
	return ""
}
