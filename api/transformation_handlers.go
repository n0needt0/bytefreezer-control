package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

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
		// Create transformation job
		job := &storage.TransformationJob{
			TenantID:  input.TenantID,
			DatasetID: input.DatasetID,
			JobType:   storage.TransformationJobTypeTest,
			Status:    storage.JobStatusPending,
			Request: map[string]interface{}{
				"tenant_id":  input.TenantID,
				"dataset_id": input.DatasetID,
				"filters":    input.Filters,
				"samples":    input.Samples,
			},
			TTL: time.Now().Add(1 * time.Hour), // 1 hour TTL
		}

		if err := api.Services.Storage.CreateTransformationJob(ctx, job); err != nil {
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
		// Create transformation job
		job := &storage.TransformationJob{
			TenantID:  input.TenantID,
			DatasetID: input.DatasetID,
			JobType:   storage.TransformationJobTypeValidate,
			Status:    storage.JobStatusPending,
			Request: map[string]interface{}{
				"tenant_id":  input.TenantID,
				"dataset_id": input.DatasetID,
				"filters":    input.Filters,
				"count":      input.Count,
			},
			TTL: time.Now().Add(1 * time.Hour), // 1 hour TTL
		}

		if err := api.Services.Storage.CreateTransformationJob(ctx, job); err != nil {
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
		// Create transformation job
		job := &storage.TransformationJob{
			TenantID:  input.TenantID,
			DatasetID: input.DatasetID,
			JobType:   storage.TransformationJobTypeActivate,
			Status:    storage.JobStatusPending,
			Request: map[string]interface{}{
				"tenant_id":  input.TenantID,
				"dataset_id": input.DatasetID,
				"filters":    input.Filters,
				"enabled":    input.Enabled,
			},
			TTL: time.Now().Add(1 * time.Hour), // 1 hour TTL
		}

		if err := api.Services.Storage.CreateTransformationJob(ctx, job); err != nil {
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
		job, err := api.Services.Storage.GetTransformationJob(ctx, input.JobID)
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

		// Extract fields array from schema data
		var schemaFields interface{}
		if fields, ok := schemaData["fields"]; ok {
			schemaFields = fields
		} else {
			// Fallback to entire schema if fields key doesn't exist
			schemaFields = schemaData
		}

		// Convert samples to response format
		sampleData := make([]interface{}, 0, len(samples))
		for _, s := range samples {
			sampleData = append(sampleData, s.SampleData)
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

// GetTransformationStats proxies stats request to piper
func (api *API) GetTransformationStats() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.PathValue("tenantId")
		datasetID := r.PathValue("datasetId")

		// Get piper URL from config
		piperURL := api.Config.Services.PiperURL
		if piperURL == "" {
			log.Error("Piper URL not configured in services config")
			http.Error(w, "Piper service not configured", http.StatusServiceUnavailable)
			return
		}
		url := fmt.Sprintf("%s/api/v1/transformations/%s/%s/stats", piperURL, tenantID, datasetID)

		req, err := http.NewRequestWithContext(r.Context(), "GET", url, nil)
		if err != nil {
			log.Errorf("Failed to create proxy request: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			log.Errorf("Failed to proxy stats request to piper: %v", err)
			http.Error(w, "Failed to connect to piper service", http.StatusServiceUnavailable)
			return
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Errorf("Failed to read piper response: %v", err)
			http.Error(w, "Failed to read response", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		w.Write(body) // #nosec G104 - write error not critical for proxy
	}
}

// GetTransformationPreview proxies preview request to piper
func (api *API) GetTransformationPreview() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.PathValue("tenantId")
		datasetID := r.PathValue("datasetId")
		count := r.URL.Query().Get("count")
		if count == "" {
			count = "10"
		}

		// Get piper URL from config
		piperURL := api.Config.Services.PiperURL
		if piperURL == "" {
			log.Error("Piper URL not configured in services config")
			http.Error(w, "Piper service not configured", http.StatusServiceUnavailable)
			return
		}
		url := fmt.Sprintf("%s/api/v1/transformations/%s/%s/preview?count=%s", piperURL, tenantID, datasetID, count)

		req, err := http.NewRequestWithContext(r.Context(), "GET", url, nil)
		if err != nil {
			log.Errorf("Failed to create proxy request: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			log.Errorf("Failed to proxy preview request to piper: %v", err)
			http.Error(w, "Failed to connect to piper service", http.StatusServiceUnavailable)
			return
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Errorf("Failed to read piper response: %v", err)
			http.Error(w, "Failed to read response", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		w.Write(body) // #nosec G104 - write error not critical for proxy
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
