# ByteFreezer Control Client Library

Go client library for interacting with the ByteFreezer Control API. This library enables piper and packer services to manage state without direct database access, which is required for on-prem deployments.

## Installation

```bash
go get github.com/n0needt0/bytefreezer-control/client
```

## Configuration

```go
import "github.com/n0needt0/bytefreezer-control/client"

config := client.Config{
    BaseURL:        "http://192.168.86.103:8082",
    APIKey:         "bytefreezer-service-api-key-8f4a2d1b-3c5e-4f6a-9b8c-7d2e1f3a4b5c",
    TimeoutSeconds: 30,
}

controlClient := client.NewClient(config)
```

## Usage Examples

### Piper Operations

#### File Locks

```go
// Acquire a file lock
lock, err := controlClient.AcquireFileLock(ctx,
    "tenant-001",           // tenantID
    "dataset-001",          // datasetID
    "path/to/file.json",    // fileKey
    "piper-instance-1",     // lockedBy
    300,                    // lockDurationSeconds
)
if err != nil {
    log.Errorf("Failed to acquire lock: %v", err)
    return
}

// Check if a file is locked
lock, err := controlClient.CheckFileLock(ctx, "tenant-001", "dataset-001", "path/to/file.json")
if err != nil {
    log.Errorf("Failed to check lock: %v", err)
} else if lock != nil {
    log.Infof("File is locked by: %s", lock.LockedBy)
} else {
    log.Info("File is not locked")
}

// Release a file lock
err = controlClient.ReleaseFileLock(ctx,
    "tenant-001",
    "dataset-001",
    "path/to/file.json",
    "piper-instance-1",
)

// Cleanup expired locks
count, err := controlClient.CleanupExpiredFileLocks(ctx)
log.Infof("Cleaned up %d expired locks", count)

// Cleanup stale locks (heartbeat older than threshold)
count, err := controlClient.CleanupStaleFileLocks(ctx, 600) // 10 minutes
log.Infof("Cleaned up %d stale locks", count)
```

#### Job Records

```go
// Create a job record
job := &client.PiperJobRecord{
    JobID:         "job-12345",
    TenantID:      "tenant-001",
    DatasetID:     "dataset-001",
    Status:        "running",
    SourceFiles:   []string{"file1.json", "file2.json"},
    ProcessorType: "ndjson-processor",
    ProcessorID:   "piper-instance-1",
}

createdJob, err := controlClient.CreatePiperJob(ctx, job)

// Update job status
err = controlClient.UpdatePiperJobStatus(ctx,
    "job-12345",            // jobID
    "completed",            // status
    "",                     // errorMessage
    "output/file.parquet",  // outputFile
    1000,                   // recordsProcessed
)

// Get job by ID
job, err := controlClient.GetPiperJob(ctx, "job-12345")

// Get jobs by status
jobs, err := controlClient.GetPiperJobsByStatus(ctx, "running", 100)

// Get jobs for tenant
jobs, err := controlClient.GetPiperJobsForTenant(ctx, "tenant-001", 100)

// Cleanup old jobs
count, err := controlClient.CleanupOldPiperJobs(ctx, 30) // older than 30 days
```

#### Pipeline Configuration Cache

```go
// Cache pipeline configuration
config := map[string]interface{}{
    "format":      "ndjson",
    "compression": "gzip",
    "batch_size":  1000,
}

err = controlClient.CachePipelineConfiguration(ctx,
    "tenant-001",
    "dataset-001",
    config,
    3600, // TTL in seconds (1 hour)
)

// Get cached configuration
cachedConfig, err := controlClient.GetCachedPipelineConfiguration(ctx, "tenant-001", "dataset-001")
if err != nil {
    log.Errorf("Failed to get cached config: %v", err)
} else if cachedConfig != nil {
    log.Infof("Using cached config: %v", cachedConfig.Configuration)
} else {
    log.Info("No cached config found, fetching from control API...")
}

// Invalidate cached configuration
err = controlClient.InvalidatePipelineConfiguration(ctx, "tenant-001", "dataset-001")

// List all cached pipelines
configs, err := controlClient.ListCachedPipelines(ctx, 100)

// Cleanup expired cache entries
count, err := controlClient.CleanupExpiredPipelineCache(ctx)
```

#### Tenant Cache

