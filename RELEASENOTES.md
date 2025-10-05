# ByteFreezer Control - Release Notes

## Phase 2: Client Library for Service Integration (2025-10-04)

### Major Features

#### Go Client Library
Created `/client` package within bytefreezer-control for easy integration with other ByteFreezer services:

**Package Structure**:
- `client/client.go` - Core HTTP client with proper headers, timeouts, error handling
- `client/types.go` - Type definitions (Account, Tenant, Dataset, request/response structs)
- `client/accounts.go` - Account management methods (Create, Get, List, Update, Delete)
- `client/tenants.go` - Tenant management methods (Create, Get, List, Update, Delete)
- `client/config_helper.go` - Configuration fallback helper with 5-minute caching
- `client/README.md` - Comprehensive usage documentation

**Key Features**:
- **Zero Dependencies**: Uses only Go standard library for maximum portability
- **Type-Safe**: Strongly typed API with proper error handling
- **Context Support**: All methods accept `context.Context` for cancellation/timeout
- **Configuration Fallback**: 3-tier fallback (Control Service API → local config → defaults)
- **Caching**: ConfigHelper caches API responses for 5 minutes to reduce load
- **Thread-Safe**: Mutex-protected caching for concurrent use

#### Client Usage Example

```go
import "github.com/n0needt0/bytefreezer-control/client"

// Create client
controlClient := client.NewClient(client.Config{
    BaseURL:        "http://bytefreezer-control:8080",
    APIKey:         "your-api-key",
    TimeoutSeconds: 30,
})

// Fetch tenants from Control Service
tenants, err := controlClient.ListTenants(ctx, accountID, 100)
if err != nil {
    // Fallback to legacy controller or dev mode
}
```

#### Configuration Helper with Fallback

```go
// Create helper with 3-tier fallback
helper := client.NewConfigHelper(
    controlClient,
    localConfig,   // From config.yaml
    defaultConfig, // Hardcoded defaults
)

// Get configuration with automatic fallback
config, err := helper.GetTenantConfig(ctx, accountID, tenantID)

// Type-safe config helpers
maxBatchSize := helper.GetConfigInt(ctx, accountID, tenantID, "max_batch_size", 500)
compression := helper.GetConfigString(ctx, accountID, tenantID, "compression", "gzip")
enabled := helper.GetConfigBool(ctx, accountID, tenantID, "enabled", true)
```

### Service Integration

#### ByteFreezer Receiver Integration
Updated `/home/andrew/workspace/bytefreezer/bytefreezer-receiver` to use Control Service:

**Changes**:
- Added `ControlServiceConfig` to `config/config.go:51`
- Updated `tenant/tenant_client.go` to use `bytefreezer-control/client`
- Implemented 3-tier tenant fetching: Control Service → Legacy Controller → Dev Mode
- Added active tenant filtering (inactive tenants skipped)
- Enhanced logging for tenant source tracking

**Fallback Priority** (`tenant/tenant_client.go:74`):
1. **Control Service API** (if enabled and accountID configured)
2. **Legacy Controller** (if controller URL configured)
3. **Dev Mode** (fake data for development)

**Key Implementation Details**:
- `tenant_client.go:107` - `fetchTenantsFromControlService()` method
- `tenant_client.go:138` - `fetchTenantsFromLegacyController()` method
- `config.go:287` - ControlService config passed to tenant client
- `go.mod:99` - Local replace directive for bytefreezer-control

### Configuration Changes

#### New Control Service Configuration
Services can now be configured to use Control Service:

```yaml
control_service:
  enabled: true
  base_url: "http://bytefreezer-control:8080"
  api_key: "your-api-key"
  timeout_seconds: 30
  account_id: "default-account-id"
  tenant_id: "default-tenant-id"
```

#### Deprecated Configuration (Still Supported)
The following configurations are deprecated but still work for backward compatibility:

