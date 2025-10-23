-- Migration 007: Add dataset test status fields
-- Created: 2025-10-23
-- Description: Adds test status fields for tracking input/output test results

-- Add input test status fields
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'control_datasets'
        AND column_name = 'input_test_status'
    ) THEN
        ALTER TABLE control_datasets ADD COLUMN input_test_status VARCHAR(50) DEFAULT 'untested';
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'control_datasets'
        AND column_name = 'input_test_message'
    ) THEN
        ALTER TABLE control_datasets ADD COLUMN input_test_message TEXT;
    END IF;
END $$;

-- Add output test status fields
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'control_datasets'
        AND column_name = 'output_test_status'
    ) THEN
        ALTER TABLE control_datasets ADD COLUMN output_test_status VARCHAR(50) DEFAULT 'untested';
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'control_datasets'
        AND column_name = 'output_test_message'
    ) THEN
        ALTER TABLE control_datasets ADD COLUMN output_test_message TEXT;
    END IF;
END $$;

-- Add last tested timestamp
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_name = 'control_datasets'
        AND column_name = 'last_tested_at'
    ) THEN
        ALTER TABLE control_datasets ADD COLUMN last_tested_at TIMESTAMP WITH TIME ZONE;
    END IF;
END $$;

-- Verify the columns were added
SELECT column_name, data_type, character_maximum_length
FROM information_schema.columns
WHERE table_name = 'control_datasets'
AND column_name IN ('input_test_status', 'input_test_message', 'output_test_status', 'output_test_message', 'last_tested_at')
ORDER BY ordinal_position;
