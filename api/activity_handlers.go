package api

import (
	"context"
	"fmt"

	"github.com/n0needt0/bytefreezer-control/middleware"
	"github.com/n0needt0/bytefreezer-control/services"
	"github.com/swaggest/usecase"
	usecaseStatus "github.com/swaggest/usecase/status"
)

// UpsertServiceOperation reports or updates an operation
func (api *API) UpsertServiceOperation() usecase.Interactor {
	type upsertServiceOperationInput struct {
		services.OperationUpdate
	}

	type upsertServiceOperationOutput struct {
		Success bool   `json:"success"`
		Message string `json:"message,omitempty"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input upsertServiceOperationInput, output *upsertServiceOperationOutput) error {
		if err := api.Services.OperationTracking.UpsertOperation(ctx, &input.OperationUpdate); err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to upsert operation: %w", err), usecaseStatus.Internal)
		}

		output.Success = true
		output.Message = "Operation reported successfully"
		return nil
	})

	u.SetTitle("Upsert Service Operation")
	u.SetDescription("Report or update a service operation for activity tracking")
	u.SetTags("Activity")
	u.SetExpectedErrors(usecaseStatus.Internal)

	return u
}

// GetActiveOperations retrieves currently running operations
func (api *API) GetActiveOperations() usecase.Interactor {
	type getActiveOperationsOutput struct {
		Operations []*services.ServiceOperation `json:"operations"`
		Count      int                          `json:"count"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, _ struct{}, output *getActiveOperationsOutput) error {
		// Get account_id from context if available (for non-system admins)
		accountID := ""
		if !middleware.IsSystemAdmin(ctx) {
			accountID = middleware.GetAccountIDFromContext(ctx)
		}

		operations, err := api.Services.OperationTracking.GetActiveOperations(ctx, accountID)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to get active operations: %w", err), usecaseStatus.Internal)
		}

		output.Operations = operations
		output.Count = len(operations)
		return nil
	})

	u.SetTitle("Get Active Operations")
	u.SetDescription("Retrieve all currently in-progress operations across services")
	u.SetTags("Activity")
	u.SetExpectedErrors(usecaseStatus.Internal)

	return u
}

// GetRecentOperations retrieves recently completed operations
func (api *API) GetRecentOperations() usecase.Interactor {
	type getRecentOperationsInput struct {
		Minutes int `query:"minutes" default:"60" description:"Number of minutes of history to retrieve"`
	}

	type getRecentOperationsOutput struct {
		Operations []*services.ServiceOperation `json:"operations"`
		Count      int                          `json:"count"`
		TimeRange  string                       `json:"time_range"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input getRecentOperationsInput, output *getRecentOperationsOutput) error {
		if input.Minutes <= 0 {
			input.Minutes = 60
		}
		if input.Minutes > 1440 { // Max 24 hours
			input.Minutes = 1440
		}

		// Get account_id from context if available (for non-system admins)
		accountID := ""
		if !middleware.IsSystemAdmin(ctx) {
			accountID = middleware.GetAccountIDFromContext(ctx)
		}

		operations, err := api.Services.OperationTracking.GetRecentOperations(ctx, accountID, input.Minutes)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to get recent operations: %w", err), usecaseStatus.Internal)
		}

		output.Operations = operations
		output.Count = len(operations)
		output.TimeRange = fmt.Sprintf("Last %d minutes", input.Minutes)
		return nil
	})

	u.SetTitle("Get Recent Operations")
	u.SetDescription("Retrieve recently completed or failed operations")
	u.SetTags("Activity")
	u.SetExpectedErrors(usecaseStatus.Internal)

	return u
}

// GetActivitySummary provides a summary of current activity
func (api *API) GetActivitySummary() usecase.Interactor {
	type activitySummaryOutput struct {
		ActiveOperations int                         `json:"active_operations"`
		RecentCompleted  int                         `json:"recent_completed"`
		RecentFailed     int                         `json:"recent_failed"`
		ByService        map[string]int              `json:"by_service"`
		ByStatus         map[string]int              `json:"by_status"`
		Timestamp        string                      `json:"timestamp"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, _ struct{}, output *activitySummaryOutput) error {
		// Get account_id from context if available (for non-system admins)
		accountID := ""
		if !middleware.IsSystemAdmin(ctx) {
			accountID = middleware.GetAccountIDFromContext(ctx)
		}

		// Get active operations
		active, err := api.Services.OperationTracking.GetActiveOperations(ctx, accountID)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to get active operations: %w", err), usecaseStatus.Internal)
		}

		// Get recent operations (last 15 minutes)
		recent, err := api.Services.OperationTracking.GetRecentOperations(ctx, accountID, 15)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to get recent operations: %w", err), usecaseStatus.Internal)
		}

		// Build summary
		output.ActiveOperations = len(active)
		output.ByService = make(map[string]int)
		output.ByStatus = make(map[string]int)

		for _, op := range active {
			output.ByService[op.ServiceType]++
			output.ByStatus["in_progress"]++
		}

		for _, op := range recent {
			output.ByService[op.ServiceType]++
			output.ByStatus[op.Status]++
			if op.Status == "completed" {
				output.RecentCompleted++
			} else if op.Status == "failed" {
				output.RecentFailed++
			}
		}

		output.Timestamp = fmt.Sprintf("%v", ctx.Value("request_time"))
		if output.Timestamp == "" || output.Timestamp == "<nil>" {
			output.Timestamp = "now"
		}

		return nil
	})

	u.SetTitle("Get Activity Summary")
	u.SetDescription("Get a summary of current and recent activity across all services")
	u.SetTags("Activity")
	u.SetExpectedErrors(usecaseStatus.Internal)

	return u
}
