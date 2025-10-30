package api

import (
	"context"
	"fmt"

	"github.com/n0needt0/bytefreezer-control/middleware"
	"github.com/n0needt0/bytefreezer-control/services"
	"github.com/n0needt0/go-goodies/log"
	"github.com/swaggest/usecase"
	usecaseStatus "github.com/swaggest/usecase/status"
)

// ErrorReportRequest is the request for reporting an error (system services)
type ErrorReportRequest struct {
	ErrorType    string                 `json:"error_type" required:"true"`
	Component    string                 `json:"component" required:"true"`
	TenantID     string                 `json:"tenant_id,omitempty"`
	DatasetID    string                 `json:"dataset_id,omitempty"`
	ErrorMessage string                 `json:"error_message" required:"true"`
	ErrorSample  map[string]interface{} `json:"error_sample,omitempty"`
	Severity     string                 `json:"severity,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// AccountErrorReportRequest is the request for reporting an error (account-scoped services like proxy)
type AccountErrorReportRequest struct {
	AccountID    string                 `path:"accountId" required:"true"`
	ErrorType    string                 `json:"error_type" required:"true"`
	Component    string                 `json:"component" required:"true"`
	TenantID     string                 `json:"tenant_id,omitempty"`
	DatasetID    string                 `json:"dataset_id,omitempty"`
	ErrorMessage string                 `json:"error_message" required:"true"`
	ErrorSample  map[string]interface{} `json:"error_sample,omitempty"`
	Severity     string                 `json:"severity,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// ErrorReportResponse is the response for error reporting
type ErrorReportResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// ListErrorsRequest is the request for listing errors
type ListErrorsRequest struct {
	Component string `query:"component"`
	TenantID  string `query:"tenant_id"`
	DatasetID string `query:"dataset_id"`
	Severity  string `query:"severity"`
	Status    string `query:"status"`
	ErrorType string `query:"error_type"`
	Limit     int    `query:"limit"`
	Offset    int    `query:"offset"`
}

// ListAccountErrorsRequest is the request for listing errors for an account
type ListAccountErrorsRequest struct {
	AccountID string `path:"accountId" required:"true"`
	Component string `query:"component"`
	TenantID  string `query:"tenant_id"`
	DatasetID string `query:"dataset_id"`
	Severity  string `query:"severity"`
	Status    string `query:"status"`
	ErrorType string `query:"error_type"`
	Limit     int    `query:"limit"`
	Offset    int    `query:"offset"`
}

// ListDatasetErrorsRequest is the request for listing errors for a dataset
type ListDatasetErrorsRequest struct {
	AccountID string `path:"accountId" required:"true"`
	DatasetID string `path:"datasetId" required:"true"`
	Component string `query:"component"`
	Severity  string `query:"severity"`
	Status    string `query:"status"`
	ErrorType string `query:"error_type"`
	Limit     int    `query:"limit"`
	Offset    int    `query:"offset"`
}

// ListErrorsResponse is the response for listing errors
type ListErrorsResponse struct {
	Errors []services.SystemError `json:"errors"`
	Total  int                    `json:"total"`
	Limit  int                    `json:"limit"`
	Offset int                    `json:"offset"`
}

// ErrorStatsRequest is the request for error statistics
type ErrorStatsRequest struct {
	AccountID string `path:"accountId"`
}

// ErrorStatsResponse is the response for error statistics
type ErrorStatsResponse struct {
	Stats map[string]interface{} `json:"stats"`
}

// ReportError handles error reporting from system services (receiver, packer, piper, soc)
func (api *API) ReportError() usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input ErrorReportRequest, output *ErrorReportResponse) error {
		api.Services.IncrementAPIRequests()

		if api.Services.ErrorReporting == nil {
			return usecaseStatus.Wrap(fmt.Errorf("error reporting service not available"), usecaseStatus.InvalidArgument)
		}

		log.Debugf("Received error report from component=%s, type=%s, tenant=%s, dataset=%s",
			input.Component, input.ErrorType, input.TenantID, input.DatasetID)

		report := services.ErrorReport{
			ErrorType:    input.ErrorType,
			Component:    input.Component,
			TenantID:     input.TenantID,
			DatasetID:    input.DatasetID,
			ErrorMessage: input.ErrorMessage,
			ErrorSample:  input.ErrorSample,
			Severity:     input.Severity,
			Metadata:     input.Metadata,
		}

		if err := api.Services.ErrorReporting.ReportError(ctx, report); err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to report error: %w", err), usecaseStatus.InvalidArgument)
		}

		output.Success = true
		output.Message = "Error reported successfully"

		return nil
	})

	u.SetTitle("Report System Error")
	u.SetDescription("Reports an error from system services (receiver, packer, piper, soc)")
	u.SetTags("errors", "monitoring")

	return u
}

