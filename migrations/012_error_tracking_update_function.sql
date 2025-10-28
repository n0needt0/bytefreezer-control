-- Migration: 012_error_tracking_update_function
-- Description: Add update function for system_errors updated_at column
-- Author: Claude
-- Date: 2025-10-28

CREATE OR REPLACE FUNCTION update_system_errors_updated_at()
RETURNS TRIGGER AS $func$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$func$ LANGUAGE plpgsql;

-- Record migration in control_migrations table
INSERT INTO control_migrations (version, name, applied_at)
VALUES (12, 'error_tracking_update_fn', NOW())
ON CONFLICT (version) DO NOTHING;
