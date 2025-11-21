# ByteFreezer AI Agent Quick Reference

Quick reference for AI agents interacting with ByteFreezer API.

## Authentication

```bash
# Login
curl -X POST https://api.bytefreezer.com/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"pass"}'

# Use token in subsequent requests
curl -H "Authorization: Bearer YOUR_TOKEN" \
  https://api.bytefreezer.com/api/v1/datasets
```

## Common Operations

### List Resources

```bash
# List all datasets (flat list)
GET /api/v1/datasets

# List datasets for specific tenant
GET /api/v1/tenants/{tenantId}/datasets

# List all tenants
GET /api/v1/tenants

# List accounts
GET /api/v1/accounts
```

### Transformation Workflow

```bash
# 1. Get schema and samples
GET /api/v1/tenants/{tenantId}/datasets/{datasetId}/transformations/schema

# 2. Test transformation
POST /api/v1/tenants/{tenantId}/datasets/{datasetId}/transformations/test
Body: {"filters": [...]}

# 3. Check job status
GET /api/v1/transformations/jobs/{jobId}

# 4. Validate on fresh data
POST /api/v1/tenants/{tenantId}/datasets/{datasetId}/transformations/validate
Body: {"filters": [...], "count": 100}

# 5. Deploy to production
POST /api/v1/tenants/{tenantId}/datasets/{datasetId}/transformations/activate
Body: {"filters": [...], "enabled": true}

# 6. Monitor performance
GET /api/v1/tenants/{tenantId}/datasets/{datasetId}/transformations/stats
```

## Transformation Filter Template

```json
{
  "type": "filter_type",
  "config": {
    "param1": "value1",
    "param2": "value2"
  },
  "enabled": true,
  "notes": "Optional description"
}
```

## Common Filter Types

### Grok (Parse Text)
```json
{
  "type": "grok",
  "config": {
    "source_field": "message",
    "pattern": "%{IP:client_ip} %{WORD:method}"
  },
  "enabled": true
}
```

### GeoIP (Enrich IP)
```json
{
  "type": "geoip",
  "config": {
    "source_field": "client_ip",
    "target_field": "geo"
  },
  "enabled": true
}
```

### Mutate (Modify Fields)
```json
{
  "type": "mutate",
  "config": {
    "add_field": {"environment": "prod"},
    "remove_field": ["temp"],
    "rename": {"old_name": "new_name"}
  },
  "enabled": true
}
```

### Drop (Filter Out Events)
```json
{
  "type": "drop",
  "config": {
    "field": "status",
    "equals": "debug"
  },
  "enabled": true
}
```

### Include (Keep Only Matching)
```json
{
  "type": "include",
  "config": {
    "field": "log_level",
    "equals": "ERROR"
  },
  "enabled": true
}
```

### Date Parse
```json
{
  "type": "date_parse",
  "config": {
    "source_field": "timestamp",
    "target_field": "@timestamp",
    "formats": ["2006-01-02 15:04:05"]
  },
  "enabled": true
}
```

### Passthrough (No-op)
```json
{
  "type": "passthrough",
  "config": {},
  "enabled": true,
  "notes": "Placeholder for future transformation"
}
```

## AI Pipeline Generation

```bash
POST /api/v1/ai/pipeline/generate
Body:
{
  "tenant_id": "tenant-id",
  "dataset_id": "dataset-id",
  "prompt": "Extract IPs and add geoip data",
  "schema": [...],
  "samples": [...]
}
```

## Error Handling

```json
{
  "error": "Error message",
  "details": "Detailed explanation"
}
```

## Status Codes

- `200` - Success
- `201` - Created
- `400` - Bad request
- `401` - Unauthorized
- `403` - Forbidden
- `404` - Not found
- `429` - Rate limited
- `500` - Server error

## Filter Catalog

```bash
# Get all available filter types with documentation
GET /api/v1/piper/filters
```

## Job Statuses

- `pending` - Job queued
- `processing` - Job running
- `completed` - Job successful
- `failed` - Job failed
- `retrying` - Job retrying
- `cancelled` - Job cancelled

## Rate Limits

- Standard: 100 req/min
- Service accounts: 1000 req/min

Check headers:
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1704110400
```

## Complete Example (Python)

```python
import requests
import time

BASE = "https://api.bytefreezer.com/api/v1"

# Auth
r = requests.post(f"{BASE}/login", json={
    "email": "ai@example.com",
    "password": "pass"
})
token = r.json()["access_token"]
h = {"Authorization": f"Bearer {token}"}

# Get schema
schema = requests.get(
    f"{BASE}/tenants/my-tenant/datasets/logs/transformations/schema",
    headers=h
).json()

# Test transformation
test = requests.post(
    f"{BASE}/tenants/my-tenant/datasets/logs/transformations/test",
    headers=h,
    json={"filters": [
        {
            "type": "grok",
            "config": {
                "source_field": "message",
                "pattern": "%{IP:ip}"
            },
            "enabled": True
        }
    ]}
).json()

# Poll job
job_id = test["job_id"]
while True:
    job = requests.get(
        f"{BASE}/transformations/jobs/{job_id}",
        headers=h
    ).json()["response"]

    if job["status"] == "completed":
        print("Success:", job["result"])
        break
    time.sleep(1)

# Deploy
requests.post(
    f"{BASE}/tenants/my-tenant/datasets/logs/transformations/activate",
    headers=h,
    json={
        "filters": [...],
        "enabled": True
    }
)
```

## Tips for AI Agents

1. Always test transformations before deploying
2. Use validation endpoint to test on fresh data
3. Monitor stats after deployment
4. Use notes field to document transformations
5. Start with passthrough filters as placeholders
6. Check filter catalog for available options
7. Handle job polling with timeouts
8. Respect rate limits
9. Use meaningful error handling
10. Keep transformations modular and simple