```yaml
dev: true                              # Use control_service.enabled: false instead
bytefreezer:
  controller: "http://legacy/tenants"  # Use control_service.base_url instead
```

### Technical Details

#### Fallback Architecture
All services now support graceful degradation with 3-tier fallback:

1. **Control Service API** (primary source)
   - Centralized configuration management
   - Multi-account, multi-tenant support
   - Real-time configuration updates
   - 5-minute caching to reduce API load

2. **Local Configuration** (from config file)
   - Falls back when Control Service unavailable
   - Allows offline operation
   - Uses settings from config.yaml

3. **Default Configuration** (hardcoded)
   - Final fallback when both above fail
   - Ensures service continues to function
   - Sensible defaults for all settings

#### Thread Safety
- Client is safe for concurrent use across goroutines
- ConfigHelper uses sync.RWMutex for thread-safe caching
- No shared mutable state between requests

### Compatibility

#### Backward Compatibility
- **No Breaking Changes**: All existing configurations continue to work
- **Graceful Fallback**: Services continue if Control Service unavailable
- **Legacy Support**: Old controller endpoints still supported
- **Dev Mode**: Development mode with fake data still works

#### Migration Path

**For New Deployments**:
Use Control Service configuration:
```yaml
control_service:
  enabled: true
  base_url: "http://bytefreezer-control:8080"
  api_key: "your-api-key"
  account_id: "your-account-id"
```

**For Existing Deployments**:
Option 1 - Keep using legacy controller:
```yaml
bytefreezer:
  controller: "http://legacy-controller/tenants"
dev: false
```

Option 2 - Enable Control Service with legacy fallback:
```yaml
control_service:
  enabled: true
  base_url: "http://bytefreezer-control:8080"
  api_key: "your-api-key"
  account_id: "your-account-id"
bytefreezer:
  controller: "http://legacy/tenants"  # Used as fallback
```

### Next Steps (Phase 3)
- Integrate piper with bytefreezer-control/client (requires PipelineClient refactor)
- Remove hardcoded dev settings from piper
- Clean up AWX templates using piper examples
- Remove unused functions across all services
- Implement APIKey authentication system
- Add dataset management endpoints and client methods
- Publish bytefreezer-control package (remove local replace directives)

### Database Population Tool

Created `examples/populate_fake_data_simple.go` to populate database with fake data matching dev mode:

**What it creates**:
- 1 Development Account (`dev-account`)
- 4 Tenants (customer-1, tenant-001, tenant-002, tenant-003)
- 11 Datasets across all tenants
- Bearer tokens matching dev mode (stored in tenant custom_settings)
- S3 destination configurations

**Usage**:
```bash
cd examples
go run populate_fake_data_simple.go "postgres://postgres:postgres@localhost:5432/bytefreezer?sslmode=disable"
```

This eliminates the need for hardcoded dev mode in services. See `examples/README.md` for complete documentation.

### Files Changed

**ByteFreezer Control**:
- `client/client.go` - New client package (HTTP client, health check)
- `client/types.go` - Type definitions for Account, Tenant, Dataset
- `client/accounts.go` - Account management methods
- `client/tenants.go` - Tenant management methods
- `client/config_helper.go` - Configuration fallback with caching
- `client/README.md` - Client usage documentation
- `examples/populate_fake_data_simple.go` - Database population script
- `examples/README.md` - Examples and usage documentation

**ByteFreezer Receiver**:
- `config/config.go` - Added ControlServiceConfig struct
- `tenant/tenant_client.go` - Updated to use bytefreezer-control/client
- `go.mod` - Added bytefreezer-control dependency with local replace
- `RELEASENOTES.md` - Documented v2.1.0 Control Service integration

**ByteFreezer Packer**:
- `config/config.go` - Added ControlServiceConfig struct
- `tenant/tenant_client.go` - Updated to use bytefreezer-control/client
- `go.mod` - Added bytefreezer-control dependency with local replace
- Implemented 3-tier fallback (Control Service → Legacy Controller → Dev Mode)

