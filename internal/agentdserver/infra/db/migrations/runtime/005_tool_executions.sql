CREATE TABLE IF NOT EXISTS tool_executions (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL,
    agent_name TEXT NOT NULL,
    tool_name TEXT NOT NULL,
    command_summary TEXT NOT NULL,
    started_at TEXT NOT NULL,
    completed_at TEXT,
    exit_code INTEGER NOT NULL DEFAULT 0,
    timed_out INTEGER NOT NULL DEFAULT 0 CHECK (timed_out IN (0, 1)),
    stdout_summary TEXT NOT NULL DEFAULT '',
    stderr_summary TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    FOREIGN KEY (run_id) REFERENCES agent_runs(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_tool_executions_run_started
ON tool_executions(run_id, started_at DESC);

CREATE INDEX IF NOT EXISTS idx_tool_executions_agent_started
ON tool_executions(agent_name, started_at DESC);
