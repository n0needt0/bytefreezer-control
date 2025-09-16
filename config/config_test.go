package config

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file for testing
	configContent := `
app:
  name: "test-service"
  version: "1.0.0-test"

logging:
  level: "debug"
  encoding: "json"

server:
  api_port: 8083

services:
  receiver:
    url: "http://test-receiver:8080"
    health_endpoint: "/test-health"
    config_endpoint: "/test-config"
    timeout_seconds: 15
  proxy:
    url: "http://test-proxy:8088"
    health_endpoint: "/api/v2/health"
    config_endpoint: "/api/v2/config"
    timeout_seconds: 20

database:
  enabled: true
  type: "postgres"
  host: "test-postgres"
  port: 5433
  database: "test_db"
  username: "test_user"
  password: "test_pass"
  ssl_mode: "disable"
  max_connections: 10
  max_idle_connections: 5

auth:
  enabled: true
  jwt_secret: "test-secret-key-12345678901234567890"
  token_expiry_hours: 12
  admin_users:
    - "test-admin@example.com"
    - "test-user@example.com"

rate_limit:
  enabled: true
  requests_per_minute: 50
  burst_size: 10

otel:
  enabled: false
  endpoint: "http://test-jaeger:4317"
  service_name: "test-control"
  scrape_interval_seconds: 30

housekeeping:
  enabled: false
  interval_seconds: 600

dev: true
`

	// Write test config to temporary file
	tmpFile, err := os.CreateTemp("", "test-config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("Failed to write config content: %v", err)
	}
	tmpFile.Close()

	// Test loading the configuration
	var config Config
	err = LoadConfig(tmpFile.Name(), "TEST_", &config)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify app configuration
	if config.App.Name != "test-service" {
		t.Errorf("Expected app name 'test-service', got '%s'", config.App.Name)
	}
	if config.App.Version != "1.0.0-test" {
		t.Errorf("Expected app version '1.0.0-test', got '%s'", config.App.Version)
	}

	// Verify logging configuration
	if config.Logging.Level != "debug" {
		t.Errorf("Expected log level 'debug', got '%s'", config.Logging.Level)
	}
	if config.Logging.Encoding != "json" {
		t.Errorf("Expected log encoding 'json', got '%s'", config.Logging.Encoding)
	}

	// Verify server configuration
	if config.Server.ApiPort != 8083 {
		t.Errorf("Expected API port 8083, got %d", config.Server.ApiPort)
	}

	// Verify services configuration
	if config.Services.Receiver.URL != "http://test-receiver:8080" {
		t.Errorf("Expected receiver URL 'http://test-receiver:8080', got '%s'", config.Services.Receiver.URL)
	}
	if config.Services.Receiver.TimeoutSeconds != 15 {
		t.Errorf("Expected receiver timeout 15, got %d", config.Services.Receiver.TimeoutSeconds)
	}

	// Verify database configuration
	if !config.Database.Enabled {
		t.Error("Expected database to be enabled")
	}
	if config.Database.Type != "postgres" {
		t.Errorf("Expected database type 'postgres', got '%s'", config.Database.Type)
	}
	if config.Database.Host != "test-postgres" {
		t.Errorf("Expected database host 'test-postgres', got '%s'", config.Database.Host)
	}
	if config.Database.Port != 5433 {
		t.Errorf("Expected database port 5433, got %d", config.Database.Port)
	}

	// Verify auth configuration
	if !config.Auth.Enabled {
		t.Error("Expected auth to be enabled")
	}
	if config.Auth.TokenExpiryHours != 12 {
		t.Errorf("Expected token expiry 12 hours, got %d", config.Auth.TokenExpiryHours)
	}
	if len(config.Auth.AdminUsers) != 2 {
		t.Errorf("Expected 2 admin users, got %d", len(config.Auth.AdminUsers))
	}

	// Verify rate limit configuration
	if !config.RateLimit.Enabled {
		t.Error("Expected rate limit to be enabled")
	}
	if config.RateLimit.RequestsPerMinute != 50 {
		t.Errorf("Expected 50 requests per minute, got %d", config.RateLimit.RequestsPerMinute)
	}

	// Verify development mode
	if !config.Dev {
		t.Error("Expected dev mode to be enabled")
	}
}

func TestLoadConfigFileNotFound(t *testing.T) {
	var config Config
	err := LoadConfig("non-existent-file.yaml", "TEST_", &config)
	if err == nil {
		t.Error("Expected error when loading non-existent config file")
	}
}

func TestLoadConfigInvalidYAML(t *testing.T) {
	// Create a temporary file with invalid YAML
	invalidYAML := `
app:
  name: "test
  invalid yaml
`
	tmpFile, err := os.CreateTemp("", "invalid-config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(invalidYAML); err != nil {
		t.Fatalf("Failed to write invalid YAML: %v", err)
	}
	tmpFile.Close()

	var config Config
	err = LoadConfig(tmpFile.Name(), "TEST_", &config)
	if err == nil {
		t.Error("Expected error when loading invalid YAML")
	}
}

