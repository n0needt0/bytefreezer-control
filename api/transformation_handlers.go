package api

import (
	"context"
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

// GetTransformationSchema proxies schema request to piper
func (api *API) GetTransformationSchema() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract path parameters from chi router context or URL
		tenantID := r.PathValue("tenantId")
		datasetID := r.PathValue("datasetId")
		count := r.URL.Query().Get("count")
		if count == "" {
			count = "10"
		}

		// TODO: Get piper URL from service discovery or config
		piperURL := "http://192.168.86.96:8090"
		url := fmt.Sprintf("%s/api/v1/transformations/%s/%s/schema?count=%s", piperURL, tenantID, datasetID, count)

		// Create request with context
		req, err := http.NewRequestWithContext(r.Context(), "GET", url, nil)
		if err != nil {
			log.Errorf("Failed to create proxy request: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Make request to piper
		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			log.Errorf("Failed to proxy schema request to piper: %v", err)
			http.Error(w, "Failed to connect to piper service", http.StatusServiceUnavailable)
			return
		}
		defer resp.Body.Close()

		// Read response body
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Errorf("Failed to read piper response: %v", err)
			http.Error(w, "Failed to read response", http.StatusInternalServerError)
			return
		}

		// Forward response status and body
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		w.Write(body) // #nosec G104 - write error not critical for proxy
	}
}

// GetTransformationStats proxies stats request to piper
func (api *API) GetTransformationStats() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.PathValue("tenantId")
		datasetID := r.PathValue("datasetId")

		piperURL := "http://192.168.86.96:8090"
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

		piperURL := "http://192.168.86.96:8090"
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