// ReportAccountError handles error reporting from account-scoped services (proxy)
func (api *API) ReportAccountError() usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input AccountErrorReportRequest, output *ErrorReportResponse) error {
		api.Services.IncrementAPIRequests()

		// Extract claims from context (set by JWT middleware)
		claims, ok := ctx.Value(middleware.JWTClaimsContextKey).(*services.JWTClaims)
		if !ok {
			return usecaseStatus.Wrap(fmt.Errorf("authentication required"), usecaseStatus.Unauthenticated)
		}

		// Verify account_id matches the authenticated account
		if claims.AccountID != input.AccountID {
			log.Warnf("Account ID mismatch: token has %s, request for %s", claims.AccountID, input.AccountID)
			return usecaseStatus.Wrap(fmt.Errorf("unauthorized: account mismatch"), usecaseStatus.PermissionDenied)
		}

		if api.Services.ErrorReporting == nil {
			return usecaseStatus.Wrap(fmt.Errorf("error reporting service not available"), usecaseStatus.InvalidArgument)
		}

		log.Debugf("Received account-scoped error report from account=%s, component=%s, type=%s, tenant=%s, dataset=%s",
			input.AccountID, input.Component, input.ErrorType, input.TenantID, input.DatasetID)

		report := services.ErrorReport{
			ErrorType:    input.ErrorType,
			Component:    input.Component,
			TenantID:     input.TenantID,
			DatasetID:    input.DatasetID,
			ErrorMessage: input.ErrorMessage,
			ErrorSample:  input.ErrorSample,
			Severity:     input.Severity,
			Metadata:     input.Metadata,
		}

		if err := api.Services.ErrorReporting.ReportError(ctx, report); err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to report error: %w", err), usecaseStatus.InvalidArgument)
		}

		output.Success = true
		output.Message = fmt.Sprintf("Error reported successfully for account %s", input.AccountID)

		return nil
	})

	u.SetTitle("Report Account Error")
	u.SetDescription("Reports an error from account-scoped services (proxy)")
	u.SetTags("errors", "monitoring", "accounts")

	return u
}

// ListErrors handles listing all errors (system admin only)
func (api *API) ListErrors() usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input ListErrorsRequest, output *ListErrorsResponse) error {
		api.Services.IncrementAPIRequests()

		// Extract claims from context
		claims, ok := ctx.Value(middleware.JWTClaimsContextKey).(*services.JWTClaims)
		if !ok {
			return usecaseStatus.Wrap(fmt.Errorf("authentication required"), usecaseStatus.Unauthenticated)
		}

		// Only system admins can see all errors
		if claims.Role != "system_admin" {
			return usecaseStatus.Wrap(fmt.Errorf("unauthorized: system admin required"), usecaseStatus.PermissionDenied)
		}

		if api.Services.ErrorReporting == nil {
			return usecaseStatus.Wrap(fmt.Errorf("error reporting service not available"), usecaseStatus.InvalidArgument)
		}

		filter := services.ErrorListFilter{
			Component: input.Component,
			TenantID:  input.TenantID,
			DatasetID: input.DatasetID,
			Severity:  input.Severity,
			Status:    input.Status,
			ErrorType: input.ErrorType,
			Limit:     input.Limit,
			Offset:    input.Offset,
		}

		errors, total, err := api.Services.ErrorReporting.ListErrors(ctx, filter)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to list errors: %w", err), usecaseStatus.InvalidArgument)
		}

		output.Errors = errors
		output.Total = total
		output.Limit = filter.Limit
		output.Offset = filter.Offset

		return nil
	})

	u.SetTitle("List All Errors")
	u.SetDescription("Lists all system errors (system admin only)")
	u.SetTags("errors", "monitoring")

	return u
}

// ListAccountErrors handles listing errors for a specific account
func (api *API) ListAccountErrors() usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input ListAccountErrorsRequest, output *ListErrorsResponse) error {
		api.Services.IncrementAPIRequests()

		// Extract claims from context
		claims, ok := ctx.Value(middleware.JWTClaimsContextKey).(*services.JWTClaims)
		if !ok {
			return usecaseStatus.Wrap(fmt.Errorf("authentication required"), usecaseStatus.Unauthenticated)
		}

		// Verify account_id matches the authenticated account (unless system admin)
		if claims.Role != "system_admin" && claims.AccountID != input.AccountID {
			log.Warnf("Account ID mismatch: token has %s, request for %s", claims.AccountID, input.AccountID)
			return usecaseStatus.Wrap(fmt.Errorf("unauthorized: account mismatch"), usecaseStatus.PermissionDenied)
		}

		if api.Services.ErrorReporting == nil {
			return usecaseStatus.Wrap(fmt.Errorf("error reporting service not available"), usecaseStatus.InvalidArgument)
		}

		filter := services.ErrorListFilter{
			Component: input.Component,
			TenantID:  input.TenantID,
			DatasetID: input.DatasetID,
			Severity:  input.Severity,
			Status:    input.Status,
			ErrorType: input.ErrorType,
			AccountID: input.AccountID, // Filter by account
			Limit:     input.Limit,
			Offset:    input.Offset,
		}

		errors, total, err := api.Services.ErrorReporting.ListErrors(ctx, filter)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to list errors: %w", err), usecaseStatus.InvalidArgument)
		}

		output.Errors = errors
		output.Total = total
		output.Limit = filter.Limit
		output.Offset = filter.Offset

		return nil
	})

	u.SetTitle("List Account Errors")
	u.SetDescription("Lists errors for a specific account")
	u.SetTags("errors", "monitoring", "accounts")

	return u
}

