# ByteFreezer Control - Release Notes

## v2.3.0: Account-Based Proxy Configuration API (2025-10-23)

### Features

#### New Proxy Configuration Endpoint for Account-Based Polling
- **Account-Level Configuration API**: New endpoint `/api/v2/proxy/config?account_id={accountID}` returns all tenants and datasets for an account
  - Single API call returns complete configuration for proxy instances
  - Eliminates need for multiple API calls per tenant
  - Supports multi-tenant proxy deployments
  - Implementation: `api/proxy_config_handlers.go:285-340`

#### Response Structure
```json
{
  "tenants": [
    {
      "tenant": {
        "id": "tenant-123",
        "account_id": "account-456",
        "name": "Production Tenant",
        ...
      },
      "datasets": [
        {
          "id": "dataset-789",
          "tenant_id": "tenant-123",
          "name": "sflow-data",
          "config": {
            "source": {
              "type": "stream",
              "custom": {
                "plugin_type": "sflow",
                "port": 2066,
                "protocol": "sflow",
                ...
              }
            }
          }
        }
      ]
    }
  ],
  "count": 1
}
```

#### Dataset Source Configuration for Plugin Generation
- **Plugin Configuration in Datasets**: Datasets now support plugin-specific configuration in `source.custom`
  - `plugin_type`: Plugin name (e.g., "sflow", "netflow", "ipfix")
  - Plugin-specific fields: port, protocol, data_hint, read_buffer_size, worker_count, etc.
  - Proxy reads these fields and dynamically generates plugin configurations

#### Configuration Reporting
- **Proxy Status Tracking**: Proxy reports applied configuration back to Control for each tenant
  - Uses existing `/api/v2/proxies/{instanceId}/config` endpoint
  - Stores which plugins are configured for each tenant
  - Enables dataset test endpoint to show which proxies have plugins active

### Technical Details

**New API Endpoint**:
```go
GET /api/v2/proxy/config?account_id={accountID}
Authorization: Bearer {token}

Response: ControlConfiguration with all tenants + datasets
```

**Implementation Flow**:
1. Proxy sends account_id to Control
2. Control queries all tenants for the account (`ListTenants`)
3. For each tenant, Control queries all datasets (`ListDatasets`)
4. Control returns nested structure: tenants → datasets
5. Proxy converts datasets to plugin configurations
6. Proxy reports applied configuration back to Control

**GetProxyConfiguration Handler**:
```go
func (api *API) GetProxyConfiguration() usecase.Interactor {
    // Query all tenants for account
    tenantResult, err := api.Services.Storage.ListTenants(ctx, input.AccountID, tenantOpts)

    // For each tenant, query datasets
    for _, tenant := range tenantResult.Items {
        datasetResult, err := api.Services.Storage.ListDatasets(ctx, tenant.ID, datasetOpts)
        tenantsWithDatasets = append(tenantsWithDatasets, tenantWithDatasets{
            Tenant:   tenant,
            Datasets: datasetResult.Items,
        })
    }

    return tenantsWithDatasets
}
```

### Files Modified

**Backend**:
- `api/api.go` - Added `/api/v2/proxy/config` route (line 155)
- `api/proxy_config_handlers.go` - GetProxyConfiguration handler (lines 285-340)
- `api/proxy_config_handlers.go` - Removed dataset_id validation (line 133-135)

### Deployment Notes

**Impact**:
- New endpoint is backwards compatible
- Existing tenant-based proxy config endpoints unchanged
- No database migration required
- Proxy configurations now accept any plugin_configs (no validation)

**Workflow Changes**:
- Create datasets with `source.type: "stream"` and `source.custom` containing plugin config
- Configure proxy with `account_id` instead of `tenant_id`
- Proxy polls new endpoint and dynamically generates plugin configs
- Proxy reports applied configuration per tenant

