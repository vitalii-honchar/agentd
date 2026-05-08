CREATE INDEX IF NOT EXISTS idx_agent_runs_log_path
ON agent_runs(log_path);

CREATE INDEX IF NOT EXISTS idx_agent_runs_latest_logs
ON agent_runs(agent_name, started_at DESC, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_agent_runs_terminal_logs
ON agent_runs(agent_name, completed_at DESC)
WHERE completed_at IS NOT NULL;
