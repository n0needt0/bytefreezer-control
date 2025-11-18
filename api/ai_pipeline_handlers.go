package api

import (
	"context"
	"fmt"

	"github.com/n0needt0/bytefreezer-control/middleware"
	"github.com/n0needt0/bytefreezer-control/services"
	"github.com/swaggest/usecase"
	usecaseStatus "github.com/swaggest/usecase/status"
)

// GenerateAIPipeline generates a pipeline configuration using AI
func (api *API) GenerateAIPipeline() usecase.Interactor {
	type generatePipelineInput struct {
		TenantID      string                      `json:"tenant_id" required:"true"`
		DatasetID     string                      `json:"dataset_id" required:"true"`
		UserMessage   string                      `json:"user_message" required:"true"`
		DataSchema    []services.SchemaField      `json:"data_schema,omitempty"`
		SampleRecords []interface{}               `json:"sample_records,omitempty"`
		History       []services.ChatMessage      `json:"history,omitempty"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input generatePipelineInput, output *services.GeneratePipelineResponse) error {
		// Check if AI service is available
		if api.Services.AIPipeline == nil {
			return usecaseStatus.Wrap(fmt.Errorf("AI pipeline service not configured"), usecaseStatus.FailedPrecondition)
		}

		// Get account_id from context for non-system admins
		accountID := ""
		if !middleware.IsSystemAdmin(ctx) {
			accountID = middleware.GetAccountIDFromContext(ctx)
		}

		// Verify user has access to this tenant/dataset
		if accountID != "" {
			// TODO: Add tenant/dataset access validation
			// For now, we'll rely on the existing middleware
		}

		// Prepare request
		req := &services.GeneratePipelineRequest{
			TenantID:      input.TenantID,
			DatasetID:     input.DatasetID,
			UserMessage:   input.UserMessage,
			DataSchema:    input.DataSchema,
			SampleRecords: input.SampleRecords,
			History:       input.History,
		}

		// Generate pipeline configuration
		response, err := api.Services.AIPipeline.GeneratePipeline(ctx, req)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to generate pipeline: %w", err), usecaseStatus.Internal)
		}

		*output = *response
		return nil
	})

	u.SetTitle("Generate AI Pipeline Configuration")
	u.SetDescription("Generate a transformation pipeline configuration using AI based on natural language description")
	u.SetTags("AI", "Pipeline", "Transformation")
	u.SetExpectedErrors(usecaseStatus.Internal, usecaseStatus.FailedPrecondition)

	return u
}

// RefreshAICatalog refreshes the plugin catalog from disk
func (api *API) RefreshAICatalog() usecase.Interactor {
	type refreshCatalogOutput struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, _ struct{}, output *refreshCatalogOutput) error {
		// Only system admins can refresh the catalog
		if !middleware.IsSystemAdmin(ctx) {
			return usecaseStatus.Wrap(fmt.Errorf("unauthorized"), usecaseStatus.PermissionDenied)
		}

		// Check if AI service is available
		if api.Services.AIPipeline == nil {
			return usecaseStatus.Wrap(fmt.Errorf("AI pipeline service not configured"), usecaseStatus.FailedPrecondition)
		}

		// Refresh catalog
		if err := api.Services.AIPipeline.RefreshCatalog(); err != nil {
			output.Success = false
			output.Message = fmt.Sprintf("Failed to refresh catalog: %v", err)
			return usecaseStatus.Wrap(err, usecaseStatus.Internal)
		}

		output.Success = true
		output.Message = "Plugin catalog refreshed successfully"
		return nil
	})

	u.SetTitle("Refresh AI Plugin Catalog")
	u.SetDescription("Reload the transformation plugin catalog from disk (system admin only)")
	u.SetTags("AI", "Admin")
	u.SetExpectedErrors(usecaseStatus.Internal, usecaseStatus.PermissionDenied, usecaseStatus.FailedPrecondition)

	return u
}

// GetAIChatHistory retrieves chat history for a conversation
func (api *API) GetAIChatHistory() usecase.Interactor {
	type getChatHistoryInput struct {
		TenantID       string `query:"tenant_id" required:"true"`
		DatasetID      string `query:"dataset_id" required:"true"`
		ConversationID string `query:"conversation_id"`
	}

	type getChatHistoryOutput struct {
		Messages []services.ChatMessage `json:"messages"`
		Count    int                    `json:"count"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input getChatHistoryInput, output *getChatHistoryOutput) error {
		// TODO: Implement chat history storage and retrieval
		// For MVP, we'll return empty history and rely on client-side storage

		output.Messages = []services.ChatMessage{}
		output.Count = 0
		return nil
	})

	u.SetTitle("Get AI Chat History")
	u.SetDescription("Retrieve conversation history for AI pipeline assistant")
	u.SetTags("AI", "Chat")
	u.SetExpectedErrors(usecaseStatus.Internal)

	return u
}

