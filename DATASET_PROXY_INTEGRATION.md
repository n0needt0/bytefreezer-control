# Dataset-Proxy Configuration Integration

## Overview

This document explains how datasets in ByteFreezer Control integrate with proxy plugin configurations, ensuring data integrity and proper reference management.

ByteFreezer supports **two configuration modes**:

1. **Legacy Tenant Mode**: Manual proxy configuration via API (pre-v4.1.0)
2. **Account-Based Mode**: Automatic configuration generation from datasets (v4.1.0+)

Both modes ensure referential integrity and proper dataset-proxy relationships.

## Quick Reference

### Legacy Tenant Mode (Pre-v4.1.0)

**Configuration Flow**:
```
User → Create Dataset → Create Proxy Config (manual) → Proxy Polls → Proxy Applies
```

**Key Characteristics**:
- Manual proxy configuration via API
- Dataset is just a data definition
- Proxy config explicitly defines plugins
- One tenant per proxy instance
- Endpoint: `GET /api/v1/proxies/{instanceId}/config?tenant_id={tenantId}`

**Proxy Config**:
```yaml
tenant_id: "customer-1"
config_mode: "hybrid"
```

### Account-Based Mode (v4.1.0+)

**Configuration Flow**:
```
User → Create Dataset with source.custom → Proxy Polls → Proxy Generates Plugins → Proxy Reports Back
```

**Key Characteristics**:
- Automatic plugin generation from datasets
- Dataset includes plugin configuration in `source.custom`
- Zero-touch proxy configuration
- Multiple tenants per proxy instance
- Endpoint: `GET /api/v1/proxy/config?account_id={accountID}`

**Proxy Config**:
```yaml
account_id: "ejq73vgnw26p"
config_mode: "control-only"
config_polling:
  enabled: true
  interval_seconds: 60
```

**Dataset Example**:
```json
{
  "config": {
    "source": {
      "type": "ebpf",
      "custom": {
        "host": "0.0.0.0",
        "port": 2056,
        "worker_count": 4,
        "read_buffer_size": 65536
      }
    }
  }
}
```

**Note**: `source.type` (e.g., "ebpf", "sflow", "netflow") becomes the plugin type.

## Architecture

### Mode 1: Legacy Tenant Mode (Pre-v4.1.0)

```
┌─────────────────────────────────────────────────────────────────┐
│                     ByteFreezer Control                          │
│                                                                  │
│  ┌──────────────┐           ┌─────────────────────────┐        │
│  │   Datasets   │           │  Proxy Configurations   │        │
│  │ (control_    │           │  (proxy_instances)      │        │
│  │  datasets)   │           │                         │        │
│  │              │           │  ┌────────────────────┐ │        │
│  │ • id         │◄──refs────┼──┤ plugin_configs[]   │ │        │
│  │ • tenant_id  │           │  │  - dataset_id ─────┼─┘        │
│  │ • name       │           │  │  - type           │           │
│  │ • config     │           │  │  - port           │           │
│  └──────────────┘           │  └────────────────────┘           │
│         ▲                   │                         │           │
│         │                   │  • instance_id          │           │
│         │ validates         │  • tenant_id            │           │
│         │ dataset_id        │  • config_version       │           │
│         │                   └─────────────────────────┘           │
│         │                              │                          │
│  ┌──────┴────────────┐                │                          │
│  │ User creates      │                │                          │
│  │ proxy config via  │                │ GET /api/v1/proxies/     │
│  │ PUT endpoint      │                │    {instanceId}/config   │
│  └───────────────────┘                │                          │
└─────────────────────────────────────────────────────────────────┘
                                       │
                                       │ polls every N seconds
                                       ▼
                            ┌─────────────────────┐
                            │  ByteFreezer Proxy  │
                            │  (tenant mode)      │
                            │                     │
                            │  Receives plugin    │
                            │  configs with       │
                            │  dataset_id refs    │
                            └─────────────────────┘
```

### Mode 2: Account-Based Mode (v4.1.0+)

