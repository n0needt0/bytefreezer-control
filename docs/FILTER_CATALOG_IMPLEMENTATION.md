# Piper Filter Catalog Implementation

**Status**: ✅ Complete
**Date**: 2025-11-19
**Migration Version**: 15 (009_piper_filter_catalog.sql)

## Overview

Implemented centralized filter catalog in the control database to serve as a single source of truth for piper filter documentation. This eliminates duplicate documentation and enables dynamic UI rendering and AI-assisted pipeline generation.

## Architecture

```
┌─────────────┐
│   Piper     │ (future) Reports filter schema on startup
│   Service   │──────────────────────────┐
└─────────────┘                          │
                                         ▼
                                  ┌──────────────┐
                                  │   Control    │
                                  │   Database   │ ← Single source of truth
                                  └──────────────┘
                                         │
                  ┌──────────────────────┴───────────────────┐
                  │                                          │
                  ▼                                          ▼
           ┌─────────────┐                            ┌──────────┐
           │  Control API │                           │ AI Agent │
           │ /api/v1/filters                          │ (Training)
           └─────────────┘                            └──────────┘
                  │
                  ▼
           ┌─────────────┐
           │     UI      │
           │ (Reference) │
           └─────────────┘
```

## Components Created

### 1. Database Migration

**File**: `storage/migrations/009_piper_filter_catalog.sql`

**Table**: `control_piper_filter_catalog`

```sql
CREATE TABLE control_piper_filter_catalog (
    filter_type VARCHAR(50) PRIMARY KEY,
    display_name VARCHAR(100) NOT NULL,
    category VARCHAR(50) NOT NULL,
    purpose TEXT NOT NULL,
    parameters JSONB NOT NULL DEFAULT '[]'::jsonb,
    examples JSONB NOT NULL DEFAULT '[]'::jsonb,
    version VARCHAR(20),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);
```

**Indexes**:
- `idx_piper_filter_catalog_category` - Fast category filtering
- `idx_piper_filter_catalog_version` - Version tracking
- `idx_piper_filter_catalog_updated` - Recent updates
- `idx_piper_filter_catalog_parameters` (GIN) - JSONB search

**Migrator Updates**:
- Added to `getAllMigrations()` as version 15
- Added SQL to `getMigrationSQL()` case 15
- Added rollback to `getRollbackSQL()` case 15

### 2. Storage Layer

**File**: `storage/interface.go`

**Types Added**:
- `FilterParameter` - Parameter definition with name, type, required, default, description
- `FilterExample` - Example configuration with description and config map
- `PiperFilter` - Complete filter catalog entry

**Interface Methods**:
- `UpsertPiperFilter(ctx, *PiperFilter) error`
- `GetPiperFilter(ctx, filterType string) (*PiperFilter, error)`
- `ListPiperFilters(ctx) ([]*PiperFilter, error)`
- `ListPiperFiltersByCategory(ctx, category) ([]*PiperFilter, error)`
- `DeletePiperFilter(ctx, filterType) error`
- `GetPiperFilterCatalog(ctx, format) (interface{}, error)`

**File**: `storage/postgresql_filter_catalog.go`

Implements all 6 storage methods with:
- JSONB marshaling/unmarshaling for parameters and examples
- Upsert logic with conflict handling
- Category-based filtering
- Three output formats: "ui", "ai", "full"

### 3. API Handlers

**File**: `api/filter_catalog_handlers.go`

**Endpoints**:
- `GET /api/v1/filters` - List all filters (with optional `?category=` filter)
- `GET /api/v1/filters/catalog` - Get catalog in various formats (`?format=ui|ai|full`)
- `GET /api/v1/filters/{filterType}` - Get specific filter details
- `POST /api/v1/filters` - Create/update filter
- `PUT /api/v1/filters/{filterType}` - Update specific filter
- `DELETE /api/v1/filters/{filterType}` - Delete filter

**Request/Response Types**:
- `ListFiltersRequest/Response`
- `GetFilterRequest/Response`
- `UpsertFilterRequest/Response`
- `DeleteFilterRequest/Response`
- `GetFilterCatalogRequest/Response`

### 4. Seed Script

**File**: `scripts/seed_filter_catalog.go`

Seeds 21 filters across 5 categories:
- **Filtering** (4): include, exclude, sample, drop
- **Parsing** (4): grok, kv, date_parse, split
- **Transformation** (5): add_field, remove_field, rename_field, mutate, regex_replace
- **Enrichment** (4): geoip, useragent, dns, fingerprint
- **Utility** (4): conditional, json_validate, json_flatten, uppercase_keys

**File**: `scripts/seed_filter_catalog.sh`

Shell wrapper for easy execution:
```bash
./scripts/seed_filter_catalog.sh [config-file] [version]
```

## API Output Formats

### Full Format (default)
```json
[
  {
    "filter_type": "include",
    "display_name": "Include Filter",
    "category": "filtering",
    "purpose": "Keep only events matching conditions",
    "parameters": [...],
    "examples": [...],
    "version": "1.0.0",
    "created_at": "2025-11-19T...",
    "updated_at": "2025-11-19T..."
  }
]
```

