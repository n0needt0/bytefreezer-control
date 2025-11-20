package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/n0needt0/bytefreezer-control/storage"
	"github.com/n0needt0/go-goodies/log"
	"github.com/swaggest/usecase"
	usecaseStatus "github.com/swaggest/usecase/status"
)

// Transformation API request/response types

type FilterConfig struct {
	Type    string                 `json:"type"`
	Config  map[string]interface{} `json:"config"`
	Enabled bool                   `json:"enabled"`
}

type TransformationSample struct {
	LineNumber int                    `json:"line_number"`
	RawData    string                 `json:"raw_data"`
	ParsedData map[string]interface{} `json:"parsed_data"`
}

type TransformationJobResponse struct {
	JobID     string    `json:"job_id"`
	TenantID  string    `json:"tenant_id"`
	DatasetID string    `json:"dataset_id"`
	JobType   string    `json:"job_type"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	Message   string    `json:"message,omitempty"`
}

type TransformationJobStatusResponse struct {
	JobID       string      `json:"job_id"`
	TenantID    string      `json:"tenant_id"`
	DatasetID   string      `json:"dataset_id"`
	JobType     string      `json:"job_type"`
	Status      string      `json:"status"`
	ProcessorID string      `json:"processor_id,omitempty"`
	Result      interface{} `json:"result,omitempty"`
	ErrorMsg    string      `json:"error_message,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
	StartedAt   *time.Time  `json:"started_at,omitempty"`
	CompletedAt *time.Time  `json:"completed_at,omitempty"`
}

// CreateTransformationTest creates a test transformation job
func (api *API) CreateTransformationTest() usecase.Interactor {
	type Input struct {
		TenantID  string                 `path:"tenantId" required:"true"`
		DatasetID string                 `path:"datasetId" required:"true"`
		Filters   []FilterConfig         `json:"filters" required:"true"`
		Samples   []TransformationSample `json:"samples" required:"true"`
	}

	type Output struct {
		Response TransformationJobResponse `json:"response"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input Input, output *Output) error {
		// Create transformation job in piper_transformation_jobs table
		job := &storage.PiperTransformationJob{
			JobID:     uuid.New().String(),
			TenantID:  input.TenantID,
			DatasetID: input.DatasetID,
			JobType:   storage.PiperTransformationJobTypeTest,
			Status:    storage.PiperJobStatusPending,
			Priority:  10, // Test jobs have priority 10
			Request: map[string]interface{}{
				"tenant_id":  input.TenantID,
				"dataset_id": input.DatasetID,
				"filters":    input.Filters,
				"samples":    input.Samples,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			TTL:       time.Now().Add(1 * time.Hour), // 1 hour TTL
		}

		if err := api.Services.Storage.CreatePiperTransformationJob(ctx, job); err != nil {
			log.Errorf("Failed to create transformation test job: %v", err)
			return fmt.Errorf("failed to create transformation job: %w", err)
		}

		log.Infof("Created transformation test job %s for %s/%s", job.JobID, input.TenantID, input.DatasetID)

		output.Response = TransformationJobResponse{
			JobID:     job.JobID,
			TenantID:  job.TenantID,
			DatasetID: job.DatasetID,
			JobType:   string(job.JobType),
			Status:    string(job.Status),
			CreatedAt: job.CreatedAt,
			Message:   "Transformation test job created successfully",
		}

		return nil
	})

	u.SetExpectedErrors(usecaseStatus.InvalidArgument)
	return u
}

// CreateTransformationValidate creates a validate transformation job
func (api *API) CreateTransformationValidate() usecase.Interactor {
	type Input struct {
		TenantID  string         `path:"tenantId" required:"true"`
		DatasetID string         `path:"datasetId" required:"true"`
		Filters   []FilterConfig `json:"filters" required:"true"`
		Count     int            `json:"count" required:"true"`
	}

	type Output struct {
		Response TransformationJobResponse `json:"response"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input Input, output *Output) error {
		// Create transformation job in piper_transformation_jobs table
		job := &storage.PiperTransformationJob{
			JobID:     uuid.New().String(),
			TenantID:  input.TenantID,
			DatasetID: input.DatasetID,
			JobType:   storage.PiperTransformationJobTypeValidate,
			Status:    storage.PiperJobStatusPending,
			Priority:  8, // Validate jobs have priority 8
			Request: map[string]interface{}{
				"tenant_id":  input.TenantID,
				"dataset_id": input.DatasetID,
				"filters":    input.Filters,
				"count":      input.Count,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			TTL:       time.Now().Add(1 * time.Hour), // 1 hour TTL
		}

		if err := api.Services.Storage.CreatePiperTransformationJob(ctx, job); err != nil {
			log.Errorf("Failed to create transformation validate job: %v", err)
			return fmt.Errorf("failed to create transformation job: %w", err)
		}

		log.Infof("Created transformation validate job %s for %s/%s", job.JobID, input.TenantID, input.DatasetID)

		output.Response = TransformationJobResponse{
			JobID:     job.JobID,
			TenantID:  job.TenantID,
			DatasetID: job.DatasetID,
			JobType:   string(job.JobType),
			Status:    string(job.Status),
			CreatedAt: job.CreatedAt,
			Message:   "Transformation validate job created successfully",
		}

		return nil
	})

	u.SetExpectedErrors(usecaseStatus.InvalidArgument)
	return u
}

