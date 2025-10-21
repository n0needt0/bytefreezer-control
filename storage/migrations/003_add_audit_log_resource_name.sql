-- Migration: Add resource_name column to control_audit_log table
-- Created: 2025-10-21
-- Description: Adds the missing resource_name column to audit log table

-- Add resource_name column if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'control_audit_log'
        AND column_name = 'resource_name'
    ) THEN
        ALTER TABLE control_audit_log ADD COLUMN resource_name VARCHAR(255);
    END IF;
END $$;

-- Verify the column was added
SELECT column_name, data_type, character_maximum_length
FROM information_schema.columns
WHERE table_name = 'control_audit_log'
ORDER BY ordinal_position;
