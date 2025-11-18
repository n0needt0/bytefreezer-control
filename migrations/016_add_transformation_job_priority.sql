-- Migration 016: Add priority field to piper transformation jobs
-- Enables high-priority test jobs to execute before normal validation/activation jobs

-- Add priority column (default 0 for normal jobs, 10 for test jobs)
ALTER TABLE piper_transformation_jobs
ADD COLUMN IF NOT EXISTS priority INTEGER NOT NULL DEFAULT 0;

-- Create index for efficient priority-based job claiming
CREATE INDEX IF NOT EXISTS idx_piper_transformation_jobs_priority_claim
    ON piper_transformation_jobs(status, priority DESC, created_at ASC)
    WHERE status = 'pending';

COMMENT ON COLUMN piper_transformation_jobs.priority IS
    'Job priority: 0 (normal/default), 10 (high priority for test jobs with 10 samples)';
