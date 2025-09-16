package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/n0needt0/bytefreezer-control/config"
)

func TestNewEcosystemMonitor(t *testing.T) {
	cfg := &config.Config{
		Services: config.ServicesConfig{
			Receiver: config.ServiceEndpoint{
				URL:            "http://receiver:8080",
				HealthEndpoint: "/health",
				TimeoutSeconds: 30,
			},
		},
	}

	monitor := NewEcosystemMonitor(cfg)
	if monitor == nil {
		t.Fatal("NewEcosystemMonitor should not return nil")
	}

	// Type assertion to access private fields for testing
	if em, ok := monitor.(*ecosystemMonitor); ok {
		if em.config != cfg {
			t.Error("Monitor should store the provided config")
		}
		if em.client == nil {
			t.Error("Monitor should create HTTP client")
		}
		if em.statuses == nil {
			t.Error("Monitor should initialize statuses map")
		}
	} else {
		t.Error("Monitor should be of type *ecosystemMonitor")
	}
}

func TestCheckSingleService_HealthyService(t *testing.T) {
	// Create a mock server that returns healthy responses
	healthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/health") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  "ok",
				"service": "test-receiver",
				"version": "1.0.0",
			})
		} else if strings.HasSuffix(r.URL.Path, "/config") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"app": map[string]interface{}{
					"name":    "test-receiver",
					"version": "1.0.0",
				},
			})
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer healthServer.Close()

	cfg := &config.Config{
		Services: config.ServicesConfig{
			Receiver: config.ServiceEndpoint{
				URL:            healthServer.URL,
				HealthEndpoint: "/health",
				ConfigEndpoint: "/config",
				TimeoutSeconds: 30,
			},
		},
	}

	monitor := NewEcosystemMonitor(cfg).(*ecosystemMonitor)

	status, err := monitor.checkSingleService("receiver", cfg.Services.Receiver)
	if err != nil {
		t.Fatalf("checkSingleService failed: %v", err)
	}

	if !status.Healthy {
		t.Error("Service should be marked as healthy")
	}
	if status.Name != "receiver" {
		t.Errorf("Expected service name 'receiver', got '%s'", status.Name)
	}
	if status.URL != healthServer.URL {
		t.Errorf("Expected service URL '%s', got '%s'", healthServer.URL, status.URL)
	}
	if status.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", status.Version)
	}
	if status.Error != "" {
		t.Errorf("Expected no error, got '%s'", status.Error)
	}
	if status.ResponseTime <= 0 {
		t.Error("Response time should be greater than 0")
	}
	if status.Config == nil {
		t.Error("Config should be populated")
	}
}

func TestCheckSingleService_UnhealthyService(t *testing.T) {
	// Create a mock server that returns unhealthy responses
	unhealthyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "error",
			"error":  "service unavailable",
		})
	}))
	defer unhealthyServer.Close()

	cfg := &config.Config{
		Services: config.ServicesConfig{
			Receiver: config.ServiceEndpoint{
				URL:            unhealthyServer.URL,
				HealthEndpoint: "/health",
				TimeoutSeconds: 30,
			},
		},
	}

	monitor := NewEcosystemMonitor(cfg).(*ecosystemMonitor)

	status, err := monitor.checkSingleService("receiver", cfg.Services.Receiver)
	if err == nil {
		t.Error("Expected error for unhealthy service")
	}

	if status.Healthy {
		t.Error("Service should be marked as unhealthy")
	}
	if !strings.Contains(status.Error, "503") {
		t.Errorf("Expected error to mention status code 503, got '%s'", status.Error)
	}
}

func TestCheckSingleService_ServiceDown(t *testing.T) {
	cfg := &config.Config{
		Services: config.ServicesConfig{
			Receiver: config.ServiceEndpoint{
				URL:            "http://non-existent-service:8080",
				HealthEndpoint: "/health",
				TimeoutSeconds: 1, // Short timeout for testing
			},
		},
	}

	monitor := NewEcosystemMonitor(cfg).(*ecosystemMonitor)

	status, err := monitor.checkSingleService("receiver", cfg.Services.Receiver)
	if err == nil {
		t.Error("Expected error for non-existent service")
	}

	if status.Healthy {
		t.Error("Service should be marked as unhealthy")
	}
	if !strings.Contains(status.Error, "Health check failed") {
		t.Errorf("Expected error about health check failure, got '%s'", status.Error)
	}
}

