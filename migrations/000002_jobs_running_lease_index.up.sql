-- Serves the stale-lease reaper (internal/queue Reap): finds running jobs whose
-- lease has expired without scanning the whole jobs table. Partial, so only
-- the small set of in-flight rows is indexed (see ADR-009).
CREATE INDEX idx_jobs_running_lease ON jobs (locked_until) WHERE status = 'running';
