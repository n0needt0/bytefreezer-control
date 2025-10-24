#!/bin/bash

# Script to clean up dataset_tested audit log entries
# Usage: ./run_cleanup_dataset_tested.sh

set -e

# Database connection settings
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-bytefreezer}"
DB_USER="${DB_USER:-bytefreezer}"
DB_PASSWORD="${DB_PASSWORD:-}"

echo "Connecting to database at $DB_HOST:$DB_PORT/$DB_NAME as $DB_USER"
echo ""

# Execute the cleanup SQL script
if [ -f "scripts/cleanup_dataset_tested_logs.sql" ]; then
    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f scripts/cleanup_dataset_tested_logs.sql
elif [ -f "cleanup_dataset_tested_logs.sql" ]; then
    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f cleanup_dataset_tested_logs.sql
else
    echo "Error: cleanup_dataset_tested_logs.sql not found"
    exit 1
fi

echo ""
echo "Cleanup completed successfully!"