**Dataset Configuration Example**:
```json
{
  "name": "sflow-data",
  "config": {
    "source": {
      "type": "stream",
      "custom": {
        "plugin_type": "sflow",
        "port": 2066,
        "protocol": "sflow",
        "data_hint": "ndjson",
        "read_buffer_size": 65536,
        "worker_count": 4
      }
    },
    "destination": {
      "type": "s3",
      "connection": {
        "bucket": "bytefreezer-intake"
      }
    }
  }
}
```

## v2.2.6: Migration System Registration (2025-10-21)

### Features

#### Registered Missing Database Migrations
- **Migration System Completeness**: Registered migrations 4, 5, and 6 in the PostgreSQL migrator
  - Migration 4: Add resource_name column to audit log
  - Migration 5: Add user_email column to audit log
  - Migration 6: Cleanup token_refresh audit log entries
  - These migrations were created as SQL files but never registered in the migrator
  - Now they will execute on next service restart
  - Implementation: `storage/postgresql_migrator.go:206-237, 457-491, 540-552`

**Migration Details**:

**Migration 4** - Add resource_name column:
- Adds `resource_name VARCHAR(255)` to control_audit_log
- Uses DO block with IF NOT EXISTS for idempotency
- Source file: `storage/migrations/003_add_audit_log_resource_name.sql`

**Migration 5** - Add user_email column:
- Adds `user_email VARCHAR(255)` to control_audit_log
- Uses DO block with IF NOT EXISTS for idempotency
- Source file: `storage/migrations/004_add_audit_log_user_email.sql`

**Migration 6** - Cleanup token_refresh logs:
- Deletes all token_refresh entries from control_audit_log
- These are noisy logs that were removed from code in v2.2.1
- Source file: `storage/migrations/005_cleanup_token_refresh_audit_logs.sql`
- Note: No rollback possible (deleted data cannot be restored)

**Rollback Support**:
- Migration 4 rollback: Drops resource_name column
- Migration 5 rollback: Drops user_email column
- Migration 6 rollback: No rollback (cleanup migration)

### Files Modified

**Backend**:
- `storage/postgresql_migrator.go` - Registered migrations 4, 5, and 6 in getAllMigrations(), getMigrationSQL(), and getRollbackSQL()

### Technical Details

**Registration Structure**:
```go
// Added to getAllMigrations()
{
    Version:     4,
    Name:        "add_audit_log_resource_name",
    Description: "Add resource_name column to control_audit_log table",
},
{
    Version:     5,
    Name:        "add_audit_log_user_email",
    Description: "Add user_email column to control_audit_log table",
},
{
    Version:     6,
    Name:        "cleanup_token_refresh_audit_logs",
    Description: "Remove noisy token_refresh entries from audit logs",
}
```

**Migration Execution**:
- Migrations run automatically on service startup
- Version tracking in control_migrations table
- Transactional execution with rollback on failure
- Idempotent SQL ensures safe re-runs

### Deployment Notes

**On Next Restart**:
1. Service will detect pending migrations 4, 5, 6
2. Apply them in order (4 → 5 → 6)
3. Record version in control_migrations table
4. Migration 6 will delete all existing token_refresh audit log entries

**Expected Log Output**:
```
INFO: Applying migration 4: add_audit_log_resource_name
INFO: Applying migration 5: add_audit_log_user_email
INFO: Applying migration 6: cleanup_token_refresh_audit_logs
INFO: Deleted N rows from control_audit_log (action='token_refresh')
```

### Testing

**Verified Migration Execution**:
- ✅ All migrations (4, 5, 6) applied successfully on service startup
- ✅ Migration 6 deleted all token_refresh entries from audit log
- ✅ Confirmed via API: No token_refresh entries remain in database
- ✅ Audit logs working properly with new columns (resource_name, user_email)
- ✅ Service starts and runs successfully with all migrations applied

