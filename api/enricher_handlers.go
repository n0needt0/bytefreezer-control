package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/n0needt0/bytefreezer-control/storage"
	"github.com/n0needt0/go-goodies/log"
	"github.com/swaggest/usecase"
	usecaseStatus "github.com/swaggest/usecase/status"
)

// CreateEnricher creates a new enricher
func (api *API) CreateEnricher() usecase.Interactor {
	type Input struct {
		TenantID     string   `path:"tenantId" required:"true"`
		Name         string   `json:"name" required:"true"`
		Description  string   `json:"description"`
		Columns      []string `json:"columns" required:"true"`
		IndexColumns []string `json:"index_columns" required:"true"`
	}

	type Output struct {
		Enricher storage.Enricher `json:"enricher"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input Input, output *Output) error {
		tenantID, err := uuid.Parse(input.TenantID)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("invalid tenant ID"), usecaseStatus.InvalidArgument)
		}

		// Build request
		req := &storage.CreateEnricherRequest{
			Name:         input.Name,
			Description:  input.Description,
			Columns:      input.Columns,
			IndexColumns: input.IndexColumns,
		}

		// Validate request
		if err := req.Validate(); err != nil {
			return usecaseStatus.Wrap(err, usecaseStatus.InvalidArgument)
		}

		// Get user ID from context
		userID := getUserIDFromContext(ctx)

		// Create enricher
		enricher, err := api.Services.Enricher.CreateEnricher(ctx, tenantID, userID, req)
		if err != nil {
			if err == storage.ErrAlreadyExists {
				return usecaseStatus.Wrap(fmt.Errorf("enricher with this name already exists"), usecaseStatus.AlreadyExists)
			}
			log.Errorf("Failed to create enricher: %v", err)
			return fmt.Errorf("failed to create enricher: %w", err)
		}

		output.Enricher = *enricher
		return nil
	})

	u.SetExpectedErrors(usecaseStatus.InvalidArgument, usecaseStatus.AlreadyExists)
	return u
}

// GetEnricher retrieves an enricher by ID
func (api *API) GetEnricher() usecase.Interactor {
	type Input struct {
		TenantID   string `path:"tenantId" required:"true"`
		EnricherID string `path:"enricherId" required:"true"`
	}

	type Output struct {
		Enricher storage.Enricher `json:"enricher"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input Input, output *Output) error {
		tenantID, err := uuid.Parse(input.TenantID)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("invalid tenant ID"), usecaseStatus.InvalidArgument)
		}

		enricherID, err := uuid.Parse(input.EnricherID)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("invalid enricher ID"), usecaseStatus.InvalidArgument)
		}

		enricher, err := api.Services.Enricher.GetEnricher(ctx, tenantID, enricherID)
		if err != nil {
			if err == storage.ErrNotFound {
				return usecaseStatus.NotFound
			}
			log.Errorf("Failed to get enricher: %v", err)
			return fmt.Errorf("failed to get enricher: %w", err)
		}

		output.Enricher = *enricher
		return nil
	})

	u.SetExpectedErrors(usecaseStatus.InvalidArgument, usecaseStatus.NotFound)
	return u
}

// ListEnrichers lists enrichers for a tenant
func (api *API) ListEnrichers() usecase.Interactor {
	type Input struct {
		TenantID string `path:"tenantId" required:"true"`
		Page     int    `query:"page"`
		PageSize int    `query:"page_size"`
	}

	type Output struct {
		Enrichers storage.EnricherListResponse `json:"response"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input Input, output *Output) error {
		tenantID, err := uuid.Parse(input.TenantID)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("invalid tenant ID"), usecaseStatus.InvalidArgument)
		}

		// Set defaults for pagination
		page := input.Page
		if page < 1 {
			page = 1
		}

		pageSize := input.PageSize
		if pageSize < 1 || pageSize > 100 {
			pageSize = 20
		}

		enrichers, err := api.Services.Enricher.ListEnrichers(ctx, tenantID, page, pageSize)
		if err != nil {
			log.Errorf("Failed to list enrichers: %v", err)
			return fmt.Errorf("failed to list enrichers: %w", err)
		}

		output.Enrichers = *enrichers
		return nil
	})

	u.SetExpectedErrors(usecaseStatus.InvalidArgument)
	return u
}

// DeleteEnricher deletes an enricher
func (api *API) DeleteEnricher() usecase.Interactor {
	type Input struct {
		TenantID   string `path:"tenantId" required:"true"`
		EnricherID string `path:"enricherId" required:"true"`
	}

	type Output struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input Input, output *Output) error {
		tenantID, err := uuid.Parse(input.TenantID)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("invalid tenant ID"), usecaseStatus.InvalidArgument)
		}

		enricherID, err := uuid.Parse(input.EnricherID)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("invalid enricher ID"), usecaseStatus.InvalidArgument)
		}

		if err := api.Services.Enricher.DeleteEnricher(ctx, tenantID, enricherID); err != nil {
			if err == storage.ErrNotFound {
				return usecaseStatus.NotFound
			}
			log.Errorf("Failed to delete enricher: %v", err)
			return fmt.Errorf("failed to delete enricher: %w", err)
		}

		output.Success = true
		output.Message = "Enricher deleted successfully"
		return nil
	})

	u.SetExpectedErrors(usecaseStatus.InvalidArgument, usecaseStatus.NotFound)
	return u
}

// UploadEnricherData uploads CSV data to an enricher
// This handler uses http.HandlerFunc because it handles multipart form data
func (api *API) UploadEnricherData() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.PathValue("tenantId")
		enricherID := r.PathValue("enricherId")

		tenantUUID, err := uuid.Parse(tenantID)
		if err != nil {
			http.Error(w, "Invalid tenant ID", http.StatusBadRequest)
			return
		}

		enricherUUID, err := uuid.Parse(enricherID)
		if err != nil {
			http.Error(w, "Invalid enricher ID", http.StatusBadRequest)
			return
		}

		// Parse multipart form (max 100MB)
		if err := r.ParseMultipartForm(100 << 20); err != nil {
			http.Error(w, "Failed to parse form", http.StatusBadRequest)
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "No file uploaded", http.StatusBadRequest)
			return
		}
		defer file.Close()

		// Get user ID from context
		userID := getUserIDFromContext(r.Context())

		// Upload data
		if err := api.Services.Enricher.UploadEnricherData(r.Context(), tenantUUID, enricherUUID, userID, file, header.Filename); err != nil {
			log.Errorf("Failed to upload enricher data: %v", err)
			http.Error(w, fmt.Sprintf("Failed to upload data: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true, "message": "Data uploaded successfully"}`)) // #nosec G104
	}
}

// Helper functions

func getUserIDFromContext(ctx context.Context) *uuid.UUID {
	if userID, ok := ctx.Value("user_id").(string); ok {
		if id, err := uuid.Parse(userID); err == nil {
			return &id
		}
	}
	return nil
}
