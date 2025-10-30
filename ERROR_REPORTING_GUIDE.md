# ByteFreezer Error Reporting System - Implementation Guide

## Overview

The ByteFreezer Error Reporting System provides centralized error tracking across all ecosystem components with automatic deduplication, adaptive sampling, and account-based filtering.

## Architecture

### Components

```
┌─────────────────┐
│   Proxy (tp1)   │──┐
├─────────────────┤  │
│   Piper (tp1)   │──┤
├─────────────────┤  │   Account JWT Auth
│ Receiver (tp2)  │──┤   ┌──────────────────────┐
├─────────────────┤  ├──→│  Control Service     │
│  Packer (tp2)   │──┤   │  (tp3)               │
├─────────────────┤  │   │  - Error Reporting   │
│   SOC (tp3)     │──┘   │  - Deduplication     │
└─────────────────┘      │  - Adaptive Sampling │
                         └──────────────────────┘
                                    ↓
                         ┌──────────────────────┐
                         │  PostgreSQL          │
                         │  system_errors table │
                         └──────────────────────┘
```

## Database Schema

### system_errors Table

```sql
CREATE TABLE system_errors (
    id BIGSERIAL PRIMARY KEY,
    error_hash VARCHAR(64) NOT NULL UNIQUE,  -- SHA256 hash for deduplication
    error_type VARCHAR(100) NOT NULL,
    component VARCHAR(50) NOT NULL,          -- proxy, receiver, piper, packer, control, soc
    tenant_id VARCHAR(255),
    dataset_id VARCHAR(255),
    error_message TEXT NOT NULL,
    error_sample JSONB,
    severity VARCHAR(20) NOT NULL DEFAULT 'error',  -- debug, info, warning, error, critical
    status VARCHAR(20) NOT NULL DEFAULT 'active',   -- active, resolved, ignored
    occurrence_count BIGINT NOT NULL DEFAULT 1,
    first_seen TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    last_seen TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    sample_rate FLOAT NOT NULL DEFAULT 1.0,
    samples_collected INTEGER NOT NULL DEFAULT 1,
    samples_dropped INTEGER NOT NULL DEFAULT 0,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMP WITH TIME ZONE
);
```

### Adaptive Sampling

The `upsert_system_error()` function automatically adjusts sampling based on occurrence count:

- **0-99 occurrences**: 100% sampling (sample_rate = 1.0)
- **100-999 occurrences**: 10% sampling (sample_rate = 0.1)
- **1,000-9,999 occurrences**: 1% sampling (sample_rate = 0.01)
- **10,000+ occurrences**: 0.1% sampling (sample_rate = 0.001)

This prevents database flooding while maintaining visibility into error patterns.

## API Endpoints

### System Services (Packer, Receiver, Piper, SOC)

**Authentication**: System API key via `Authorization: Bearer {api_key}`

#### Report Error
```http
POST /api/v1/errors
Content-Type: application/json
Authorization: Bearer {system_api_key}

{
  "error_type": "s3_upload_failure",
  "component": "packer",
  "tenant_id": "tenant-001",
  "dataset_id": "dataset-001",
  "error_message": "Failed to upload file to S3",
  "error_sample": {
    "bucket": "destination-bucket",
    "key": "path/to/file.parquet",
    "error": "AccessDenied"
  },
  "severity": "error",
  "metadata": {
    "file_size": 1048576,
    "retry_count": 3
  }
}
```

#### List All Errors (System Admin Only)
```http
GET /api/v1/errors?component=packer&severity=error&status=active&limit=100
Authorization: Bearer {admin_jwt_token}
```

#### Get Error Statistics
```http
GET /api/v1/errors/stats
Authorization: Bearer {admin_jwt_token}
```

### Account-Scoped Services (Proxy)

**Authentication**: Account JWT token (same as health reporting)

#### Report Error
```http
POST /api/v1/accounts/{accountId}/errors
Content-Type: application/json
Authorization: Bearer {account_jwt_token}

{
  "error_type": "udp_parse_failure",
  "component": "proxy",
  "tenant_id": "tenant-001",
  "dataset_id": "dataset-001",
  "error_message": "Failed to parse UDP packet",
  "severity": "warning"
}
```

#### List Account Errors
```http
GET /api/v1/accounts/{accountId}/errors?severity=error&status=active
Authorization: Bearer {account_jwt_token}
```

#### Get Account Error Statistics
```http
GET /api/v1/accounts/{accountId}/errors/stats
Authorization: Bearer {account_jwt_token}
```

#### List Dataset Errors
```http
GET /api/v1/accounts/{accountId}/datasets/{datasetId}/errors
Authorization: Bearer {account_jwt_token}
```

## Client Implementation

### System Services (Go)

**File**: `errors/error_reporter.go`

```go
import "github.com/n0needt0/bytefreezer-{service}/errors"

// Initialize error reporter
reporter := errors.NewErrorReporter(
    controlURL,      // e.g., "http://192.168.86.103:8082"
    apiKey,          // System API key
    "packer",        // Component name
    true,            // Enabled
)

// Report simple error
ctx := context.Background()
err := reporter.ReportErrorSimple(
    ctx,
    "processing_failure",     // error_type
    "Failed to process file", // error_message
    "error",                  // severity
    "tenant-001",             // tenant_id
    "dataset-001",            // dataset_id
)

// Report critical error
err := reporter.ReportCritical(
    ctx,
    "database_connection_lost",
    "PostgreSQL connection failed",
    "",  // tenant_id (optional)
    "",  // dataset_id (optional)
)

// Report with sample data
sample := map[string]interface{}{
    "file_path": "/path/to/file.ndjson",
    "line_number": 12345,
    "error_detail": "Invalid JSON",
}
err := reporter.ReportErrorWithSample(
    ctx,
    "json_parse_error",
    "Failed to parse JSON line",
    "warning",
    "tenant-001",
    "dataset-001",
    sample,
)
```

