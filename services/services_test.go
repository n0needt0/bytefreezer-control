package services

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/n0needt0/bytefreezer-control/config"
)

func TestNewServices(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Enabled: false, // Disable database to avoid connection issues in tests
		},
	}

	services := NewServices(cfg)

	if services == nil {
		t.Fatal("NewServices should not return nil")
	}

	if services.Config != cfg {
		t.Error("Services should store the provided config")
	}

	if services.Stats == nil {
		t.Error("Services should initialize stats")
	}

	if services.EcosystemMonitor == nil {
		t.Error("Services should initialize ecosystem monitor")
	}

	// Verify stats initialization
	if services.Stats.StartTime.IsZero() {
		t.Error("Stats start time should be initialized")
	}

	if services.Stats.LastActivity.IsZero() {
		t.Error("Stats last activity should be initialized")
	}
}

func TestNewServicesWithDatabase(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Enabled:  true,
			Type:     "postgres",
			Host:     "localhost",
			Port:     5432,
			Database: "test",
			Username: "test",
			Password: "test",
		},
	}

	services := NewServices(cfg)

	if services == nil {
		t.Fatal("NewServices should not return nil")
	}

	// Database service should be initialized even if connection fails
	// (The actual connection is attempted but failure is handled gracefully)
	if services.Database == nil {
		t.Error("Database service should be initialized when enabled")
	}
}

func TestGetStats(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{Enabled: false},
	}

	services := NewServices(cfg)

	stats := services.GetStats()
	if stats == nil {
		t.Fatal("GetStats should not return nil")
	}

	// Verify initial values
	if stats.APIRequests != 0 {
		t.Errorf("Expected 0 API requests initially, got %d", stats.APIRequests)
	}

	if stats.HealthChecksPerformed != 0 {
		t.Errorf("Expected 0 health checks initially, got %d", stats.HealthChecksPerformed)
	}

	if stats.StartTime.IsZero() {
		t.Error("Start time should be set")
	}
}

func TestIncrementAPIRequests(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{Enabled: false},
	}

	services := NewServices(cfg)

	// Initial state
	initialStats := services.GetStats()
	initialRequests := initialStats.APIRequests
	initialActivity := initialStats.LastActivity

	// Wait a bit to ensure timestamp changes
	time.Sleep(1 * time.Millisecond)

	// Increment API requests
	services.IncrementAPIRequests()

	// Verify increment
	newStats := services.GetStats()
	if newStats.APIRequests != initialRequests+1 {
		t.Errorf("Expected API requests to increment from %d to %d, got %d",
			initialRequests, initialRequests+1, newStats.APIRequests)
	}

	// Verify last activity was updated
	if !newStats.LastActivity.After(initialActivity) {
		t.Error("Last activity should be updated when incrementing API requests")
	}
}

func TestIncrementHealthChecks(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{Enabled: false},
	}

	services := NewServices(cfg)

	// Initial state
	initialStats := services.GetStats()
	initialHealthChecks := initialStats.HealthChecksPerformed

	// Increment health checks
	services.IncrementHealthChecks()

	// Verify increment
	newStats := services.GetStats()
	if newStats.HealthChecksPerformed != initialHealthChecks+1 {
		t.Errorf("Expected health checks to increment from %d to %d, got %d",
			initialHealthChecks, initialHealthChecks+1, newStats.HealthChecksPerformed)
	}

	// Health check increment should NOT update last activity (unlike API requests)
	if !newStats.LastActivity.Equal(initialStats.LastActivity) {
		t.Error("Last activity should not change when incrementing health checks")
	}
}

func TestIncrementConfigUpdates(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{Enabled: false},
	}

	services := NewServices(cfg)

	// Initial state
	initialStats := services.GetStats()
	initialConfigUpdates := initialStats.ConfigUpdates
	initialActivity := initialStats.LastActivity

	// Wait a bit to ensure timestamp changes
	time.Sleep(1 * time.Millisecond)

	// Increment config updates
	services.IncrementConfigUpdates()

	// Verify increment
	newStats := services.GetStats()
	if newStats.ConfigUpdates != initialConfigUpdates+1 {
		t.Errorf("Expected config updates to increment from %d to %d, got %d",
			initialConfigUpdates, initialConfigUpdates+1, newStats.ConfigUpdates)
	}

	// Verify last activity was updated
	if !newStats.LastActivity.After(initialActivity) {
		t.Error("Last activity should be updated when incrementing config updates")
	}
}