### Testing
✅ Control client package compiles successfully
✅ Receiver builds successfully with new client integration
✅ Packer builds successfully with new client integration
⏳ End-to-end testing with live Control Service pending
⏳ Piper integration pending (different architecture - uses PipelineClient)

---

## Phase 1: Account Management & Configuration Hierarchy (2025-10-03)

### Major Features

#### Account → Tenant → Dataset Hierarchy
Implemented a three-tier configuration hierarchy to support multi-account, multi-tenant architecture:

- **Accounts**: Top-level entities representing ByteFreezer customers/organizations
  - Short GUID for system use (primary key)
  - Human-readable name and email
  - Account-level configuration (tier, limits, custom fields)
  - Support for free, pro, and enterprise tiers

- **Tenants**: Second-level entities belonging to accounts
  - Short GUID for system use
  - Human-readable name and description
  - Tenant-level configuration (organization, subscription, notifications, security, billing)
  - Multiple tenants per account

- **Datasets**: Processing pipelines belonging to tenants
  - Short GUID for system use
  - Human-readable name and description
  - Dataset-level configuration (source, processing, destination, schedule, monitoring)
  - Multiple datasets per tenant

#### Database Schema
- PostgreSQL-based storage with comprehensive schema
- Tables: `accounts`, `tenants`, `datasets`, `api_keys`
- Foreign key relationships with CASCADE delete
- JSONB columns for flexible configuration storage
- Performance indexes (regular + GIN for JSONB queries)
- Migration system with rollback support (3 migrations total)

#### Storage Layer
New storage abstraction layer with PostgreSQL implementation:

**Account Operations** (`storage/interface.go:91-96`, `storage/postgresql.go:57-240`):
- `CreateAccount(ctx, account)` - Create new account with default config (free tier, 5 tenants, 20 datasets)
- `GetAccount(ctx, id)` - Retrieve account by ID
- `GetAccountByEmail(ctx, email)` - Retrieve account by email (unique)
- `UpdateAccount(ctx, account)` - Update account details
- `DeleteAccount(ctx, id)` - Delete account (CASCADE deletes tenants/datasets)
- `ListAccounts(ctx, opts)` - List accounts with pagination

**Tenant Operations** (`storage/interface.go:98-108`, `storage/postgresql.go:242-405`):
- `CreateTenant(ctx, tenant)` - Create tenant scoped to account
- `GetTenant(ctx, accountID, tenantID)` - Retrieve tenant (account-scoped)
- `UpdateTenant(ctx, tenant)` - Update tenant details
- `DeleteTenant(ctx, accountID, tenantID)` - Delete tenant (CASCADE deletes datasets)
- `ListTenants(ctx, accountID, opts)` - List tenants for account

**Advanced Queries** (`storage/postgresql.go:407-461`):
- `FindTenantsBySubscriptionTier(ctx, tier)` - JSONB query by subscription tier
- `FindTenantsByOrganizationSize(ctx, size)` - JSONB query by org size
- `GetTenantsWithNotificationEnabled(ctx, type)` - JSONB query for notification settings

**API Key Operations** (stub implementations for Phase 2):
- API key CRUD operations defined in interface
- Stub implementations ready for authentication system

#### REST API Endpoints
Complete REST API for account and tenant management:

**Account Endpoints** (`api/api.go:80-85`, `api/handlers.go:310-502`):
- `GET /api/v1/accounts` - List all accounts (pagination support)
- `POST /api/v1/accounts` - Create new account (requires name, email)
- `GET /api/v1/accounts/{accountId}` - Get specific account
- `PUT /api/v1/accounts/{accountId}` - Update account (name, email, active status)
- `DELETE /api/v1/accounts/{accountId}` - Delete account (CASCADE)

