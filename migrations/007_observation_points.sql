-- Migration: 007_observation_points
-- Description: Update dataset_metrics to support observation point naming for inter-component data flow tracking
-- Author: Claude
-- Date: 2025-10-26

-- Drop the old CHECK constraint on component
ALTER TABLE dataset_metrics DROP CONSTRAINT IF EXISTS dataset_metrics_component_check;

-- Add new CHECK constraint with observation point names
ALTER TABLE dataset_metrics ADD CONSTRAINT dataset_metrics_component_check
    CHECK (component IN ('proxy-to-receiver', 'receiver-to-piper', 'piper-to-packer', 'packer-to-storage', 'control'));

-- Update table comment
COMMENT ON TABLE dataset_metrics IS 'Stores inter-component data flow metrics at observation points (proxy→receiver, receiver→piper, piper→packer, packer→storage) for monitoring and analytics';

-- Update component column comment
COMMENT ON COLUMN dataset_metrics.component IS 'Observation point: proxy-to-receiver, receiver-to-piper, piper-to-packer, packer-to-storage, or control';

-- Update metrics column comments to reflect inter-component flow
COMMENT ON COLUMN dataset_metrics.input_bytes IS 'Number of input bytes received at this observation point';
COMMENT ON COLUMN dataset_metrics.output_bytes IS 'Number of output bytes sent from this observation point';
COMMENT ON COLUMN dataset_metrics.lines_processed IS 'Number of log lines transferred through this observation point';

-- Insert migration record
INSERT INTO control_migrations (version, name, applied_at)
VALUES (7, 'observation_points', NOW())
ON CONFLICT (version) DO NOTHING;
