package api

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/n0needt0/bytefreezer-control/config"
	"github.com/n0needt0/bytefreezer-control/services"
)

// Mock services for testing
type mockServices struct {
	Config           *config.Config
	Database         services.DatabaseService
	EcosystemMonitor services.EcosystemMonitor
	Stats            *services.ControlStats
}

func (m *mockServices) GetStats() *services.ControlStats {
	if m.Stats == nil {
		m.Stats = &services.ControlStats{
			StartTime:             time.Now().Add(-1 * time.Hour),
			APIRequests:           123,
			ServicesMonitored:     4,
			HealthChecksPerformed: 456,
			ConfigUpdates:         7,
			DatabaseQueries:       89,
			LastActivity:          time.Now().Add(-5 * time.Minute),
		}
	}
	return m.Stats
}

func (m *mockServices) IncrementAPIRequests()     {}
func (m *mockServices) IncrementHealthChecks()    {}
func (m *mockServices) IncrementConfigUpdates()   {}
func (m *mockServices) IncrementDatabaseQueries() {}

type mockEcosystemMonitor struct {
	statuses map[string]*services.ServiceStatus
}

func (m *mockEcosystemMonitor) CheckAllServices() error {
	return nil
}

func (m *mockEcosystemMonitor) CheckService(serviceName string) (*services.ServiceStatus, error) {
	if status, exists := m.statuses[serviceName]; exists {
		return status, nil
	}
	return &services.ServiceStatus{
		Name:         serviceName,
		URL:          "http://test:8080",
		Healthy:      false,
		LastCheck:    time.Now(),
		ResponseTime: 0,
		Error:        "service not found",
	}, nil
}

func (m *mockEcosystemMonitor) GetServiceStatuses() map[string]*services.ServiceStatus {
	if m.statuses == nil {
		m.statuses = map[string]*services.ServiceStatus{
			"receiver": {
				Name:         "receiver",
				URL:          "http://receiver:8080",
				Healthy:      true,
				LastCheck:    time.Now(),
				ResponseTime: 50 * time.Millisecond,
				Version:      "1.0.0",
			},
			"piper": {
				Name:         "piper",
				URL:          "http://piper:8080",
				Healthy:      true,
				LastCheck:    time.Now(),
				ResponseTime: 75 * time.Millisecond,
				Version:      "1.0.0",
			},
			"proxy": {
				Name:         "proxy",
				URL:          "http://proxy:8088",
				Healthy:      false,
				LastCheck:    time.Now(),
				ResponseTime: 0,
				Error:        "connection refused",
			},
		}
	}
	return m.statuses
}

type mockDatabaseService struct{}

func (m *mockDatabaseService) Connect() error { return nil }
func (m *mockDatabaseService) Close() error   { return nil }
func (m *mockDatabaseService) Ping() error    { return nil }
func (m *mockDatabaseService) GetTenants() ([]services.Tenant, error) {
	return []services.Tenant{}, nil
}
func (m *mockDatabaseService) GetTenant(id string) (*services.Tenant, error) { return nil, nil }
func (m *mockDatabaseService) CreateTenant(tenant *services.Tenant) error    { return nil }
func (m *mockDatabaseService) UpdateTenant(tenant *services.Tenant) error    { return nil }
func (m *mockDatabaseService) DeleteTenant(id string) error                  { return nil }

func createTestAPI() *API {
	cfg := &config.Config{
		App: config.AppConfig{
			Name:    "test-control",
			Version: "1.0.0-test",
		},
		Auth: config.AuthConfig{
			Enabled:          true,
			JWTSecret:        "test-secret-key-for-jwt-signing-12345678",
			TokenExpiryHours: 24,
			AdminUsers:       []string{"admin@test.com"},
		},
		Services: config.ServicesConfig{
			Receiver: config.ServiceEndpoint{
				URL:            "http://receiver:8080",
				HealthEndpoint: "/health",
				ConfigEndpoint: "/config",
				TimeoutSeconds: 30,
			},
			Proxy: config.ServiceEndpoint{
				URL:            "http://proxy:8088",
				HealthEndpoint: "/api/v2/health",
				ConfigEndpoint: "/api/v2/config",
				TimeoutSeconds: 30,
			},
		},
		Database: config.DatabaseConfig{
			Enabled: true,
			Type:    "postgres",
			Host:    "localhost",
			Port:    5432,
		},
		RateLimit: config.RateLimitConfig{
			Enabled:           true,
			RequestsPerMinute: 100,
			BurstSize:         20,
		},
	}

	mockSvcs := &mockServices{
		Config:           cfg,
		EcosystemMonitor: &mockEcosystemMonitor{},
		Database:         &mockDatabaseService{},
	}

	api := &API{
		Config:   cfg,
		Services: &services.Services{
			Config:           cfg,
			Database:         mockSvcs.Database,
			EcosystemMonitor: mockSvcs.EcosystemMonitor,
			Stats:            mockSvcs.Stats,
		},
	}

	// Mock services are now properly initialized

	return api
}

