# ByteFreezer Control

Centralized control plane and ecosystem management service for the ByteFreezer platform. This service provides unified monitoring, configuration management, and orchestration for all ByteFreezer components.

## Overview

ByteFreezer Control is the **management layer** in the ByteFreezer four-service architecture:

1. **bytefreezer-receiver**: Raw data ingestion → S3 raw/
2. **bytefreezer-piper**: Data processing pipeline → S3 processed/  
3. **bytefreezer-packer**: Parquet optimization → S3 parquet/
4. **bytefreezer-control** (this service): Ecosystem monitoring and management

### Key Features

- **Ecosystem Health Monitoring** - Real-time health checks for all ByteFreezer services
- **Centralized Configuration Management** - Unified configuration oversight and validation
- **Tenant Management** - Multi-tenant administration and lifecycle management
- **Service Discovery and Orchestration** - Service registration and coordination
- **Authentication and Authorization** - JWT-based security with role-based access control
- **OpenTelemetry Integration** - Comprehensive monitoring and observability
- **Database-Backed Storage** - PostgreSQL for persistent tenant and configuration data
- **RESTful API** - Complete REST API for programmatic management
- **Rate Limiting** - Built-in rate limiting for API protection
- **Housekeeping Automation** - Automated maintenance and health check scheduling

### Control Plane Architecture

ByteFreezer Control acts as the central nervous system for the ByteFreezer ecosystem:

- **Port 8082**: Management API server for control plane operations
- **Service Monitoring**: Continuous health monitoring of all ecosystem services
- **Tenant Isolation**: Complete tenant lifecycle management with strict isolation
- **Configuration Validation**: Centralized validation of service configurations
- **Security Layer**: JWT authentication with admin role management

## Installation

### Docker (Recommended)

Pull and run the latest version:
```bash
# Pull the latest image
docker pull ghcr.io/n0needt0/bytefreezer-control:latest

# Run with default configuration
docker run -p 8082:8082 ghcr.io/n0needt0/bytefreezer-control:latest
```

With custom configuration:
```bash
# Create your config file
wget https://raw.githubusercontent.com/n0needt0/bytefreezer-control/main/config.yaml

# Run with custom config
docker run -p 8082:8082 -v $(pwd)/config.yaml:/config.yaml ghcr.io/n0needt0/bytefreezer-control:latest
```

### Binary Extraction

Extract the binary from the container for direct use:
```bash
# Extract binary
docker run --rm -v $(pwd):/output ghcr.io/n0needt0/bytefreezer-control:latest sh -c "cp /bytefreezer-control /output/"

# Make executable and run
chmod +x bytefreezer-control
./bytefreezer-control --config config.yaml
```

### Production Deployment

**Docker Compose with Database:**
```yaml
version: '3.8'
services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: bytefreezer_control
      POSTGRES_USER: bytefreezer
      POSTGRES_PASSWORD: secure_password
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./init-db.sql:/docker-entrypoint-initdb.d/init-db.sql
    ports:
      - "5432:5432"

  bytefreezer-control:
    image: ghcr.io/n0needt0/bytefreezer-control:latest
    ports:
      - "8082:8082"  # Management API
    volumes:
      - ./control-config.yaml:/config.yaml
      - ./logs:/var/log/bytefreezer-control
    depends_on:
      - postgres
    environment:
      - BYTEFREEZER_CONTROL_DEV=false
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8082/api/v1/health"]
      interval: 30s
      timeout: 10s
      retries: 3

volumes:
  postgres_data:
```

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    ByteFreezer Control                          │
├─────────────────────────────────────────────────────────────────┤
│  Management API (Port 8082)                                    │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐ │
│  │   Health        │  │   Tenants       │  │   Services      │ │
│  │   Monitoring    │  │   Management    │  │   Discovery     │ │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘ │
├─────────────────────────────────────────────────────────────────┤
│  Core Services Layer                                           │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐ │
│  │   Ecosystem     │  │   Database      │  │   Auth &        │ │
│  │   Monitor       │  │   Service       │  │   Rate Limit    │ │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘ │
├─────────────────────────────────────────────────────────────────┤
│  External Integrations                                         │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐ │
│  │   PostgreSQL    │  │   OpenTelemetry │  │   ByteFreezer   │ │
│  │   Database      │  │   Monitoring    │  │   Services      │ │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