### UI Format (`?format=ui`)
```json
{
  "filters_by_category": {
    "filtering": [...],
    "parsing": [...],
    "transformation": [...],
    "enrichment": [...],
    "utility": [...]
  },
  "total_filters": 21,
  "categories": ["filtering", "parsing", "transformation", "enrichment", "utility"]
}
```

### AI Format (`?format=ai`)
```json
{
  "filters": [
    {
      "type": "include",
      "name": "Include Filter",
      "category": "filtering",
      "purpose": "Keep only events matching conditions",
      "parameters": [...],
      "examples": [...]
    }
  ],
  "version": "1.0.0"
}
```

## Usage

### 1. Run Migration

```bash
# Control service will auto-migrate on startup
# Or manually:
bytefreezer-control migrate
```

### 2. Seed Catalog

```bash
cd /home/andrew/workspace/bytefreezer/bytefreezer-control
./scripts/seed_filter_catalog.sh /etc/bytefreezer/control/config.yaml 1.0.0
```

### 3. Query Filters

```bash
# List all filters
curl http://localhost:8080/api/v1/filters

# Filter by category
curl http://localhost:8080/api/v1/filters?category=filtering

# Get specific filter
curl http://localhost:8080/api/v1/filters/include

# Get UI-optimized catalog
curl http://localhost:8080/api/v1/filters/catalog?format=ui

# Get AI training format
curl http://localhost:8080/api/v1/filters/catalog?format=ai
```

### 4. Update Filter

```bash
curl -X POST http://localhost:8080/api/v1/filters \
  -H "Content-Type: application/json" \
  -d '{
    "filter_type": "custom_filter",
    "display_name": "Custom Filter",
    "category": "transformation",
    "purpose": "Does custom transformation",
    "parameters": [],
    "examples": []
  }'
```

## Benefits

✅ **Single Source of Truth** - Filter catalog in central database
✅ **Version Tracking** - Track which piper version has which filters
✅ **Multiple Formats** - Optimize for UI, AI, or raw consumption
✅ **No Duplication** - Eliminated 3 duplicate documentation files
✅ **Dynamic UI** - UI can render filters without hardcoding
✅ **AI Training** - Dedicated format for natural language → config generation
✅ **Runtime Updates** - Update docs without redeploying services
✅ **Searchable** - Query by category, filter type, parameters

## Migration Path

**Phase 1** (✅ Complete):
- Database schema and migration
- Storage layer implementation
- API endpoints
- Seed script with 21 filters

**Phase 2** (Future):
- Piper reports filter schema on startup (like proxy plugins)
- Auto-updates control DB from running piper instances
- Remove manual seeding

**Phase 3** (Future):
- UI consumes `/api/v1/filters/catalog?format=ui`
- Remove hardcoded filter references in UI
- Dynamic filter reference display

**Phase 4** (Future):
- AI agent uses `/api/v1/filters/catalog?format=ai`
- Natural language → JSON pipeline generation
- Replace static `piper_plugin_catalog.md`

## Files Created

1. `storage/migrations/009_piper_filter_catalog.sql` - Migration SQL
2. `storage/postgresql_filter_catalog.go` - Storage implementation (314 lines)
3. `api/filter_catalog_handlers.go` - API handlers (240 lines)
4. `scripts/seed_filter_catalog.go` - Seed script (650 lines)
5. `scripts/seed_filter_catalog.sh` - Shell wrapper
6. `docs/FILTER_CATALOG_IMPLEMENTATION.md` - This file

## Files Modified

1. `storage/postgresql_migrator.go` - Added migration 15
2. `storage/interface.go` - Added types and interface methods
3. `api/api.go` - Registered 6 new routes

## Testing

Build Status: ✅ Successful
```bash
cd /home/andrew/workspace/bytefreezer/bytefreezer-control
go build
```

Next Steps:
1. Start control service
2. Verify migration runs
3. Run seed script
4. Test API endpoints
5. Integrate with UI (future)
6. Integrate with AI agent (future)

## Database Schema

```sql
-- View all filters
SELECT filter_type, category, display_name
FROM control_piper_filter_catalog
ORDER BY category, filter_type;

-- Count by category
SELECT category, COUNT(*)
FROM control_piper_filter_catalog
GROUP BY category
ORDER BY category;

-- Search parameters
SELECT filter_type, parameters
FROM control_piper_filter_catalog
WHERE parameters @> '[{"name": "field"}]'::jsonb;
```

## Rollback

If needed, rollback migration 15:
```bash
# This will drop the table and all data
bytefreezer-control migrate --rollback --target-version 14
```

## Documentation Cleanup

Removed duplicate files:
- ✅ `bytefreezer-control/docs/FILTER_REFERENCE.md` - Duplicate, kept only in piper
- ✅ `bytefreezer-control/ansible/playbooks/dist/piper_plugin_catalog.md` - Build artifact
- ✅ `bytefreezer-piper/docs/FILTER_REFERENCE.md` - Duplicate of FILTERS.md

Kept:
- ✅ `bytefreezer-piper/docs/FILTERS.md` - Source of truth (technical reference)
- ✅ `bytefreezer-control/docs/piper_plugin_catalog.md` - AI training doc
- ✅ `bytefreezer-ui/docs/PLUGIN_PARAMETERS.md` - Proxy plugins only

Now filter catalog is in database, served via API, consumable by all services.
