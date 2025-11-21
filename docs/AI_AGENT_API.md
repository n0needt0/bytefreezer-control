# ByteFreezer API Documentation for AI Agents

This document provides comprehensive API documentation for AI agents to interact with ByteFreezer programmatically.

## Base URL

```
https://your-bytefreezer-instance.com/api/v1
```

## Authentication

All API requests require JWT authentication via the `Authorization` header:

```
Authorization: Bearer <your-jwt-token>
```

### Obtaining an Access Token

```bash
POST /api/v1/login
Content-Type: application/json

{
  "email": "your-email@example.com",
  "password": "your-password"
}
```

**Response:**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_in": 3600,
  "user": {
    "id": "user-uuid",
    "email": "your-email@example.com",
    "role": "admin"
  }
}
```

### Refreshing Tokens

```bash
POST /api/v1/refresh
Content-Type: application/json

{
  "refresh_token": "your-refresh-token"
}
```

## Core Resources

### Accounts

Accounts are top-level organizational units in ByteFreezer.

#### List All Accounts

```bash
GET /api/v1/accounts
Authorization: Bearer <token>
```

**Response:**
```json
{
  "accounts": [
    {
      "id": "account-uuid",
      "name": "Production Account",
      "description": "Main production environment",
      "active": true,
      "created_at": "2025-01-01T00:00:00Z"
    }
  ]
}
```

#### Create Account

```bash
POST /api/v1/accounts
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "New Account",
  "description": "Account description",
  "active": true
}
```

### Tenants

Tenants represent isolated data spaces within an account.

#### List All Tenants

```bash
GET /api/v1/tenants
Authorization: Bearer <token>
```

#### List Tenants for Account

```bash
GET /api/v1/accounts/{accountId}/tenants
Authorization: Bearer <token>
```

#### Create Tenant

```bash
POST /api/v1/accounts/{accountId}/tenants
Authorization: Bearer <token>
Content-Type: application/json

{
  "id": "my-tenant-id",
  "name": "Production Tenant",
  "description": "Main production tenant",
  "active": true
}
```

### Datasets

Datasets are collections of data within a tenant.

#### List All Datasets

```bash
GET /api/v1/datasets
Authorization: Bearer <token>
```

**Response:**
```json
{
  "items": [
    {
      "id": "dataset-id",
      "tenant_id": "tenant-id",
      "name": "Web Logs",
      "description": "Production web server logs",
      "active": true,
      "created_at": "2025-01-01T00:00:00Z"
    }
  ]
}
```

#### List Datasets for Tenant

```bash
GET /api/v1/tenants/{tenantId}/datasets
Authorization: Bearer <token>
```

#### Create Dataset

```bash
POST /api/v1/tenants/{tenantId}/datasets
Authorization: Bearer <token>
Content-Type: application/json

{
  "id": "my-dataset",
  "name": "Application Logs",
  "description": "Application event logs",
  "active": true
}
```

#### Update Dataset

```bash
PUT /api/v1/tenants/{tenantId}/datasets/{datasetId}
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "Updated Name",
  "description": "Updated description",
  "active": true
}
```

## Data Transformations

ByteFreezer provides powerful data transformation capabilities via the Piper service.

### Understanding Transformations

Transformations are defined as a JSON array of filter configurations:

```json
[
  {
    "type": "grok",
    "config": {
      "source_field": "message",
      "pattern": "%{IP:client_ip} %{WORD:method} %{URIPATH:request}"
    },
    "enabled": true,
    "notes": "Parse web server access log format"
  },
  {
    "type": "geoip",
    "config": {
      "source_field": "client_ip",
      "target_field": "geo"
    },
    "enabled": true,
    "notes": "Add geographic data for client IPs"
  }
]
```

### Available Filter Types

- **grok** - Parse unstructured text using grok patterns
- **mutate** - Add, remove, or rename fields
- **drop** - Conditionally drop events
- **kv** - Parse key-value pairs
- **date_parse** - Parse date/time strings
- **split** - Split arrays into separate events
- **useragent** - Parse user agent strings
- **dns** - Perform DNS lookups
- **geoip** - Add geographic information for IP addresses
- **fingerprint** - Generate hash fingerprints for deduplication
- **regex_replace** - Find and replace using regex
- **include** - Keep only matching events
- **exclude** - Drop matching events
- **sample** - Random sampling
- **passthrough** - Pass data through without modification

### Get Dataset Schema

Retrieve the current schema and sample data for a dataset:

```bash
GET /api/v1/tenants/{tenantId}/datasets/{datasetId}/transformations/schema
Authorization: Bearer <token>
```

**Response:**
```json
{
  "schema": [
    {
      "name": "timestamp",
      "type": "string",
      "count": 1000,
      "nullable": false,
      "sample": "2025-01-01T12:00:00Z"
    },
    {
      "name": "client_ip",
      "type": "string",
      "count": 1000,
      "nullable": false,
      "sample": "192.168.1.100"
    }
  ],
  "samples": [
    {
      "line_number": 1,
      "raw_data": "{\"timestamp\":\"2025-01-01T12:00:00Z\",\"client_ip\":\"192.168.1.100\"}",
      "parsed_data": {
        "timestamp": "2025-01-01T12:00:00Z",
        "client_ip": "192.168.1.100"
      }
    }
  ],
  "schema_updated_at": "2025-01-01T12:00:00Z",
  "dataset_enabled": true
}
```

### Test Transformation

Test a transformation configuration without deploying it:

```bash
POST /api/v1/tenants/{tenantId}/datasets/{datasetId}/transformations/test
Authorization: Bearer <token>
Content-Type: application/json

