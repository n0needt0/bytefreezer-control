package storage

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
)

// ExampleMigration demonstrates how to use the migration system
// This would typically be in a separate cmd/ directory
func ExampleMigration() {
	var (
		command       = flag.String("command", "status", "Migration command: status, migrate, rollback, reset")
		freshInstall  = flag.Bool("fresh-install", false, "Fresh install (drop and recreate)")
		resetData     = flag.Bool("reset-data", false, "Reset data only (preserve schema)")
		targetVersion = flag.Int("target-version", 0, "Target migration version (0 = latest)")
		dryRun        = flag.Bool("dry-run", false, "Show what would be done")
		force         = flag.Bool("force", false, "Force migration despite conflicts")
	)
	flag.Parse()

	// Simple config for demonstration
	config := Config{
		Type:     "postgresql",
		URI:      "postgres://bytefreezer:bytefreezer123@localhost:5432/bytefreezer_control_example?sslmode=disable",
		Database: "bytefreezer_control_example",
	}

	// Create storage
	store, err := NewStorage(config)
	if err != nil {
		log.Fatalf("Failed to create storage: %v", err)
	}
	defer store.Close()

	// Get migrator
	migrator := store.GetMigrator()
	if migrator == nil {
		log.Fatal("Migrator not available for this storage backend")
	}

	ctx := context.Background()

	switch *command {
	case "status":
		status, err := migrator.GetStatus(ctx)
		if err != nil {
			log.Fatalf("Failed to get migration status: %v", err)
		}
		printStatus(status)

	case "migrate":
		opts := MigrationOptions{
			FreshInstall:  *freshInstall,
			ResetData:     *resetData,
			TargetVersion: *targetVersion,
			DryRun:        *dryRun,
			Force:         *force,
		}

		if *dryRun {
			fmt.Println("=== DRY RUN MODE ===")
		}

		err := migrator.Migrate(ctx, opts)
		if err != nil {
			log.Fatalf("Migration failed: %v", err)
		}

		if !*dryRun {
			fmt.Println("Migration completed successfully!")
		}

	case "rollback":
		if *targetVersion == 0 {
			log.Fatal("Target version must be specified for rollback")
		}

		fmt.Printf("Rolling back to version %d...\n", *targetVersion)
		err := migrator.Rollback(ctx, *targetVersion)
		if err != nil {
			log.Fatalf("Rollback failed: %v", err)
		}
		fmt.Println("Rollback completed successfully!")

	case "reset":
		fmt.Println("Resetting database (fresh install)...")
		err := migrator.Reset(ctx)
		if err != nil {
			log.Fatalf("Reset failed: %v", err)
		}
		fmt.Println("Database reset completed!")

	default:
		fmt.Printf("Unknown command: %s\n", *command)
		fmt.Println("Available commands: status, migrate, rollback, reset")
		os.Exit(1)
	}
}

func printStatus(status *MigrationStatus) {
	fmt.Printf("=== Database Migration Status ===\n")
	fmt.Printf("Current Version: %d\n", status.CurrentVersion)
	fmt.Printf("Total Migrations: %d\n", len(status.Migrations))
	fmt.Printf("Applied: %d\n", len(status.Applied))
	fmt.Printf("Pending: %d\n", len(status.Pending))
	fmt.Println()

	if len(status.Applied) > 0 {
		fmt.Println("Applied Migrations:")
		for _, migration := range status.Applied {
			fmt.Printf("  ✓ %03d: %s (%s)\n", 
				migration.Version, 
				migration.Name, 
				migration.AppliedAt.Format("2006-01-02 15:04:05"))
		}
		fmt.Println()
	}

	if len(status.Pending) > 0 {
		fmt.Println("Pending Migrations:")
		for _, migration := range status.Pending {
			fmt.Printf("  ○ %03d: %s\n", migration.Version, migration.Name)
			fmt.Printf("      %s\n", migration.Description)
		}
		fmt.Println()
	}

	if len(status.Pending) == 0 {
		fmt.Println("✅ Database is up to date!")
	}
}

/*
Example usage:

# Check migration status
go run migrate_example.go -command=status

# Apply all pending migrations
go run migrate_example.go -command=migrate

# Apply migrations up to version 2
go run migrate_example.go -command=migrate -target-version=2

# Dry run to see what would be applied
go run migrate_example.go -command=migrate -dry-run

# Fresh install (drop and recreate all tables)
go run migrate_example.go -command=migrate -fresh-install

# Reset data only (preserve schema)
go run migrate_example.go -command=migrate -reset-data

# Rollback to version 1
go run migrate_example.go -command=rollback -target-version=1

# Complete reset
go run migrate_example.go -command=reset

Expected output:

=== Database Migration Status ===
Current Version: 0
Total Migrations: 3
Applied: 0
Pending: 3

Pending Migrations:
  ○ 001: initial_schema
      Create initial tenants and datasets tables
  ○ 002: add_tenant_settings
      Add additional tenant settings and metadata
  ○ 003: add_dataset_metrics
      Add dataset metrics and processing status

After running migrate:

=== Database Migration Status ===
Current Version: 3
Total Migrations: 3
Applied: 3
Pending: 0

Applied Migrations:
  ✓ 001: initial_schema (2024-01-15 10:30:45)
  ✓ 002: add_tenant_settings (2024-01-15 10:30:45)
  ✓ 003: add_dataset_metrics (2024-01-15 10:30:45)

✅ Database is up to date!
*/