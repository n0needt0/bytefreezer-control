package storage

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestPostgreSQLStorage(t *testing.T) {
	// Create PostgreSQL storage for testing
	config := Config{
		Type:     "postgresql",
		URI:      "postgres://bytefreezer:bytefreezer123@localhost:5432/bytefreezer_control_test?sslmode=disable",
		Database: "bytefreezer_control_test",
	}

	store, err := NewStorage(config)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Migrate schema
	err = store.Migrate(ctx)
	if err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	// Health check
	err = store.HealthCheck(ctx)
	if err != nil {
		t.Fatalf("Health check failed: %v", err)
	}

	// Create a tenant
	tenant := &Tenant{
		ID:    uuid.New().String(),
		Name:  "Test Company",
		Email: "test@example.com",
		Active: true,
		Config: TenantConfig{
			Organization: OrganizationConfig{
				Name: "Test Company",
				Size: "small",
			},
			Subscription: SubscriptionConfig{
				Tier:        "basic",
				MaxDatasets: 10,
			},
		},
	}

	err = store.CreateTenant(ctx, tenant)
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}

	// Get tenant by ID
	retrieved, err := store.GetTenant(ctx, tenant.ID)
	if err != nil {
		t.Fatalf("Failed to get tenant: %v", err)
	}

	if retrieved.Name != tenant.Name {
		t.Errorf("Expected name %s, got %s", tenant.Name, retrieved.Name)
	}

	// Get tenant by email
	byEmail, err := store.GetTenantByEmail(ctx, tenant.Email)
	if err != nil {
		t.Fatalf("Failed to get tenant by email: %v", err)
	}

	if byEmail.ID != tenant.ID {
		t.Errorf("Expected ID %s, got %s", tenant.ID, byEmail.ID)
	}

	// Create a dataset
	dataset := &Dataset{
		ID:          uuid.New().String(),
		TenantID:    tenant.ID,
		Name:        "Test Dataset",
		Description: "A test dataset",
		Active:      true,
		Config: DatasetConfig{
			Source: SourceConfig{
				Type: "api",
			},
			Destination: DestinationConfig{
				Type: "s3",
			},
			Transform: TransformConfig{
				Enabled: true,
				Rules: []TransformRule{
					{Name: "validate", Type: "field_map", Enabled: true},
					{Name: "clean", Type: "regex", Enabled: true},
				},
			},
		},
	}

	err = store.CreateDataset(ctx, dataset)
	if err != nil {
		t.Fatalf("Failed to create dataset: %v", err)
	}

	// Get dataset
	retrievedDataset, err := store.GetDataset(ctx, tenant.ID, dataset.ID)
	if err != nil {
		t.Fatalf("Failed to get dataset: %v", err)
	}

	if retrievedDataset.Name != dataset.Name {
		t.Errorf("Expected dataset name %s, got %s", dataset.Name, retrievedDataset.Name)
	}

	// List datasets
	datasets, err := store.ListDatasets(ctx, tenant.ID, ListOptions{Limit: 10})
	if err != nil {
		t.Fatalf("Failed to list datasets: %v", err)
	}

	if len(datasets.Items) != 1 {
		t.Errorf("Expected 1 dataset, got %d", len(datasets.Items))
	}

	// List tenants
	tenants, err := store.ListTenants(ctx, ListOptions{Limit: 10})
	if err != nil {
		t.Fatalf("Failed to list tenants: %v", err)
	}

	if len(tenants.Items) != 1 {
		t.Errorf("Expected 1 tenant, got %d", len(tenants.Items))
	}

	t.Logf("Test completed successfully!")
	t.Logf("Tenant: %s (%s)", retrieved.Name, retrieved.Email)
	t.Logf("Dataset: %s - %s", retrievedDataset.Name, retrievedDataset.Description)
}

// Example usage patterns
func ExampleNewStorage() {
	// PostgreSQL (document-oriented storage with JSONB)
	postgresConfig := Config{
		Type:     "postgresql",
		URI:      "postgres://bytefreezer:bytefreezer123@localhost:5432/bytefreezer_control?sslmode=disable",
		Database: "bytefreezer_control",
	}

	// Create storage
	store, err := NewStorage(postgresConfig)
	if err != nil {
		panic(err)
	}
	defer store.Close()

	// Initialize schema and indexes
	ctx := context.Background()
	store.Migrate(ctx)

	// Create tenant with rich configuration
	tenant := &Tenant{
		ID:     uuid.New().String(),
		Name:   "Example Corp",
		Email:  "admin@example.com",
		Active: true,
		Config: TenantConfig{
			Organization: OrganizationConfig{
				Name: "Example Corporation",
				Size: "medium",
			},
			Subscription: SubscriptionConfig{
				Tier:        "enterprise",
				MaxDatasets: 100,
			},
		},
	}

	store.CreateTenant(ctx, tenant)

	// Query using document fields
	enterpriseTenants, _ := store.FindTenantsBySubscriptionTier(ctx, "enterprise")
	_ = enterpriseTenants
}