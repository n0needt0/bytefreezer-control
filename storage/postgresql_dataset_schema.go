package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	"github.com/n0needt0/go-goodies/log"
)

// UpsertDatasetSchema stores or updates the inferred schema for a dataset
func (s *PostgreSQLStorage) UpsertDatasetSchema(ctx context.Context, tenantID, datasetID, schemaType string, schema interface{}) error {
	// Marshal schema to JSON
	schemaJSON, err := sonic.Marshal(schema)
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", err)
	}

	query := `
		INSERT INTO dataset_schema (tenant_id, dataset_id, schema_type, schema_data, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (tenant_id, dataset_id, schema_type)
		DO UPDATE SET
			schema_data = EXCLUDED.schema_data,
			updated_at = EXCLUDED.updated_at
	`

	_, err = s.db.ExecContext(ctx, query,
		tenantID,
		datasetID,
		schemaType,
		schemaJSON,
		time.Now(),
	)
	if err != nil {
		return fmt.Errorf("failed to upsert dataset schema: %w", err)
	}

	log.Infof("Upserted %s schema for %s/%s", schemaType, tenantID, datasetID)
	return nil
}

// GetDatasetSchemaWithMetadata retrieves the cached schema with metadata for a dataset
func (s *PostgreSQLStorage) GetDatasetSchemaWithMetadata(ctx context.Context, tenantID, datasetID, schemaType string) (*DatasetSchema, error) {
	query := `
		SELECT id, tenant_id, dataset_id, schema_type, schema_data, updated_at
		FROM dataset_schema
		WHERE tenant_id = $1 AND dataset_id = $2 AND schema_type = $3
	`

	var schema DatasetSchema
	err := s.db.QueryRowContext(ctx, query, tenantID, datasetID, schemaType).Scan(
		&schema.ID,
		&schema.TenantID,
		&schema.DatasetID,
		&schema.SchemaType,
		&schema.SchemaData,
		&schema.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil // No schema cached
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get dataset schema: %w", err)
	}

	return &schema, nil
}

// GetDatasetSchema retrieves the cached schema for a dataset
func (s *PostgreSQLStorage) GetDatasetSchema(ctx context.Context, tenantID, datasetID, schemaType string) ([]byte, error) {
	query := `
		SELECT schema_data
		FROM dataset_schema
		WHERE tenant_id = $1 AND dataset_id = $2 AND schema_type = $3
	`

	var schemaData []byte
	err := s.db.QueryRowContext(ctx, query, tenantID, datasetID, schemaType).Scan(&schemaData)
	if err == sql.ErrNoRows {
		return nil, nil // No schema cached
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get dataset schema: %w", err)
	}

	return schemaData, nil
}

// UpsertDatasetSamples stores samples and keeps only the latest N samples per type
func (s *PostgreSQLStorage) UpsertDatasetSamples(ctx context.Context, tenantID, datasetID, sampleType string, samples []DatasetSample, keepCount int) error {
	if len(samples) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert new samples
	insertQuery := `
		INSERT INTO dataset_samples (tenant_id, dataset_id, sample_type, line_number, sample_data, batch_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	for _, sample := range samples {
		sampleJSON, err := sonic.Marshal(sample.SampleData)
		if err != nil {
			log.Errorf("Failed to marshal sample data: %v", err)
			continue
		}

		_, err = tx.ExecContext(ctx, insertQuery,
			sample.TenantID,
			sample.DatasetID,
			sample.SampleType,
			sample.LineNumber,
			sampleJSON,
			sample.BatchID,
			sample.CreatedAt,
		)
		if err != nil {
			log.Errorf("Failed to insert sample: %v", err)
			continue
		}
	}

	// Delete old samples, keeping only the latest keepCount
	deleteQuery := `
		DELETE FROM dataset_samples
		WHERE tenant_id = $1
		  AND dataset_id = $2
		  AND sample_type = $3
		  AND id NOT IN (
			SELECT id FROM dataset_samples
			WHERE tenant_id = $1
			  AND dataset_id = $2
			  AND sample_type = $3
			ORDER BY created_at DESC
			LIMIT $4
		  )
	`
	_, err = tx.ExecContext(ctx, deleteQuery, tenantID, datasetID, sampleType, keepCount)
	if err != nil {
		return fmt.Errorf("failed to delete old samples: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Infof("Upserted %d %s samples for %s/%s (keeping %d latest)",
		len(samples), sampleType, tenantID, datasetID, keepCount)

	return nil
}

// GetDatasetSamples retrieves the latest N samples for a given type
func (s *PostgreSQLStorage) GetDatasetSamples(ctx context.Context, tenantID, datasetID, sampleType string, limit int) ([]DatasetSample, error) {
	query := `
		SELECT id, tenant_id, dataset_id, sample_type, line_number, sample_data, batch_id, created_at
		FROM dataset_samples
		WHERE tenant_id = $1 AND dataset_id = $2 AND sample_type = $3
		ORDER BY created_at DESC
		LIMIT $4
	`

	rows, err := s.db.QueryContext(ctx, query, tenantID, datasetID, sampleType, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query dataset samples: %w", err)
	}
	defer rows.Close()

	samples := []DatasetSample{}
	for rows.Next() {
		var sample DatasetSample
		var sampleDataJSON []byte

		err := rows.Scan(
			&sample.ID,
			&sample.TenantID,
			&sample.DatasetID,
			&sample.SampleType,
			&sample.LineNumber,
			&sampleDataJSON,
			&sample.BatchID,
			&sample.CreatedAt,
		)
		if err != nil {
			log.Errorf("Failed to scan sample row: %v", err)
			continue
		}

		// Unmarshal JSON data
		if err := sonic.Unmarshal(sampleDataJSON, &sample.SampleData); err != nil {
			log.Errorf("Failed to unmarshal sample data: %v", err)
			continue
		}

		samples = append(samples, sample)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating samples: %w", err)
	}

	return samples, nil
}