**Test Commands**:
```bash
# Start service (migrations run automatically)
./bytefreezer-control -config config.yaml

# Verify no token_refresh entries remain
curl -s http://localhost:8082/api/v1/audit-logs?limit=100 | grep token_refresh
# Returns: (empty - no matches found)

# Check audit logs are working
curl -s http://localhost:8082/api/v1/audit-logs?limit=10
# Returns: login, user_created, user_updated, etc. (no token_refresh)
```

### Binary

**Build Information**:
- Binary: `bytefreezer-control`
- Compiled successfully with all changes
- Tested: Migrations execute successfully on startup

---

## v2.2.5: Audit Log NULL Handling Fix (2025-10-21)

### Bug Fixes

#### Fix Audit Log Display with NULL user_email
- **NULL Handling**: Fixed audit log listing failure when `user_email` column contains NULL values
  - Root cause: Older audit logs or system actions may have NULL user_email
  - Error: "converting NULL to string is unsupported" when scanning user_email column
  - Solution: Use `sql.NullString` for user_email field, just like resource_name and error_message
  - Implementation: `storage/postgresql.go:1128-1143, 1233-1252`

**Error Fixed**:
```
failed to scan audit log: sql: Scan error on column index 2, name "user_email":
converting NULL to string is unsupported
```

**Before**: Audit log queries failed when encountering NULL user_email values
**After**: Audit logs display correctly, with empty string for NULL user_email

### Files Modified

**Backend**:
- `storage/postgresql.go` - Use sql.NullString for user_email in both ListAuditLogs and GetAuditLog

### Technical Details

**Fix Implementation**:
```go
// Before (Failed on NULL)
err := rows.Scan(&log.ID, &log.UserID, &log.UserEmail, ...)

// After (Handles NULL)
var userEmail sql.NullString
err := rows.Scan(&log.ID, &log.UserID, &userEmail, ...)
if userEmail.Valid {
    log.UserEmail = userEmail.String
}
```

**When user_email is NULL**:
- System-generated actions (automated cleanup, migrations, etc.)
- Audit logs created before user_email column was added
- Actions performed by background processes

### Binary

**Build Information**:
- Binary: `bytefreezer-control`
- Compiled successfully with all changes

---

## v2.2.4: MinIO S3 Delete Compatibility Fix (2025-10-21)

### Bug Fixes

#### Fix MinIO S3 Object Deletion (MissingContentMD5 Error)
- **MinIO Compatibility**: Fixed dataset deletion failures with MinIO S3 storage
  - Root cause: MinIO requires `Content-MD5` header for batch `DeleteObjects` operations
  - AWS SDK v2 doesn't include this header by default for DeleteObjects
  - Solution: Switch from batch deletes to individual `DeleteObject` calls for MinIO compatibility
  - Implementation: `storage/s3_helper.go:90-108`

**Error Fixed**:
```
operation error S3: DeleteObjects, https response error StatusCode: 400,
api error MissingContentMD5: Missing required header for this request: Content-Md5
```

**Before**: Used batch DeleteObjects (up to 1000 objects per request) which failed on MinIO
**After**: Use individual DeleteObject calls (one per object) which works with MinIO

**Performance Note**: Individual deletes are slightly slower than batch deletes, but necessary for MinIO compatibility. For AWS S3, batch deletes would be more efficient, but MinIO is the primary deployment target.

### Files Modified

**Backend**:
- `storage/s3_helper.go` - Changed from batch DeleteObjects to individual DeleteObject calls
- `storage/s3_helper.go` - Removed unused `types` import

### Technical Details

**Old Implementation** (Batch Delete):
```go
_, err := s.client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
    Bucket: aws.String(bucket),
    Delete: &types.Delete{
        Objects: objectIdentifiers,  // Up to 1000 objects
        Quiet:   aws.Bool(true),
    },
})
```

**New Implementation** (Individual Delete):
```go
for _, obj := range page.Contents {
    _, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
        Bucket: aws.String(bucket),
        Key:    obj.Key,
    })
    // Continue on error instead of failing entire cleanup
}
```