// CreateTransformationActivate creates an activate transformation job
func (api *API) CreateTransformationActivate() usecase.Interactor {
	type Input struct {
		TenantID  string         `path:"tenantId" required:"true"`
		DatasetID string         `path:"datasetId" required:"true"`
		Filters   []FilterConfig `json:"filters" required:"true"`
		Enabled   bool           `json:"enabled"`
	}

	type Output struct {
		Response TransformationJobResponse `json:"response"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input Input, output *Output) error {
		// Create transformation job in piper_transformation_jobs table
		job := &storage.PiperTransformationJob{
			JobID:     uuid.New().String(),
			TenantID:  input.TenantID,
			DatasetID: input.DatasetID,
			JobType:   storage.PiperTransformationJobTypeActivate,
			Status:    storage.PiperJobStatusPending,
			Priority:  5, // Activate jobs have priority 5
			Request: map[string]interface{}{
				"tenant_id":  input.TenantID,
				"dataset_id": input.DatasetID,
				"filters":    input.Filters,
				"enabled":    input.Enabled,
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			TTL:       time.Now().Add(1 * time.Hour), // 1 hour TTL
		}

		if err := api.Services.Storage.CreatePiperTransformationJob(ctx, job); err != nil {
			log.Errorf("Failed to create transformation activate job: %v", err)
			return fmt.Errorf("failed to create transformation job: %w", err)
		}

		log.Infof("Created transformation activate job %s for %s/%s", job.JobID, input.TenantID, input.DatasetID)

		output.Response = TransformationJobResponse{
			JobID:     job.JobID,
			TenantID:  job.TenantID,
			DatasetID: job.DatasetID,
			JobType:   string(job.JobType),
			Status:    string(job.Status),
			CreatedAt: job.CreatedAt,
			Message:   "Transformation activate job created successfully",
		}

		return nil
	})

	u.SetExpectedErrors(usecaseStatus.InvalidArgument)
	return u
}

// GetTransformationJobStatus retrieves transformation job status and results
func (api *API) GetTransformationJobStatus() usecase.Interactor {
	type Input struct {
		JobID string `path:"jobId" required:"true"`
	}

	type Output struct {
		Response TransformationJobStatusResponse `json:"response"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input Input, output *Output) error {
		job, err := api.Services.Storage.GetPiperTransformationJob(ctx, input.JobID)
		if err != nil {
			log.Errorf("Failed to get transformation job %s: %v", input.JobID, err)
			return fmt.Errorf("failed to get transformation job: %w", err)
		}

		if job == nil {
			return usecaseStatus.NotFound
		}

		output.Response = TransformationJobStatusResponse{
			JobID:       job.JobID,
			TenantID:    job.TenantID,
			DatasetID:   job.DatasetID,
			JobType:     string(job.JobType),
			Status:      string(job.Status),
			ProcessorID: job.ProcessorID,
			Result:      job.Result,
			ErrorMsg:    job.ErrorMsg,
			CreatedAt:   job.CreatedAt,
			UpdatedAt:   job.UpdatedAt,
			StartedAt:   job.StartedAt,
			CompletedAt: job.CompletedAt,
		}

		return nil
	})

	u.SetExpectedErrors(usecaseStatus.NotFound)
	return u
}

