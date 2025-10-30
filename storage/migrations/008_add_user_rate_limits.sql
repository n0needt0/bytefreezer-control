-- Migration 008: Add rate limiting fields to control_users
-- This enables per-user/token rate limiting instead of IP-based rate limiting

-- Add rate limit fields to control_users table
ALTER TABLE control_users
ADD COLUMN rate_limit_enabled BOOLEAN DEFAULT true,
ADD COLUMN rate_limit_requests_per_minute INTEGER DEFAULT 60,
ADD COLUMN rate_limit_burst_size INTEGER DEFAULT 100;

-- Add index for efficient lookup by rate limiting middleware
CREATE INDEX idx_control_users_rate_limit ON control_users(id, rate_limit_enabled);

-- Add comments for documentation
COMMENT ON COLUMN control_users.rate_limit_enabled IS 'Enable/disable rate limiting for this user';
COMMENT ON COLUMN control_users.rate_limit_requests_per_minute IS 'Maximum requests per minute for this user';
COMMENT ON COLUMN control_users.rate_limit_burst_size IS 'Burst capacity for this user';

-- Set sensible defaults for existing users
UPDATE control_users
SET
    rate_limit_enabled = true,
    rate_limit_requests_per_minute = 60,
    rate_limit_burst_size = 100
WHERE rate_limit_enabled IS NULL;

-- Set higher limits for system admins
UPDATE control_users
SET
    rate_limit_requests_per_minute = 300,
    rate_limit_burst_size = 500
WHERE role = 'system_admin';