```
┌──────────────────────────────────────────────────────────────────┐
│                     ByteFreezer Control                           │
│                                                                   │
│  ┌──────────────────────────────┐    ┌──────────────────────┐   │
│  │   Datasets                   │    │ Proxy Configurations │   │
│  │  (control_datasets)          │    │ (proxy_instances)    │   │
│  │                              │    │                      │   │
│  │ • id                         │    │  plugin_configs[]    │   │
│  │ • tenant_id                  │    │  - dataset_id        │   │
│  │ • config:                    │    │  - type              │   │
│  │   - source:                  │    │  - port              │   │
│  │     - type: "stream"         │    │                      │   │
│  │     - custom:                │    └──────────────────────┘   │
│  │       * plugin_type: "sflow" │                │               │
│  │       * port: 2066           │                │               │
│  │       * protocol: "sflow"    │                │               │
│  └──────────────────────────────┘                │               │
│         │                                         │               │
│         │ GET /api/v1/proxy/config?              │               │
│         │     account_id=XXX                      │               │
│         │                                         │               │
│         │ Returns all tenants + datasets         │               │
│         │                                         │               │
│         └─────────────────────────────────────────┘               │
│                                   │                               │
│                                   │ polls every N seconds         │
│                                   ▼                               │
│                        ┌────────────────────────┐                │
│                        │  ByteFreezer Proxy     │                │
│                        │  (account mode)        │                │
│                        │                        │                │
│                        │ 1. Polls for tenants   │                │
│                        │    + datasets          │                │
│                        │ 2. Generates plugin    │                │
│                        │    configs from        │                │
│                        │    source.custom       │                │
│                        │ 3. Starts plugins      │                │
│                        │ 4. Reports config ─────┼────────────────┤
│                        │    via PUT endpoint    │  Validates &   │
│                        └────────────────────────┘  Stores        │
│                                                                   │
└──────────────────────────────────────────────────────────────────┘
```

## Integration Mechanism

### Mode 1: Legacy Tenant Mode (Manual Configuration)

**How It Works**:

1. **Dataset Creation**
   - User creates a dataset in Control via `/api/v1/tenants/{tenantId}/datasets`
   - Dataset gets a unique ID (e.g., "sflow-data", "netflow-data")
   - Dataset stored in `control_datasets` table

2. **Proxy Configuration Creation** (Manual)
   - User creates/updates proxy configuration via `/api/v1/proxies/{instanceId}/config`
   - Proxy configuration includes plugin configs that reference datasets by `dataset_id`
   - Example plugin config:
     ```json
     {
       "type": "sflow",
       "name": "sflow-listener",
       "config": {
         "host": "0.0.0.0",
         "port": 2066,
         "dataset_id": "sflow-data",
         "protocol": "sflow",
         "data_hint": "ndjson"
       }
     }
     ```

3. **Validation**
   - Control validates that `dataset_id` references exist before saving
   - Returns HTTP 400 if any referenced dataset doesn't exist
   - This ensures referential integrity

4. **Configuration Polling**
   - Proxy polls Control: `GET /api/v1/proxies/{instanceId}/config?tenant_id={tenantId}`
   - Receives pre-built plugin configs with valid `dataset_id` references
   - Proxy applies configuration and starts plugins

5. **Proxy Configuration**: `config.yaml`
   ```yaml
   tenant_id: "customer-1"
   bearer_token: "your-token"
   control_url: "http://192.168.86.103:8082"
   config_mode: "hybrid"
   ```

### Mode 2: Account-Based Mode (Automatic Generation - v4.1.0+)

**How It Works**:

1. **Dataset Creation with Plugin Configuration**
   - User creates dataset with `source.type` indicating plugin type
   - Plugin-specific parameters go in `source.custom`
   - Example:
     ```json
     {
       "name": "ebpf-data",
       "config": {
         "source": {
           "type": "ebpf",
           "custom": {
             "host": "0.0.0.0",
             "port": 2056,
             "worker_count": 4,
             "read_buffer_size": 65536
           }
         }
       }
     }
     ```

2. **Account-Based Polling**
   - Proxy polls: `GET /api/v1/proxy/config?account_id={accountID}`
   - Control returns **all tenants + datasets** for the account
   - Response includes datasets with `source.custom` plugin configurations

