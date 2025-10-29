package api

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/n0needt0/bytefreezer-control/middleware"
	"github.com/n0needt0/bytefreezer-control/services"
	"github.com/n0needt0/go-goodies/log"
	"github.com/swaggest/usecase"
	usecaseStatus "github.com/swaggest/usecase/status"
)

// ServiceRegistrationRequest represents a service registration request
type ServiceRegistrationRequest struct {
	ServiceType   string                 `json:"service_type" required:"true"`
	InstanceID    string                 `json:"instance_id,omitempty"`
	InstanceAPI   string                 `json:"instance_api" required:"true"`
	Status        string                 `json:"status,omitempty"`
	Configuration map[string]interface{} `json:"configuration,omitempty"`
}

// ServiceRegistrationResponse represents a service registration response
type ServiceRegistrationResponse struct {
	Success     bool   `json:"success"`
	Message     string `json:"message"`
	InstanceID  string `json:"instance_id"`
	ServiceType string `json:"service_type"`
}

// HealthStatusResponse represents the health status response
type HealthStatusResponse struct {
	Services []services.HealthRecord `json:"services"`
	Summary  map[string]interface{}  `json:"summary"`
}

// HealthSummaryResponse represents just the summary data
type HealthSummaryResponse struct {
	Summary map[string]interface{} `json:"summary"`
}

// RegisterService handles service registration requests
func (api *API) RegisterService() usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input ServiceRegistrationRequest, output *ServiceRegistrationResponse) error {
		api.Services.IncrementAPIRequests()

		log.Infof("Received registration request for service type: %s, instance_id: %s, instance_api: %s",
			input.ServiceType, input.InstanceID, input.InstanceAPI)

		if api.Services.HealthService == nil {
			log.Error("Health service is not available for registration")
			return usecaseStatus.Wrap(fmt.Errorf("health service not available"), usecaseStatus.InvalidArgument)
		}

		// Get hostname if instance_id not provided
		instanceID := input.InstanceID
		if instanceID == "" {
			hostname, err := os.Hostname()
			if err != nil {
				log.Errorf("Failed to get hostname for registration: %v", err)
				return usecaseStatus.Wrap(fmt.Errorf("failed to get hostname: %w", err), usecaseStatus.InvalidArgument)
			}
			instanceID = hostname
		}

		// Use provided status or default to "Starting"
		status := input.Status
		if status == "" {
			status = "Starting"
		}

		registration := services.ServiceRegistration{
			ServiceType:   input.ServiceType,
			InstanceID:    instanceID,
			InstanceAPI:   input.InstanceAPI,
			Status:        status,
			Configuration: input.Configuration,
			Timestamp:     time.Now(),
		}

		err := api.Services.HealthService.RegisterService(registration)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to register service: %w", err), usecaseStatus.InvalidArgument)
		}

		output.Success = true
		output.Message = fmt.Sprintf("Service %s registered successfully", input.ServiceType)
		output.InstanceID = instanceID
		output.ServiceType = input.ServiceType

		log.Infof("Service registered: %s instance %s at %s", input.ServiceType, instanceID, input.InstanceAPI)

		return nil
	})

	u.SetTitle("Register Service")
	u.SetDescription("Registers a new service instance for health monitoring")
	u.SetTags("health", "registration")

	return u
}

// GetHealthStatus returns the current health status of all services
func (api *API) GetHealthStatus() usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input struct{}, output *HealthStatusResponse) error {
		api.Services.IncrementAPIRequests()

		if api.Services.HealthService == nil {
			return usecaseStatus.Wrap(fmt.Errorf("health service not available"), usecaseStatus.InvalidArgument)
		}

		// Get all health records
		records, err := api.Services.HealthService.GetAllHealthRecords()
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to get health records: %w", err), usecaseStatus.InvalidArgument)
		}

		// Get summary
		summary, err := api.Services.HealthService.GetHealthSummary()
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to get health summary: %w", err), usecaseStatus.InvalidArgument)
		}

		output.Services = records
		output.Summary = summary

		return nil
	})

	u.SetTitle("Get Health Status")
	u.SetDescription("Returns the current health status of all registered services")
	u.SetTags("health", "monitoring")

	return u
}

// GetHealthSummary returns just the health summary for dashboard
func (api *API) GetHealthSummary() usecase.Interactor {
	u := usecase.NewInteractor(func(ctx context.Context, input struct{}, output *HealthSummaryResponse) error {
		api.Services.IncrementAPIRequests()

		if api.Services.HealthService == nil {
			return usecaseStatus.Wrap(fmt.Errorf("health service not available"), usecaseStatus.InvalidArgument)
		}

		summary, err := api.Services.HealthService.GetHealthSummary()
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to get health summary: %w", err), usecaseStatus.InvalidArgument)
		}

		output.Summary = summary

		return nil
	})

	u.SetTitle("Get Health Summary")
	u.SetDescription("Returns health summary statistics for the dashboard")
	u.SetTags("health", "monitoring", "dashboard")

	return u
}

