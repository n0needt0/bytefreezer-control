package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/n0needt0/go-goodies/log"
)

// SaveTransformationHistory saves a transformation configuration to history
func (s *PostgreSQLStorage) SaveTransformationHistory(ctx context.Context, history *TransformationHistory) error {
	if history == nil {
		return fmt.Errorf("history cannot be nil")
	}

	if history.TenantID == "" || history.DatasetID == "" {
		return fmt.Errorf("tenant_id and dataset_id are required")
	}

	// Marshal config to JSONB
	configJSON, err := json.Marshal(history.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	query := `
		INSERT INTO control_transformation_history (
			tenant_id, dataset_id, config, deployed, deployed_at, created_at, created_by, label
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at`

	now := time.Now()
	var deployedAt interface{} = nil
	if history.Deployed && history.DeployedAt != nil {
		deployedAt = history.DeployedAt
	} else if history.Deployed {
		deployedAt = now
	}

	err = s.db.QueryRowContext(ctx, query,
		history.TenantID,
		history.DatasetID,
		configJSON,
		history.Deployed,
		deployedAt,
		now,
		history.CreatedBy,
		history.Label,
	).Scan(&history.ID, &history.CreatedAt)

	if err != nil {
		log.Errorf("Failed to save transformation history for %s/%s: %v", history.TenantID, history.DatasetID, err)
		return fmt.Errorf("failed to save transformation history: %w", err)
	}

	log.Infof("Saved transformation history for %s/%s (id: %d, deployed: %v)",
		history.TenantID, history.DatasetID, history.ID, history.Deployed)

	return nil
}

// GetTransformationHistory retrieves transformation history for a dataset
func (s *PostgreSQLStorage) GetTransformationHistory(ctx context.Context, tenantID, datasetID string, limit int) ([]*TransformationHistory, error) {
	if tenantID == "" || datasetID == "" {
		return nil, fmt.Errorf("tenant_id and dataset_id are required")
	}

	if limit <= 0 {
		limit = 10
	}

	query := `
		SELECT
			id, tenant_id, dataset_id, config, deployed, deployed_at, created_at, created_by, label
		FROM control_transformation_history
		WHERE tenant_id = $1 AND dataset_id = $2
		ORDER BY created_at DESC
		LIMIT $3`

	rows, err := s.db.QueryContext(ctx, query, tenantID, datasetID, limit)
	if err != nil {
		log.Errorf("Failed to query transformation history for %s/%s: %v", tenantID, datasetID, err)
		return nil, fmt.Errorf("failed to query transformation history: %w", err)
	}
	defer rows.Close()

	var histories []*TransformationHistory
	for rows.Next() {
		var history TransformationHistory
		var configJSON []byte
		var deployedAt sql.NullTime
		var createdBy, label sql.NullString

		err := rows.Scan(
			&history.ID,
			&history.TenantID,
			&history.DatasetID,
			&configJSON,
			&history.Deployed,
			&deployedAt,
			&history.CreatedAt,
			&createdBy,
			&label,
		)
		if err != nil {
			log.Errorf("Failed to scan transformation history row: %v", err)
			continue
		}

		// Unmarshal config
		if err := json.Unmarshal(configJSON, &history.Config); err != nil {
			log.Errorf("Failed to unmarshal config for history %d: %v", history.ID, err)
			continue
		}

		if deployedAt.Valid {
			history.DeployedAt = &deployedAt.Time
		}
		if createdBy.Valid {
			history.CreatedBy = createdBy.String
		}
		if label.Valid {
			history.Label = label.String
		}

		histories = append(histories, &history)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating transformation history rows: %w", err)
	}

	log.Debugf("Retrieved %d transformation history entries for %s/%s", len(histories), tenantID, datasetID)
	return histories, nil
}

// CleanupOldHistory removes old history entries keeping only the most recent ones
func (s *PostgreSQLStorage) CleanupOldHistory(ctx context.Context, tenantID, datasetID string, keepCount int) error {
	if tenantID == "" || datasetID == "" {
		return fmt.Errorf("tenant_id and dataset_id are required")
	}

	if keepCount <= 0 {
		keepCount = 10
	}

	query := `
		DELETE FROM control_transformation_history
		WHERE id IN (
			SELECT id FROM control_transformation_history
			WHERE tenant_id = $1 AND dataset_id = $2
			ORDER BY created_at DESC
			OFFSET $3
		)`

	result, err := s.db.ExecContext(ctx, query, tenantID, datasetID, keepCount)
	if err != nil {
		log.Errorf("Failed to cleanup old transformation history for %s/%s: %v", tenantID, datasetID, err)
		return fmt.Errorf("failed to cleanup old transformation history: %w", err)
	}

	rowsDeleted, _ := result.RowsAffected()
	if rowsDeleted > 0 {
		log.Infof("Cleaned up %d old transformation history entries for %s/%s (kept %d)",
			rowsDeleted, tenantID, datasetID, keepCount)
	}

	return nil
}
