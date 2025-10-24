package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
	"github.com/n0needt0/go-goodies/log"
)

func main() {
	// Get database connection info from environment
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}
	dbPassword := os.Getenv("DB_PASSWORD")
	if dbPassword == "" {
		log.Fatal("DB_PASSWORD environment variable is required")
	}

	// Build connection string
	connStr := fmt.Sprintf("host=%s port=5432 user=bytefreezer password=%s dbname=bytefreezer sslmode=disable",
		dbHost, dbPassword)

	// Connect to database
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Check how many records will be deleted
	var count int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM control_audit_log WHERE action = 'dataset_tested'").Scan(&count)
	if err != nil {
		log.Fatalf("Failed to count records: %v", err)
	}

	log.Infof("Found %d dataset_tested audit log entries", count)

	if count == 0 {
		log.Info("No dataset_tested audit log entries to delete")
		return
	}

	// Delete the records
	result, err := db.ExecContext(ctx, "DELETE FROM control_audit_log WHERE action = 'dataset_tested'")
	if err != nil {
		log.Fatalf("Failed to delete records: %v", err)
	}

	rowsAffected, _ := result.RowsAffected()
	log.Infof("Successfully deleted %d dataset_tested audit log entries", rowsAffected)

	// Verify deletion
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM control_audit_log WHERE action = 'dataset_tested'").Scan(&count)
	if err != nil {
		log.Fatalf("Failed to verify deletion: %v", err)
	}

	if count == 0 {
		log.Info("Verification successful: No dataset_tested entries remain in the database")
	} else {
		log.Warnf("Warning: %d dataset_tested entries still exist in the database", count)
	}
}