// GetServiceHealth returns health status for a specific service type
func (api *API) GetServiceHealth() usecase.Interactor {
	type ServiceHealthRequest struct {
		ServiceType string `path:"serviceType" required:"true"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input ServiceHealthRequest, output *HealthStatusResponse) error {
		api.Services.IncrementAPIRequests()

		if api.Services.HealthService == nil {
			return usecaseStatus.Wrap(fmt.Errorf("health service not available"), usecaseStatus.InvalidArgument)
		}

		records, err := api.Services.HealthService.GetHealthRecordsByService(input.ServiceType)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to get health records for service %s: %w", input.ServiceType, err), usecaseStatus.InvalidArgument)
		}

		// Get overall summary as well
		summary, err := api.Services.HealthService.GetHealthSummary()
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to get health summary: %w", err), usecaseStatus.InvalidArgument)
		}

		output.Services = records
		output.Summary = summary

		return nil
	})

	u.SetTitle("Get Service Health")
	u.SetDescription("Returns health status for a specific service type")
	u.SetTags("health", "monitoring")

	return u
}

// RegisterAccountService handles service registration for account-scoped services (proxy)
func (api *API) RegisterAccountService() usecase.Interactor {
	type AccountServiceRegistrationRequest struct {
		AccountID     string                 `path:"accountId" required:"true"`
		ServiceType   string                 `json:"service_type" required:"true"`
		InstanceID    string                 `json:"instance_id,omitempty"`
		InstanceAPI   string                 `json:"instance_api" required:"true"`
		Status        string                 `json:"status,omitempty"`
		Configuration map[string]interface{} `json:"configuration,omitempty"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input AccountServiceRegistrationRequest, output *ServiceRegistrationResponse) error {
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

		log.Infof("Received account-scoped registration for account %s, service: %s, instance: %s",
			input.AccountID, input.ServiceType, input.InstanceID)

		if api.Services.HealthService == nil {
			return usecaseStatus.Wrap(fmt.Errorf("health service not available"), usecaseStatus.InvalidArgument)
		}

		// Get hostname if instance_id not provided
		instanceID := input.InstanceID
		if instanceID == "" {
			hostname, err := os.Hostname()
			if err != nil {
				log.Errorf("Failed to get hostname for registration: %v", err)
				return usecaseStatus.Wrap(fmt.Errorf("failed to get hostname: %w", err), usecaseStatus.InvalidArgument)
			}
			instanceID = hostname
		}

		// Use provided status or default to "Starting"
		status := input.Status
		if status == "" {
			status = "Starting"
		}

		registration := services.ServiceRegistration{
			ServiceType:   input.ServiceType,
			InstanceID:    instanceID,
			InstanceAPI:   input.InstanceAPI,
			AccountID:     input.AccountID, // Include account_id
			Status:        status,
			Configuration: input.Configuration,
			Timestamp:     time.Now(),
		}

		err := api.Services.HealthService.RegisterService(registration)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to register service: %w", err), usecaseStatus.InvalidArgument)
		}

		output.Success = true
		output.Message = fmt.Sprintf("Service %s registered successfully for account %s", input.ServiceType, input.AccountID)
		output.InstanceID = instanceID
		output.ServiceType = input.ServiceType

		log.Infof("Account-scoped service registered: %s instance %s at %s (account: %s)",
			input.ServiceType, instanceID, input.InstanceAPI, input.AccountID)

		return nil
	})

	u.SetTitle("Register Account Service")
	u.SetDescription("Registers a service instance scoped to a specific account (for proxy)")
	u.SetTags("health", "registration", "accounts")

	return u
}