**Key Changes**:
- Individual deletes work with MinIO (no Content-MD5 requirement)
- Continues deleting remaining objects even if one fails
- Logs warnings for failed deletes instead of failing entire operation
- More resilient to partial failures

### Binary

**Build Information**:
- Binary: `bytefreezer-control`
- Compiled successfully with all changes

---

## v2.2.3: Authentication Configuration Fixes (2025-10-21)

### Features

#### Token Expiry Hours Configuration Actually Works
- **Configurable Token Expiry**: Fixed `auth.token_expiry_hours` configuration to actually control JWT token expiration
  - Previously, tokens were hardcoded to 1 hour regardless of config setting
  - Now tokens expire based on `auth.token_expiry_hours` setting (default: 24 hours)
  - Implementation: `services/auth.go:17-20, 72-82, 147-150`
  - Service initialization logs token expiry setting: `services/services.go:147-148`
  - Added validation: defaults to 1 hour if invalid value provided

**Before**: Tokens always expired in 1 hour (hardcoded)
**After**: Tokens expire based on config.yaml `auth.token_expiry_hours` setting

**Configuration** (`config.yaml:85-88`):
```yaml
auth:
  enabled: false
  jwt_secret: "your-jwt-secret-key-here"
  token_expiry_hours: 24  # Now actually used!
```

#### Password Reset Now Uses Database Roles
- **Database-Driven Admin Check**: Password reset now checks database user roles instead of config file
  - Previously, password reset checked `auth.admin_users` list in config file
  - Now queries database for users with `system_admin` or `account_admin` roles
  - Eliminates inconsistency between config-based admin list and database roles
  - Implementation: `api/handlers.go:1118-1131`

**Before**: Checked `auth.admin_users` array in config.yaml
**After**: Queries `control_users` table for `role IN ('system_admin', 'account_admin')`

**Note**: The `auth.admin_users` config setting is now deprecated but still present in config for backward compatibility. It is no longer used functionally.

### Bug Fixes

#### Fix Token Expiry Hours Inconsistency
- **Configuration Not Used**: Fixed bug where `token_expiry_hours` config was read but never applied
  - Root cause: AuthService struct didn't store or use the config value
  - Solution: Added `tokenExpiryHours` field to AuthService and updated token generation
  - Tokens now properly expire based on configuration instead of hardcoded 1 hour
  - Default to 1 hour if invalid value provided (≤0)

#### Fix Admin Users Config vs Database Inconsistency
- **Redundant Admin List**: Fixed password reset to use database roles instead of config list
  - Root cause: Password reset checked config file while actual roles are in database
  - Solution: Query database for user roles instead of checking config array
  - Aligns password reset with the database-driven role system
  - Reduces configuration complexity and potential inconsistencies

### Files Modified

**Backend**:
- `services/auth.go` - Added tokenExpiryHours field, updated NewAuthService, updated GenerateTokenPair to use configured expiry
- `services/services.go` - Pass token_expiry_hours from config to AuthService initialization
- `api/handlers.go` - Updated RequestPasswordReset to check database roles instead of config admin_users

**Configuration Files**:
- `config.yaml` - Removed deprecated admin_users array, added comment about database-driven roles
- `ansible/playbooks/templates/config.yaml.j2` - Removed admin_users template section, added database role comment
- `ansible/playbooks/group_vars/all.yml` - Removed admin_users variable definition, added database role comment

### Technical Details

**Token Expiry Implementation**:
- AuthService struct now stores tokenExpiryHours from config
- NewAuthService validates and defaults to 1 hour if invalid
- GenerateTokenPair uses `time.Duration(a.tokenExpiryHours) * time.Hour`
- Startup log shows configured token expiry: "Authentication service initialized (token expiry: 24 hours)"

