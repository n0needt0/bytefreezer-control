# Piper and Packer Control API Design

## Overview

This document outlines the API design for migrating piper and packer direct database operations to the control service. This enables on-prem deployments where piper/packer services cannot access the central PostgreSQL database directly.

## Authentication

All endpoints require authentication via the service API key (same as other control endpoints). System accounts can access all data; account-scoped requests filter by account_id.

## Piper API Endpoints

### 1. File Locks

**Acquire File Lock**
```
POST /api/v1/piper/locks/files
Body: {
  "tenant_id": "string",
  "dataset_id": "string",
  "file_key": "string",
  "locked_by": "string",  // instance ID
  "lock_duration_seconds": int
}
Response: {
  "lock_id": "string",
  "ttl": "timestamp",
  "success": bool
}
```

**Release File Lock**
```
DELETE /api/v1/piper/locks/files
Body: {
  "tenant_id": "string",
  "dataset_id": "string",
  "file_key": "string",
  "locked_by": "string"
}
Response: { "success": bool }
```

**Check File Lock**
```
GET /api/v1/piper/locks/files/{tenant_id}/{dataset_id}/{file_key}
Response: {
  "is_locked": bool,
  "lock": {
    "locked_by": "string",
    "lock_timestamp": "timestamp",
    "ttl": "timestamp"
  }
}
```

**Cleanup Expired File Locks**
```
DELETE /api/v1/piper/locks/files/cleanup/expired
Response: {
  "deleted_count": int
}
```

**Cleanup Stale File Locks**
```
DELETE /api/v1/piper/locks/files/cleanup/stale
Query: threshold_minutes=5
Response: {
  "deleted_count": int
}
```

### 2. Job Records

**Create Job Record**
```
POST /api/v1/piper/jobs
Body: {
  "job_id": "string",
  "tenant_id": "string",
  "dataset_id": "string",
  "status": "string",  // pending, processing, completed, failed
  "source_files": ["string"],
  "processor_type": "string",
  "processor_id": "string",
  "created_at": "timestamp"
}
Response: { "success": bool }
```

**Update Job Status**
```
PUT /api/v1/piper/jobs/{job_id}/status
Body: {
  "status": "string",
  "processor_id": "string",
  "output_file": "string",
  "error_message": "string",
  "records_processed": int
}
Response: { "success": bool }
```

**Get Jobs by Status**
```
GET /api/v1/piper/jobs?status={status}
Response: {
  "jobs": [{job_record}]
}
```

**Get Job by ID**
```
GET /api/v1/piper/jobs/{job_id}
Response: {job_record}
```

**Get Jobs for Tenant**
```
GET /api/v1/piper/jobs/tenant/{tenant_id}
Response: {
  "jobs": [{job_record}]
}
```

**Cleanup Old Jobs**
```
DELETE /api/v1/piper/jobs/cleanup/old
Query: older_than_days=30
Response: {
  "deleted_count": int
}
```

### 3. Pipeline Configuration Cache

**Cache Pipeline Configuration**
```
POST /api/v1/piper/cache/pipelines
Body: {
  "tenant_id": "string",
  "dataset_id": "string",
  "configuration": {object},
  "ttl_hours": int
}
Response: { "success": bool }
```

**Get Cached Pipeline Configuration**
```
GET /api/v1/piper/cache/pipelines/{tenant_id}/{dataset_id}
Response: {
  "configuration": {object},
  "cached_at": "timestamp",
  "expires_at": "timestamp"
}
```

**Invalidate Pipeline Configuration**
```
DELETE /api/v1/piper/cache/pipelines/{tenant_id}/{dataset_id}
Response: { "success": bool }
```

**Get All Cached Pipelines**
```
GET /api/v1/piper/cache/pipelines
Response: {
  "pipelines": [{
    "tenant_id": "string",
    "dataset_id": "string",
    "cached_at": "timestamp",
    "expires_at": "timestamp"
  }]
}
```

**Cleanup Expired Pipeline Configurations**
```
DELETE /api/v1/piper/cache/pipelines/cleanup/expired
Response: {
  "deleted_count": int
}
```