func TestCheckService(t *testing.T) {
	// Create mock servers for different services
	receiverServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "ok",
			"service": "bytefreezer-receiver",
		})
	}))
	defer receiverServer.Close()

	cfg := &config.Config{
		Services: config.ServicesConfig{
			Receiver: config.ServiceEndpoint{
				URL:            receiverServer.URL,
				HealthEndpoint: "/health",
				TimeoutSeconds: 30,
			},
			Proxy: config.ServiceEndpoint{
				URL:            "http://proxy:8088",
				HealthEndpoint: "/health",
				TimeoutSeconds: 30,
			},
		},
	}

	monitor := NewEcosystemMonitor(cfg)

	// Test checking existing service
	status, err := monitor.CheckService("receiver")
	if err != nil {
		t.Fatalf("CheckService failed for receiver: %v", err)
	}
	if !status.Healthy {
		t.Error("Receiver should be healthy")
	}

	// Test checking non-existent service
	_, err = monitor.CheckService("unknown")
	if err == nil {
		t.Error("Expected error for unknown service")
	}
	if !strings.Contains(err.Error(), "unknown service") {
		t.Errorf("Expected 'unknown service' error, got '%s'", err.Error())
	}
}

func TestCheckAllServices(t *testing.T) {
	// Create mock servers for different services
	receiverServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "ok",
			"service": "receiver",
		})
	}))
	defer receiverServer.Close()

	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "ok",
			"service": "proxy",
		})
	}))
	defer proxyServer.Close()

	cfg := &config.Config{
		Services: config.ServicesConfig{
			Receiver: config.ServiceEndpoint{
				URL:            receiverServer.URL,
				HealthEndpoint: "/health",
				TimeoutSeconds: 30,
			},
			Proxy: config.ServiceEndpoint{
				URL:            proxyServer.URL,
				HealthEndpoint: "/health",
				TimeoutSeconds: 30,
			},
			SOC: config.ServiceEndpoint{
				URL:            "http://down-service:8089",
				HealthEndpoint: "/health",
				TimeoutSeconds: 1, // Short timeout
			},
			Packer: config.ServiceEndpoint{
				URL:            "http://another-down-service:8090",
				HealthEndpoint: "/health",
				TimeoutSeconds: 1, // Short timeout
			},
		},
	}

	monitor := NewEcosystemMonitor(cfg)

	// Check all services
	err := monitor.CheckAllServices()
	if err != nil {
		t.Fatalf("CheckAllServices failed: %v", err)
	}

	// Verify statuses were updated
	statuses := monitor.GetServiceStatuses()

	// Should have 4 services
	expectedServices := []string{"receiver", "proxy", "soc", "packer"}
	if len(statuses) != len(expectedServices) {
		t.Errorf("Expected %d services, got %d", len(expectedServices), len(statuses))
	}

	// Check that healthy services are marked correctly
	if status, exists := statuses["receiver"]; exists {
		if !status.Healthy {
			t.Error("Receiver should be healthy")
		}
	} else {
		t.Error("Receiver status not found")
	}

	if status, exists := statuses["proxy"]; exists {
		if !status.Healthy {
			t.Error("Proxy should be healthy")
		}
	} else {
		t.Error("Proxy status not found")
	}

	// Check that down services are marked unhealthy
	if status, exists := statuses["soc"]; exists {
		if status.Healthy {
			t.Error("SOC should be unhealthy")
		}
	} else {
		t.Error("SOC status not found")
	}

	if status, exists := statuses["packer"]; exists {
		if status.Healthy {
			t.Error("Packer should be unhealthy")
		}
	} else {
		t.Error("Packer status not found")
	}
}

func TestGetServiceStatuses(t *testing.T) {
	cfg := &config.Config{
		Services: config.ServicesConfig{
			Receiver: config.ServiceEndpoint{
				URL:            "http://receiver:8080",
				HealthEndpoint: "/health",
				TimeoutSeconds: 30,
			},
		},
	}

	monitor := NewEcosystemMonitor(cfg).(*ecosystemMonitor)

	// Initially, statuses should be empty
	statuses := monitor.GetServiceStatuses()
	if len(statuses) != 0 {
		t.Errorf("Expected 0 statuses initially, got %d", len(statuses))
	}

	// Add a status manually for testing
	testStatus := &ServiceStatus{
		Name:         "receiver",
		URL:          "http://receiver:8080",
		Healthy:      true,
		LastCheck:    time.Now(),
		ResponseTime: 100 * time.Millisecond,
		Version:      "1.0.0",
	}

	monitor.statuses["receiver"] = testStatus

	// Get statuses and verify deep copy
	statuses = monitor.GetServiceStatuses()
	if len(statuses) != 1 {
		t.Errorf("Expected 1 status, got %d", len(statuses))
	}

	receivedStatus, exists := statuses["receiver"]
	if !exists {
		t.Fatal("Receiver status not found")
	}

	// Verify it's a copy (different pointer)
	if receivedStatus == testStatus {
		t.Error("GetServiceStatuses should return a copy, not the original")
	}

	// Verify the content is the same
	if receivedStatus.Name != testStatus.Name {
		t.Errorf("Expected name '%s', got '%s'", testStatus.Name, receivedStatus.Name)
	}
	if receivedStatus.Healthy != testStatus.Healthy {
		t.Errorf("Expected healthy %v, got %v", testStatus.Healthy, receivedStatus.Healthy)
	}
}