// ReceiveAccountServiceReport receives health reports for account-scoped services
func (api *API) ReceiveAccountServiceReport() usecase.Interactor {
	type AccountServiceReport struct {
		AccountID     string                 `path:"accountId" required:"true"`
		ServiceName   string                 `json:"service_name,omitempty"`
		ServiceType   string                 `json:"service_type,omitempty"`
		ServiceID     string                 `json:"service_id,omitempty"`
		InstanceID    string                 `json:"instance_id,omitempty"`
		InstanceAPI   string                 `json:"instance_api,omitempty"`
		Healthy       bool                   `json:"healthy"`
		Status        string                 `json:"status,omitempty"`
		Configuration map[string]interface{} `json:"configuration,omitempty"`
		Metrics       map[string]interface{} `json:"metrics,omitempty"`
	}

	type ServiceReportResponse struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input AccountServiceReport, output *ServiceReportResponse) error {
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

		// Handle both naming conventions
		serviceName := input.ServiceName
		if serviceName == "" && input.ServiceType != "" {
			serviceName = input.ServiceType
		}

		serviceID := input.ServiceID
		if serviceID == "" && input.InstanceID != "" {
			serviceID = input.InstanceID
		}

		// Handle both healthy (bool) and status (string) fields
		healthy := input.Healthy
		if !healthy && input.Status != "" {
			healthy = (input.Status == "Healthy")
		}

		log.Debugf("Received account-scoped health report from service %s (%s) for account %s: healthy=%v",
			serviceName, serviceID, input.AccountID, healthy)

		if api.Services.HealthService == nil {
			return usecaseStatus.Wrap(fmt.Errorf("health service not available"), usecaseStatus.InvalidArgument)
		}

		// Get current record to check if service is still starting
		records, err := api.Services.HealthService.GetHealthRecordsByAccountAndService(input.AccountID, serviceName)
		currentStatus := ""
		if err == nil {
			for _, record := range records {
				if record.InstanceID == serviceID {
					currentStatus = record.Status
					break
				}
			}
		}

		// Determine status from health report
		status := "Unhealthy"
		if healthy {
			status = "Healthy"
		}

		// If service is currently "Starting" and reports unhealthy, keep it "Starting"
		if currentStatus == "Starting" && !healthy {
			log.Debugf("Service %s:%s (account %s) is still starting - keeping status as Starting",
				serviceName, serviceID, input.AccountID)
			status = "Starting"
		}

		// Update health status with account_id
		err = api.Services.HealthService.UpdateServiceHealth(
			serviceName,
			serviceID,
			input.InstanceAPI,
			status,
			input.AccountID, // Include account_id
			input.Configuration,
			input.Metrics,
			nil,
		)

		if err != nil {
			log.Warnf("Failed to update health status for %s:%s (account %s): %v",
				serviceName, serviceID, input.AccountID, err)
			// Don't fail the request, just log the error
		}

		output.Success = true
		output.Message = fmt.Sprintf("Health report received for service %s (account %s)", serviceName, input.AccountID)

		return nil
	})

	u.SetTitle("Receive Account Service Report")
	u.SetDescription("Receives health reports from account-scoped services (for proxy)")
	u.SetTags("health", "monitoring", "accounts")

	return u
}

// ListAccountServices returns all services for a specific account
func (api *API) ListAccountServices() usecase.Interactor {
	type ListAccountServicesRequest struct {
		AccountID string `path:"accountId" required:"true"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input ListAccountServicesRequest, output *HealthStatusResponse) error {
		api.Services.IncrementAPIRequests()

		// Extract claims from context (set by JWT middleware)
		claims, ok := ctx.Value(middleware.JWTClaimsContextKey).(*services.JWTClaims)
		if !ok {
			return usecaseStatus.Wrap(fmt.Errorf("authentication required"), usecaseStatus.Unauthenticated)
		}

		// Verify account_id matches the authenticated account (unless system admin)
		if claims.Role != "system_admin" && claims.AccountID != input.AccountID {
			log.Warnf("Account ID mismatch: token has %s, request for %s", claims.AccountID, input.AccountID)
			return usecaseStatus.Wrap(fmt.Errorf("unauthorized: account mismatch"), usecaseStatus.PermissionDenied)
		}

		if api.Services.HealthService == nil {
			return usecaseStatus.Wrap(fmt.Errorf("health service not available"), usecaseStatus.InvalidArgument)
		}

		// Get health records for this account
		records, err := api.Services.HealthService.GetHealthRecordsByAccount(input.AccountID)
		if err != nil {
			return usecaseStatus.Wrap(fmt.Errorf("failed to get health records for account %s: %w", input.AccountID, err), usecaseStatus.InvalidArgument)
		}

		// Get summary (filtered by account)
		summary := map[string]interface{}{
			"account_id":   input.AccountID,
			"total":        len(records),
			"healthy":      0,
			"unhealthy":    0,
			"starting":     0,
			"last_updated": time.Now(),
		}

		for _, record := range records {
			switch record.Status {
			case "Healthy":
				summary["healthy"] = summary["healthy"].(int) + 1
			case "Unhealthy":
				summary["unhealthy"] = summary["unhealthy"].(int) + 1
			case "Starting":
				summary["starting"] = summary["starting"].(int) + 1
			}
		}

		output.Services = records
		output.Summary = summary

		return nil
	})

	u.SetTitle("List Account Services")
	u.SetDescription("Returns all service instances for a specific account")
	u.SetTags("health", "monitoring", "accounts")

	return u
}

