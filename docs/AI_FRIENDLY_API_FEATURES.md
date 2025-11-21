# AI-Friendly API Features

ByteFreezer API includes several features specifically designed for AI agent consumption.

## API Discovery Endpoint

The root API endpoint provides comprehensive discovery information:

```bash
GET /api/v1
```

**Response:**
```json
{
  "version": "1.0.0",
  "documentation": {
    "comprehensive_api": "/api/v1/docs/ai-agent",
    "quick_reference": "/api/v1/docs/ai-agent/quick-reference",
    "filter_catalog": "/api/v1/filters/catalog",
    "openapi_spec": "/api/v1/openapi.json",
    "swagger_ui": "/v1/docs"
  },
  "endpoints": {
    "authentication": [
      "/api/v1/login",
      "/api/v1/refresh",
      "/api/v1/password-reset"
    ],
    "resources": [
      "/api/v1/accounts",
      "/api/v1/tenants",
      "/api/v1/datasets",
      "/api/v1/users"
    ],
    "transformations": [
      "/api/v1/tenants/{tenantId}/datasets/{datasetId}/transformations/schema",
      "/api/v1/tenants/{tenantId}/datasets/{datasetId}/transformations/test",
      "/api/v1/tenants/{tenantId}/datasets/{datasetId}/transformations/validate",
      "/api/v1/tenants/{tenantId}/datasets/{datasetId}/transformations/activate",
      "/api/v1/transformations/jobs/{jobId}"
    ],
    "ai_assistant": [
      "/api/v1/ai/pipeline/generate",
      "/api/v1/ai/pipeline/validate",
      "/api/v1/filters/catalog"
    ],
    "monitoring": [
      "/api/v1/health",
      "/api/v1/stats",
      "/api/v1/ecosystem/health"
    ]
  },
  "rate_limit": {
    "default_limit": 100,
    "service_limit": 1000,
    "window_duration": "1 minute"
  },
  "features": [
    "async_job_processing",
    "ai_pipeline_generation",
    "real_time_transformation_testing",
    "comprehensive_filter_catalog",
    "rate_limiting",
    "jwt_authentication",
    "openapi_specification"
  ]
}
```

## Enhanced Error Responses

All API errors follow a consistent, informative format:

```json
{
  "error": {
    "code": "INVALID_FILTER_TYPE",
    "message": "Filter type 'unknown_filter' is not recognized",
    "details": {
      "requested_type": "unknown_filter",
      "available_types": ["grok", "mutate", "drop", "geoip", "passthrough"],
      "similar_types": ["drop", "sample"]
    },
    "suggestion": "Use GET /api/v1/filters/catalog to see all available filter types with complete documentation and examples",
    "documentation_url": "/api/v1/docs/ai-agent#filter-types",
    "request_id": "req-abc-123"
  }
}
```

### Error Code Reference

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `INVALID_REQUEST` | 400 | Request format or content is invalid |
| `UNAUTHORIZED` | 401 | Missing or invalid authentication |
| `FORBIDDEN` | 403 | Insufficient permissions |
| `NOT_FOUND` | 404 | Resource not found |
| `CONFLICT` | 409 | Resource conflict (e.g., duplicate) |
| `RATE_LIMIT_EXCEEDED` | 429 | Too many requests |
| `INTERNAL_ERROR` | 500 | Server error |
| `INVALID_FILTER_TYPE` | 400 | Unknown transformation filter type |
| `INVALID_JSON` | 400 | JSON parsing failed |
| `MISSING_PARAMETER` | 400 | Required parameter not provided |
| `INVALID_PARAMETER` | 400 | Parameter value is invalid |
| `JOB_NOT_FOUND` | 404 | Transformation job not found |
| `JOB_FAILED` | 500 | Transformation job failed |
| `VALIDATION_FAILED` | 400 | Input validation failed |

## Standardized Response Format

API responses include metadata for better context:

```json
{
  "data": {
    "items": [
      {"id": "dataset-1", "name": "Web Logs"},
      {"id": "dataset-2", "name": "App Metrics"}
    ]
  },
  "meta": {
    "endpoint": "/api/v1/datasets",
    "method": "GET",
    "version": "1.0.0",
    "request_id": "req-xyz-789",
    "pagination": {
      "page": 1,
      "page_size": 50,
      "total_items": 150,
      "total_pages": 3,
      "has_next": true,
      "has_prev": false,
      "next_url": "/api/v1/datasets?page=2&page_size=50",
      "prev_url": null
    }
  }
}
```

## Rate Limiting Headers

All responses include rate limit information:

```http
HTTP/1.1 200 OK
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1704110400
Content-Type: application/json
```

When rate limit is exceeded:

```http
HTTP/1.1 429 Too Many Requests
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1704110460
Retry-After: 60
```

## Asynchronous Job Pattern

For long-running operations (transformations, validation):

### 1. Submit Job

```bash
POST /api/v1/tenants/my-tenant/datasets/logs/transformations/test
{
  "filters": [...]
}
```

**Response (201 Created):**
```json
{
  "job_id": "test-abc-123",
  "tenant_id": "my-tenant",
  "dataset_id": "logs",
  "job_type": "test",
  "status": "pending",
  "created_at": "2025-01-21T12:00:00Z",
  "poll_url": "/api/v1/transformations/jobs/test-abc-123",
  "estimated_duration": "2-5 seconds"
}
```

### 2. Poll for Results

```bash
GET /api/v1/transformations/jobs/test-abc-123
```