// ValidateAIPipeline validates a generated pipeline configuration
func (api *API) ValidateAIPipeline() usecase.Interactor {
	type validatePipelineInput struct {
		TenantID  string                          `json:"tenant_id" required:"true"`
		DatasetID string                          `json:"dataset_id" required:"true"`
		Filters   []services.FilterConfig         `json:"filters" required:"true"`
	}

	type validatePipelineOutput struct {
		Valid           bool     `json:"valid"`
		Errors          []string `json:"errors,omitempty"`
		Warnings        []string `json:"warnings,omitempty"`
		FilterCount     int      `json:"filter_count"`
		EstimatedImpact string   `json:"estimated_impact,omitempty"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input validatePipelineInput, output *validatePipelineOutput) error {
		// Basic validation
		errors := []string{}
		warnings := []string{}

		if len(input.Filters) == 0 {
			errors = append(errors, "Pipeline must have at least one filter")
		}

		// Validate each filter
		for i, filter := range input.Filters {
			if filter.Type == "" {
				errors = append(errors, fmt.Sprintf("Filter %d: type is required", i+1))
			}
			if filter.Config == nil {
				errors = append(errors, fmt.Sprintf("Filter %d: config is required", i+1))
			}

			// Check for common issues
			switch filter.Type {
			case "grok":
				if _, hasPattern := filter.Config["pattern"]; !hasPattern {
					if _, hasPatterns := filter.Config["patterns"]; !hasPatterns {
						errors = append(errors, fmt.Sprintf("Filter %d (grok): pattern or patterns required", i+1))
					}
				}
			case "geoip":
				if _, hasDB := filter.Config["database"]; !hasDB {
					warnings = append(warnings, fmt.Sprintf("Filter %d (geoip): no database specified, will use default", i+1))
				}
			case "remove_field":
				if _, hasField := filter.Config["field"]; !hasField {
					if _, hasFields := filter.Config["fields"]; !hasFields {
						errors = append(errors, fmt.Sprintf("Filter %d (remove_field): field or fields required", i+1))
					}
				}
			}
		}

		// Estimate impact
		impact := "Low"
		if len(input.Filters) > 10 {
			impact = "High"
		} else if len(input.Filters) > 5 {
			impact = "Medium"
		}

		// Check for expensive operations
		for _, filter := range input.Filters {
			if filter.Type == "grok" || filter.Type == "dns" || filter.Type == "database" {
				if impact == "Low" {
					impact = "Medium"
				} else if impact == "Medium" {
					impact = "High"
				}
				break
			}
		}

		output.Valid = len(errors) == 0
		output.Errors = errors
		output.Warnings = warnings
		output.FilterCount = len(input.Filters)
		output.EstimatedImpact = impact

		return nil
	})

	u.SetTitle("Validate AI Pipeline Configuration")
	u.SetDescription("Validate a generated pipeline configuration for correctness")
	u.SetTags("AI", "Pipeline", "Validation")
	u.SetExpectedErrors(usecaseStatus.Internal)

	return u
}
