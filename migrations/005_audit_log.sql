-- Migration 005: Audit Log
-- Description: Adds comprehensive audit logging for all user actions
-- Created: 2025-10-21

-- Create audit log table
CREATE TABLE IF NOT EXISTS control_audit_log (
    id SERIAL PRIMARY KEY,
    user_id TEXT NOT NULL,                   -- User who performed the action
    account_id TEXT NOT NULL,                -- Account the user belongs to
    action TEXT NOT NULL,                    -- Action type (e.g., "delete_dataset", "create_dataset", "update_dataset")
    resource_type TEXT NOT NULL,             -- Resource type (e.g., "dataset", "tenant", "account")
    resource_id TEXT NOT NULL,               -- ID of the resource affected
    resource_name TEXT,                      -- Name of the resource (for display)
    details JSONB DEFAULT '{}',              -- Additional details about the action
    ip_address TEXT,                         -- IP address of the request
    user_agent TEXT,                         -- User agent string
    status TEXT NOT NULL DEFAULT 'success',  -- Action status: success, failed, partial
    error_message TEXT,                      -- Error message if status=failed
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for efficient querying
CREATE INDEX idx_audit_log_account_id ON control_audit_log(account_id);
CREATE INDEX idx_audit_log_user_id ON control_audit_log(user_id);
CREATE INDEX idx_audit_log_action ON control_audit_log(action);
CREATE INDEX idx_audit_log_resource_type ON control_audit_log(resource_type);
CREATE INDEX idx_audit_log_resource_id ON control_audit_log(resource_id);
CREATE INDEX idx_audit_log_created_at ON control_audit_log(created_at DESC);
CREATE INDEX idx_audit_log_account_created ON control_audit_log(account_id, created_at DESC);

-- Create composite index for common query patterns
CREATE INDEX idx_audit_log_account_resource ON control_audit_log(account_id, resource_type, created_at DESC);

-- Add comment
COMMENT ON TABLE control_audit_log IS 'Audit log for all user actions in the control plane';
COMMENT ON COLUMN control_audit_log.user_id IS 'User who performed the action';
COMMENT ON COLUMN control_audit_log.account_id IS 'Account the user belongs to';
COMMENT ON COLUMN control_audit_log.action IS 'Action type (e.g., delete_dataset, create_dataset)';
COMMENT ON COLUMN control_audit_log.resource_type IS 'Type of resource affected';
COMMENT ON COLUMN control_audit_log.resource_id IS 'ID of the resource affected';
COMMENT ON COLUMN control_audit_log.resource_name IS 'Name of the resource for display';
COMMENT ON COLUMN control_audit_log.details IS 'Additional details about the action in JSON format';
COMMENT ON COLUMN control_audit_log.status IS 'Action status: success, failed, partial';
