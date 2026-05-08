ALTER TABLE agent_runs
ADD COLUMN result TEXT NOT NULL DEFAULT '';

ALTER TABLE agent_runs
ADD COLUMN result_summary TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_agent_runs_terminal_results
ON agent_runs(agent_name, completed_at DESC)
WHERE completed_at IS NOT NULL;