**Tenant Endpoints** (`api/api.go:87-92`, `api/handlers.go:504-704`):
- `GET /api/v1/accounts/{accountId}/tenants` - List tenants for account
- `POST /api/v1/accounts/{accountId}/tenants` - Create tenant (requires name, optional description)
- `GET /api/v1/accounts/{accountId}/tenants/{tenantId}` - Get specific tenant
- `PUT /api/v1/accounts/{accountId}/tenants/{tenantId}` - Update tenant
- `DELETE /api/v1/accounts/{accountId}/tenants/{tenantId}` - Delete tenant (CASCADE)

All endpoints use usecase pattern with proper error handling (404, 400, etc.)

#### Service Integration
Storage layer integrated into service lifecycle (`services/services.go:91-139`):
- Automatic storage initialization on startup (if database enabled)
- PostgreSQL URI construction from config (host, port, user, password, database, sslmode)
- Automatic database migration on startup
- Graceful fallback if storage initialization fails
- Legacy database service maintained for backward compatibility

### Technical Details

**Migration System** (`storage/postgresql_migrator.go`):
- Version-based migrations with up/down support
- Migration 1: Create all tables with relationships
- Migration 2: Add performance indexes
- Migration 3: Add GIN indexes for JSONB queries
- Transactional migrations (rollback on failure)
- Migration status tracking in `migrations` table

**Default Configurations**:
- Account: free tier, max 5 tenants, max 20 datasets
- Tenant: Default TenantConfig from `config_types.go`
- Dataset: Default DatasetConfig from `config_types.go`

**Error Handling**:
- `ErrNotFound` for missing resources (used consistently)
- Unique constraint violations return descriptive errors
- Foreign key violations handled by PostgreSQL CASCADE

### Architecture Changes

**Before**: Flat tenant structure with hardcoded dev settings
**After**: Account → Tenant → Dataset hierarchy with centralized configuration

**Benefits**:
- Multi-tenancy support (multiple accounts, each with multiple tenants)
- Centralized configuration management (no more hardcoded settings)
- Scalable architecture (GUIDs + human-readable descriptions)
- Flexible JSONB configuration (extensible without schema changes)
- Proper authentication foundation (api_keys table ready)

### Files Changed

**Database & Storage**:
- `storage/postgresql_migrator.go` - Migration system with 3 migrations
- `storage/interface.go` - Storage interface with Account/Tenant/Dataset/APIKey operations
- `storage/postgresql.go` - PostgreSQL implementation (~850 lines)
- `storage/config_types.go` - Rich configuration schemas (existing)

**API Layer**:
- `api/api.go` - Route registration for account/tenant endpoints
- `api/handlers.go` - Complete handlers for all account/tenant operations
- Added `storage` import

**Service Layer**:
- `services/services.go` - Storage initialization, migration execution
- Added `Storage` field to Services struct
- URI construction from database config

### Breaking Changes
- Tenant API endpoints moved from `/api/v1/tenants/*` to `/api/v1/accounts/{accountId}/tenants/*`
- Tenant operations now require accountID in path
- GetTenant signature changed from `(ctx, id)` to `(ctx, accountID, tenantID)`

### Migration Path
For existing installations:
1. Ensure database config is correct in `config.yaml`
2. Restart service - migrations run automatically
3. Create initial account via `POST /api/v1/accounts`
4. Migrate existing tenants to use new account structure
5. Update client applications to use new account-scoped tenant endpoints

### Next Steps (Phase 2)
- Implement APIKey authentication system
- Create control-client library (go-goodies)
- Integrate receiver with Control Service APIs
- Integrate piper with Control Service APIs
- Integrate packer with Control Service APIs
- Remove hardcoded dev settings from all services
- Clean up AWX templates using piper examples
- Remove unused functions across all services

### Testing
✅ All code compiles successfully
⏳ End-to-end testing pending (requires PostgreSQL instance)
⏳ API integration testing pending

### Documentation
- Database schema documented in migration files
- API endpoints documented with usecase descriptions
- Code comments added for complex operations
- JSONB query examples in advanced queries
