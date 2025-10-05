# ByteFreezer Control Client

Go client library for interacting with the ByteFreezer Control Service API.

## Features

- **Account Management**: Create, read, update, delete accounts
- **Tenant Management**: Manage tenants within accounts
- **Dataset Management**: Manage datasets within tenants
- **Configuration Fallback**: Automatic fallback from API → local config → defaults
- **Caching**: Built-in configuration caching (5 minute default)
- **Type-Safe**: Strongly typed API with proper error handling
- **Zero Dependencies**: Uses only Go standard library

## Installation

```bash
go get github.com/n0needt0/bytefreezer-control/client
```

## Quick Start

### Basic Client Usage

```go
import (
    "context"
    "github.com/n0needt0/bytefreezer-control/client"
)

// Create client
controlClient := client.NewClient(client.Config{
    BaseURL:        "http://localhost:8080",
    APIKey:         "your-api-key",
    TimeoutSeconds: 30,
})

// Health check
ctx := context.Background()
if err := controlClient.HealthCheck(ctx); err != nil {
    log.Fatal("Control service unavailable:", err)
}

// Create account
account, err := controlClient.CreateAccount(ctx, client.CreateAccountRequest{
    Name:  "Acme Corp",
    Email: "admin@acme.com",
})

// Create tenant
tenant, err := controlClient.CreateTenant(ctx, account.ID, client.CreateTenantRequest{
    Name:        "production",
    Description: "Production environment",
})
```

### Configuration Helper with Fallback

```go
import (
    "context"
    "github.com/n0needt0/bytefreezer-control/client"
)

// Local configuration (fallback if API unavailable)
localConfig := map[string]interface{}{
    "max_batch_size": 1000,
    "compression":    "gzip",
}

// Default configuration (final fallback)
defaultConfig := map[string]interface{}{
    "max_batch_size": 500,
    "compression":    "none",
    "retention_days": 30,
}

// Create client
controlClient := client.NewClient(client.Config{
    BaseURL: "http://bytefreezer-control:8080",
    APIKey:  "your-api-key",
})

// Create config helper
helper := client.NewConfigHelper(controlClient, localConfig, defaultConfig)

// Get configuration with automatic fallback
// Priority: API → localConfig → defaultConfig
config, err := helper.GetTenantConfig(ctx, accountID, tenantID)

// Get specific values with type-safe helpers
maxBatchSize := helper.GetConfigInt(ctx, accountID, tenantID, "max_batch_size", 500)
compression := helper.GetConfigString(ctx, accountID, tenantID, "compression", "none")
enabled := helper.GetConfigBool(ctx, accountID, tenantID, "enabled", true)
```

## API Methods

### Accounts

```go
// Create account
account, err := controlClient.CreateAccount(ctx, client.CreateAccountRequest{
    Name:  "Company Name",
    Email: "admin@company.com",
})

// Get account by ID
account, err := controlClient.GetAccount(ctx, accountID)

// List accounts
accounts, err := controlClient.ListAccounts(ctx, 10) // limit 10

// Update account
account, err := controlClient.UpdateAccount(ctx, accountID, client.UpdateAccountRequest{
    Name: "New Name",
})

// Delete account (cascades to tenants/datasets)
err := controlClient.DeleteAccount(ctx, accountID)
```

### Tenants

```go
// Create tenant
tenant, err := controlClient.CreateTenant(ctx, accountID, client.CreateTenantRequest{
    Name:        "production",
    Description: "Production environment",
})

// Get tenant
tenant, err := controlClient.GetTenant(ctx, accountID, tenantID)

// List tenants for account
tenants, err := controlClient.ListTenants(ctx, accountID, 20) // limit 20

// Update tenant
tenant, err := controlClient.UpdateTenant(ctx, accountID, tenantID, client.UpdateTenantRequest{
    Description: "Updated description",
})

// Delete tenant (cascades to datasets)
err := controlClient.DeleteTenant(ctx, accountID, tenantID)
```

### Configuration Helper

```go
// Create helper
helper := client.NewConfigHelper(controlClient, localConfig, defaultConfig)

// Get full configuration
config, err := helper.GetTenantConfig(ctx, accountID, tenantID)

// Get typed values
str := helper.GetConfigString(ctx, accountID, tenantID, "key", "default")
num := helper.GetConfigInt(ctx, accountID, tenantID, "key", 100)
bool := helper.GetConfigBool(ctx, accountID, tenantID, "key", true)

// Invalidate cache (force refresh)
helper.InvalidateCache()

// Change cache duration
helper.SetCacheDuration(10 * time.Minute)
```

## Integration Examples

### ByteFreezer Receiver

```go
import "github.com/n0needt0/bytefreezer-control/client"

// Initialize control client
controlClient := client.NewClient(client.Config{
    BaseURL:        config.ControlService.BaseURL,
    APIKey:         config.ControlService.APIKey,
    TimeoutSeconds: config.ControlService.TimeoutSeconds,
})

// Fetch tenants from Control Service
tenants, err := controlClient.ListTenants(ctx, accountID, 100)
if err != nil {
    log.Warnf("Failed to fetch tenants from Control Service: %v", err)
    // Fallback to legacy controller or dev mode
}
```

## Error Handling

```go
account, err := controlClient.GetAccount(ctx, accountID)
if err != nil {
    // Check for specific errors
    if strings.Contains(err.Error(), "status 404") {
        // Account not found
    } else if strings.Contains(err.Error(), "status 401") {
        // Unauthorized
    } else {
        // Other error (network, timeout, etc.)
    }
}
```

## Caching Behavior

The `ConfigHelper` caches configuration for 5 minutes by default to reduce API calls:

- First call: Fetches from API, caches result
- Subsequent calls (within 5 min): Returns cached result
- After 5 min: Fetches fresh data from API
- API failure: Falls back to local → default config
- Manual invalidation: `helper.InvalidateCache()`

## Fallback Priority

1. **Control Service API** (primary source)
2. **Local Configuration** (from config file)
3. **Default Configuration** (hardcoded fallback)

This ensures services continue to function even if the Control Service is unavailable.

## Thread Safety

The client is safe for concurrent use. The ConfigHelper uses internal mutexes for thread-safe caching.

## License

Same as ByteFreezer project