The control service follows a layered architecture:
- `api/` - REST API handlers and routing with OpenAPI documentation
- `config/` - Configuration management with environment variable override
- `services/` - Core business services (monitoring, database, tenant management)
- `middleware/` - HTTP middleware for authentication, rate limiting, and logging
- Database integration for persistent tenant and configuration storage
- Service discovery and health monitoring for the ByteFreezer ecosystem

## Configuration

The service is configured via `config.yaml` file. Key configuration sections:

### Basic Service Configuration
```yaml
# Application configuration
app:
  name: "bytefreezer-control"
  version: "1.0.0"

# Logging configuration
logging:
  level: "info"         # debug, info, warn, error
  encoding: "console"   # console, json

# Server configuration
server:
  api_port: 8082        # Management API port
```

### Ecosystem Services Configuration
```yaml
# ByteFreezer services endpoints for monitoring
services:
  receiver:
    url: "http://bytefreezer-receiver:8080"
    health_endpoint: "/health"
    config_endpoint: "/config"
    timeout_seconds: 30
  
  piper:
    url: "http://bytefreezer-piper:8080"
    health_endpoint: "/health"
    config_endpoint: "/config"
    timeout_seconds: 30
    
  packer:
    url: "http://bytefreezer-packer:8080"
    health_endpoint: "/health"
    config_endpoint: "/config"
    timeout_seconds: 30

  proxy:
    url: "http://bytefreezer-proxy:8088"
    health_endpoint: "/api/v1/health"
    config_endpoint: "/api/v1/config"
    timeout_seconds: 30

  soc:
    url: "http://bytefreezer-soc:8089"
    health_endpoint: "/api/v1/health"
    config_endpoint: "/api/v1/config"
    timeout_seconds: 30
```

### Database Configuration
```yaml
# PostgreSQL database for tenant management
database:
  enabled: true
  type: "postgres"
  host: "postgres"
  port: 5432
  database: "bytefreezer_control"
  username: "bytefreezer"
  password: "secure_password"
  ssl_mode: "require"     # disable, require, verify-ca, verify-full
  max_connections: 25
  max_idle_connections: 10
```

### Security Configuration
```yaml
# Authentication and authorization
auth:
  enabled: true
  jwt_secret: "your-256-bit-secret-key-here"
  token_expiry_hours: 24
  admin_users:
    - "admin@company.com"
    - "ops-team@company.com"

# API rate limiting
rate_limit:
  enabled: true
  requests_per_minute: 100
  burst_size: 20
```

### Monitoring Configuration
```yaml
# Automated housekeeping and health checks
housekeeping:
  enabled: true
  interval_seconds: 300  # 5 minutes

# OpenTelemetry integration
otel:
  enabled: true
  endpoint: "http://jaeger:4317"
  service_name: "bytefreezer-control"
  scrape_interval_seconds: 60

# Development mode
dev: false  # Set to true for development features
```

## API Endpoints

### Core Management API

#### Health and Status
- `GET /api/v1/health` - Service health check with uptime and statistics
- `GET /api/v1/config` - Current service configuration (sanitized)
- `GET /api/v1/stats` - Detailed service statistics and metrics

#### Ecosystem Monitoring
- `GET /api/v1/ecosystem/health` - Complete ecosystem health overview
- `GET /api/v1/services` - Status of all monitored ByteFreezer services
- `GET /api/v1/services/{serviceName}` - Detailed status of a specific service
- `POST /api/v1/services/{serviceName}/restart` - Restart a specific service (if supported)