func TestHealthResponseParsing(t *testing.T) {
	testCases := []struct {
		name            string
		responseBody    string
		responseStatus  int
		expectedHealthy bool
		expectedVersion string
	}{
		{
			name: "Valid health response with ok status",
			responseBody: `{
				"status": "ok",
				"service": "test-service",
				"version": "1.2.3"
			}`,
			responseStatus:  http.StatusOK,
			expectedHealthy: true,
			expectedVersion: "1.2.3",
		},
		{
			name: "Valid health response with error status",
			responseBody: `{
				"status": "error",
				"service": "test-service",
				"version": "1.2.3"
			}`,
			responseStatus:  http.StatusOK,
			expectedHealthy: false,
			expectedVersion: "1.2.3",
		},
		{
			name:            "Invalid JSON response but 200 status",
			responseBody:    `invalid json`,
			responseStatus:  http.StatusOK,
			expectedHealthy: true,
			expectedVersion: "",
		},
		{
			name:            "Empty response but 200 status",
			responseBody:    ``,
			responseStatus:  http.StatusOK,
			expectedHealthy: true,
			expectedVersion: "",
		},
		{
			name: "Valid JSON but non-200 status",
			responseBody: `{
				"status": "ok",
				"version": "1.2.3"
			}`,
			responseStatus:  http.StatusServiceUnavailable,
			expectedHealthy: false,
			expectedVersion: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.responseStatus)
				w.Write([]byte(tc.responseBody))
			}))
			defer server.Close()

			cfg := &config.Config{}
			monitor := NewEcosystemMonitor(cfg).(*ecosystemMonitor)

			serviceConfig := config.ServiceEndpoint{
				URL:            server.URL,
				HealthEndpoint: "/health",
				TimeoutSeconds: 30,
			}

			status, _ := monitor.checkSingleService("test-service", serviceConfig)

			if status.Healthy != tc.expectedHealthy {
				t.Errorf("Expected healthy %v, got %v", tc.expectedHealthy, status.Healthy)
			}
			if status.Version != tc.expectedVersion {
				t.Errorf("Expected version '%s', got '%s'", tc.expectedVersion, status.Version)
			}
		})
	}
}

func TestConcurrentHealthChecks(t *testing.T) {
	// Test that concurrent health checks don't cause race conditions
	cfg := &config.Config{
		Services: config.ServicesConfig{
			Receiver: config.ServiceEndpoint{
				URL:            "http://receiver:8080",
				HealthEndpoint: "/health",
				TimeoutSeconds: 1,
			},
			Proxy: config.ServiceEndpoint{
				URL:            "http://proxy:8088",
				HealthEndpoint: "/health",
				TimeoutSeconds: 1,
			},
		},
	}

	monitor := NewEcosystemMonitor(cfg)

	// Run multiple concurrent health checks
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			monitor.CheckAllServices()
			monitor.GetServiceStatuses()
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// This test passes if no race conditions occur
	statuses := monitor.GetServiceStatuses()
	if len(statuses) < 2 {
		t.Errorf("Expected at least 2 service statuses, got %d", len(statuses))
	}
}

func TestServiceTimeout(t *testing.T) {
	// Create a slow server that takes longer than the timeout
	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer slowServer.Close()

	cfg := &config.Config{
		Services: config.ServicesConfig{
			Receiver: config.ServiceEndpoint{
				URL:            slowServer.URL,
				HealthEndpoint: "/health",
				TimeoutSeconds: 1, // Short timeout
			},
		},
	}

	monitor := NewEcosystemMonitor(cfg)

	start := time.Now()
	status, err := monitor.CheckService("receiver")
	elapsed := time.Since(start)

	// Should timeout and return error
	if err == nil {
		t.Error("Expected timeout error")
	}
	if status.Healthy {
		t.Error("Service should be marked as unhealthy due to timeout")
	}

	// Should complete within reasonable time (not wait for full 2 seconds)
	if elapsed > 2*time.Second {
		t.Errorf("Health check took too long: %v", elapsed)
	}
}
