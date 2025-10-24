-- Clean up dataset_tested audit log entries
-- These events are no longer needed and should not be stored

-- First, check how many records will be deleted
SELECT COUNT(*) as total_dataset_tested_events
FROM control_audit_log
WHERE action = 'dataset_tested';

-- Delete all dataset_tested events
DELETE FROM control_audit_log
WHERE action = 'dataset_tested';

-- Verify deletion
SELECT COUNT(*) as remaining_dataset_tested_events
FROM control_audit_log
WHERE action = 'dataset_tested';
