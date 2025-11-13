-- Migration 009: Add deployment type to accounts for multi-deployment support
-- This enables support for managed, on-prem with central control, and air-gapped deployments

-- Add deployment_type to accounts table
-- Types: 'managed', 'on_prem', 'air_gapped'
ALTER TABLE control_accounts
ADD COLUMN IF NOT EXISTS deployment_type VARCHAR(20) NOT NULL DEFAULT 'managed';

-- Add check constraint to ensure valid deployment types
ALTER TABLE control_accounts
ADD CONSTRAINT check_deployment_type
CHECK (deployment_type IN ('managed', 'on_prem', 'air_gapped'));

-- Add index for filtering accounts by deployment type
CREATE INDEX IF NOT EXISTS idx_control_accounts_deployment_type
ON control_accounts(deployment_type);

-- Add comment to explain the field
COMMENT ON COLUMN control_accounts.deployment_type IS
'Deployment type: managed (ByteFreezer hosted), on_prem (customer hosted with central control), air_gapped (fully customer managed)';
