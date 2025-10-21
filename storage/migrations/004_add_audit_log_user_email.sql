-- Migration: Add user_email column to control_audit_log table
-- Created: 2025-10-21
-- Description: Adds user_email column to make audit logs more readable

-- Add user_email column if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'control_audit_log'
        AND column_name = 'user_email'
    ) THEN
        ALTER TABLE control_audit_log ADD COLUMN user_email VARCHAR(255);
    END IF;
END $$;

-- Verify the column was added
SELECT column_name, data_type, character_maximum_length
FROM information_schema.columns
WHERE table_name = 'control_audit_log'
AND column_name = 'user_email';
