# ByteFreezer Control

## Project Description
This project provides a control plane API for managing the entire ByteFreezer ecosystem. It monitors and manages all ByteFreezer services including receiver, proxy, SOC, and packer components.

## Key Features
- **Health Monitoring**: Monitor health status of all ecosystem services
- **Configuration Management**: Centralized configuration management
- **Tenant Management**: Database-backed tenant administration
- **Observability**: OpenTelemetry integration for metrics and tracing
- **Authentication**: JWT-based authentication with admin controls
- **Rate Limiting**: Configurable request rate limiting

## API Endpoints
- Health check: `/api/v2/health`
- Configuration: `/api/v2/config`
- Service status: `/api/v2/services/status`
- Tenant management: `/api/v2/tenants`

## Development Commands
```bash
# Build the service
go build .

# Run with config validation
./bytefreezer-control --validate-config

# Run the service
./bytefreezer-control --config config.yaml
```