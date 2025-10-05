package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/n0needt0/bytefreezer-control/storage"
)

// Simplified script to populate database with fake data matching dev mode
// Uses minimal config with bearer tokens in custom fields

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run populate_fake_data_simple.go <postgres_uri>")
		fmt.Println("Example: go run populate_fake_data_simple.go 'postgres://postgres:postgres@localhost:5432/bytefreezer?sslmode=disable'")
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
				"environment": "development",
			},
		},
	}

	if err := store.CreateAccount(ctx, account); err != nil {
		log.Fatalf("Failed to create account: %v", err)
	}
	fmt.Printf("✅ Created account: %s (%s)\n", account.Name, account.ID)

	// Tenant data matching fake data
	tenants := []struct {
		ID          string
		Name        string
		BearerToken string
		Datasets    []struct {
			ID   string
			Name string
		}
	}{
		{
			ID:          "customer-1",
			Name:        "Customer 1 - Production eBPF Data",
			BearerToken: "eb4ba9e3236eaefae736495bd79f0d6de753bd7b6547b184ac0c439ff76982cf",
			Datasets: []struct {
				ID   string
				Name string
			}{
				{ID: "ebpf-data", Name: "eBPF Observability Data"},
				{ID: "sflow-data", Name: "sFlow Network Monitoring Data"},
			},
		},
		{
			ID:          "tenant-001",
			Name:        "Development Tenant Alpha",
			BearerToken: "bearer-token-tenant-001-dev",
			Datasets: []struct {
				ID   string
				Name string
			}{
				{ID: "dataset-001", Name: "Web Analytics"},
				{ID: "dataset-004", Name: "User Events"},
				{ID: "dataset-007", Name: "Sales Data"},
			},
		},
		{
			ID:          "tenant-002",
			Name:        "Development Tenant Beta",
			BearerToken: "bearer-token-tenant-002-dev",
			Datasets: []struct {
				ID   string
				Name string
			}{
				{ID: "dataset-002", Name: "User Events"},
				{ID: "dataset-005", Name: "Web Analytics"},
				{ID: "dataset-008", Name: "Sales Data"},
			},
		},
		{
			ID:          "tenant-003",
			Name:        "Development Tenant Gamma",
			BearerToken: "bearer-token-tenant-003-dev",
			Datasets: []struct {
				ID   string
				Name string
			}{
				{ID: "dataset-003", Name: "Sales Data"},
				{ID: "dataset-006", Name: "Web Analytics"},
				{ID: "dataset-009", Name: "User Events"},
			},
		},
	}

	// Create tenants and datasets
	totalDatasets := 0
	for _, t := range tenants {
		tenant := &storage.Tenant{
			ID:          t.ID,
			AccountID:   account.ID,
			Name:        t.Name,
			Description: fmt.Sprintf("Development tenant: %s", t.Name),
			Config: storage.TenantConfig{
				CustomSettings: map[string]interface{}{
					"bearer_token": t.BearerToken,
				},
				Subscription: storage.SubscriptionConfig{
					Tier:       "pro",
					MaxDatasets: 10,
				},
			},
		}

		if err := store.CreateTenant(ctx, tenant); err != nil {
			log.Fatalf("Failed to create tenant %s: %v", t.ID, err)
		}
		fmt.Printf("  ✅ Created tenant: %s (%s)\n", tenant.Name, tenant.ID)

		// Create datasets
		for _, d := range t.Datasets {
			dataset := &storage.Dataset{
				ID:          d.ID,
				TenantID:    tenant.ID,
				Name:        d.Name,
				Description: fmt.Sprintf("%s for %s", d.Name, t.Name),
				Status:      "active",
				Config: storage.DatasetConfig{
					Source: storage.SourceConfig{
						Type:   "webhook",
						Format: "ndjson",
					},
					Processing: storage.ProcessingConfig{
						Enabled: true,
					},
					Destination: storage.DestinationConfig{
						Type:        "s3",
						Compression: "gzip",
						Custom: map[string]interface{}{
							"bucket_name": "bytefreezer-data",
							"prefix":      fmt.Sprintf("%s/%s/", t.ID, d.ID),
							"region":      "us-east-1",
						},
					},
				},
			}

			if err := store.CreateDataset(ctx, dataset); err != nil {
				log.Fatalf("Failed to create dataset %s: %v", d.ID, err)
			}
			fmt.Printf("    ✅ Created dataset: %s (%s)\n", dataset.Name, dataset.ID)
			totalDatasets++
		}
	}

	// Print summary
	fmt.Println("\n📊 Database Population Summary:")
	fmt.Printf("  Account: %s (%s)\n", account.Name, account.ID)
	fmt.Printf("  Tenants: %d\n", len(tenants))
	fmt.Printf("  Datasets: %d\n", totalDatasets)

	// Print service configuration
	fmt.Println("\n⚙️  Service Configuration:")
	fmt.Println("control_service:")
	fmt.Println("  enabled: true")
	fmt.Println("  base_url: http://localhost:8080")
	fmt.Printf("  account_id: %s\n", account.ID)

	// Print API examples
	fmt.Println("\n🔗 Test APIs:")
	fmt.Printf("  curl http://localhost:8080/api/v2/accounts/%s/tenants\n", account.ID)
	fmt.Printf("  curl http://localhost:8080/api/v2/accounts/%s/tenants/customer-1\n", account.ID)

	fmt.Println("\n✅ Database population complete!")
	fmt.Println("   Start services with control_service.enabled=true and they will fetch this data")
}