### 4. Tenant Cache

**Cache Tenant**
```
POST /api/v1/piper/cache/tenants
Body: {
  "tenant_id": "string",
  "tenant_data": {object},
  "ttl_hours": int
}
Response: { "success": bool }
```

**Get Cached Tenants**
```
GET /api/v1/piper/cache/tenants
Response: {
  "tenants": [{
    "tenant_id": "string",
    "tenant_data": {object},
    "cached_at": "timestamp",
    "expires_at": "timestamp"
  }]
}
```

**Invalidate Tenant Cache**
```
DELETE /api/v1/piper/cache/tenants
Response: { "success": bool }
```

**Cleanup Expired Tenant Cache**
```
DELETE /api/v1/piper/cache/tenants/cleanup/expired
Response: {
  "deleted_count": int
}
```

### 5. Transformation Jobs

**Create Transformation Job**
```
POST /api/v1/piper/transformations
Body: {
  "tenant_id": "string",
  "dataset_id": "string",
  "job_type": "string",  // test, validate, activate
  "configuration": {object},
  "source_file_key": "string",
  "requested_by": "string"
}
Response: {
  "job_id": "string"
}
```

**Claim Transformation Job**
```
POST /api/v1/piper/transformations/claim
Body: {
  "processor_id": "string"
}
Response: {
  "job": {transformation_job} or null
}
```

**Update Transformation Job**
```
PUT /api/v1/piper/transformations/{job_id}
Body: {
  "status": "string",
  "processor_id": "string",
  "output_data": {object},
  "error_message": "string",
  "records_processed": int
}
Response: { "success": bool }
```

**Get Transformation Jobs by Status**
```
GET /api/v1/piper/transformations?status={status}
Response: {
  "jobs": [{transformation_job}]
}
```

**Cleanup Completed Jobs**
```
DELETE /api/v1/piper/transformations/cleanup/completed
Query: older_than_hours=24
Response: {
  "deleted_count": int
}
```

## Packer API Endpoints

### 1. Tenant Locks

**Acquire Tenant Lock**
```
POST /api/v1/packer/locks/tenants
Body: {
  "tenant_id": "string",
  "locked_by": "string",  // instance ID
  "lock_duration_seconds": int
}
Response: {
  "lock": {
    "tenant_id": "string",
    "locked_by": "string",
    "lock_timestamp": "timestamp",
    "ttl": int64
  }
}
```

**Release Tenant Lock**
```
DELETE /api/v1/packer/locks/tenants/{tenant_id}
Body: {
  "locked_by": "string"
}
Response: { "success": bool }
```

**Update Heartbeat**
```
PUT /api/v1/packer/locks/tenants/{tenant_id}/heartbeat
Body: {
  "locked_by": "string"
}
Response: { "success": bool }
```

**Check Tenant Lock**
```
GET /api/v1/packer/locks/tenants/{tenant_id}
Response: {
  "is_locked": bool,
  "lock": {
    "tenant_id": "string",
    "locked_by": "string",
    "lock_timestamp": "timestamp",
    "ttl": int64
  }
}
```

**Cleanup Expired Locks**
```
DELETE /api/v1/packer/locks/tenants/cleanup/expired
Response: {
  "deleted_count": int
}
```

**Clear All Locks**
```
DELETE /api/v1/packer/locks/tenants/cleanup/all
Response: {
  "deleted_count": int
}
```

**Cleanup Stale Locks**
```
DELETE /api/v1/packer/locks/tenants/cleanup/stale
Query: threshold_minutes=5
Response: {
  "deleted_count": int
}
```

### 2. Parquet Metadata

**Upsert File Metadata**
```
POST /api/v1/packer/metadata/files
Body: {
  "tenant_id": "string",
  "dataset_id": "string",
  "file_path": "string",
  "partition_path": "string",
  "file_size_bytes": int64,
  "row_count": int64,
  "created_at": "timestamp",
  "last_modified": "timestamp",
  "schema_json": "string",
  "column_stats": "string",
  "file_checksum": "string",
  "instance_id": "string"
}
Response: {
  "id": int64,
  "version": int
}
```