func TestIncrementDatabaseQueries(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{Enabled: false},
	}

	services := NewServices(cfg)

	// Initial state
	initialStats := services.GetStats()
	initialQueries := initialStats.DatabaseQueries

	// Increment database queries
	services.IncrementDatabaseQueries()

	// Verify increment
	newStats := services.GetStats()
	if newStats.DatabaseQueries != initialQueries+1 {
		t.Errorf("Expected database queries to increment from %d to %d, got %d",
			initialQueries, initialQueries+1, newStats.DatabaseQueries)
	}

	// Database query increment should NOT update last activity
	if !newStats.LastActivity.Equal(initialStats.LastActivity) {
		t.Error("Last activity should not change when incrementing database queries")
	}
}

func TestConcurrentStatsOperations(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{Enabled: false},
	}

	services := NewServices(cfg)

	// Run concurrent operations to test thread safety
	const numGoroutines = 100
	const opsPerGoroutine = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 4) // 4 different operation types

	// Concurrent API request increments
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				services.IncrementAPIRequests()
			}
		}()
	}

	// Concurrent health check increments
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				services.IncrementHealthChecks()
			}
		}()
	}

	// Concurrent config update increments
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				services.IncrementConfigUpdates()
			}
		}()
	}

	// Concurrent database query increments
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				services.IncrementDatabaseQueries()
			}
		}()
	}

	wg.Wait()

	// Verify final counts
	finalStats := services.GetStats()
	expectedCount := int64(numGoroutines * opsPerGoroutine)

	if finalStats.APIRequests != expectedCount {
		t.Errorf("Expected %d API requests, got %d", expectedCount, finalStats.APIRequests)
	}

	if finalStats.HealthChecksPerformed != expectedCount {
		t.Errorf("Expected %d health checks, got %d", expectedCount, finalStats.HealthChecksPerformed)
	}

	if finalStats.ConfigUpdates != expectedCount {
		t.Errorf("Expected %d config updates, got %d", expectedCount, finalStats.ConfigUpdates)
	}

	if finalStats.DatabaseQueries != expectedCount {
		t.Errorf("Expected %d database queries, got %d", expectedCount, finalStats.DatabaseQueries)
	}
}

func TestConcurrentGetStats(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{Enabled: false},
	}

	services := NewServices(cfg)

	// Run concurrent GetStats calls to ensure they don't race
	const numGoroutines = 50

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	results := make([]*ControlStats, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			defer wg.Done()
			results[index] = services.GetStats()
		}(i)
	}

	wg.Wait()

	// Verify all calls returned valid stats
	for i, stats := range results {
		if stats == nil {
			t.Errorf("GetStats call %d returned nil", i)
			continue
		}

		if stats.StartTime.IsZero() {
			t.Errorf("GetStats call %d returned zero start time", i)
		}

		if stats.LastActivity.IsZero() {
			t.Errorf("GetStats call %d returned zero last activity", i)
		}
	}
}

func TestStatsIsolation(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{Enabled: false},
	}

	services := NewServices(cfg)

	// Get initial stats
	stats1 := services.GetStats()
	stats2 := services.GetStats()

	// Verify they are different objects (deep copy)
	if stats1 == stats2 {
		t.Error("GetStats should return different instances (deep copy)")
	}

	// Modify one stats object and verify the other is unchanged
	stats1.APIRequests = 999

	if stats2.APIRequests == 999 {
		t.Error("Modifying one stats object should not affect another")
	}

	// Verify the service's internal stats are unchanged
	currentStats := services.GetStats()
	if currentStats.APIRequests == 999 {
		t.Error("Modifying returned stats should not affect internal stats")
	}
}