**Response (In Progress):**
```json
{
  "response": {
    "job_id": "test-abc-123",
    "status": "processing",
    "processor_id": "piper-1",
    "started_at": "2025-01-21T12:00:01Z",
    "progress": {
      "current": 50,
      "total": 100
    }
  }
}
```

**Response (Completed):**
```json
{
  "response": {
    "job_id": "test-abc-123",
    "status": "completed",
    "result": {
      "results": [...],
      "success_count": 100,
      "error_count": 0,
      "total_time_ms": 1250.5
    },
    "completed_at": "2025-01-21T12:00:05Z"
  }
}
```

## Best Practices for AI Agents

### 1. Start with Discovery
```python
# First request - discover API capabilities
response = requests.get("https://api.bytefreezer.com/api/v1")
api_info = response.json()

# Use discovered endpoints
docs_url = api_info["documentation"]["comprehensive_api"]
```

### 2. Handle Errors Gracefully
```python
response = requests.post(url, json=data)

if response.status_code != 200:
    error = response.json()["error"]

    # Use structured error information
    print(f"Error: {error['message']}")

    # Follow suggestions
    if error.get("suggestion"):
        print(f"Suggestion: {error['suggestion']}")

    # Check documentation
    if error.get("documentation_url"):
        docs = requests.get(error["documentation_url"])
```

### 3. Respect Rate Limits
```python
response = requests.get(url)

# Check remaining requests
remaining = int(response.headers.get("X-RateLimit-Remaining", 0))
reset_time = int(response.headers.get("X-RateLimit-Reset", 0))

if remaining < 10:
    wait_time = reset_time - time.time()
    time.sleep(max(0, wait_time))
```

### 4. Poll Async Jobs Efficiently
```python
job_id = create_job()

max_attempts = 60  # 1 minute with 1s intervals
for attempt in range(max_attempts):
    job = get_job_status(job_id)

    if job["status"] == "completed":
        return job["result"]
    elif job["status"] == "failed":
        error = job.get("error_message")
        raise Exception(f"Job failed: {error}")

    time.sleep(1)
```

### 5. Use Pagination
```python
all_items = []
page = 1

while True:
    response = requests.get(f"{url}?page={page}&page_size=50")
    data = response.json()

    all_items.extend(data["data"]["items"])

    if not data["meta"]["pagination"]["has_next"]:
        break

    # Or use next_url directly
    next_url = data["meta"]["pagination"]["next_url"]
    page += 1
```

## Public Endpoints (No Authentication Required)

These endpoints are accessible without a JWT token:

- `GET /api/v1` - API discovery
- `GET /api/v1/health` - Health check
- `GET /api/v1/docs` - Documentation index
- `GET /api/v1/docs/ai-agent` - Comprehensive API docs
- `GET /api/v1/docs/ai-agent/quick-reference` - Quick reference
- `POST /api/v1/login` - Authentication
- `POST /api/v1/refresh` - Token refresh
- `POST /api/v1/password-reset` - Password reset request

All other endpoints require authentication via `Authorization: Bearer <token>` header.

## Example: Complete AI Agent Workflow

```python
import requests
import time

BASE_URL = "https://api.bytefreezer.com/api/v1"

# 1. Discover API
discovery = requests.get(BASE_URL).json()
print(f"API Version: {discovery['version']}")
print(f"Features: {', '.join(discovery['features'])}")

# 2. Authenticate
auth = requests.post(f"{BASE_URL}/login", json={
    "email": "ai-agent@example.com",
    "password": "secure-password"
}).json()

headers = {"Authorization": f"Bearer {auth['access_token']}"}

# 3. Get transformation schema
schema = requests.get(
    f"{BASE_URL}/tenants/prod/datasets/logs/transformations/schema",
    headers=headers
).json()

# 4. Generate transformation using AI
ai_response = requests.post(
    f"{BASE_URL}/ai/pipeline/generate",
    headers=headers,
    json={
        "tenant_id": "prod",
        "dataset_id": "logs",
        "prompt": "Extract client IPs and add geoip data",
        "schema": schema["schema"],
        "samples": schema["samples"][:5]
    }
).json()

filters = ai_response["filters"]

# 5. Test transformation
test = requests.post(
    f"{BASE_URL}/tenants/prod/datasets/logs/transformations/test",
    headers=headers,
    json={"filters": filters}
).json()

# 6. Poll for results
job_id = test["job_id"]
for _ in range(60):
    job = requests.get(
        f"{BASE_URL}/transformations/jobs/{job_id}",
        headers=headers
    ).json()["response"]

    if job["status"] == "completed":
        print(f"Test passed: {job['result']['success_count']} records")
        break
    elif job["status"] == "failed":
        print(f"Test failed: {job.get('error_message')}")
        break

    time.sleep(1)

# 7. Deploy if successful
if job["status"] == "completed" and job["result"]["error_count"] == 0:
    deploy = requests.post(
        f"{BASE_URL}/tenants/prod/datasets/logs/transformations/activate",
        headers=headers,
        json={"filters": filters, "enabled": True}
    ).json()
    print(f"Deployed: {deploy}")
```

## Additional Resources

- **Comprehensive API Documentation**: `/api/v1/docs/ai-agent`
- **Quick Reference Guide**: `/api/v1/docs/ai-agent/quick-reference`
- **Filter Catalog**: `/api/v1/filters/catalog`
- **OpenAPI Specification**: `/api/v1/openapi.json`
- **Interactive API Explorer**: `/v1/docs`