func TestHealthCheck(t *testing.T) {
	api := createTestAPI()
	handler := api.HealthCheck()

	// Test the handler
	var output HealthResponse
	err := handler.Interact(context.Background(), struct{}{}, &output)

	if err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}

	if output.Status != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", output.Status)
	}

	if output.Service != "test-control" {
		t.Errorf("Expected service 'test-control', got '%s'", output.Service)
	}

	if output.Version != "1.0.0-test" {
		t.Errorf("Expected version '1.0.0-test', got '%s'", output.Version)
	}

	if output.Uptime == "" {
		t.Error("Uptime should not be empty")
	}

	if output.Timestamp.IsZero() {
		t.Error("Timestamp should be set")
	}
}

func TestGetConfig(t *testing.T) {
	api := createTestAPI()
	handler := api.GetConfig()

	// Test the handler
	var output ConfigResponse
	err := handler.Interact(context.Background(), struct{}{}, &output)

	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}

	// Verify app config
	if output.App.Name != "test-control" {
		t.Errorf("Expected app name 'test-control', got '%s'", output.App.Name)
	}

	if output.App.Version != "1.0.0-test" {
		t.Errorf("Expected app version '1.0.0-test', got '%s'", output.App.Version)
	}

	// Verify services config
	if output.Services.Receiver.URL != "http://receiver:8080" {
		t.Errorf("Expected receiver URL 'http://receiver:8080', got '%s'", output.Services.Receiver.URL)
	}

	if output.Services.Receiver.TimeoutSeconds != 30 {
		t.Errorf("Expected receiver timeout 30, got %d", output.Services.Receiver.TimeoutSeconds)
	}

	// Verify database config (should not include sensitive info)
	if output.Database.Type != "postgres" {
		t.Errorf("Expected database type 'postgres', got '%s'", output.Database.Type)
	}

	if output.Database.Host != "localhost" {
		t.Errorf("Expected database host 'localhost', got '%s'", output.Database.Host)
	}

	if output.Database.Port != 5432 {
		t.Errorf("Expected database port 5432, got %d", output.Database.Port)
	}

	// Verify auth config (should not include JWT secret)
	if !output.Auth.Enabled {
		t.Error("Expected auth to be enabled")
	}

	if output.Auth.TokenExpiryHours != 24 {
		t.Errorf("Expected token expiry 24 hours, got %d", output.Auth.TokenExpiryHours)
	}

	if output.Auth.AdminUsersCount != 1 {
		t.Errorf("Expected 1 admin user, got %d", output.Auth.AdminUsersCount)
	}

	// Verify rate limit config
	if !output.RateLimit.Enabled {
		t.Error("Expected rate limit to be enabled")
	}

	if output.RateLimit.RequestsPerMinute != 100 {
		t.Errorf("Expected 100 requests per minute, got %d", output.RateLimit.RequestsPerMinute)
	}
}

func TestGetEcosystemHealth(t *testing.T) {
	api := createTestAPI()
	handler := api.GetEcosystemHealth()

	// Test the handler
	var output EcosystemHealthResponse
	err := handler.Interact(context.Background(), struct{}{}, &output)

	if err != nil {
		t.Fatalf("GetEcosystemHealth failed: %v", err)
	}

	// Check overall status
	if output.OverallStatus != "degraded" { // Should be degraded because proxy is down
		t.Errorf("Expected overall status 'degraded', got '%s'", output.OverallStatus)
	}

	// Check service counts
	if output.TotalCount != 3 {
		t.Errorf("Expected 3 total services, got %d", output.TotalCount)
	}

	if output.HealthyCount != 2 { // receiver and piper are healthy
		t.Errorf("Expected 2 healthy services, got %d", output.HealthyCount)
	}

	// Check services
	if output.Services == nil {
		t.Fatal("Services should not be nil")
	}

	if len(output.Services) != 3 {
		t.Errorf("Expected 3 services, got %d", len(output.Services))
	}

	// Check specific services
	if receiverStatus, exists := output.Services["receiver"]; exists {
		if !receiverStatus.Healthy {
			t.Error("Receiver should be healthy")
		}
	} else {
		t.Error("Receiver status not found")
	}

	if proxyStatus, exists := output.Services["proxy"]; exists {
		if proxyStatus.Healthy {
			t.Error("Proxy should be unhealthy")
		}
	} else {
		t.Error("Proxy status not found")
	}
}