3. **Dynamic Plugin Generation** (Proxy-Side)
   - Proxy uses `dataset.config.source.type` as the plugin type
   - Proxy generates plugin configuration dynamically:
     ```go
     pluginConfig := {
       "type": dataset.Config.Source.Type,  // "ebpf", "sflow", "netflow"
       "name": dataset.Name + "-" + tenant.Name,
       "config": {
         "dataset_id": dataset.ID,
         "tenant_id": tenant.ID,
         "port": custom["port"],              // All fields from source.custom
         "host": custom["host"],
         "worker_count": custom["worker_count"],
         // ... etc
       }
     }
     ```

4. **Plugin Startup**
   - Proxy starts plugins based on generated configs
   - Only **active datasets** with non-empty `source.type` are processed
   - Inactive datasets are skipped

5. **Configuration Reporting**
   - Proxy reports generated config back to Control
   - Uses same endpoint: `PUT /api/v1/proxies/{instanceId}/config`
   - Control validates `dataset_id` references in reported config

6. **Proxy Configuration**: `config.yaml`
   ```yaml
   account_id: "ejq73vgnw26p"
   bearer_token: "your-token"
   control_url: "http://192.168.86.103:8082"
   config_mode: "control-only"

   config_polling:
     enabled: true
     interval_seconds: 60
   ```

### Common Steps (Both Modes)

**Dataset ID Usage**:
- Proxy uses `dataset_id` for:
  - Spool directory structure: `/var/spool/bytefreezer-proxy/{tenant}/{dataset_id}/`
  - Data organization and routing
  - Processing pipeline identification

**Dataset Testing**:
- Test endpoint `/api/v1/tenants/{tenantId}/datasets/{datasetId}/test` checks:
  - Which proxy instances have plugins configured for the dataset
  - Whether proxy has applied the configuration
  - Returns status: "active", "untested", or "degraded"

**Dataset Deletion Protection**:
- Control prevents deletion of datasets referenced by proxy configs
- Returns error identifying which proxy instance has the reference
- User must remove dataset from proxy config before deletion

## API Examples

### Mode 1: Legacy Tenant Mode Examples

#### Creating a Dataset (Legacy Mode)

```bash
curl -X POST http://control:8080/api/v1/tenants/ejq73vgnw26p/datasets \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "sflow-data",
    "display_name": "sFlow Network Data",
    "description": "Network flow data from switches",
    "active": true,
    "config": {
      "source": {
        "type": "stream"
      },
      "destination": {
        "type": "s3",
        "connection": {
          "bucket": "bytefreezer-intake",
          "region": "us-east-1"
        }
      }
    }
  }'
```

### Creating Proxy Configuration with Dataset Reference

```bash
curl -X PUT http://control:8080/api/v1/proxies/proxy-01/config \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "tenant_id": "ejq73vgnw26p",
    "instance_api": "proxy-01.example.com:8081",
    "config_mode": "hybrid",
    "plugin_configs": [
      {
        "type": "sflow",
        "name": "sflow-listener",
        "config": {
          "host": "0.0.0.0",
          "port": 2066,
          "dataset_id": "sflow-data",
          "protocol": "sflow",
          "data_hint": "ndjson",
          "read_buffer_size": 65536,
          "worker_count": 4
        }
      }
    ]
  }'
```

### Testing Dataset Configuration

```bash
curl http://control:8080/api/v1/tenants/ejq73vgnw26p/datasets/sflow-data/test \
  -H "Authorization: Bearer $TOKEN"
```

Response when configured and applied:
```json
{
  "success": true,
  "input_test_status": "active",
  "input_test_message": "Plugin configured in proxy proxy-01",
  "output_test_status": "active",
  "output_test_message": "S3 connection verified",
  "tested_at": "2025-10-23T10:30:00Z"
}
```

Response when not configured:
```json
{
  "success": false,
  "input_test_status": "degraded",
  "input_test_message": "No proxy configuration found for this dataset",
  "output_test_status": "active",
  "output_test_message": "S3 connection verified",
  "tested_at": "2025-10-23T10:30:00Z"
}
```

### Attempting to Delete Referenced Dataset

```bash
curl -X DELETE http://control:8080/api/v1/tenants/ejq73vgnw26p/datasets/sflow-data \
  -H "Authorization: Bearer $TOKEN"
```

Response (HTTP 400):
```json
{
  "error": "cannot delete dataset 'sflow-data': referenced by proxy instance 'proxy-01' (remove from proxy config first)"
}
```

### Mode 2: Account-Based Mode Examples (v4.1.0+)

#### Creating a Dataset with Plugin Configuration

