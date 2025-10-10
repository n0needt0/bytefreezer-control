package api

import (
	"context"
	"fmt"
	"os"
	"time"

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

		if api.Services.HealthService == nil {
			return usecaseStatus.Wrap(fmt.Errorf("health service not available"), usecaseStatus.InvalidArgument)
		}

		// Get hostname if instance_id not provided
		instanceID := input.InstanceID
		if instanceID == "" {
			hostname, err := os.Hostname()
			if err != nil {
				return usecaseStatus.Wrap(fmt.Errorf("failed to get hostname: %w", err), usecaseStatus.InvalidArgument)
			}
			instanceID = hostname
		}

		registration := services.ServiceRegistration{
			ServiceType:   input.ServiceType,
			InstanceID:    instanceID,
			InstanceAPI:   input.InstanceAPI,
			Status:        "Unhealthy", // Initial status
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