**Admin Role Check**:
- Password reset iterates all users from database
- Checks if email matches AND role is 'system_admin' or 'account_admin'
- No longer uses api.Config.Auth.AdminUsers array
- Prevents email enumeration attacks (always returns success message)

### Removed Configuration

The following configuration setting has been removed:
```yaml
auth:
  admin_users:
    - "admin@company.com"
```

**Reason**: Admin roles are now stored in the database `control_users.role` field with values: `system_admin`, `account_admin`, `account_readonly`.

**Migration**:
- Remove `admin_users` array from your config.yaml if present (won't cause errors but is no longer read)
- Admin users should be created via the API with appropriate roles: `POST /api/v1/users`
- Existing config files will continue to work - the setting is simply ignored if present

### Binary

**Build Information**:
- Binary: `bytefreezer-control`
- Compiled successfully with all changes

---

## v2.2.2: S3 Configuration for Dataset Cleanup (2025-10-21)

### Features

#### Centralized S3 Configuration for Dataset Cleanup
- **S3 Configuration in Control Service**: Added centralized S3 configuration for dataset deletion operations
  - S3 credentials and settings now configured in `config.yaml` instead of per-dataset
  - Eliminates EC2 IMDS credential chain fallback that was causing errors
  - Explicit credential configuration prevents AWS SDK from attempting IAM role lookups
  - Configurable bucket names for intake and piper buckets
  - Implementation: `config/config.go:55-64`, `storage/interface.go:223-233`

**Configuration Structure** (`config.yaml:26-35`):
```yaml
s3:
  enabled: true
  intake_bucket: "intake"
  piper_bucket: "piper"
  region: "us-east-1"
  endpoint: "192.168.86.125:9000"  # MinIO endpoint
  access_key: "AKIAQNXBAHUKIMJP26P6"
  secret_key: "SECRET_KEY_AKIAQNXBAHUKIMJP26P6"
  use_ssl: false
```

**Dataset Deletion Paths**:
- Deletes from `{intake_bucket}/{tenant_id}/{dataset_id}/`
- Deletes from `{piper_bucket}/{tenant_id}/{dataset_id}/`
- Preserves packer output bucket (customer data)

**Key Changes**:
- `NewS3Cleaner()` now requires explicit credentials and never falls back to AWS default credential chain
- `CleanupDatasetStorage()` accepts configurable bucket names instead of hardcoded values
- `DeleteDataset()` uses control service S3 config instead of dataset config
- S3 cleanup only runs if `s3.enabled: true` in control config

### Bug Fixes

#### Fix Dataset Deletion S3 Credential Errors
- **Resolved EC2 IMDS Errors**: Fixed "no EC2 IMDS role found" errors during dataset deletion
  - Root cause: AWS SDK was falling back to EC2 Instance Metadata Service for credentials
  - Solution: Use explicit static credentials only, disable default credential chain
  - Error no longer attempts to contact 169.254.169.254 (EC2 metadata endpoint)
  - Implementation: `storage/s3_helper.go:21-37`

### Files Modified

**Backend**:
- `config/config.go` - Added S3Config struct with bucket names
- `storage/interface.go` - Added S3Config to storage Config
- `storage/s3_helper.go` - Updated NewS3Cleaner to use only explicit credentials
- `storage/postgresql.go` - Updated DeleteDataset to use control config S3 settings
- `services/services.go` - Pass S3 config from control config to storage
- `config.yaml` - Added S3 configuration section

**Ansible/AWX Templates**:
- `ansible/playbooks/templates/config.yaml.j2` - Added S3 configuration template section
- `ansible/playbooks/group_vars/all.yml` - Added S3 configuration variables

### Binary

**Build Information**:
- Binary: `bytefreezer-control`
- Size: 34 MB
- MD5: `4937b8ce1f0c3286445638a12e1f9af1`

---

## v2.2.1: Audit Log Improvements (2025-10-21)

### Features

#### Add User Email to Audit Logs
- **User Email Field Added**: Added `user_email` column to audit logs for better readability
  - Audit logs now display the user's email address directly
  - Eliminates need to join with users table to see who performed actions
  - Makes audit log viewing more user-friendly in the UI
  - Database migration: `storage/migrations/004_add_audit_log_user_email.sql`
  - Column: `user_email VARCHAR(255)`

### Bug Fixes

#### Remove token_refresh from Audit Log
- **Token Refresh Logging Removed**: Removed audit logging for token refresh operations to reduce log noise
  - Token refresh is an automated, frequent operation that doesn't require audit logging
  - Reduces audit log table size and improves query performance
  - Initial login events are still logged for security tracking
  - Implementation: `api/handlers.go:1097-1100` (removed)
  - Deleted 142 existing token_refresh entries from database

### Files Modified

**Backend**:
- `storage/interface.go` - Added `UserEmail` field to AuditLog struct
- `services/audit_log.go` - Updated LogAction() to accept userEmail parameter
- `api/handlers.go` - Updated all LogAction() calls to pass user email
- `storage/postgresql.go` - Updated SELECT queries and Scan calls for user_email
- `storage/migrations/004_add_audit_log_user_email.sql` - New migration file

**Frontend**:
- `src/lib/api.ts` - Added `user_email` field to AuditLog interface

---

## v2.2.0: Proxy Configuration Management (2025-10-21)

### Major Features

#### 🎛️ Proxy Instance Configuration Management
Comprehensive system for managing proxy instance configurations centrally.

**Database Schema** (`migrations/004_proxy_configuration.sql`):
- `proxy_instances` table: Stores current proxy configurations
- `proxy_config_history` table: Maintains configuration audit trail
- Automatic configuration versioning with each update
- SHA256 hash-based change detection
- Trigger-based automatic history archiving

**API Endpoints** (`/api/v2/proxies/*`):
- `GET /api/v2/proxies` - List all proxy instances (filterable by tenant)
- `GET /api/v2/proxies/{instanceId}/config?tenant_id={tid}` - Get proxy configuration
- `PUT /api/v2/proxies/{instanceId}/config` - Create/update proxy configuration
- `POST /api/v2/proxies/{instanceId}/config/applied` - Mark configuration as applied
- `GET /api/v2/proxies/{instanceId}/config/history?tenant_id={tid}` - Get configuration history
- `DELETE /api/v2/proxies/{instanceId}?tenant_id={tid}` - Delete proxy instance

#### 📋 Configuration Structure
Each proxy instance configuration includes:
- **Instance Identification**: instance_id (hostname), tenant_id, instance_api (hostname:port)
- **Configuration Mode**: local-only, control-only, or hybrid
- **Plugin Configurations**: Array of plugin configurations (type, name, config)
- **Proxy Settings**: receiver, batching, spooling, housekeeping, otel, soc settings
- **Versioning**: config_version (auto-incremented), config_hash (SHA256)
- **Application Tracking**: config_applied, config_applied_at timestamps

#### 🔄 Configuration Workflow

1. **Create/Update Configuration**:
   ```bash
   PUT /api/v2/proxies/{instanceId}/config
   {
     "tenant_id": "my-tenant",
     "instance_api": "proxy-01:8080",
     "config_mode": "hybrid",
     "plugin_configs": [...],
     "proxy_settings": {...}
   }
   ```
   - Auto-increments version
   - Calculates SHA256 hash
   - Resets `config_applied` to false

2. **Proxy Polls Configuration**:
   ```bash
   GET /api/v2/proxies/{instanceId}/config?tenant_id={tid}
   ```
   - Returns current configuration with version and hash
   - Proxy compares hash to detect changes

3. **Proxy Reports Applied Status**:
   ```bash
   POST /api/v2/proxies/{instanceId}/config/applied
   {
     "tenant_id": "my-tenant",
     "config_version": 3
   }
   ```
   - Marks configuration as applied
   - Updates timestamp

4. **View Configuration History**:
   ```bash
   GET /api/v2/proxies/{instanceId}/config/history?tenant_id={tid}&limit=10
   ```
   - Returns last N configuration versions
   - Includes change timestamps and version numbers

#### 🛠️ Implementation Details

**Storage Layer** (`storage/`):
- `config_types.go`: ProxyInstanceConfig, ProxySettings, and component settings types
- `interface.go`: Storage interface methods for proxy config operations
- `postgresql_proxy_config.go`: PostgreSQL implementation with JSONB support

**API Layer** (`api/`):
- `api.go`: Route definitions for proxy config endpoints
- `proxy_config_handlers.go`: HTTP handlers using swaggest/usecase pattern

**Database Functions** (SQL):
- `upsert_proxy_config()`: Create or update with automatic versioning
- `mark_proxy_config_applied()`: Update application status
- `get_proxy_config()`: Retrieve current configuration
- `cleanup_old_proxy_config_history()`: Maintain history retention (90 days, keep last 10)

#### 📊 Configuration History & Audit Trail
- All configuration changes automatically archived
- Triggered on UPDATE when plugin_configs or proxy_settings change
- Stores: version, configs, hash, changed_by, change_reason, timestamp
- Cleanup policy: Keep last 10 versions per instance, or 90 days, whichever is more recent

#### 🔐 Security & Validation
- SHA256 hash verification for change detection
- Version-based concurrency control
- Tenant isolation in all queries
- Configuration mode validation (local-only, control-only, hybrid)
- Instance-tenant uniqueness constraints

#### 🎯 Use Cases

**Centralized Management**:
- Manage all proxy configurations from control UI/API
- Update configurations without proxy restarts
- Roll back to previous configurations using history

**Multi-Instance Coordination**:
- Consistent configurations across proxy fleet
- Per-instance customization when needed
- Track which instances have applied latest config

**Operational Visibility**:
- See current vs. applied configuration
- Configuration change audit trail
- Identify instances with outdated configurations

### Database Schema
```sql
proxy_instances (
    id SERIAL PRIMARY KEY,
    instance_id VARCHAR(255),
    tenant_id VARCHAR(255),
    instance_api VARCHAR(255),
    config_mode VARCHAR(50),
    plugin_configs JSONB,
    proxy_settings JSONB,
    config_version INTEGER,
    config_applied BOOLEAN,
    config_applied_at TIMESTAMP,
    config_hash VARCHAR(64),
    active BOOLEAN,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    UNIQUE(instance_id, tenant_id)
)
```

### Files Added/Modified

**New Files**:
- `migrations/004_proxy_configuration.sql`
- `storage/postgresql_proxy_config.go`
- `api/proxy_config_handlers.go`

**Modified Files**:
- `storage/config_types.go` - Added ProxyInstanceConfig and related types
- `storage/interface.go` - Added proxy config interface methods
- `api/api.go` - Added proxy config routes

### Integration with Proxy
This control-side implementation works with bytefreezer-proxy v4.0.0 which includes:
- Configuration polling service
- Dynamic plugin reload
- Local configuration caching
- Port conflict resolution

See bytefreezer-proxy RELEASENOTES.md for proxy-side details.

---

## Phase 2.1: Health Status & Consistency (2025-10-11)

### Bug Fixes

#### Health Registration Status Consistency
- **Consistent Startup Status**: Changed control service self-registration to use "Starting" status instead of "Healthy"
  - All services now consistently register with "Starting" status on startup
  - Aligns control service behavior with all other ByteFreezer components (proxy, receiver, piper, packer)
  - Services transition to "Healthy" after first successful health check
  - Implementation: `services/health.go:422`

### Behavior Changes
- **Before**: Control service registered itself as "Healthy" immediately on startup
- **After**: Control service registers as "Starting" and transitions to "Healthy" after health checks confirm service readiness
- **Benefit**: More accurate health status representation during service initialization phase

---

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