```go
// Cache tenant information
tenantData := map[string]interface{}{
    "name":   "Customer A",
    "active": true,
    "config": map[string]interface{}{
        "retention_days": 90,
    },
}

err = controlClient.CacheTenant(ctx, "tenant-001", tenantData, 3600)

// Get all cached tenants
tenants, err := controlClient.GetCachedTenants(ctx, 100)

// Invalidate tenant cache
err = controlClient.InvalidateTenantCache(ctx, "tenant-001")  // specific tenant
err = controlClient.InvalidateTenantCache(ctx, "")            // all tenants

// Cleanup expired tenant cache
count, err := controlClient.CleanupExpiredTenantCache(ctx)
```

### Packer Operations

#### Tenant Locks

```go
// Acquire tenant lock
lock, err := controlClient.AcquireTenantLock(ctx,
    "tenant-001",          // tenantID
    "packer-instance-1",   // lockedBy
    300,                   // lockDurationSeconds
)

// Update heartbeat
err = controlClient.UpdateTenantLockHeartbeat(ctx, "tenant-001", "packer-instance-1")

// Check tenant lock
lock, err := controlClient.CheckTenantLock(ctx, "tenant-001")

// Release tenant lock
err = controlClient.ReleaseTenantLock(ctx, "tenant-001", "packer-instance-1")

// Cleanup expired locks
count, err := controlClient.CleanupExpiredTenantLocks(ctx)

// Clear all locks (use with caution)
count, err := controlClient.ClearAllTenantLocks(ctx)

// Cleanup stale locks
count, err := controlClient.CleanupStaleTenantLocks(ctx, 600) // 10 minutes
```

#### Parquet Metadata

```go
// Upsert parquet file metadata
metadata := &client.PackerParquetFileMetadata{
    TenantID:      "tenant-001",
    DatasetID:     "dataset-001",
    FilePath:      "s3://bucket/path/file.parquet",
    PartitionPath: "year=2024/month=11/day=12",
    FileSizeBytes: 1024000,
    RowCount:      1000,
    SchemaJSON: map[string]interface{}{
        "fields": []interface{}{
            map[string]interface{}{"name": "id", "type": "string"},
            map[string]interface{}{"name": "timestamp", "type": "int64"},
        },
    },
    ColumnStats:  map[string]interface{}{},
    FileChecksum: "abc123",
    InstanceID:   "packer-instance-1",
}

result, err := controlClient.UpsertParquetFileMetadata(ctx, metadata)
log.Infof("Upserted metadata with ID: %d, version: %d", result.ID, result.MetadataVersion)

// Get metadata by partition
files, err := controlClient.GetParquetFileMetadataByPartition(ctx,
    "tenant-001",
    "dataset-001",
    "year=2024/month=11/day=12",
    100, // limit
)

// Get all metadata for tenant/dataset
allFiles, err := controlClient.GetAllParquetFileMetadata(ctx, "tenant-001", "dataset-001", 1000)

// Delete metadata
err = controlClient.DeleteParquetFileMetadata(ctx,
    "tenant-001",
    "dataset-001",
    "s3://bucket/path/file.parquet",
)

// Cleanup orphaned metadata
count, err := controlClient.CleanupOrphanedParquetMetadata(ctx, "tenant-001", "dataset-001")

// Cleanup expired metadata
count, err := controlClient.CleanupExpiredParquetMetadata(ctx)
```

#### Metadata Generation Status

```go
// Update generation status
status := &client.PackerMetadataGenerationStatus{
    TenantID:          "tenant-001",
    DatasetID:         "dataset-001",
    PartitionPath:     "year=2024/month=11/day=12",
    FileCount:         10,
    TotalRows:         10000,
    TotalSizeBytes:    1024000,
    NeedsRegeneration: false,
    CurrentSchemaHash: "abc123",
    SchemaVersion:     1,
}

err = controlClient.UpdateMetadataGenerationStatus(ctx, status)

// Get generation status
status, err := controlClient.GetMetadataGenerationStatus(ctx,
    "tenant-001",
    "dataset-001",
    "year=2024/month=11/day=12",
)
```

#### Metadata Summary