In account-based mode, the `source.type` indicates the plugin type (ebpf, sflow, netflow, etc.):

```bash
curl -X POST http://control:8080/api/v1/tenants/tenant1/datasets \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "id": "ebpf-data",
    "name": "ebpf-data",
    "display_name": "eBPF Network Data",
    "description": "Network data collected via eBPF",
    "active": true,
    "config": {
      "source": {
        "type": "ebpf",
        "custom": {
          "host": "0.0.0.0",
          "port": 2056,
          "worker_count": 4,
          "read_buffer_size": 65536
        }
      },
      "destination": {
        "type": "minio",
        "connection": {
          "bucket": "bytefreezer-data",
          "endpoint": "192.168.86.125:9000",
          "region": "us-east-1"
        }
      }
    }
  }'
```

**Key Fields**:
- `source.type` - **Required** - Plugin type (ebpf, sflow, netflow, ipfix, http, etc.)
- `source.custom.*` - Plugin-specific configuration parameters
- `active: true` - Must be active for proxy to generate plugin

#### Polling for Account Configuration

Proxy polls this endpoint (automatically):

```bash
curl http://control:8080/api/v1/proxy/config?account_id=ejq73vgnw26p \
  -H "Authorization: Bearer $TOKEN"
```

Response:
```json
{
  "tenants": [
    {
      "tenant": {
        "id": "tenant1",
        "account_id": "ejq73vgnw26p",
        "name": "tenant1",
        "display_name": "Tenant 1",
        "active": true
      },
      "datasets": [
        {
          "id": "sflow-data",
          "tenant_id": "tenant1",
          "name": "sflow-data",
          "active": true,
          "config": {
            "source": {
              "type": "ebpf",
              "custom": {
                "host": "0.0.0.0",
                "port": 2056,
                "worker_count": 4,
                "read_buffer_size": 65536
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

#### Proxy Generates and Reports Configuration

After receiving tenants + datasets, proxy:
1. Generates plugin configs from `source.custom`
2. Starts plugins
3. Reports applied configuration to Control:

```bash
# This is done automatically by proxy, but here's what it sends:
curl -X PUT http://control:8080/api/v1/proxies/proxy-01/config \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "instance_id": "proxy-01",
    "tenant_id": "tenant1",
    "instance_api": "proxy-01:8088",
    "config_mode": "control-only",
    "plugin_configs": [
      {
        "type": "ebpf",
        "name": "ebpf-data-tenant1",
        "config": {
          "tenant_id": "tenant1",
          "dataset_id": "ebpf-data",
          "host": "0.0.0.0",
          "port": 2056,
          "worker_count": 4,
          "read_buffer_size": 65536,
          "bearer_token": "..."
        }
      }
    ],
    "proxy_settings": {}
  }'
```

**Note**: The `dataset_id` reference is validated at this point, just like legacy mode.

#### Creating Multiple Datasets for Multi-Plugin Setup

Create NetFlow dataset:

```bash
curl -X POST http://control:8080/api/v1/tenants/tenant1/datasets \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "id": "netflow-data",
    "name": "netflow-data",
    "active": true,
    "config": {
      "source": {
        "type": "stream",
        "custom": {
          "plugin_type": "netflow",
          "port": 2068,
          "protocol": "netflow",
          "data_hint": "ndjson"
        }
      }
    }
  }'
```

Create IPFIX dataset:

```bash
curl -X POST http://control:8080/api/v1/tenants/tenant1/datasets \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "id": "ipfix-data",
    "name": "ipfix-data",
    "active": true,
    "config": {
      "source": {
        "type": "stream",
        "custom": {
          "plugin_type": "ipfix",
          "port": 2070,
          "protocol": "ipfix",
          "data_hint": "ndjson"
        }
      }
    }
  }'
```

**Result**: On next poll (60 seconds), proxy automatically:
- Detects 3 datasets with `plugin_type`
- Generates 3 plugin configs
- Starts listeners on ports 2066, 2068, 2070
- Reports all 3 configs to Control

#### Deactivating a Dataset

To stop a plugin without deleting the dataset:

```bash
curl -X PUT http://control:8080/api/v1/tenants/tenant1/datasets/sflow-data \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "sflow-data",
    "active": false,
    "config": { ... }
  }'
