-- Migration 009: Add piper filter catalog table
-- Created: 2025-11-19
-- Description: Create table for storing piper filter catalog (types, parameters, examples)

-- Create piper_filter_catalog table
CREATE TABLE IF NOT EXISTS control_piper_filter_catalog (
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

-- Add indexes for fast lookups
CREATE INDEX IF NOT EXISTS idx_piper_filter_catalog_category ON control_piper_filter_catalog(category);
CREATE INDEX IF NOT EXISTS idx_piper_filter_catalog_version ON control_piper_filter_catalog(version);
CREATE INDEX IF NOT EXISTS idx_piper_filter_catalog_updated ON control_piper_filter_catalog(updated_at DESC);

-- Add GIN index for JSONB parameter search
CREATE INDEX IF NOT EXISTS idx_piper_filter_catalog_parameters ON control_piper_filter_catalog USING GIN(parameters);

-- Create function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_piper_filter_catalog_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger for automatic timestamp updates
DROP TRIGGER IF EXISTS trigger_update_piper_filter_catalog_updated_at ON control_piper_filter_catalog;
CREATE TRIGGER trigger_update_piper_filter_catalog_updated_at
    BEFORE UPDATE ON control_piper_filter_catalog
    FOR EACH ROW
    EXECUTE FUNCTION update_piper_filter_catalog_updated_at();

-- Verify the table was created
SELECT
    table_name,
    column_name,
    data_type,
    is_nullable
FROM information_schema.columns
WHERE table_name = 'control_piper_filter_catalog'
ORDER BY ordinal_position;