```go
// Get metadata summary
summary, err := controlClient.GetParquetMetadataSummary(ctx,
    "tenant-001",
    "dataset-001",
    "year=2024/month=11/day=12",
)
if err != nil {
    log.Errorf("Failed to get summary: %v", err)
} else if summary != nil {
    log.Infof("Partition has %d files, %d total rows, %d bytes",
        summary.FileCount,
        summary.TotalRows,
        summary.TotalSizeBytes,
    )
}
```

### Account/Tenant/Dataset Operations

```go
// List accounts
accounts, err := controlClient.ListAccounts(ctx, 100)

// List tenants for account
tenants, err := controlClient.ListTenants(ctx, "account-001", 100)

// List datasets for tenant
datasets, err := controlClient.ListDatasets(ctx, "tenant-001", 100)
```

## Error Handling

All methods return errors that should be checked. Common error patterns:

```go
// Lock already held
lock, err := controlClient.AcquireFileLock(ctx, ...)
if err != nil {
    if strings.Contains(err.Error(), "409") {
        log.Info("File is already locked by another process")
    } else {
        log.Errorf("Failed to acquire lock: %v", err)
    }
}

// Not found
lock, err := controlClient.CheckFileLock(ctx, ...)
if err != nil {
    log.Errorf("Error checking lock: %v", err)
} else if lock == nil {
    log.Info("No lock exists")
}
```

## Authentication

All requests require a service API key. The key should be configured in the control service's configuration and passed to the client constructor.

The API key is sent as a Bearer token in the Authorization header:
```
Authorization: Bearer bytefreezer-service-api-key-8f4a2d1b-3c5e-4f6a-9b8c-7d2e1f3a4b5c
```

## Thread Safety

The client is safe for concurrent use from multiple goroutines. Each instance uses a shared http.Client with connection pooling.

## Timeouts

Configure timeouts using the `TimeoutSeconds` field in the Config struct. Default is 30 seconds if not specified.

## Best Practices

1. **Reuse Client Instances**: Create one client instance and reuse it throughout your application for connection pooling benefits.

2. **Lock Cleanup**: Run periodic cleanup of expired/stale locks to prevent lock table growth:
   ```go
   go func() {
       ticker := time.NewTicker(5 * time.Minute)
       for range ticker.C {
           controlClient.CleanupExpiredFileLocks(context.Background())
           controlClient.CleanupStaleFileLocks(context.Background(), 600)
       }
   }()
   ```

3. **Cache TTL**: Set appropriate TTL values for cached configurations based on update frequency.

4. **Error Handling**: Always check for errors and handle lock conflicts gracefully.

5. **Context Usage**: Pass request-specific contexts for proper cancellation and timeouts.

## Retry Logic

The current implementation does not include automatic retries. Applications should implement retry logic for transient failures:

```go
import "github.com/avast/retry-go"

err := retry.Do(
    func() error {
        return controlClient.AcquireFileLock(ctx, ...)
    },
    retry.Attempts(3),
    retry.Delay(time.Second),
)
```

## Migration from Direct Database Access

### Piper Service

Replace direct PostgreSQL calls in `storage/postgresql_state_manager.go`:

```go
// Before (direct DB):
err := stateManager.AcquireFileLock(ctx, lock)

// After (control API):
lock, err := controlClient.AcquireFileLock(ctx, tenantID, datasetID, fileKey, lockedBy, duration)
```

### Packer Service

Replace direct PostgreSQL calls in `storage/postgres_lock_client.go` and `storage/postgres_metadata_client.go`:

```go
// Before (direct DB):
err := lockClient.AcquireTenantLock(ctx, tenantID, lockedBy, duration)

// After (control API):
lock, err := controlClient.AcquireTenantLock(ctx, tenantID, lockedBy, duration)
```

## API Endpoints

All endpoints are under `/api/v1`:

**Piper Endpoints**:
- File Locks: `/piper/locks/files/*`
- Job Records: `/piper/jobs/*`
- Pipeline Cache: `/piper/cache/pipelines/*`
- Tenant Cache: `/piper/cache/tenants/*`

**Packer Endpoints**:
- Tenant Locks: `/packer/locks/tenants/*`
- Parquet Metadata: `/packer/metadata/files/*`
- Generation Status: `/packer/metadata/generation/status/*`
- Metadata Summary: `/packer/metadata/summary/*`

See API documentation at `http://control-api:8082/v1/docs` for full endpoint details.