{
  "filters": [
    {
      "type": "grok",
      "config": {
        "source_field": "message",
        "pattern": "%{IP:client_ip}"
      },
      "enabled": true
    }
  ]
}
```

**Response:**
```json
{
  "job_id": "test-job-uuid",
  "tenant_id": "tenant-id",
  "dataset_id": "dataset-id",
  "job_type": "test",
  "status": "pending",
  "created_at": "2025-01-01T12:00:00Z"
}
```

### Check Job Status

Poll for transformation job results:

```bash
GET /api/v1/transformations/jobs/{jobId}
Authorization: Bearer <token>
```

**Response:**
```json
{
  "response": {
    "job_id": "test-job-uuid",
    "status": "completed",
    "result": {
      "results": [
        {
          "input": {"message": "192.168.1.100"},
          "output": {"message": "192.168.1.100", "client_ip": "192.168.1.100"},
          "applied": ["grok"],
          "skipped": false,
          "duration_ms": 1.5
        }
      ],
      "success_count": 1,
      "error_count": 0,
      "skipped_count": 0,
      "total_time_ms": 1.5
    }
  }
}
```

### Validate on Fresh Data

Validate transformation on recent production data:

```bash
POST /api/v1/tenants/{tenantId}/datasets/{datasetId}/transformations/validate
Authorization: Bearer <token>
Content-Type: application/json

{
  "filters": [
    {
      "type": "grok",
      "config": {
        "source_field": "message",
        "pattern": "%{IP:client_ip}"
      },
      "enabled": true,
      "notes": "Extract client IP from message field"
    }
  ],
  "count": 100
}
```

### Activate Transformation

Deploy transformation to production:

```bash
POST /api/v1/tenants/{tenantId}/datasets/{datasetId}/transformations/activate
Authorization: Bearer <token>
Content-Type: application/json

{
  "filters": [
    {
      "type": "grok",
      "config": {
        "source_field": "message",
        "pattern": "%{IP:client_ip}"
      },
      "enabled": true,
      "notes": "Production IP extraction"
    }
  ],
  "enabled": true
}
```

### Get Transformation Stats

View transformation performance metrics:

```bash
GET /api/v1/tenants/{tenantId}/datasets/{datasetId}/transformations/stats
Authorization: Bearer <token>
```

**Response:**
```json
{
  "stats": {
    "tenant_id": "tenant-id",
    "dataset_id": "dataset-id",
    "enabled": true,
    "filter_count": 3,
    "total_processed": 10000,
    "success_count": 9950,
    "error_count": 50,
    "skipped_count": 0,
    "avg_rows_per_sec": 1500.5,
    "last_processed": "2025-01-01T12:00:00Z"
  }
}
```

### Get Filter Catalog

Retrieve all available filter types with parameters and examples:

```bash
GET /api/v1/piper/filters
Authorization: Bearer <token>
```

**Response:**
```json
{
  "filters": [
    {
      "filter_type": "grok",
      "display_name": "Grok Pattern Parser",
      "category": "parsing",
      "purpose": "Parse unstructured log data using grok patterns",
      "parameters": [
        {
          "name": "source_field",
          "type": "string",
          "required": true,
          "description": "Field containing text to parse"
        },
        {
          "name": "pattern",
          "type": "string",
          "required": true,
          "description": "Grok pattern to match"
        }
      ],
      "examples": [
        {
          "description": "Parse Apache access log",
          "config": {
            "source_field": "message",
            "pattern": "%{COMBINEDAPACHELOG}"
          }
        }
      ]
    }
  ]
}
```

## AI Pipeline Assistant

ByteFreezer includes an AI-powered pipeline assistant to generate transformation configurations.

### Generate Transformation Pipeline

```bash
POST /api/v1/ai/pipeline/generate
Authorization: Bearer <token>
Content-Type: application/json