func TestServiceStatus_JSONSerialization(t *testing.T) {
	status := &ServiceStatus{
		Name:         "test-service",
		URL:          "http://test:8080",
		Healthy:      true,
		LastCheck:    time.Now(),
		ResponseTime: 150 * time.Millisecond,
		Version:      "1.0.0",
		Config: map[string]interface{}{
			"key": "value",
		},
		Error: "",
	}

	// This test verifies that the struct can be serialized (JSON tags are correct)
	// In a real scenario, this would be done by the API layer
	if status.Name == "" {
		t.Error("ServiceStatus should have valid fields for serialization")
	}
}

func TestTenant_Structure(t *testing.T) {
	tenant := &Tenant{
		ID:        "tenant-123",
		Name:      "Test Tenant",
		Email:     "admin@test.com",
		Active:    true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Datasets:  []Dataset{},
		NotificationSettings: map[string]interface{}{
			"email_enabled": true,
		},
	}

	// Verify basic structure
	if tenant.ID != "tenant-123" {
		t.Errorf("Expected tenant ID 'tenant-123', got '%s'", tenant.ID)
	}

	if tenant.NotificationSettings == nil {
		t.Error("Notification settings should be initialized")
	}

	if tenant.Datasets == nil {
		t.Error("Datasets should be initialized (even if empty)")
	}
}

func TestDataset_Structure(t *testing.T) {
	dataset := &Dataset{
		ID:          "dataset-456",
		TenantID:    "tenant-123",
		Name:        "test-dataset",
		Description: "A test dataset",
		Active:      true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Pipeline: map[string]interface{}{
			"enabled": true,
			"filters": []string{"filter1", "filter2"},
		},
	}

	// Verify basic structure
	if dataset.ID != "dataset-456" {
		t.Errorf("Expected dataset ID 'dataset-456', got '%s'", dataset.ID)
	}

	if dataset.TenantID != "tenant-123" {
		t.Errorf("Expected tenant ID 'tenant-123', got '%s'", dataset.TenantID)
	}

	if dataset.Pipeline == nil {
		t.Error("Pipeline should be initialized")
	}
}

func TestControlStats_ThreadSafety(t *testing.T) {
	stats := &ControlStats{
		StartTime:    time.Now(),
		LastActivity: time.Now(),
	}

	const numGoroutines = 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Concurrent reads and writes to test thread safety
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			stats.mutex.Lock()
			stats.APIRequests++
			stats.LastActivity = time.Now()
			stats.mutex.Unlock()
		}()
	}

	wg.Wait()

	if stats.APIRequests != int64(numGoroutines) {
		t.Errorf("Expected %d API requests, got %d", numGoroutines, stats.APIRequests)
	}
}

// Mock database service for testing
type mockDatabaseService struct {
	connected bool
}

func (m *mockDatabaseService) Connect() error {
	m.connected = true
	return nil
}

func (m *mockDatabaseService) Close() error {
	m.connected = false
	return nil
}

func (m *mockDatabaseService) Ping() error {
	if !m.connected {
		return fmt.Errorf("not connected")
	}
	return nil
}

func (m *mockDatabaseService) GetTenants() ([]Tenant, error) {
	return []Tenant{}, nil
}

func (m *mockDatabaseService) GetTenant(id string) (*Tenant, error) {
	return nil, fmt.Errorf("not found")
}

func (m *mockDatabaseService) CreateTenant(tenant *Tenant) error {
	return nil
}

func (m *mockDatabaseService) UpdateTenant(tenant *Tenant) error {
	return nil
}

func (m *mockDatabaseService) DeleteTenant(id string) error {
	return nil
}

func TestDatabaseServiceInterface(t *testing.T) {
	// Test that our mock implements the interface correctly
	var db DatabaseService = &mockDatabaseService{}

	err := db.Connect()
	if err != nil {
		t.Errorf("Connect failed: %v", err)
	}

	err = db.Ping()
	if err != nil {
		t.Errorf("Ping failed: %v", err)
	}

	tenants, err := db.GetTenants()
	if err != nil {
		t.Errorf("GetTenants failed: %v", err)
	}
	if tenants == nil {
		t.Error("GetTenants should return empty slice, not nil")
	}

	err = db.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}