### Proxy (Account-Scoped)

**File**: `errors/error_reporter.go`

```go
// Initialize error reporter with account credentials
reporter := errors.NewErrorReporter(
    controlURL,      // e.g., "http://192.168.86.103:8082"
    accountID,       // Account ID
    jwtToken,        // Account JWT token (same as health reporting)
    "proxy",         // Component name
    true,            // Enabled
)

// Usage is identical to system services
err := reporter.ReportErrorSimple(
    ctx,
    "udp_packet_dropped",
    "Dropped malformed UDP packet",
    "warning",
    tenantID,
    datasetID,
)
```

## Configuration

### Enable Error Reporting

**config.yaml**:
```yaml
error_tracking:
  enabled: true

control_service:
  enabled: true
  base_url: "http://192.168.86.103:8082"
  api_key: "your-system-api-key"  # For system services
  timeout_seconds: 30
```

### Proxy Configuration

Proxy uses the account JWT token from config polling (no separate API key needed).

## UI Implementation

### System Errors Page

**URL**: `/dashboard/errors`

**Features**:
- Real-time error display with auto-refresh (30s)
- Statistics cards: Total Errors, Total Occurrences, Critical Count, Active Count
- Filtering: Severity, Component, Status
- Expandable error details with samples and metadata
- Account-based filtering (non-admin users see only their errors)
- Responsive design with dark mode support

**Navigation**: Added to sidebar as "System Errors" with AlertCircle icon

### Statistics Page Integration

The `total_errors` field from the aggregated metrics response can be linked to the errors page:

```typescript
// In statistics page, link total_errors to error page
<Link href="/dashboard/errors">
  <div className="stat-card">
    <p className="text-2xl font-bold">{formatNumber(totalErrors)}</p>
    <p className="text-sm">Total Errors</p>
  </div>
</Link>
```

## Best Practices

### When to Report Errors

**DO Report**:
- Processing failures (file corruption, parsing errors)
- External service failures (S3, database, API calls)
- Configuration errors
- Resource exhaustion (disk full, memory limits)
- Security issues (authentication failures, unauthorized access)

**DON'T Report**:
- Expected validation failures (user input errors)
- Successful operations
- Debug logging
- High-frequency events that don't indicate problems

### Error Types

Use descriptive, consistent error types:
- `s3_upload_failure`
- `database_connection_lost`
- `json_parse_error`
- `authentication_failed`
- `resource_exhausted`
- `configuration_invalid`

### Severity Levels

- **critical**: Service is down or data loss is occurring
- **error**: Operation failed but service continues
- **warning**: Degraded operation or potential issue
- **info**: Informational events
- **debug**: Detailed debugging information

### Error Samples

Include relevant context but avoid:
- Sensitive data (passwords, tokens, PII)
- Large payloads (truncate to ~1KB)
- Binary data

Good sample:
```json
{
  "file_path": "/data/tenant-001/file.ndjson",
  "line_number": 12345,
  "error_detail": "Expected opening bracket",
  "file_size": 1048576
}
```

## Monitoring & Alerts

### Key Metrics to Monitor

1. **Critical Error Count**: Alert if > 0
2. **Active Error Growth**: Alert if increasing rapidly
3. **Error Rate by Component**: Identify problem services
4. **Sampling Rate**: If many errors have sample_rate < 0.1, system may be overwhelmed

### SOC Integration

The SOC service can query error statistics and send alerts:

```go
stats, err := errorReporter.GetErrorStats(ctx, "")
if stats.CriticalCount > 0 {
    socClient.SendCriticalAlert(
        "Critical Errors Detected",
        fmt.Sprintf("Found %d critical errors", stats.CriticalCount),
        "Immediate attention required",
    )
}
```

## Troubleshooting

### Errors Not Appearing

1. **Check configuration**: Verify `error_tracking.enabled = true`
2. **Check network**: Ensure service can reach control service
3. **Check authentication**: Verify API key or JWT token is valid
4. **Check logs**: Look for error reporting failures in service logs

### High Sample Drop Rate

If `samples_dropped` is very high relative to `samples_collected`:
- This is normal for high-frequency errors (by design)
- Check if the same error is occurring excessively
- Investigate root cause of the error

### Performance Impact

Error reporting is designed to be lightweight:
- Async reporting (doesn't block operations)
- Adaptive sampling prevents database overload
- 10-second timeout prevents hanging
- Deduplication reduces storage

Typical overhead: <1ms per error report

## Migration Checklist

- [x] Database migrations 11-14 applied
- [x] Control service updated and restarted
- [x] Error reporter client added to packer
- [x] Error reporter client added to receiver
- [x] Error reporter client added to piper
- [x] Error reporter client added to proxy (account-scoped)
- [x] UI System Errors page implemented
- [x] Navigation updated with Errors link
- [ ] Configuration updated in all services
- [ ] Services restarted with error reporting enabled
- [ ] End-to-end testing completed

## Support

For issues or questions:
1. Check control service logs: `journalctl -u bytefreezer-control -f`
2. Check database: `SELECT COUNT(*) FROM system_errors WHERE status = 'active'`
3. Verify API endpoints: `curl http://control-url/api/v1/errors -H "Authorization: Bearer {token}"`