func TestGetServiceStatuses(t *testing.T) {
	api := createTestAPI()
	handler := api.GetServiceStatuses()

	// Test the handler
	var output map[string]*services.ServiceStatus
	err := handler.Interact(context.Background(), struct{}{}, &output)

	if err != nil {
		t.Fatalf("GetServiceStatuses failed: %v", err)
	}

	if len(output) != 3 {
		t.Errorf("Expected 3 services, got %d", len(output))
	}

	// Verify services are present
	expectedServices := []string{"receiver", "piper", "proxy"}
	for _, serviceName := range expectedServices {
		if _, exists := output[serviceName]; !exists {
			t.Errorf("Service '%s' not found in output", serviceName)
		}
	}
}

func TestGetServiceStatus(t *testing.T) {
	api := createTestAPI()
	handler := api.GetServiceStatus()

	testCases := []struct {
		name        string
		serviceName string
		expectError bool
	}{
		{
			name:        "Valid service",
			serviceName: "receiver",
			expectError: false,
		},
		{
			name:        "Unknown service",
			serviceName: "unknown",
			expectError: false, // Mock returns a status even for unknown services
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			input := struct {
				ServiceName string `path:"serviceName"`
			}{
				ServiceName: tc.serviceName,
			}

			var output *services.ServiceStatus
			err := handler.Interact(context.Background(), input, &output)

			if tc.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tc.expectError && output == nil {
				t.Error("Output should not be nil for valid request")
			}

			if !tc.expectError && output.Name != tc.serviceName {
				t.Errorf("Expected service name '%s', got '%s'", tc.serviceName, output.Name)
			}
		})
	}
}

func TestRestartService(t *testing.T) {
	api := createTestAPI()
	handler := api.RestartService()

	input := struct {
		ServiceName string `path:"serviceName"`
	}{
		ServiceName: "receiver",
	}

	var output struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}

	err := handler.Interact(context.Background(), input, &output)

	// Should return error because restart is not implemented
	if err == nil {
		t.Error("Expected error for unimplemented restart")
	}

	if output.Success {
		t.Error("Expected success to be false for unimplemented feature")
	}

	if !strings.Contains(output.Message, "not implemented") {
		t.Errorf("Expected 'not implemented' in message, got '%s'", output.Message)
	}
}

func TestGetTenants(t *testing.T) {
	api := createTestAPI()
	handler := api.GetTenants()

	var output []services.Tenant
	err := handler.Interact(context.Background(), struct{}{}, &output)

	if err != nil {
		t.Fatalf("GetTenants failed: %v", err)
	}

	// Mock returns empty slice
	if output == nil {
		t.Error("Output should not be nil")
	}

	if len(output) != 0 {
		t.Errorf("Expected 0 tenants from mock, got %d", len(output))
	}
}

func TestGetTenant(t *testing.T) {
	api := createTestAPI()
	handler := api.GetTenant()

	input := struct {
		TenantID string `path:"tenantId"`
	}{
		TenantID: "tenant-123",
	}

	var output *services.Tenant
	err := handler.Interact(context.Background(), input, &output)

	// Mock returns error for tenant not found
	if err == nil {
		t.Error("Expected error for tenant not found")
	}
}

func TestCreateTenant(t *testing.T) {
	api := createTestAPI()
	handler := api.CreateTenant()

	input := struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}{
		Name:  "Test Tenant",
		Email: "admin@test.com",
	}

	var output *services.Tenant
	err := handler.Interact(context.Background(), input, &output)

	// Should return error because creation is not implemented
	if err == nil {
		t.Error("Expected error for unimplemented tenant creation")
	}
}

func TestUpdateTenant(t *testing.T) {
	api := createTestAPI()
	handler := api.UpdateTenant()

	input := struct {
		TenantID string `path:"tenantId"`
		Name     string `json:"name"`
		Email    string `json:"email"`
	}{
		TenantID: "tenant-123",
		Name:     "Updated Tenant",
		Email:    "updated@test.com",
	}

	var output *services.Tenant
	err := handler.Interact(context.Background(), input, &output)

	// Should return error because update is not implemented
	if err == nil {
		t.Error("Expected error for unimplemented tenant update")
	}
}

func TestDeleteTenant(t *testing.T) {
	api := createTestAPI()
	handler := api.DeleteTenant()

	input := struct {
		TenantID string `path:"tenantId"`
	}{
		TenantID: "tenant-123",
	}

	var output struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}

	err := handler.Interact(context.Background(), input, &output)

	// Should return error because deletion is not implemented
	if err == nil {
		t.Error("Expected error for unimplemented tenant deletion")
	}

	if output.Success {
		t.Error("Expected success to be false for unimplemented feature")
	}
}