#### Tenant Management
- `GET /api/v1/tenants` - List all tenants with pagination
- `GET /api/v1/tenants/{tenantId}` - Get specific tenant details
- `POST /api/v1/tenants` - Create a new tenant
- `PUT /api/v1/tenants/{tenantId}` - Update tenant information
- `DELETE /api/v1/tenants/{tenantId}` - Delete a tenant and all associated data

#### Authentication
- `POST /api/v1/auth/login` - Authenticate user and receive JWT token
- `POST /api/v1/auth/refresh` - Refresh JWT token
- `GET /api/v1/auth/profile` - Get current user profile (requires authentication)

### API Usage Examples

**Health Check:**
```bash
curl http://localhost:8082/api/v1/health
```

**Ecosystem Overview:**
```bash
curl http://localhost:8082/api/v1/ecosystem/health
```

**Create Tenant:**
```bash
curl -X POST http://localhost:8082/api/v1/tenants \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <your-jwt-token>" \
  -d '{
    "name": "Acme Corporation",
    "email": "admin@acme-corp.com"
  }'
```

**Monitor Specific Service:**
```bash
curl http://localhost:8082/api/v1/services/receiver
```

## Database Schema

### Initial Database Setup

**init-db.sql:**
```sql
-- Create tenants table
CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    notification_settings JSONB DEFAULT '{}'::jsonb
);

-- Create datasets table
CREATE TABLE IF NOT EXISTS datasets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    pipeline JSONB DEFAULT '{}'::jsonb,
    UNIQUE(tenant_id, name)
);

-- Create service_configs table for centralized configuration management
CREATE TABLE IF NOT EXISTS service_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    service_name VARCHAR(100) NOT NULL,
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    config_data JSONB NOT NULL,
    version INTEGER DEFAULT 1,
    active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by VARCHAR(255),
    UNIQUE(service_name, tenant_id)
);

-- Create audit_log table for tracking changes
CREATE TABLE IF NOT EXISTS audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type VARCHAR(100) NOT NULL,
    entity_id UUID NOT NULL,
    action VARCHAR(50) NOT NULL,
    changes JSONB,
    performed_by VARCHAR(255),
    performed_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_tenants_email ON tenants(email);
CREATE INDEX IF NOT EXISTS idx_datasets_tenant_id ON datasets(tenant_id);
CREATE INDEX IF NOT EXISTS idx_service_configs_service_tenant ON service_configs(service_name, tenant_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_entity ON audit_log(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_performed_at ON audit_log(performed_at);

-- Create function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers for updating timestamps
CREATE TRIGGER update_tenants_updated_at BEFORE UPDATE ON tenants
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_datasets_updated_at BEFORE UPDATE ON datasets
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_service_configs_updated_at BEFORE UPDATE ON service_configs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
```

## Building and Running

### Build from Source
```bash
# Install dependencies
go mod tidy

# Build for current platform
go build -o bytefreezer-control .

# Cross-compile for Linux
GOOS=linux GOARCH=amd64 go build -o bytefreezer-control-linux .

# Run with default config
./bytefreezer-control

# Run with custom config
./bytefreezer-control --config production-config.yaml

# Validate configuration
./bytefreezer-control --validate-config

# Show version
./bytefreezer-control --version
```

### Development Setup
```bash
# Install development dependencies
go mod download

# Run with hot reload (requires air)
air

# Run tests
go test -v ./...

# Run tests with coverage
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run linting
golangci-lint run

# Format code
go fmt ./...
```

### Docker Development
```bash
# Build development image
docker build -t bytefreezer-control:dev .

# Run development environment
docker-compose -f docker-compose.dev.yml up -d

# View logs
docker-compose logs -f bytefreezer-control
```

## Monitoring & Observability

The control service provides comprehensive monitoring capabilities:

### Health Monitoring
- **Service Health**: Continuous monitoring of all ByteFreezer services
- **Database Health**: PostgreSQL connection and query monitoring
- **API Health**: Request/response monitoring with detailed statistics
- **Ecosystem Overview**: Unified health dashboard for the entire platform

### Metrics and Statistics
- **API Request Metrics**: Request counts, response times, and error rates
- **Service Discovery Metrics**: Service availability and response times
- **Database Metrics**: Connection pool usage and query performance
- **Tenant Metrics**: Tenant counts, activity levels, and resource usage

### OpenTelemetry Integration
```yaml
otel:
  enabled: true
  endpoint: "http://jaeger:4317"
  service_name: "bytefreezer-control"
  attributes:
    environment: "production"
    version: "1.0.0"
```

### Prometheus Metrics
The service exposes Prometheus-compatible metrics:
- `bytefreezer_control_requests_total` - Total API requests
- `bytefreezer_control_request_duration` - Request duration histogram
- `bytefreezer_control_services_monitored` - Number of monitored services
- `bytefreezer_control_tenants_total` - Total number of tenants
- `bytefreezer_control_database_connections` - Database connection metrics

## Security

### Authentication Flow
1. **Login**: Users authenticate via `/api/v1/auth/login` with credentials
2. **JWT Token**: Service returns signed JWT with user claims and permissions
3. **Authorization**: Protected endpoints validate JWT tokens via middleware
4. **Refresh**: Tokens can be refreshed before expiration

### Role-Based Access Control
- **Admin Users**: Full access to all control plane operations
- **Tenant Users**: Limited access to their own tenant resources
- **Service Accounts**: API access for other ByteFreezer services

### Security Best Practices
- **JWT Secrets**: Use strong, randomly generated secrets for JWT signing
- **Database Encryption**: Enable TLS for database connections
- **API Rate Limiting**: Prevent abuse with configurable rate limits
- **Input Validation**: Comprehensive validation of all API inputs
- **Audit Logging**: Complete audit trail of all administrative actions

## Error Handling & Reliability

### Service Resilience
- **Circuit Breaker**: Automatic circuit breaker for failing service calls
- **Retry Logic**: Configurable retry policies for transient failures
- **Graceful Degradation**: Continue operating even if some services are unavailable
- **Health Check Scheduling**: Configurable health check intervals and timeouts

### Database Reliability
- **Connection Pooling**: Managed database connection pools
- **Transaction Management**: Proper transaction handling for data consistency
- **Migration Support**: Database schema migration capabilities
- **Backup Integration**: Hooks for automated backup processes

### API Reliability
- **Request Validation**: Comprehensive input validation with detailed error messages
- **Error Standardization**: Consistent error response format across all endpoints
- **Timeout Handling**: Configurable timeouts for all external service calls
- **Logging and Debugging**: Detailed logging for troubleshooting and debugging

## Integration with ByteFreezer Ecosystem

### Service Discovery
The control service automatically discovers and monitors all ByteFreezer services:
- Periodic health checks ensure service availability
- Configuration validation prevents deployment of invalid configurations
- Service dependency tracking ensures proper startup ordering

### Tenant Management Integration
- **Receiver Integration**: Tenant configurations synchronized with receiver service
- **Piper Integration**: Processing pipeline configurations managed per tenant
- **Packer Integration**: Optimization settings coordinated across tenants
- **Cross-Service Validation**: Ensures tenant consistency across all services

### Configuration Management
- **Centralized Configuration**: Single source of truth for tenant and service configurations
- **Version Control**: Configuration changes tracked with full audit history
- **Validation Pipeline**: Multi-stage validation before configuration deployment
- **Rollback Capability**: Quick rollback to previous configuration versions

This control service provides the essential management layer for operating ByteFreezer at scale, with comprehensive monitoring, tenant management, and ecosystem orchestration capabilities.