```

**Result**: On next poll, proxy detects dataset is inactive and removes the plugin.

## Validation Rules

### Dataset Reference Validation (Upsert Proxy Config)

**Location**: `api/proxy_config_handlers.go:133-146`

**Applies to**: Both modes (legacy and account-based)

**When it Runs**:
- **Legacy Mode**: When user creates/updates proxy config via `PUT /api/v1/proxies/{instanceId}/config`
- **Account-Based Mode**: When proxy reports generated config via `PUT /api/v1/proxies/{instanceId}/config`

**Rules**:
1. Extract all `dataset_id` fields from plugin configs
2. For each non-empty `dataset_id`:
   - Check if dataset exists via `GetDataset(ctx, tenantID, datasetID)`
   - If not found, return HTTP 400 with error message
   - Error includes plugin config index for easy debugging

**Note**: In account-based mode, validation occurs AFTER proxy generates configs, not during dataset creation. This is intentional - datasets can exist without being used by a proxy.

**Error Format**:
```
plugin config at index 0 references non-existent dataset_id 'invalid-dataset'
```

### Dataset Deletion Protection

**Location**: `api/handlers.go:1532-1546`

**Rules**:
1. Before deleting dataset, list all proxy configs for tenant
2. Scan all plugin configs in each proxy configuration
3. If any plugin config has matching `dataset_id`:
   - Return HTTP 400 with error message
   - Error includes proxy instance ID
   - Dataset deletion is blocked

**Error Format**:
```
cannot delete dataset 'sflow-data': referenced by proxy instance 'proxy-01' (remove from proxy config first)
```

## Database Schema

### Datasets Table
```sql
CREATE TABLE control_datasets (
    id VARCHAR(255) PRIMARY KEY,
    tenant_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    description TEXT,
    active BOOLEAN DEFAULT true,
    config JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

### Proxy Instances Table
```sql
CREATE TABLE proxy_instances (
    id SERIAL PRIMARY KEY,
    instance_id VARCHAR(255) NOT NULL,
    tenant_id VARCHAR(255) NOT NULL,
    instance_api VARCHAR(255) NOT NULL,
    config_mode VARCHAR(50) NOT NULL DEFAULT 'hybrid',
    plugin_configs JSONB DEFAULT '[]'::jsonb,  -- Contains dataset_id references
    proxy_settings JSONB DEFAULT '{}'::jsonb,
    config_version INTEGER NOT NULL DEFAULT 1,
    config_applied BOOLEAN DEFAULT false,
    config_applied_at TIMESTAMP WITH TIME ZONE,
    config_hash VARCHAR(64),
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE (instance_id, tenant_id)
);
```

## Plugin Configuration Schema

### Required Fields

All plugin configurations must include:
- `type`: Plugin type (e.g., "sflow", "netflow", "ipfix", "http")
- `name`: Unique name for the plugin instance
- `config`: Plugin-specific configuration object

### Common Config Fields

Most plugins include:
- `dataset_id`: Reference to dataset in control_datasets (REQUIRED for data routing)
- `host`: Listening host (default: "0.0.0.0")
- `port`: Listening port (plugin-specific)
- `protocol`: Protocol name (e.g., "sflow", "netflow")
- `data_hint`: Format hint for downstream processing (e.g., "ndjson", "raw")

### Example: sFlow Plugin Config

```json
{
  "type": "sflow",
  "name": "sflow-listener",
  "config": {
    "host": "0.0.0.0",
    "port": 2066,
    "dataset_id": "sflow-data",
    "protocol": "sflow",
    "data_hint": "ndjson",
    "read_buffer_size": 65536,
    "worker_count": 4
  }
}
```

### Example: NetFlow Plugin Config

```json
{
  "type": "netflow",
  "name": "netflow-listener",
  "config": {
    "host": "0.0.0.0",
    "port": 2068,
    "dataset_id": "netflow-data",
    "protocol": "netflow",
    "data_hint": "ndjson",
    "read_buffer_size": 65536,
    "worker_count": 4
  }
}
```

### Example: HTTP Webhook Plugin Config

```json
{
  "type": "http",
  "name": "webhook-receiver",
  "config": {
    "host": "0.0.0.0",
    "port": 8082,
    "dataset_id": "webhook-data",
    "path": "/webhook",
    "method": "POST",
    "data_hint": "json"
  }
}
```

## Workflow Examples

### Scenario 1: New Dataset with Proxy Configuration

**Steps**:
1. Create dataset "metrics-data" in Control
2. Create proxy configuration with plugin referencing "metrics-data"
3. Proxy polls and receives configuration
4. Proxy applies configuration and starts listening
5. Proxy reports config_applied=true to Control
6. Dataset test shows "active" status

**Commands**:
```bash
# 1. Create dataset
curl -X POST http://control:8080/api/v1/tenants/ejq73vgnw26p/datasets \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name": "metrics-data", "display_name": "Metrics", "active": true, "config": {...}}'

# 2. Create proxy config
curl -X PUT http://control:8080/api/v1/proxies/proxy-01/config \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "tenant_id": "ejq73vgnw26p",
    "instance_api": "proxy-01:8081",
    "config_mode": "hybrid",
    "plugin_configs": [{
      "type": "http",
      "name": "metrics-receiver",
      "config": {"dataset_id": "metrics-data", "port": 8082}
    }]
  }'

# 3. Wait for proxy to poll (housekeeping interval)
# 4. Proxy applies config automatically
# 5. Proxy reports applied status

# 6. Test dataset
curl http://control:8080/api/v1/tenants/ejq73vgnw26p/datasets/metrics-data/test \
  -H "Authorization: Bearer $TOKEN"
```

### Scenario 2: Attempting Invalid Dataset Reference

**Steps**:
1. Try to create proxy config with non-existent dataset_id
2. Control returns HTTP 400 error
3. User creates dataset first
4. User retries proxy config creation
5. Success

**Commands**:
```bash
# 1. Try with invalid dataset
curl -X PUT http://control:8080/api/v1/proxies/proxy-01/config \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "tenant_id": "ejq73vgnw26p",
    "instance_api": "proxy-01:8081",
    "config_mode": "hybrid",
    "plugin_configs": [{
      "type": "sflow",
      "name": "sflow-listener",
      "config": {"dataset_id": "invalid-dataset", "port": 2066}
    }]
  }'

# Response: HTTP 400
# {"error": "plugin config at index 0 references non-existent dataset_id 'invalid-dataset'"}

# 2. Create dataset first
curl -X POST http://control:8080/api/v1/tenants/ejq73vgnw26p/datasets \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name": "sflow-data", ...}'

# 3. Retry proxy config (now succeeds)
curl -X PUT http://control:8080/api/v1/proxies/proxy-01/config ...
```

### Scenario 3: Deleting Dataset in Use

**Steps**:
1. Dataset is referenced by proxy configuration
2. User attempts to delete dataset
3. Control returns HTTP 400 error with proxy instance ID
4. User removes dataset from proxy config
5. User deletes dataset successfully

**Commands**:
```bash
# 1. Try to delete dataset
curl -X DELETE http://control:8080/api/v1/tenants/ejq73vgnw26p/datasets/sflow-data \
  -H "Authorization: Bearer $TOKEN"

# Response: HTTP 400
# {"error": "cannot delete dataset 'sflow-data': referenced by proxy instance 'proxy-01' (remove from proxy config first)"}

# 2. Update proxy config to remove the plugin
curl -X PUT http://control:8080/api/v1/proxies/proxy-01/config \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "tenant_id": "ejq73vgnw26p",
    "instance_api": "proxy-01:8081",
    "config_mode": "hybrid",
    "plugin_configs": []
  }'

# 3. Delete dataset (now succeeds)
curl -X DELETE http://control:8080/api/v1/tenants/ejq73vgnw26p/datasets/sflow-data \
  -H "Authorization: Bearer $TOKEN"

# Response: HTTP 200
# {"success": true, "message": "Dataset sflow-data deleted successfully"}
```

## Troubleshooting

### Issue: "plugin config at index 0 references non-existent dataset_id"

**Cause**: Trying to create proxy config with dataset_id that doesn't exist

**Solution**:
1. Create the dataset first via `/api/v1/tenants/{tenantId}/datasets`
2. Then create/update the proxy configuration

### Issue: "cannot delete dataset: referenced by proxy instance"

**Cause**: Dataset is still referenced by proxy configuration

**Solution**:
1. List proxy configs: `GET /api/v1/proxies?tenant_id={tenantId}`
2. Find the proxy instance mentioned in the error
3. Update proxy config to remove the plugin referencing the dataset
4. Retry dataset deletion

### Issue: Dataset test shows "degraded" status for input

**Cause**: No proxy configuration has a plugin for this dataset

**Solution**:
1. Create proxy configuration with plugin referencing the dataset
2. Wait for proxy to poll and apply configuration
3. Retest dataset

### Issue: Dataset test shows "untested" status for input

**Cause**: Proxy has configuration but hasn't applied it yet

**Solution**:
1. Wait for proxy housekeeping interval (default: 60 seconds)
2. Check proxy logs for any errors
3. Verify proxy can reach Control API
4. Retest dataset after proxy applies config

## Best Practices

### 1. Create Datasets Before Proxy Configurations
Always create datasets in Control before referencing them in proxy configurations. This ensures validation passes.

### 2. Use Descriptive Dataset IDs
Use clear, descriptive dataset IDs that indicate the data type:
- `sflow-data` for sFlow network data
- `netflow-data` for NetFlow data
- `webhook-events` for HTTP webhook events
- `metrics-timeseries` for metrics data

### 3. Test Dataset Configuration
After creating proxy configuration, use the test endpoint to verify:
- Proxy has the configuration
- Proxy has applied the configuration
- S3 connectivity is working

### 4. Document Dataset-Proxy Relationships
Maintain documentation of which datasets are used by which proxy instances and for what purpose.

### 5. Use Configuration History
When troubleshooting, check proxy configuration history:
```bash
GET /api/v1/proxies/{instanceId}/config/history?tenant_id={tenantId}
```

### 6. Monitor Configuration Application
Regularly check that proxies are applying configurations:
- Look for `config_applied=true` in proxy instance records
- Check `config_applied_at` timestamp
- Monitor proxy logs for configuration updates

## Security Considerations

### Dataset Access Control
- Datasets are scoped to tenants
- Proxy configurations are scoped to tenants
- Cross-tenant dataset references are prevented by API design

### Validation Importance
- Dataset reference validation prevents broken configurations
- Deletion protection ensures data pipeline integrity
- Test endpoint helps identify misconfigurations early

### Audit Trail
- All dataset operations are logged in audit log
- Proxy configuration changes are recorded in history table
- Failed validation attempts are logged

## Choosing Between Modes

### When to Use Legacy Tenant Mode

**Use Case**:
- Single tenant deployments
- Need manual control over proxy plugin configurations
- Complex proxy setups requiring custom plugin parameters
- Existing deployments (pre-v4.1.0)

**Advantages**:
- Full control over proxy configuration
- Can create proxy configs before datasets exist
- Explicit validation at config creation time
- Supports custom plugin configurations not tied to datasets

**Disadvantages**:
- Manual configuration management
- Configuration must be updated when adding/removing datasets
- More API calls required to set up data ingestion
- Requires deep knowledge of plugin configuration schema

### When to Use Account-Based Mode

**Use Case**:
- Multi-tenant deployments
- Account-level management (one proxy serves multiple tenants)
- Dynamic dataset management (frequent adds/removes)
- Simplified configuration (datasets define their own plugin settings)

**Advantages**:
- **Zero-touch proxy configuration** - datasets self-configure
- **Dynamic updates** - add dataset, plugin starts automatically
- **Multi-tenant by default** - one proxy serves entire account
- **Simplified management** - configuration lives with data definition
- **DDoS protection** - exponential backoff for inactive accounts

**Disadvantages**:
- Less granular control over plugin parameters
- Dataset schema must include `source.custom` configuration
- Validation happens after proxy reports config (not at dataset creation)
- Requires proxy v4.1.0 or later

### Comparison Table

| Feature | Legacy Tenant Mode | Account-Based Mode |
|---------|-------------------|-------------------|
| **Configuration Source** | User via API | Dataset `source.custom` |
| **Plugin Generation** | Manual | Automatic |
| **Multi-tenant** | One tenant per proxy | Multiple tenants per proxy |
| **Dataset Changes** | Manual config update | Auto-detected on poll |
| **Validation Point** | Config creation | Config reporting |
| **Proxy Restart Required** | For config changes | No (dynamic reload) |
| **DDoS Protection** | Not applicable | Exponential backoff |
| **Setup Complexity** | High | Low |
| **Control Granularity** | High | Medium |
| **Recommended For** | Legacy systems | New deployments |

## Migration Guide

### Migrating from Legacy to Account-Based Mode

#### Step 1: Update Datasets

Ensure datasets have proper `source.type` set:

```bash
# For each dataset, ensure source.type matches the plugin type you want
curl -X PUT http://control:8080/api/v1/tenants/tenant1/datasets/ebpf-data \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "ebpf-data",
    "active": true,
    "config": {
      "source": {
        "type": "ebpf",
        "custom": {
          "host": "0.0.0.0",
          "port": 2056,
          "worker_count": 4,
          "read_buffer_size": 65536
        }
      },
      "destination": { ... }
    }
  }'
```

**Tip**: Extract plugin parameters from existing proxy configuration:

```bash
# Get current proxy config
curl http://control:8080/api/v1/proxies/proxy-01/config?tenant_id=tenant1 \
  -H "Authorization: Bearer $TOKEN"

# Extract plugin type from plugin_configs[].type → becomes source.type
# Extract plugin parameters from plugin_configs[].config → becomes source.custom
```

#### Step 2: Update Proxy Configuration

Change proxy `config.yaml`:

**Before (Legacy Mode)**:
```yaml
tenant_id: "tenant1"
bearer_token: "your-token"
control_url: "http://192.168.86.103:8082"
config_mode: "hybrid"

# Local plugin configs (will be overridden by Control)
inputs: []
```

**After (Account-Based Mode)**:
```yaml
account_id: "ejq73vgnw26p"
bearer_token: "your-token"
control_url: "http://192.168.86.103:8082"
config_mode: "control-only"

# Empty - plugins generated from datasets
inputs: []

config_polling:
  enabled: true
  interval_seconds: 60
```

#### Step 3: Restart Proxy

```bash
sudo systemctl restart bytefreezer-proxy
sudo journalctl -u bytefreezer-proxy -f
```

Expected logs:
```
INFO Polling configuration from control
INFO Received configuration for account ejq73vgnw26p: X tenants, Y datasets
INFO Converted Y datasets to Z plugin configs
INFO Configuration changed, applying Z plugin configs
INFO Starting sflow plugin on 0.0.0.0:2066
```

#### Step 4: Verify Migration

Check proxy reported configuration:

```bash
PGPASSWORD=bytefreezer123 psql -h 192.168.86.137 -U bytefreezer -d bytefreezer -c "
SELECT instance_id, tenant_id, config_applied,
       jsonb_pretty(plugin_configs) as plugins
FROM proxy_instances
WHERE instance_id = 'proxy-01';"
```

Check listening ports:

```bash
sudo netstat -tuln | grep -E '206[0-9]'
```

Test dataset:

```bash
curl http://control:8080/api/v1/tenants/tenant1/datasets/sflow-data/test \
  -H "Authorization: Bearer $TOKEN"
```

#### Step 5: Remove Legacy Proxy Config (Optional)

Once verified working, you can remove the manually-created proxy config:

```bash
curl -X DELETE http://control:8080/api/v1/proxies/proxy-01/config?tenant_id=tenant1 \
  -H "Authorization: Bearer $TOKEN"
```

**Note**: Proxy will recreate it automatically on next poll from dataset configs.

### Migrating from Account-Based to Legacy Mode

This is less common, but possible:

1. Note all plugin configurations from datasets' `source.custom`
2. Create manual proxy configuration via `PUT /api/v1/proxies/{instanceId}/config`
3. Update proxy `config.yaml` to use `tenant_id` instead of `account_id`
4. Restart proxy
5. Optionally remove `source.custom` from datasets

## Future Enhancements

### Automatic Proxy Config Generation (✅ Implemented in v4.1.0)
- ~~Auto-generate proxy configuration when dataset is created~~ **DONE**
- ~~Suggest appropriate plugin types based on dataset config~~ **DONE via source.custom.plugin_type**

### Dataset Reference Reports
- List all proxy instances using a specific dataset
- Show dataset usage statistics across proxies

### Configuration Templates
- Predefined templates for common dataset-proxy patterns
- Validation rules based on data types

### Advanced Testing
- Real data flow testing through the pipeline
- Performance metrics for dataset ingestion
- Automatic health checks and alerts
