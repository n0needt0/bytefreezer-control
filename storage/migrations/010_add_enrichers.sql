-- Migration: Add enrichers tables
-- Description: Tables for customer-uploaded enrichment lookup data

-- Enrichers table
CREATE TABLE IF NOT EXISTS enrichers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(128) NOT NULL,
    description TEXT,

    -- Column configuration
    columns JSONB NOT NULL, -- Array of column names (max 20)
    index_columns JSONB NOT NULL, -- Array of 1-3 column names used for lookup

    -- Data info
    row_count INTEGER NOT NULL DEFAULT 0,
    file_size BIGINT NOT NULL DEFAULT 0,
    file_data BYTEA, -- Binary enrichment data (JSON lines format, max 100MB)

    -- Status
    status VARCHAR(32) NOT NULL DEFAULT 'pending', -- pending, processing, active, error
    error_message TEXT,

    -- Metadata
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_by UUID REFERENCES users(id),

    -- Constraints
    CONSTRAINT enrichers_name_unique UNIQUE (tenant_id, name),
    CONSTRAINT enrichers_status_check CHECK (status IN ('pending', 'processing', 'active', 'error', 'disabled'))
);

-- Create indexes
CREATE INDEX idx_enrichers_tenant_id ON enrichers(tenant_id);
CREATE INDEX idx_enrichers_status ON enrichers(status);
CREATE INDEX idx_enrichers_created_at ON enrichers(created_at DESC);

-- Enricher versions table (for tracking uploads/updates)
CREATE TABLE IF NOT EXISTS enricher_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    enricher_id UUID NOT NULL REFERENCES enrichers(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,

    -- Upload info
    original_filename VARCHAR(255) NOT NULL,
    file_size BIGINT NOT NULL,
    row_count INTEGER NOT NULL,

    -- Processing info
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    error_message TEXT,
    processed_at TIMESTAMP WITH TIME ZONE,

    -- Metadata
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    uploaded_by UUID REFERENCES users(id),

    -- Constraints
    CONSTRAINT enricher_versions_unique UNIQUE (enricher_id, version),
    CONSTRAINT enricher_versions_status_check CHECK (status IN ('pending', 'processing', 'active', 'error'))
);

-- Create indexes
CREATE INDEX idx_enricher_versions_enricher_id ON enricher_versions(enricher_id);
CREATE INDEX idx_enricher_versions_status ON enricher_versions(status);

-- Audit log for enricher operations
CREATE TABLE IF NOT EXISTS enricher_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    enricher_id UUID REFERENCES enrichers(id) ON DELETE SET NULL,

    -- Action info
    action VARCHAR(64) NOT NULL, -- create, upload, update, delete, enable, disable
    actor_id UUID REFERENCES users(id),
    actor_email VARCHAR(255),

    -- Details
    details JSONB,

    -- Metadata
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes
CREATE INDEX idx_enricher_audit_tenant_id ON enricher_audit_log(tenant_id);
CREATE INDEX idx_enricher_audit_enricher_id ON enricher_audit_log(enricher_id);
CREATE INDEX idx_enricher_audit_created_at ON enricher_audit_log(created_at DESC);

-- Comments
COMMENT ON TABLE enrichers IS 'Customer-uploaded enrichment lookup data tables';
COMMENT ON COLUMN enrichers.columns IS 'Array of column names (max 20 columns, max 64 chars each)';
COMMENT ON COLUMN enrichers.index_columns IS 'Array of 1-3 column names used as lookup keys';
COMMENT ON COLUMN enrichers.file_data IS 'Binary enrichment data in JSON lines format (max 100MB)';
COMMENT ON COLUMN enrichers.row_count IS 'Number of rows in the lookup table (max 1,000,000)';
COMMENT ON COLUMN enrichers.file_size IS 'Size of the processed binary file in bytes (max 100MB for CSV upload)';
