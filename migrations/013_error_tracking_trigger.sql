-- Migration: 013_error_tracking_trigger
-- Description: Add update trigger for system_errors table
-- Author: Claude
-- Date: 2025-10-28

CREATE TRIGGER trigger_update_system_errors_updated_at
BEFORE UPDATE ON system_errors
FOR EACH ROW
EXECUTE FUNCTION update_system_errors_updated_at();

-- Record migration in control_migrations table
INSERT INTO control_migrations (version, name, applied_at)
VALUES (13, 'error_tracking_update_trigger', NOW())
ON CONFLICT (version) DO NOTHING;
