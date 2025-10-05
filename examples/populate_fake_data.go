package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/n0needt0/bytefreezer-control/storage"
)

// This script populates the database with fake data that matches the dev mode data
// used in receiver, piper, and packer services

func main() {
	// Check for database URI argument
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run populate_fake_data.go <postgres_uri>")
		fmt.Println("Example: go run populate_fake_data.go 'postgres://user:pass@localhost:5432/bytefreezer?sslmode=disable'")
		os.Exit(1)
	}

	dbURI := os.Args[1]
	ctx := context.Background()

	// Initialize storage
	store, err := storage.NewPostgreSQLStorage(storage.Config{
		Type:           "postgresql",
		URI:            dbURI,
		TimeoutSeconds: 30,
		SSLMode:        "disable",
	})
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}

	// Run migrations
	if err := store.Migrate(ctx); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	fmt.Println("✅ Migrations completed")

	// Create development account
	account := &storage.Account{
		ID:    "dev-account",
		Name:  "Development Account",
		Email: "dev@bytefreezer.local",
		Config: storage.AccountConfig{
			Tier:        "free",
			MaxTenants:  10,
			MaxDatasets: 50,
			CustomFields: map[string]interface{}{
				"environment":          "development",
				"allow_fake_data":      true,
				"retention_days":       30,
				"max_ingestion_rate_mb": 100,
			},
		},
	}

	if err := store.CreateAccount(ctx, account); err != nil {
		log.Fatalf("Failed to create account: %v", err)
	}
	fmt.Printf("✅ Created account: %s (%s)\n", account.Name, account.ID)

	// Tenant configurations matching receiver, piper, packer fake data
	tenants := []struct {
		ID          string
		Name        string
		Description string
		Config      map[string]interface{}
		Datasets    []struct {
			ID          string
			Name        string
			Description string
			Config      map[string]interface{}
		}
	}{
		{
			ID:          "customer-1",
			Name:        "Customer 1 - Production eBPF Data",
			Description: "Primary customer with eBPF and sFlow data streams",
			Config: map[string]interface{}{
				"bearer_token":   "eb4ba9e3236eaefae736495bd79f0d6de753bd7b6547b184ac0c439ff76982cf",
				"organization":   "Customer 1 Corp",
				"subscription":   map[string]interface{}{"tier": "enterprise", "active": true},
				"notifications":  map[string]interface{}{"email": "alerts@customer1.com", "slack_enabled": true},
				"security":       map[string]interface{}{"ip_whitelist": []string{"10.0.0.0/8"}},
			},
			Datasets: []struct {
				ID          string
				Name        string
				Description string
				Config      map[string]interface{}
			}{
				{
					ID:          "ebpf-data",
					Name:        "eBPF Observability Data",
					Description: "Real-time eBPF network observability data",
					Config: map[string]interface{}{
						"source": map[string]interface{}{
							"type":      "webhook",
							"format":    "ndjson",
							"data_hint": "ebpf",
						},
						"processing": map[string]interface{}{
							"enable_raw_storage":   true,
							"partitioning_scheme":  "date",
							"compression":          "gzip",
							"enable_deduplication": false,
						},
						"destination": map[string]interface{}{
							"type":        "s3",
							"bucket_name": "bytefreezer-data",
							"prefix":      "customer-1/ebpf-data/",
							"region":      "us-east-1",
						},
					},
				},
				{
					ID:          "sflow-data",
					Name:        "sFlow Network Monitoring Data",
					Description: "sFlow network traffic analysis data",
					Config: map[string]interface{}{
						"source": map[string]interface{}{
							"type":      "webhook",
							"format":    "sflow",
							"data_hint": "sflow",
						},
						"processing": map[string]interface{}{
							"enable_raw_storage":  true,
							"partitioning_scheme": "date",
							"compression":         "gzip",
						},
						"destination": map[string]interface{}{
							"type":        "s3",
							"bucket_name": "bytefreezer-data",
							"prefix":      "customer-1/sflow-data/",
							"region":      "us-east-1",
						},
					},
				},
			},
		},
		{
			ID:          "tenant-001",
			Name:        "Development Tenant Alpha",
			Description: "First development tenant for testing",
			Config: map[string]interface{}{
				"bearer_token":  "bearer-token-tenant-001-dev",
				"organization":  "Dev Team Alpha",
				"subscription":  map[string]interface{}{"tier": "pro", "active": true},
				"notifications": map[string]interface{}{"email": "dev-alpha@bytefreezer.local"},
			},
			Datasets: []struct {
				ID          string
				Name        string
				Description string
				Config      map[string]interface{}
			}{
				{
					ID:          "dataset-001",
					Name:        "Web Analytics",
					Description: "Website analytics and user behavior data",
					Config: map[string]interface{}{
						"source":      map[string]interface{}{"type": "webhook", "format": "json"},
						"processing":  map[string]interface{}{"enable_raw_storage": true, "partitioning_scheme": "date"},
						"destination": map[string]interface{}{"type": "s3", "bucket_name": "bytefreezer-data", "prefix": "tenant-001/web-analytics/"},
					},
				},
				{
					ID:          "dataset-004",
					Name:        "User Events",
					Description: "User interaction events",
					Config: map[string]interface{}{
						"source":      map[string]interface{}{"type": "webhook", "format": "ndjson"},
						"processing":  map[string]interface{}{"enable_raw_storage": true, "partitioning_scheme": "date"},
						"destination": map[string]interface{}{"type": "s3", "bucket_name": "bytefreezer-data", "prefix": "tenant-001/user-events/"},
					},
				},
				{
					ID:          "dataset-007",
					Name:        "Sales Data",
					Description: "E-commerce sales transactions",
					Config: map[string]interface{}{
						"source":      map[string]interface{}{"type": "webhook", "format": "json"},
						"processing":  map[string]interface{}{"enable_raw_storage": true, "partitioning_scheme": "date"},
						"destination": map[string]interface{}{"type": "s3", "bucket_name": "bytefreezer-data", "prefix": "tenant-001/sales-data/"},
					},
				},
			},
		},
		{
			ID:          "tenant-002",
			Name:        "Development Tenant Beta",
			Description: "Second development tenant for testing",
			Config: map[string]interface{}{
				"bearer_token":  "bearer-token-tenant-002-dev",
				"organization":  "Dev Team Beta",
				"subscription":  map[string]interface{}{"tier": "pro", "active": true},
				"notifications": map[string]interface{}{"email": "dev-beta@bytefreezer.local"},
			},
			Datasets: []struct {
				ID          string
				Name        string
				Description string
				Config      map[string]interface{}
			}{
				{
					ID:          "dataset-002",
					Name:        "User Events",
					Description: "User interaction events",
					Config: map[string]interface{}{
						"source":      map[string]interface{}{"type": "webhook", "format": "ndjson"},
						"processing":  map[string]interface{}{"enable_raw_storage": true, "partitioning_scheme": "date"},
						"destination": map[string]interface{}{"type": "s3", "bucket_name": "bytefreezer-data", "prefix": "tenant-002/user-events/"},
					},
				},
				{
					ID:          "dataset-005",
					Name:        "Web Analytics",
					Description: "Website analytics and user behavior data",
					Config: map[string]interface{}{
						"source":      map[string]interface{}{"type": "webhook", "format": "json"},
						"processing":  map[string]interface{}{"enable_raw_storage": true, "partitioning_scheme": "date"},
						"destination": map[string]interface{}{"type": "s3", "bucket_name": "bytefreezer-data", "prefix": "tenant-002/web-analytics/"},
					},
				},
				{
					ID:          "dataset-008",
					Name:        "Sales Data",
					Description: "E-commerce sales transactions",
					Config: map[string]interface{}{
						"source":      map[string]interface{}{"type": "webhook", "format": "json"},
						"processing":  map[string]interface{}{"enable_raw_storage": true, "partitioning_scheme": "date"},
						"destination": map[string]interface{}{"type": "s3", "bucket_name": "bytefreezer-data", "prefix": "tenant-002/sales-data/"},
					},
				},
			},
		},
		{
			ID:          "tenant-003",
			Name:        "Development Tenant Gamma",
			Description: "Third development tenant for testing",
			Config: map[string]interface{}{
				"bearer_token":  "bearer-token-tenant-003-dev",
				"organization":  "Dev Team Gamma",
				"subscription":  map[string]interface{}{"tier": "free", "active": true},
				"notifications": map[string]interface{}{"email": "dev-gamma@bytefreezer.local"},
			},
			Datasets: []struct {
				ID          string
				Name        string
				Description string
				Config      map[string]interface{}
			}{
				{
					ID:          "dataset-003",
					Name:        "Sales Data",
					Description: "E-commerce sales transactions",
					Config: map[string]interface{}{
						"source":      map[string]interface{}{"type": "webhook", "format": "json"},
						"processing":  map[string]interface{}{"enable_raw_storage": true, "partitioning_scheme": "date"},
						"destination": map[string]interface{}{"type": "s3", "bucket_name": "bytefreezer-data", "prefix": "tenant-003/sales-data/"},
					},
				},
				{
					ID:          "dataset-006",
					Name:        "Web Analytics",
					Description: "Website analytics and user behavior data",
					Config: map[string]interface{}{
						"source":      map[string]interface{}{"type": "webhook", "format": "json"},
						"processing":  map[string]interface{}{"enable_raw_storage": true, "partitioning_scheme": "date"},
						"destination": map[string]interface{}{"type": "s3", "bucket_name": "bytefreezer-data", "prefix": "tenant-003/web-analytics/"},
					},
				},
				{
					ID:          "dataset-009",
					Name:        "User Events",
					Description: "User interaction events",
					Config: map[string]interface{}{
						"source":      map[string]interface{}{"type": "webhook", "format": "ndjson"},
						"processing":  map[string]interface{}{"enable_raw_storage": true, "partitioning_scheme": "date"},
						"destination": map[string]interface{}{"type": "s3", "bucket_name": "bytefreezer-data", "prefix": "tenant-003/user-events/"},
					},
				},
			},
		},
	}

	// Create tenants and datasets
	for _, t := range tenants {
		tenant := &storage.Tenant{
			ID:          t.ID,
			AccountID:   createdAccount.ID,
			Name:        t.Name,
			Description: t.Description,
			Config:      t.Config,
		}

		createdTenant, err := store.CreateTenant(ctx, tenant)
		if err != nil {
			log.Fatalf("Failed to create tenant %s: %v", t.ID, err)
		}
		fmt.Printf("  ✅ Created tenant: %s (%s)\n", createdTenant.Name, createdTenant.ID)

		// Create datasets for this tenant
		for _, d := range t.Datasets {
			dataset := &storage.Dataset{
				ID:          d.ID,
				AccountID:   createdAccount.ID,
				TenantID:    createdTenant.ID,
				Name:        d.Name,
				Description: d.Description,
				Config:      d.Config,
			}

			createdDataset, err := store.CreateDataset(ctx, dataset)
			if err != nil {
				log.Fatalf("Failed to create dataset %s: %v", d.ID, err)
			}
			fmt.Printf("    ✅ Created dataset: %s (%s)\n", createdDataset.Name, createdDataset.ID)
		}
	}

	// Print summary
	fmt.Println("\n📊 Database Population Summary:")
	fmt.Printf("  Account: %s (%s)\n", createdAccount.Name, createdAccount.ID)
	fmt.Printf("  Tenants: %d\n", len(tenants))

	totalDatasets := 0
	for _, t := range tenants {
		totalDatasets += len(t.Datasets)
	}
	fmt.Printf("  Datasets: %d\n", totalDatasets)

	// Print example API usage
	fmt.Println("\n🔗 Control Service API Usage:")
	fmt.Printf("  List tenants:   GET http://localhost:8080/api/v2/accounts/%s/tenants\n", createdAccount.ID)
	fmt.Printf("  Get tenant:     GET http://localhost:8080/api/v2/accounts/%s/tenants/customer-1\n", createdAccount.ID)
	fmt.Printf("  List datasets:  GET http://localhost:8080/api/v2/accounts/%s/tenants/customer-1/datasets\n", createdAccount.ID)

	// Print example service configuration
	fmt.Println("\n⚙️  Service Configuration:")
	fmt.Println("control_service:")
	fmt.Println("  enabled: true")
	fmt.Println("  base_url: http://localhost:8080")
	fmt.Printf("  account_id: %s\n", createdAccount.ID)
	fmt.Println("  api_key: your-api-key")

	// Export data for reference
	summary := map[string]interface{}{
		"account":  createdAccount,
		"tenants":  tenants,
		"api_base": "http://localhost:8080/api/v2",
	}

	summaryJSON, _ := json.MarshalIndent(summary, "", "  ")
	if err := os.WriteFile("fake_data_summary.json", summaryJSON, 0644); err != nil {
		log.Printf("Warning: Failed to write summary file: %v", err)
	} else {
		fmt.Println("\n💾 Summary exported to: fake_data_summary.json")
	}

	fmt.Println("\n✅ Database population complete!")
}