{
  "tenant_id": "tenant-id",
  "dataset_id": "dataset-id",
  "prompt": "I want to parse nginx access logs, extract client IPs, and add geoip data for non-local IPs",
  "schema": [
    {
      "name": "message",
      "type": "string"
    }
  ],
  "samples": [
    {
      "line_number": 1,
      "parsed_data": {
        "message": "192.168.1.100 - - [01/Jan/2025:12:00:00 +0000] \"GET /index.html HTTP/1.1\" 200 1234"
      }
    }
  ]
}
```

**Response:**
```json
{
  "filters": [
    {
      "type": "grok",
      "config": {
        "source_field": "message",
        "pattern": "%{IP:client_ip} - - \\[%{HTTPDATE:timestamp}\\] \"%{WORD:method} %{URIPATHPARAM:request} HTTP/%{NUMBER:http_version}\" %{NUMBER:status} %{NUMBER:bytes}"
      },
      "enabled": true,
      "notes": "Parse nginx access log format"
    },
    {
      "type": "geoip",
      "config": {
        "source_field": "client_ip",
        "target_field": "geo",
        "skip_private": true
      },
      "enabled": true,
      "notes": "Add geographic data for public IPs only"
    }
  ],
  "explanation": "This pipeline first parses the nginx access log to extract fields, then enriches public IP addresses with geographic data."
}
```

## Error Handling

All API endpoints return standard HTTP status codes:

- **200 OK** - Request successful
- **201 Created** - Resource created successfully
- **400 Bad Request** - Invalid request parameters
- **401 Unauthorized** - Missing or invalid authentication
- **403 Forbidden** - Insufficient permissions
- **404 Not Found** - Resource not found
- **429 Too Many Requests** - Rate limit exceeded
- **500 Internal Server Error** - Server error

Error responses include details:

```json
{
  "error": "Invalid transformation configuration",
  "details": "Filter type 'unknown_filter' is not recognized"
}
```

## Rate Limiting

API requests are rate-limited per token. Default limits:
- **100 requests per minute** for standard users
- **1000 requests per minute** for service accounts

Rate limit headers are included in responses:
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1704110400
```

## Pagination

List endpoints support pagination:

```bash
GET /api/v1/datasets?page=1&page_size=50
```

## Complete Workflow Example

Here's a complete example of an AI agent creating and deploying a transformation:

```python
import requests
import time

BASE_URL = "https://bytefreezer.example.com/api/v1"

# 1. Authenticate
auth_response = requests.post(f"{BASE_URL}/login", json={
    "email": "ai-agent@example.com",
    "password": "secure-password"
})
token = auth_response.json()["access_token"]
headers = {"Authorization": f"Bearer {token}"}

# 2. Get schema and samples
schema_response = requests.get(
    f"{BASE_URL}/tenants/my-tenant/datasets/web-logs/transformations/schema",
    headers=headers
)
schema_data = schema_response.json()

# 3. Generate transformation using AI
ai_response = requests.post(
    f"{BASE_URL}/ai/pipeline/generate",
    headers=headers,
    json={
        "tenant_id": "my-tenant",
        "dataset_id": "web-logs",
        "prompt": "Extract client IPs and add geoip data",
        "schema": schema_data["schema"],
        "samples": schema_data["samples"][:5]
    }
)
filters = ai_response.json()["filters"]

# 4. Test transformation
test_response = requests.post(
    f"{BASE_URL}/tenants/my-tenant/datasets/web-logs/transformations/test",
    headers=headers,
    json={"filters": filters}
)
job_id = test_response.json()["job_id"]

# 5. Poll for test results
while True:
    job_response = requests.get(
        f"{BASE_URL}/transformations/jobs/{job_id}",
        headers=headers
    )
    status = job_response.json()["response"]["status"]

    if status == "completed":
        result = job_response.json()["response"]["result"]
        if result["error_count"] == 0:
            print(f"Test successful: {result['success_count']} records processed")
            break
        else:
            print(f"Test failed: {result['error_count']} errors")
            break
    elif status == "failed":
        print("Test job failed")
        break

    time.sleep(1)

# 6. Deploy to production
activate_response = requests.post(
    f"{BASE_URL}/tenants/my-tenant/datasets/web-logs/transformations/activate",
    headers=headers,
    json={
        "filters": filters,
        "enabled": true
    }
)
print("Transformation deployed:", activate_response.json())
```

## Additional Resources

- **OpenAPI Schema**: Available at `/api/v1/openapi.json`
- **Swagger UI**: Available at `/api/v1/docs`
- **Filter Documentation**: See `/docs/filters/` directory for detailed filter documentation

## Support

For API support and questions:
- Documentation: https://docs.bytefreezer.com
- Issues: https://github.com/bytefreezer/bytefreezer/issues