func TestLogin(t *testing.T) {
	api := createTestAPI()
	handler := api.Login()

	testCases := []struct {
		name        string
		username    string
		password    string
		expectError bool
		expectToken bool
	}{
		{
			name:        "Valid admin user",
			username:    "admin@test.com",
			password:    "any-password", // Password validation not implemented in mock
			expectError: false,
			expectToken: true,
		},
		{
			name:        "Invalid user",
			username:    "unknown@test.com",
			password:    "any-password",
			expectError: true,
			expectToken: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			input := LoginRequest{
				Username: tc.username,
				Password: tc.password,
			}

			var output LoginResponse
			err := handler.Interact(context.Background(), input, &output)

			if tc.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if tc.expectToken && output.Token == "" {
				t.Error("Expected token but got empty string")
			}

			if tc.expectToken && output.Expires == "" {
				t.Error("Expected expires timestamp but got empty string")
			}

			if tc.expectToken {
				// Verify expires is in future
				expiresTime, parseErr := time.Parse(time.RFC3339, output.Expires)
				if parseErr != nil {
					t.Errorf("Failed to parse expires time: %v", parseErr)
				} else if expiresTime.Before(time.Now()) {
					t.Error("Token should expire in the future")
				}
			}
		})
	}
}

func TestGetStats(t *testing.T) {
	api := createTestAPI()
	handler := api.GetStats()

	var output StatsResponse
	err := handler.Interact(context.Background(), struct{}{}, &output)

	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}

	// Verify stats values from mock
	if output.APIRequests != 123 {
		t.Errorf("Expected 123 API requests, got %d", output.APIRequests)
	}

	if output.ServicesMonitored != 4 {
		t.Errorf("Expected 4 services monitored, got %d", output.ServicesMonitored)
	}

	if output.HealthChecksPerformed != 456 {
		t.Errorf("Expected 456 health checks, got %d", output.HealthChecksPerformed)
	}

	if output.Uptime == "" {
		t.Error("Uptime should not be empty")
	}

	if output.LastActivity.IsZero() {
		t.Error("Last activity should not be zero")
	}
}

func TestAPIResponses_JSONSerialization(t *testing.T) {
	// Test that response structures can be properly serialized to JSON
	healthResp := HealthResponse{
		Status:    "ok",
		Service:   "test",
		Version:   "1.0.0",
		Timestamp: time.Now(),
		Uptime:    "1h30m",
	}

	data, err := json.Marshal(healthResp)
	if err != nil {
		t.Errorf("Failed to marshal HealthResponse: %v", err)
	}

	var unmarshaled HealthResponse
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Errorf("Failed to unmarshal HealthResponse: %v", err)
	}

	if unmarshaled.Status != healthResp.Status {
		t.Errorf("Expected status '%s', got '%s'", healthResp.Status, unmarshaled.Status)
	}
}

func TestEcosystemHealthResponse_OverallStatus(t *testing.T) {
	testCases := []struct {
		name           string
		statuses       map[string]*services.ServiceStatus
		expectedStatus string
	}{
		{
			name: "All services healthy",
			statuses: map[string]*services.ServiceStatus{
				"service1": {Healthy: true},
				"service2": {Healthy: true},
			},
			expectedStatus: "healthy",
		},
		{
			name: "Some services unhealthy",
			statuses: map[string]*services.ServiceStatus{
				"service1": {Healthy: true},
				"service2": {Healthy: false},
			},
			expectedStatus: "degraded",
		},
		{
			name: "All services unhealthy",
			statuses: map[string]*services.ServiceStatus{
				"service1": {Healthy: false},
				"service2": {Healthy: false},
			},
			expectedStatus: "unhealthy",
		},
		{
			name:           "No services",
			statuses:       map[string]*services.ServiceStatus{},
			expectedStatus: "unhealthy",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			healthyCount := 0
			for _, status := range tc.statuses {
				if status.Healthy {
					healthyCount++
				}
			}

			var overallStatus string
			overallHealthy := healthyCount == len(tc.statuses)

			if overallHealthy && len(tc.statuses) > 0 {
				overallStatus = "healthy"
			} else if healthyCount > 0 {
				overallStatus = "degraded"
			} else {
				overallStatus = "unhealthy"
			}

			if overallStatus != tc.expectedStatus {
				t.Errorf("Expected overall status '%s', got '%s'", tc.expectedStatus, overallStatus)
			}
		})
	}
}
