-- Migration 006: Add system user for audit logging
-- This user is used as a fallback when audit logs are created without authentication

-- Create system account first
INSERT INTO control_accounts (id, name, email, active, created_at, updated_at)
VALUES (
    'system_account',
    'System Account',
    'system@bytefreezer.local',
    false,  -- Inactive so it can't be used for login
    NOW(),
    NOW()
)
ON CONFLICT (id) DO NOTHING;

-- Create system user with id='system' to match middleware fallback
INSERT INTO control_users (id, account_id, email, password_hash, role, first_name, last_name, active, email_verified, created_at, updated_at)
VALUES (
    'system',
    'system_account',
    'system@bytefreezer.local',
    '$2a$10$XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX',  -- Invalid hash, can't login
    'system_admin',
    'System',
    'Service',
    false,  -- Inactive so it can't be used for login
    true,
    NOW(),
    NOW()
)
ON CONFLICT (id) DO NOTHING;
