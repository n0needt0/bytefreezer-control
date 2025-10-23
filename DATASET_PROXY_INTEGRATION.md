# Dataset-Proxy Configuration Integration

## Overview

This document explains how datasets in ByteFreezer Control integrate with proxy plugin configurations, ensuring data integrity and proper reference management.

## Architecture

### Data Flow

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
│                             │                         │           │
│                             │  • instance_id          │           │
│                             │  • tenant_id            │           │
│                             │  • config_version       │           │
│                             └─────────────────────────┘           │
└─────────────────────────────────────────────────────────────────┘
                                       │
                                       │ polls every N seconds
                                       ▼
                            ┌─────────────────────┐
                            │  ByteFreezer Proxy  │
                            │                     │
                            │  Receives plugin    │
                            │  configs with       │
                            │  dataset_id refs    │
                            └─────────────────────┘
```

## Integration Mechanism

### How It Works

1. **Dataset Creation**
   - User creates a dataset in Control via `/api/v1/tenants/{tenantId}/datasets`
   - Dataset gets a unique ID (e.g., "sflow-data", "netflow-data")
   - Dataset stored in `control_datasets` table

2. **Proxy Configuration Creation**
   - User creates/updates proxy configuration via `/api/v2/proxies/{instanceId}/config`
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
   - Proxy polls Control for configuration updates
   - Receives plugin configs with valid `dataset_id` references
   - Proxy uses `dataset_id` for:
     - Spool directory structure: `/var/spool/bytefreezer-proxy/{tenant}/{dataset_id}/`
     - Data organization and routing
     - Processing pipeline identification

5. **Dataset Testing**
   - Test endpoint `/api/v1/tenants/{tenantId}/datasets/{datasetId}/test` checks:
     - Which proxy instances have plugins configured for the dataset
     - Whether proxy has applied the configuration
     - Returns status: "active", "untested", or "degraded"

6. **Dataset Deletion Protection**
   - Control prevents deletion of datasets referenced by proxy configs
   - Returns error identifying which proxy instance has the reference
   - User must remove dataset from proxy config before deletion

## API Examples

### Creating a Dataset

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
curl -X PUT http://control:8080/api/v2/proxies/proxy-01/config \
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

## Validation Rules

### Dataset Reference Validation (Upsert Proxy Config)

**Location**: `api/proxy_config_handlers.go:133-146`

**Rules**:
1. Extract all `dataset_id` fields from plugin configs
2. For each non-empty `dataset_id`:
   - Check if dataset exists via `GetDataset(ctx, tenantID, datasetID)`
   - If not found, return HTTP 400 with error message
   - Error includes plugin config index for easy debugging

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
curl -X PUT http://control:8080/api/v2/proxies/proxy-01/config \
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
curl -X PUT http://control:8080/api/v2/proxies/proxy-01/config \
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
curl -X PUT http://control:8080/api/v2/proxies/proxy-01/config ...
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
curl -X PUT http://control:8080/api/v2/proxies/proxy-01/config \
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
1. List proxy configs: `GET /api/v2/proxies?tenant_id={tenantId}`
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
GET /api/v2/proxies/{instanceId}/config/history?tenant_id={tenantId}
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

## Future Enhancements

### Automatic Proxy Config Generation
- Auto-generate proxy configuration when dataset is created
- Suggest appropriate plugin types based on dataset config

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
