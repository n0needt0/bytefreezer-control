#!/bin/bash

# Development helper script for ByteFreezer Control

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_DIR"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Helper functions
log() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

# Check if docker and docker-compose are available
check_docker() {
    if ! command -v docker &> /dev/null; then
        error "Docker is not installed or not in PATH"
    fi
    
    if ! command -v docker-compose &> /dev/null; then
        error "Docker Compose is not installed or not in PATH"
    fi
}

# Setup development environment
setup() {
    log "Setting up development environment..."
    
    # Create necessary directories
    mkdir -p logs monitoring/grafana/{provisioning,dashboards} otel sql
    
    # Create basic monitoring configuration if it doesn't exist
    if [[ ! -f monitoring/prometheus.yml ]]; then
        cat > monitoring/prometheus.yml << EOF
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'bytefreezer-control'
    static_configs:
      - targets: ['bytefreezer-control:8080']
    metrics_path: '/metrics'
    scrape_interval: 5s

  - job_name: 'otel-collector'
    static_configs:
      - targets: ['otel-collector:8889']
EOF
    fi
    
    # Create OTEL collector config if it doesn't exist
    if [[ ! -f otel/otel-collector-config.yml ]]; then
        cat > otel/otel-collector-config.yml << EOF
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318

processors:
  batch:

exporters:
  prometheus:
    endpoint: "0.0.0.0:8889"
  logging:
    loglevel: debug

service:
  pipelines:
    metrics:
      receivers: [otlp]
      processors: [batch]
      exporters: [prometheus, logging]
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [logging]
EOF
    fi
    
    # Create database init script if it doesn't exist
    if [[ ! -f sql/init.sql ]]; then
        cat > sql/init.sql << EOF
-- ByteFreezer Control Database Schema

CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    notification_settings JSONB DEFAULT '{}'
);

CREATE TABLE IF NOT EXISTS datasets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    pipeline JSONB DEFAULT '{}'
);

CREATE INDEX idx_tenants_email ON tenants(email);
CREATE INDEX idx_datasets_tenant_id ON datasets(tenant_id);
CREATE INDEX idx_datasets_active ON datasets(active);
EOF
    fi
    
    log "Development environment setup complete!"
}

# Start development environment
start() {
    log "Starting development environment..."
    check_docker
    setup
    
    docker-compose up -d
    
    log "Waiting for services to be healthy..."
    sleep 10
    
    # Wait for control service to be ready
    for i in {1..30}; do
        if curl -s http://localhost:8080/api/v1/health > /dev/null; then
            log "ByteFreezer Control is ready!"
            break
        fi
        sleep 2
    done
    
    log "Development environment is running!"
    log "  - ByteFreezer Control: http://localhost:8080"
    log "  - API Docs: http://localhost:8080/v1/docs"
    log "  - Grafana: http://localhost:3000 (admin/admin)"
    log "  - Prometheus: http://localhost:9090"
    log "  - PostgreSQL: localhost:5432 (bytefreezer/password)"
}

# Stop development environment
stop() {
    log "Stopping development environment..."
    check_docker
    docker-compose down
    log "Development environment stopped!"
}

# Clean up development environment
clean() {
    log "Cleaning up development environment..."
    check_docker
    docker-compose down -v --remove-orphans
    docker system prune -f
    log "Development environment cleaned!"
}

# Show logs
logs() {
    check_docker
    docker-compose logs -f "$@"
}

# Run tests
test() {
    log "Running tests..."
    make test
}

# Build and run locally
run() {
    log "Building and running locally..."
    make build
    ./bytefreezer-control "$@"
}

# Show status
status() {
    check_docker
    docker-compose ps
}

# Show help
help() {
    echo "ByteFreezer Control Development Helper"
    echo ""
    echo "Usage: $0 <command> [options]"
    echo ""
    echo "Commands:"
    echo "  setup     Setup development environment"
    echo "  start     Start development environment with Docker Compose"
    echo "  stop      Stop development environment"
    echo "  clean     Clean up development environment (removes volumes)"
    echo "  logs      Show logs (use -f to follow, specify service name)"
    echo "  test      Run tests"
    echo "  run       Build and run locally (outside Docker)"
    echo "  status    Show status of Docker services"
    echo "  help      Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 start"
    echo "  $0 logs -f bytefreezer-control"
    echo "  $0 run --dev"
    echo ""
}

# Main command dispatch
case "${1:-help}" in
    setup)
        setup
        ;;
    start)
        start
        ;;
    stop)
        stop
        ;;
    clean)
        clean
        ;;
    logs)
        shift
        logs "$@"
        ;;
    test)
        test
        ;;
    run)
        shift
        run "$@"
        ;;
    status)
        status
        ;;
    help)
        help
        ;;
    *)
        error "Unknown command: $1. Use '$0 help' for available commands."
        ;;
esac