**Get File Metadata by Partition**
```
GET /api/v1/packer/metadata/files/{tenant_id}/{dataset_id}?partition={partition_path}
Response: {
  "files": [{parquet_file_metadata}]
}
```

**Get All File Metadata**
```
GET /api/v1/packer/metadata/files/{tenant_id}/{dataset_id}/all
Response: {
  "files": [{parquet_file_metadata}]
}
```

**Delete File Metadata**
```
DELETE /api/v1/packer/metadata/files/{tenant_id}/{dataset_id}
Body: {
  "file_path": "string"
}
Response: { "success": bool }
```

**Cleanup Orphaned Metadata**
```
POST /api/v1/packer/metadata/files/cleanup/orphaned
Body: {
  "tenant_id": "string",
  "dataset_id": "string",
  "existing_files": ["string"]
}
Response: {
  "deleted_count": int
}
```

**Cleanup Expired Metadata**
```
DELETE /api/v1/packer/metadata/files/cleanup/expired
Response: {
  "files_deleted": int,
  "status_deleted": int
}
```

### 3. Metadata Generation Status

**Update Generation Status**
```
POST /api/v1/packer/metadata/generation/status
Body: {
  "tenant_id": "string",
  "dataset_id": "string",
  "partition_path": "string",
  "last_generated_at": "timestamp",
  "file_count": int,
  "total_rows": int64,
  "total_size_bytes": int64,
  "needs_regeneration": bool,
  "current_schema_hash": "string",
  "schema_version": int
}
Response: { "success": bool }
```

**Get Generation Status**
```
GET /api/v1/packer/metadata/generation/status/{tenant_id}/{dataset_id}/{partition_path}
Response: {
  "tenant_id": "string",
  "dataset_id": "string",
  "partition_path": "string",
  "last_generated_at": "timestamp",
  "file_count": int,
  "total_rows": int64,
  "total_size_bytes": int64,
  "needs_regeneration": bool,
  "current_schema_hash": "string",
  "schema_version": int
}
```

### 4. Metadata Summary

**Get Metadata Summary**
```
GET /api/v1/packer/metadata/summary/{tenant_id}/{dataset_id}/{partition_path}
Response: {
  "tenant_id": "string",
  "dataset_id": "string",
  "partition_path": "string",
  "file_count": int,
  "total_rows": int64,
  "total_size_bytes": int64,
  "first_file_created": "timestamp",
  "last_file_modified": "timestamp",
  "metadata_last_updated": "timestamp"
}
```

## Database Schema

### Piper Tables

```sql
-- File locks
CREATE TABLE IF NOT EXISTS piper_file_locks (
  lock_id SERIAL PRIMARY KEY,
  tenant_id VARCHAR(255) NOT NULL,
  dataset_id VARCHAR(255) NOT NULL,
  file_key TEXT NOT NULL,
  locked_by VARCHAR(255) NOT NULL,
  lock_timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
  last_heartbeat TIMESTAMP WITH TIME ZONE NOT NULL,
  ttl TIMESTAMP WITH TIME ZONE NOT NULL,
  UNIQUE(tenant_id, dataset_id, file_key)
);

-- Job records
CREATE TABLE IF NOT EXISTS piper_job_records (
  job_id VARCHAR(255) PRIMARY KEY,
  tenant_id VARCHAR(255) NOT NULL,
  dataset_id VARCHAR(255) NOT NULL,
  status VARCHAR(50) NOT NULL,
  source_files JSONB,
  processor_type VARCHAR(100),
  processor_id VARCHAR(255),
  output_file TEXT,
  error_message TEXT,
  records_processed BIGINT DEFAULT 0,
  created_at TIMESTAMP WITH TIME ZONE NOT NULL,
  updated_at TIMESTAMP WITH TIME ZONE NOT NULL
);

-- Pipeline configuration cache
CREATE TABLE IF NOT EXISTS piper_pipeline_configurations (
  config_key VARCHAR(512) PRIMARY KEY,
  tenant_id VARCHAR(255) NOT NULL,
  dataset_id VARCHAR(255) NOT NULL,
  configuration JSONB NOT NULL,
  cached_at TIMESTAMP WITH TIME ZONE NOT NULL,
  expires_at TIMESTAMP WITH TIME ZONE NOT NULL
);

-- Tenant cache
CREATE TABLE IF NOT EXISTS piper_tenants_cache (
  tenant_id VARCHAR(255) PRIMARY KEY,
  tenant_data JSONB NOT NULL,
  cached_at TIMESTAMP WITH TIME ZONE NOT NULL,
  expires_at TIMESTAMP WITH TIME ZONE NOT NULL
);

-- Transformation jobs (already exists in control)
```

