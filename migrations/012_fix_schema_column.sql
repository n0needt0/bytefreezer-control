-- Migration: 012_fix_schema_column
-- Description: Fix dataset_schema column names to match piper code
-- Date: 2025-11-16

-- Rename schema_fields to schema_data if it exists
DO $$ 
BEGIN
    IF EXISTS (
        SELECT 1 
        FROM information_schema.columns 
        WHERE table_name = 'dataset_schema' AND column_name = 'schema_fields'
    ) THEN
        ALTER TABLE dataset_schema RENAME COLUMN schema_fields TO schema_data;
    END IF;
END $$;

-- Rename last_updated to updated_at if it exists
DO $$ 
BEGIN
    IF EXISTS (
        SELECT 1 
        FROM information_schema.columns 
        WHERE table_name = 'dataset_schema' AND column_name = 'last_updated'
    ) THEN
        ALTER TABLE dataset_schema RENAME COLUMN last_updated TO updated_at;
    END IF;
END $$;

-- Drop sample_count column if it exists (not used)
DO $$ 
BEGIN
    IF EXISTS (
        SELECT 1 
        FROM information_schema.columns 
        WHERE table_name = 'dataset_schema' AND column_name = 'sample_count'
    ) THEN
        ALTER TABLE dataset_schema DROP COLUMN sample_count;
    END IF;
END $$;

-- Insert migration record
INSERT INTO control_migrations (version, name, applied_at)
VALUES (12, 'fix_schema_column', NOW())
ON CONFLICT (version) DO NOTHING;