// ListTransformationJobs retrieves all transformation jobs for a dataset
func (api *API) ListTransformationJobs() usecase.Interactor {
	type Input struct {
		TenantID  string `path:"tenantId" required:"true"`
		DatasetID string `path:"datasetId" required:"true"`
	}

	type Output struct {
		Jobs []TransformationJobStatusResponse `json:"jobs"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input Input, output *Output) error {
		jobs, err := api.Services.Storage.ListTransformationJobs(ctx, input.TenantID, input.DatasetID)
		if err != nil {
			log.Errorf("Failed to list transformation jobs for %s/%s: %v", input.TenantID, input.DatasetID, err)
			return fmt.Errorf("failed to list transformation jobs: %w", err)
		}

		// Convert to response format
		output.Jobs = make([]TransformationJobStatusResponse, 0, len(jobs))
		for _, job := range jobs {
			output.Jobs = append(output.Jobs, TransformationJobStatusResponse{
				JobID:       job.JobID,
				TenantID:    job.TenantID,
				DatasetID:   job.DatasetID,
				JobType:     string(job.JobType),
				Status:      string(job.Status),
				ProcessorID: job.ProcessorID,
				Result:      job.Result,
				ErrorMsg:    job.ErrorMsg,
				CreatedAt:   job.CreatedAt,
				UpdatedAt:   job.UpdatedAt,
				StartedAt:   job.StartedAt,
				CompletedAt: job.CompletedAt,
			})
		}

		return nil
	})

	return u
}

// GetTransformationSchema retrieves cached schema and samples from database
func (api *API) GetTransformationSchema() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract path parameters
		tenantID := r.PathValue("tenantId")
		datasetID := r.PathValue("datasetId")

		// Schema type from query param (default to "output")
		schemaType := r.URL.Query().Get("type")
		if schemaType == "" {
			schemaType = "output"
		}

		// Get dataset configuration to check if enabled
		dataset, err := api.Services.Storage.GetDataset(r.Context(), tenantID, datasetID)
		if err != nil {
			log.Errorf("Failed to get dataset configuration: %v", err)
			http.Error(w, "Failed to retrieve dataset configuration", http.StatusInternalServerError)
			return
		}

		datasetEnabled := dataset != nil && dataset.Active

		// Get schema with metadata from database
		schemaMetadata, err := api.Services.Storage.GetDatasetSchemaWithMetadata(r.Context(), tenantID, datasetID, schemaType)
		if err != nil {
			log.Errorf("Failed to get dataset schema: %v", err)
			http.Error(w, "Failed to retrieve schema", http.StatusInternalServerError)
			return
		}

		// Build response with metadata
		type SchemaResponse struct {
			Schema           interface{} `json:"schema,omitempty"`
			Samples          []interface{} `json:"samples,omitempty"`
			SchemaUpdatedAt  *time.Time  `json:"schema_updated_at,omitempty"`
			DatasetEnabled   bool        `json:"dataset_enabled"`
			LastBatchTime    *time.Time  `json:"last_batch_time,omitempty"`
			SchemaAgeSeconds *int64      `json:"schema_age_seconds,omitempty"`
			Error            string      `json:"error,omitempty"`
		}

		if schemaMetadata == nil {
			// No schema cached yet - differentiate based on dataset enabled status
			response := SchemaResponse{
				DatasetEnabled: datasetEnabled,
			}

			if !datasetEnabled {
				response.Error = "Pipeline not configured. Please configure the pipeline for this dataset."
			} else {
				response.Error = "Schema not available yet. Piper will submit schema after processing the first batch."
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(response) // #nosec G104
			return
		}

		// Get samples from database
		samples, err := api.Services.Storage.GetDatasetSamples(r.Context(), tenantID, datasetID, schemaType, 10)
		if err != nil {
			log.Errorf("Failed to get dataset samples: %v", err)
			// Continue without samples, schema is more important
			samples = []storage.DatasetSample{}
		}

		// Get latest batch processing time from metrics
		var lastBatchTime *time.Time
		endTime := time.Now()
		startTime := endTime.Add(-7 * 24 * time.Hour) // Look back 7 days

		metricsFilter := storage.MetricsQueryFilter{
			TenantID:  tenantID,
			DatasetID: datasetID,
			StartTime: startTime,
			EndTime:   endTime,
			Component: "piper", // Filter for piper component
			Limit:     1,       // Only need the most recent
		}

		metrics, err := api.Services.Storage.QueryDatasetMetrics(r.Context(), metricsFilter)
		if err != nil {
			log.Debugf("Failed to query dataset metrics: %v", err)
			// Continue without last batch time
		} else if len(metrics) > 0 {
			lastBatchTime = &metrics[0].RecordedAt
		}

		// Parse schema data which has structure: {"fields": [...], "inferred_at": "...", "sample_count": 10}
		var schemaData map[string]interface{}
		if err := json.Unmarshal(schemaMetadata.SchemaData, &schemaData); err != nil {
			log.Errorf("Failed to unmarshal schema: %v", err)
			http.Error(w, "Failed to parse schema", http.StatusInternalServerError)
			return
		}

		// Transform piper schema format to UI expected format
		// Piper format: {"name": "field", "types": ["string", "null"]}
		// UI format: {"name": "field", "type": "string", "count": 10, "nullable": true, "sample": "value"}
		var schemaFields []map[string]interface{}
		if fieldsRaw, ok := schemaData["fields"]; ok {
			if fieldsList, ok := fieldsRaw.([]interface{}); ok {
				schemaFields = make([]map[string]interface{}, 0, len(fieldsList))
				for _, fieldRaw := range fieldsList {
					if field, ok := fieldRaw.(map[string]interface{}); ok {
						transformedField := make(map[string]interface{})

						// Copy name
						if name, ok := field["name"]; ok {
							transformedField["name"] = name
						}

						// Transform types array to single type + nullable flag
						nullable := false
						primaryType := "unknown"
						if types, ok := field["types"].([]interface{}); ok && len(types) > 0 {
							for _, t := range types {
								if typeStr, ok := t.(string); ok {
									if typeStr == "null" {
										nullable = true
									} else {
										primaryType = typeStr
									}
								}
							}
						}
						transformedField["type"] = primaryType
						transformedField["nullable"] = nullable

						// Add count (sample_count from schema data)
						if sampleCount, ok := schemaData["sample_count"]; ok {
							transformedField["count"] = sampleCount
						} else {
							transformedField["count"] = 0
						}

						// Add sample if available
						if sample, ok := field["sample"]; ok {
							transformedField["sample"] = sample
						}

						schemaFields = append(schemaFields, transformedField)
					}
				}
			}
		}

		// Convert samples to response format with line numbers
		sampleData := make([]interface{}, 0, len(samples))
		for _, s := range samples {
			sampleData = append(sampleData, map[string]interface{}{
				"line_number":  s.LineNumber,
				"parsed_data":  s.SampleData,
			})
		}

		// Calculate schema age in seconds if we have last batch time
		var schemaAgeSeconds *int64
		if lastBatchTime != nil && !schemaMetadata.UpdatedAt.IsZero() {
			ageSeconds := int64(lastBatchTime.Sub(schemaMetadata.UpdatedAt).Seconds())
			schemaAgeSeconds = &ageSeconds
		}

		response := SchemaResponse{
			Schema:           schemaFields,
			Samples:          sampleData,
			SchemaUpdatedAt:  &schemaMetadata.UpdatedAt,
			DatasetEnabled:   datasetEnabled,
			LastBatchTime:    lastBatchTime,
			SchemaAgeSeconds: schemaAgeSeconds,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response) // #nosec G104
	}
}

// GetTransformationJSONSchema returns a JSON Schema (draft-07) format schema
func (api *API) GetTransformationJSONSchema() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract path parameters
		tenantID := r.PathValue("tenantId")
		datasetID := r.PathValue("datasetId")

		// Schema type from query param (default to "output")
		schemaType := r.URL.Query().Get("type")
		if schemaType == "" {
			schemaType = "output"
		}

		// Get dataset configuration
		dataset, err := api.Services.Storage.GetDataset(r.Context(), tenantID, datasetID)
		if err != nil {
			log.Errorf("Failed to get dataset configuration: %v", err)
			http.Error(w, "Failed to retrieve dataset configuration", http.StatusInternalServerError)
			return
		}

		if dataset == nil || !dataset.Active {
			http.Error(w, "Dataset not found or not active", http.StatusNotFound)
			return
		}

		// Get schema with metadata from database
		schemaMetadata, err := api.Services.Storage.GetDatasetSchemaWithMetadata(r.Context(), tenantID, datasetID, schemaType)
		if err != nil {
			log.Errorf("Failed to get dataset schema: %v", err)
			http.Error(w, "Failed to retrieve schema", http.StatusInternalServerError)
			return
		}

		if schemaMetadata == nil {
			http.Error(w, "Schema not available yet", http.StatusNotFound)
			return
		}

		// Parse schema data
		var schemaData map[string]interface{}
		if err := json.Unmarshal(schemaMetadata.SchemaData, &schemaData); err != nil {
			log.Errorf("Failed to unmarshal schema: %v", err)
			http.Error(w, "Failed to parse schema", http.StatusInternalServerError)
			return
		}

		// Get samples for example values
		samples, _ := api.Services.Storage.GetDatasetSamples(r.Context(), tenantID, datasetID, schemaType, 1)

		// Map types to JSON Schema types
		mapToJSONSchemaType := func(types []interface{}) interface{} {
			if len(types) == 0 {
				return []string{"string", "null"}
			}

			var schemaTypes []string
			for _, t := range types {
				if typeStr, ok := t.(string); ok {
					switch typeStr {
					case "string", "number", "boolean", "array", "object", "null":
						schemaTypes = append(schemaTypes, typeStr)
					default:
						schemaTypes = append(schemaTypes, "string")
					}
				}
			}

			if len(schemaTypes) == 1 {
				return schemaTypes[0]
			}
			return schemaTypes
		}

		// Build JSON Schema properties
		properties := make(map[string]interface{})

		if fieldsRaw, ok := schemaData["fields"]; ok {
			if fieldsList, ok := fieldsRaw.([]interface{}); ok {
				for _, fieldRaw := range fieldsList {
					if field, ok := fieldRaw.(map[string]interface{}); ok {
						fieldName, _ := field["name"].(string)
						types, _ := field["types"].([]interface{})

						fieldSchema := map[string]interface{}{
							"type": mapToJSONSchemaType(types),
						}

						// Add example value from sample if available
						if len(samples) > 0 {
							if exampleVal, ok := samples[0].SampleData[fieldName]; ok {
								fieldSchema["examples"] = []interface{}{exampleVal}
							}
						}

						properties[fieldName] = fieldSchema
					}
				}
			}
		}

		// Get total records count
		var totalRecords int64
		if sampleCount, ok := schemaData["sample_count"].(float64); ok {
			totalRecords = int64(sampleCount)
		}

		// Create JSON Schema document
		jsonSchema := map[string]interface{}{
			"$schema":     "http://json-schema.org/draft-07/schema#",
			"$id":         fmt.Sprintf("https://bytefreezer.io/schemas/%s/%s", tenantID, datasetID),
			"title":       fmt.Sprintf("%s Dataset Schema", datasetID),
			"description": fmt.Sprintf("Schema for %s/%s dataset", tenantID, datasetID),
			"type":        "object",
			"properties":  properties,
			"additionalProperties": true,
			"metadata": map[string]interface{}{
				"tenant_id":         tenantID,
				"dataset_id":        datasetID,
				"schema_type":       schemaType,
				"generated_at":      time.Now().Format(time.RFC3339),
				"schema_updated_at": schemaMetadata.UpdatedAt.Format(time.RFC3339),
				"total_fields":      len(properties),
				"total_records":     totalRecords,
			},
		}

		w.Header().Set("Content-Type", "application/schema+json")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s-%s-schema.json", tenantID, datasetID))
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(jsonSchema) // #nosec G104
	}
}

// getPiperURLForTenant gets the correct piper URL based on tenant's account deployment type
func (api *API) getPiperURLForTenant(ctx context.Context, tenantID string) (string, error) {
	// Get tenant to find account_id
	tenant, err := api.Services.Storage.GetTenantByID(ctx, tenantID)
	if err != nil {
		return "", fmt.Errorf("failed to get tenant: %w", err)
	}

	// Get account to check deployment type
	account, err := api.Services.Storage.GetAccount(ctx, tenant.AccountID)
	if err != nil {
		return "", fmt.Errorf("failed to get account: %w", err)
	}

	// Route to correct piper based on deployment type
	var piperURL string
	switch account.DeploymentType {
	case storage.DeploymentTypeManaged:
		piperURL = api.Config.Services.PiperManagedURL
		if piperURL == "" {
			piperURL = api.Config.Services.PiperURL // fallback to legacy config
		}
	case storage.DeploymentTypeOnPrem, storage.DeploymentTypeAirGapped:
		piperURL = api.Config.Services.PiperOnPremURL
		if piperURL == "" {
			piperURL = api.Config.Services.PiperURL // fallback to legacy config
		}
	default:
		piperURL = api.Config.Services.PiperURL
	}

	if piperURL == "" {
		return "", fmt.Errorf("piper URL not configured for deployment type: %s", account.DeploymentType)
	}

	return piperURL, nil
}

// GetTransformationStats reads stats from database (submitted by piper)
func (api *API) GetTransformationStats() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.PathValue("tenantId")
		datasetID := r.PathValue("datasetId")

		// Query stats from database
		query := `
			SELECT tenant_id, dataset_id, enabled, filter_count, total_processed,
			       success_count, error_count, skipped_count, avg_rows_per_sec, last_processed
			FROM piper_transformation_stats
			WHERE tenant_id = $1 AND dataset_id = $2`

		var stats struct {
			TenantID       string     `json:"tenant_id"`
			DatasetID      string     `json:"dataset_id"`
			Enabled        bool       `json:"enabled"`
			FilterCount    int        `json:"filter_count"`
			TotalProcessed int64      `json:"total_processed"`
			SuccessCount   int64      `json:"success_count"`
			ErrorCount     int64      `json:"error_count"`
			SkippedCount   int64      `json:"skipped_count"`
			AvgRowsPerSec  float64    `json:"avg_rows_per_sec"`
			LastProcessed  *time.Time `json:"last_processed"`
		}

		// Type assert to PostgreSQLStorage to access DB method
		psqlStorage, ok := api.Services.Storage.(*storage.PostgreSQLStorage)
		if !ok {
			log.Error("Storage is not PostgreSQLStorage")
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		err := psqlStorage.GetDB().QueryRowContext(r.Context(), query, tenantID, datasetID).Scan(
			&stats.TenantID,
			&stats.DatasetID,
			&stats.Enabled,
			&stats.FilterCount,
			&stats.TotalProcessed,
			&stats.SuccessCount,
			&stats.ErrorCount,
			&stats.SkippedCount,
			&stats.AvgRowsPerSec,
			&stats.LastProcessed,
		)

		if err != nil {
			if err.Error() == "sql: no rows in result set" {
				// No stats available yet
				response := map[string]interface{}{
					"stats": map[string]interface{}{
						"tenant_id":        tenantID,
						"dataset_id":       datasetID,
						"enabled":          false,
						"filter_count":     0,
						"total_processed":  0,
						"success_count":    0,
						"error_count":      0,
						"skipped_count":    0,
						"avg_rows_per_sec": 0,
						"last_processed":   nil,
					},
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response) // #nosec G104
				return
			}
			log.Errorf("Failed to query transformation stats: %v", err)
			http.Error(w, "Failed to retrieve stats", http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"stats": stats,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response) // #nosec G104
	}
}

// GetTransformationPreview reads preview data from database (submitted by piper)
func (api *API) GetTransformationPreview() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.PathValue("tenantId")
		datasetID := r.PathValue("datasetId")
		count := r.URL.Query().Get("count")
		if count == "" {
			count = "10"
		}

		// Query preview samples from database
		query := `
			SELECT line_number, original_data, transformed_data, created_at
			FROM piper_transformation_preview
			WHERE tenant_id = $1 AND dataset_id = $2
			ORDER BY created_at DESC
			LIMIT $3`

		// Type assert to PostgreSQLStorage to access DB method
		psqlStorage, ok := api.Services.Storage.(*storage.PostgreSQLStorage)
		if !ok {
			log.Error("Storage is not PostgreSQLStorage")
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		rows, err := psqlStorage.GetDB().QueryContext(r.Context(), query, tenantID, datasetID, count)
		if err != nil {
			log.Errorf("Failed to query transformation preview: %v", err)
			http.Error(w, "Failed to retrieve preview", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		type PreviewSample struct {
			Input      map[string]interface{} `json:"input"`
			Output     map[string]interface{} `json:"output"`
			Applied    []string               `json:"applied"`
			Skipped    bool                   `json:"skipped"`
			DurationMs int                    `json:"duration_ms"`
		}

		samples := []PreviewSample{}
		for rows.Next() {
			var lineNumber int
			var originalJSON, transformedJSON []byte
			var createdAt time.Time

			err := rows.Scan(&lineNumber, &originalJSON, &transformedJSON, &createdAt)
			if err != nil {
				log.Errorf("Failed to scan preview row: %v", err)
				continue
			}

			var sample PreviewSample
			// Parse JSON
			if err := json.Unmarshal(originalJSON, &sample.Input); err != nil {
				log.Errorf("Failed to unmarshal original data: %v", err)
				continue
			}
			if err := json.Unmarshal(transformedJSON, &sample.Output); err != nil {
				log.Errorf("Failed to unmarshal transformed data: %v", err)
				continue
			}
			// TODO: These fields should come from piper when it submits preview data
			sample.Applied = []string{} // Will be populated when piper submits
			sample.Skipped = false
			sample.DurationMs = 0

			samples = append(samples, sample)
		}

		response := map[string]interface{}{
			"samples": samples,
			"count":   len(samples),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response) // #nosec G104
	}
}

// SubmitDatasetSchema accepts schema and samples from piper after batch processing
func (api *API) SubmitDatasetSchema() usecase.Interactor {
	type SampleData struct {
		LineNumber int                    `json:"line_number"`
		SampleData map[string]interface{} `json:"sample_data"`
		BatchID    string                 `json:"batch_id"`
	}

	type Input struct {
		TenantID   string                 `path:"tenantId" required:"true"`
		DatasetID  string                 `path:"datasetId" required:"true"`
		SchemaType string                 `json:"schema_type" required:"true"` // "input" or "output"
		Schema     interface{}            `json:"schema" required:"true"`      // The inferred schema
		Samples    []SampleData           `json:"samples" required:"true"`     // Sample records (max 10)
	}

	type Output struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input Input, output *Output) error {
		// Validate schema type
		if input.SchemaType != "input" && input.SchemaType != "output" {
			return fmt.Errorf("schema_type must be 'input' or 'output'")
		}

		// Validate sample count (should be 10)
		if len(input.Samples) > 10 {
			return fmt.Errorf("maximum 10 samples allowed")
		}

		// Store schema
		if err := api.Services.Storage.UpsertDatasetSchema(ctx, input.TenantID, input.DatasetID, input.SchemaType, input.Schema); err != nil {
			log.Errorf("Failed to store dataset schema: %v", err)
			return fmt.Errorf("failed to store schema: %w", err)
		}

		// Convert and store samples
		if len(input.Samples) > 0 {
			samples := make([]storage.DatasetSample, 0, len(input.Samples))
			for _, s := range input.Samples {
				samples = append(samples, storage.DatasetSample{
					TenantID:   input.TenantID,
					DatasetID:  input.DatasetID,
					SampleType: input.SchemaType,
					LineNumber: s.LineNumber,
					SampleData: s.SampleData,
					BatchID:    s.BatchID,
					CreatedAt:  time.Now(),
				})
			}

			if err := api.Services.Storage.UpsertDatasetSamples(ctx, input.TenantID, input.DatasetID, input.SchemaType, samples, 10); err != nil {
				log.Errorf("Failed to store dataset samples: %v", err)
				return fmt.Errorf("failed to store samples: %w", err)
			}
		}

		log.Infof("Stored %s schema and %d samples for %s/%s", input.SchemaType, len(input.Samples), input.TenantID, input.DatasetID)

		output.Success = true
		output.Message = fmt.Sprintf("Stored schema and %d samples successfully", len(input.Samples))
		return nil
	})

	return u
}
