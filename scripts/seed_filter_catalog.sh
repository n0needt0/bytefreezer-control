#!/bin/bash

# Seed the piper filter catalog from FILTERS.md documentation
# Usage: ./seed_filter_catalog.sh [config-file] [version]

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG_FILE="${1:-/etc/bytefreezer/control/config.yaml}"
VERSION="${2:-1.0.0}"

echo "Seeding filter catalog..."
echo "Config: $CONFIG_FILE"
echo "Version: $VERSION"
echo ""

cd "$SCRIPT_DIR/.." || exit 1

go run ./scripts/seed_filter_catalog.go \
    -config "$CONFIG_FILE" \
    -version "$VERSION"

if [ $? -eq 0 ]; then
    echo ""
    echo "Filter catalog seeded successfully!"
    echo ""
    echo "You can now query the filters:"
    echo "  curl http://localhost:8080/api/v1/filters"
    echo "  curl http://localhost:8080/api/v1/filters/catalog?format=ui"
    echo "  curl http://localhost:8080/api/v1/filters/catalog?format=ai"
else
    echo ""
    echo "Failed to seed filter catalog!"
    exit 1
fi