// ListDatasetErrors handles listing errors for a specific dataset
func (api *API) ListDatasetErrors() usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input ListDatasetErrorsRequest, output *ListErrorsResponse) error {
		api.Services.IncrementAPIRequests()

		// Extract claims from context
		claims, ok := ctx.Value(middleware.JWTClaimsContextKey).(*services.JWTClaims)
		if !ok {
			return usecaseStatus.Wrap(fmt.Errorf("authentication required"), usecaseStatus.Unauthenticated)
		}

		// Verify account_id matches the authenticated account (unless system admin)
		if claims.Role != "system_admin" && claims.AccountID != input.AccountID {
			log.Warnf("Account ID mismatch: token has %s, request for %s", claims.AccountID, input.AccountID)
			return usecaseStatus.Wrap(fmt.Errorf("unauthorized: account mismatch"), usecaseStatus.PermissionDenied)
		}

		if api.Services.ErrorReporting == nil {
			return usecaseStatus.Wrap(fmt.Errorf("error reporting service not available"), usecaseStatus.InvalidArgument)
		}

		filter := services.ErrorListFilter{
			Component: input.Component,
			DatasetID: input.DatasetID,
			Severity:  input.Severity,
			Status:    input.Status,
			ErrorType: input.ErrorType,
			AccountID: input.AccountID, // Filter by account
			Limit:     input.Limit,
			Offset:    input.Offset,
		}

		errors, total, err := api.Services.ErrorReporting.ListErrors(ctx, filter)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to list errors: %w", err), usecaseStatus.InvalidArgument)
		}

		output.Errors = errors
		output.Total = total
		output.Limit = filter.Limit
		output.Offset = filter.Offset

		return nil
	})

	u.SetTitle("List Dataset Errors")
	u.SetDescription("Lists errors for a specific dataset")
	u.SetTags("errors", "monitoring", "datasets")

	return u
}

// GetErrorStats handles getting error statistics (system-wide, admin only)
func (api *API) GetErrorStats() usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input struct{}, output *ErrorStatsResponse) error {
		api.Services.IncrementAPIRequests()

		// Extract claims from context
		claims, ok := ctx.Value(middleware.JWTClaimsContextKey).(*services.JWTClaims)
		if !ok {
			return usecaseStatus.Wrap(fmt.Errorf("authentication required"), usecaseStatus.Unauthenticated)
		}

		// Only system admins can see all error stats
		if claims.Role != "system_admin" {
			return usecaseStatus.Wrap(fmt.Errorf("unauthorized: system admin required"), usecaseStatus.PermissionDenied)
		}

		if api.Services.ErrorReporting == nil {
			return usecaseStatus.Wrap(fmt.Errorf("error reporting service not available"), usecaseStatus.InvalidArgument)
		}

		stats, err := api.Services.ErrorReporting.GetErrorStats(ctx, "")
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to get error stats: %w", err), usecaseStatus.InvalidArgument)
		}

		output.Stats = stats

		return nil
	})

	u.SetTitle("Get Error Statistics")
	u.SetDescription("Gets error statistics for all errors (system admin only)")
	u.SetTags("errors", "monitoring", "statistics")

	return u
}

// GetAccountErrorStats handles getting error statistics for a specific account
func (api *API) GetAccountErrorStats() usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input ErrorStatsRequest, output *ErrorStatsResponse) error {
		api.Services.IncrementAPIRequests()

		// Extract claims from context
		claims, ok := ctx.Value(middleware.JWTClaimsContextKey).(*services.JWTClaims)
		if !ok {
			return usecaseStatus.Wrap(fmt.Errorf("authentication required"), usecaseStatus.Unauthenticated)
		}

		// Verify account_id matches the authenticated account (unless system admin)
		if claims.Role != "system_admin" && claims.AccountID != input.AccountID {
			return usecaseStatus.Wrap(fmt.Errorf("unauthorized: account mismatch"), usecaseStatus.PermissionDenied)
		}

		if api.Services.ErrorReporting == nil {
			return usecaseStatus.Wrap(fmt.Errorf("error reporting service not available"), usecaseStatus.InvalidArgument)
		}

		stats, err := api.Services.ErrorReporting.GetErrorStats(ctx, input.AccountID)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to get error stats: %w", err), usecaseStatus.InvalidArgument)
		}

		output.Stats = stats

		return nil
	})

	u.SetTitle("Get Account Error Statistics")
	u.SetDescription("Gets error statistics for a specific account")
	u.SetTags("errors", "monitoring", "statistics", "accounts")

	return u
}