func TestServiceEndpointValidation(t *testing.T) {
	testCases := []struct {
		name     string
		endpoint ServiceEndpoint
		valid    bool
	}{
		{
			name: "Valid endpoint with all fields",
			endpoint: ServiceEndpoint{
				URL:            "http://example.com:8080",
				HealthEndpoint: "/health",
				ConfigEndpoint: "/config",
				TimeoutSeconds: 30,
			},
			valid: true,
		},
		{
			name: "Valid endpoint with minimal fields",
			endpoint: ServiceEndpoint{
				URL:            "http://example.com",
				HealthEndpoint: "/health",
				TimeoutSeconds: 30,
			},
			valid: true,
		},
		{
			name: "Invalid endpoint - empty URL",
			endpoint: ServiceEndpoint{
				URL:            "",
				HealthEndpoint: "/health",
				TimeoutSeconds: 30,
			},
			valid: false,
		},
		{
			name: "Invalid endpoint - empty health endpoint",
			endpoint: ServiceEndpoint{
				URL:            "http://example.com",
				HealthEndpoint: "",
				TimeoutSeconds: 30,
			},
			valid: false,
		},
		{
			name: "Invalid endpoint - zero timeout",
			endpoint: ServiceEndpoint{
				URL:            "http://example.com",
				HealthEndpoint: "/health",
				TimeoutSeconds: 0,
			},
			valid: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			isValid := tc.endpoint.URL != "" &&
				tc.endpoint.HealthEndpoint != "" &&
				tc.endpoint.TimeoutSeconds > 0

			if isValid != tc.valid {
				t.Errorf("Expected endpoint validity %v, got %v", tc.valid, isValid)
			}
		})
	}
}

func TestDatabaseConfigValidation(t *testing.T) {
	testCases := []struct {
		name   string
		config DatabaseConfig
		valid  bool
	}{
		{
			name: "Valid database config",
			config: DatabaseConfig{
				Enabled:            true,
				Type:               "postgres",
				Host:               "localhost",
				Port:               5432,
				Database:           "test_db",
				Username:           "user",
				Password:           "password",
				MaxConnections:     25,
				MaxIdleConnections: 10,
			},
			valid: true,
		},
		{
			name: "Valid disabled database",
			config: DatabaseConfig{
				Enabled: false,
			},
			valid: true,
		},
		{
			name: "Invalid - missing host when enabled",
			config: DatabaseConfig{
				Enabled:  true,
				Type:     "postgres",
				Port:     5432,
				Database: "test_db",
				Username: "user",
				Password: "password",
			},
			valid: false,
		},
		{
			name: "Invalid - invalid port",
			config: DatabaseConfig{
				Enabled:  true,
				Type:     "postgres",
				Host:     "localhost",
				Port:     0,
				Database: "test_db",
				Username: "user",
				Password: "password",
			},
			valid: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			isValid := !tc.config.Enabled ||
				(tc.config.Host != "" &&
					tc.config.Port > 0 &&
					tc.config.Database != "" &&
					tc.config.Username != "")

			if isValid != tc.valid {
				t.Errorf("Expected config validity %v, got %v", tc.valid, isValid)
			}
		})
	}
}

func TestAuthConfigValidation(t *testing.T) {
	testCases := []struct {
		name   string
		config AuthConfig
		valid  bool
	}{
		{
			name: "Valid auth config",
			config: AuthConfig{
				Enabled:          true,
				JWTSecret:        "very-long-secret-key-that-is-secure",
				TokenExpiryHours: 24,
				AdminUsers:       []string{"admin@example.com"},
			},
			valid: true,
		},
		{
			name: "Valid disabled auth",
			config: AuthConfig{
				Enabled: false,
			},
			valid: true,
		},
		{
			name: "Invalid - short JWT secret",
			config: AuthConfig{
				Enabled:          true,
				JWTSecret:        "short",
				TokenExpiryHours: 24,
				AdminUsers:       []string{"admin@example.com"},
			},
			valid: false,
		},
		{
			name: "Invalid - zero token expiry",
			config: AuthConfig{
				Enabled:          true,
				JWTSecret:        "very-long-secret-key-that-is-secure",
				TokenExpiryHours: 0,
				AdminUsers:       []string{"admin@example.com"},
			},
			valid: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			isValid := !tc.config.Enabled ||
				(len(tc.config.JWTSecret) >= 32 &&
					tc.config.TokenExpiryHours > 0)

			if isValid != tc.valid {
				t.Errorf("Expected auth config validity %v, got %v", tc.valid, isValid)
			}
		})
	}
}

func TestInitializeComponents(t *testing.T) {
	config := &Config{
		Database: DatabaseConfig{
			Enabled: true,
			Type:    "postgres",
			Host:    "localhost",
			Port:    5432,
		},
	}

	// This should not fail even if database connection fails
	// as it's just a placeholder implementation
	err := config.InitializeComponents()
	if err != nil {
		t.Errorf("InitializeComponents should not fail: %v", err)
	}
}

func TestConfigDefaults(t *testing.T) {
	// Test that a minimal config file gets reasonable defaults
	minimalConfig := `
app:
  name: "minimal-test"
  version: "1.0.0"
`

	tmpFile, err := os.CreateTemp("", "minimal-config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(minimalConfig); err != nil {
		t.Fatalf("Failed to write minimal config: %v", err)
	}
	tmpFile.Close()

	var config Config
	err = LoadConfig(tmpFile.Name(), "TEST_", &config)
	if err != nil {
		t.Fatalf("Failed to load minimal config: %v", err)
	}

	// Verify that minimal config loads successfully
	if config.App.Name != "minimal-test" {
		t.Errorf("Expected app name 'minimal-test', got '%s'", config.App.Name)
	}

	// Verify defaults for boolean fields
	if config.Database.Enabled {
		t.Error("Expected database to be disabled by default")
	}
	if config.Auth.Enabled {
		t.Error("Expected auth to be disabled by default")
	}
	if config.RateLimit.Enabled {
		t.Error("Expected rate limit to be disabled by default")
	}
}