### Packer Tables

```sql
-- Tenant locks
CREATE TABLE IF NOT EXISTS packer_tenant_locks (
  tenant_id VARCHAR(255) PRIMARY KEY,
  locked_by VARCHAR(255) NOT NULL,
  lock_timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
  last_heartbeat TIMESTAMP WITH TIME ZONE NOT NULL,
  ttl TIMESTAMP WITH TIME ZONE NOT NULL
);

-- Parquet file metadata
CREATE TABLE IF NOT EXISTS packer_parquet_file_metadata (
  id SERIAL PRIMARY KEY,
  tenant_id VARCHAR(255) NOT NULL,
  dataset_id VARCHAR(255) NOT NULL,
  file_path VARCHAR(1000) NOT NULL,
  partition_path VARCHAR(500),
  file_size_bytes BIGINT NOT NULL,
  row_count BIGINT NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE NOT NULL,
  last_modified TIMESTAMP WITH TIME ZONE NOT NULL,
  schema_json JSONB NOT NULL,
  column_stats JSONB,
  file_checksum VARCHAR(64),
  instance_id VARCHAR(255),
  metadata_version INTEGER DEFAULT 1,
  ttl TIMESTAMP WITH TIME ZONE,
  inserted_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  UNIQUE(tenant_id, dataset_id, file_path)
);

-- Metadata generation status
CREATE TABLE IF NOT EXISTS packer_metadata_generation_status (
  tenant_id VARCHAR(255) NOT NULL,
  dataset_id VARCHAR(255) NOT NULL,
  partition_path VARCHAR(500) NOT NULL,
  last_generated_at TIMESTAMP WITH TIME ZONE,
  file_count INTEGER DEFAULT 0,
  total_rows BIGINT DEFAULT 0,
  total_size_bytes BIGINT DEFAULT 0,
  needs_regeneration BOOLEAN DEFAULT false,
  current_schema_hash VARCHAR(64),
  schema_version INTEGER DEFAULT 1,
  ttl TIMESTAMP WITH TIME ZONE,
  PRIMARY KEY (tenant_id, dataset_id, partition_path)
);

-- Metadata summary view
CREATE OR REPLACE VIEW packer_parquet_metadata_summary AS
SELECT
  tenant_id,
  dataset_id,
  partition_path,
  COUNT(*) as file_count,
  SUM(row_count) as total_rows,
  SUM(file_size_bytes) as total_size_bytes,
  MIN(created_at) as first_file_created,
  MAX(last_modified) as last_file_modified,
  MAX(updated_at) as metadata_last_updated
FROM packer_parquet_file_metadata
GROUP BY tenant_id, dataset_id, partition_path;
```

## Implementation Plan

1. **Storage Layer**: Add methods to `storage/interface.go` and implement in `storage/postgresql.go`
2. **Service Layer**: Add service methods in `services/` for business logic
3. **API Handlers**: Create `api/piper_handlers.go` and `api/packer_handlers.go`
4. **Routes**: Register routes in `api/api.go` under `/api/v1/piper/*` and `/api/v1/packer/*`
5. **Migrations**: Add database migration for new tables
6. **Client Libraries**: Update piper and packer to use control API instead of direct DB access

## Access Control

- System service API key: Full access to all tenants/datasets
- Account-scoped requests: Filter by account_id from JWT claims
- On-prem deployments: Use service API key with optional account_id filter in config
