CREATE TABLE IF NOT EXISTS agent_runs (
    id TEXT PRIMARY KEY,
    agent_name TEXT NOT NULL,
    agent_revision TEXT NOT NULL,
    trigger TEXT NOT NULL CHECK (trigger IN ('schedule', 'manual', 'recovery')),
    status TEXT NOT NULL CHECK (
        status IN ('queued', 'running', 'completed', 'failed', 'stopping', 'stopped', 'interrupted')
    ),
    due_at TEXT,
    started_at TEXT,
    completed_at TEXT,
    work_dir TEXT NOT NULL,
    log_path TEXT NOT NULL,
    provider_request_id TEXT,
    error_code TEXT,
    error_message TEXT,
    stop_requested_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_agent_runs_agent_due
ON agent_runs(agent_name, due_at)
WHERE due_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_agent_runs_started_at
ON agent_runs(started_at DESC);

CREATE INDEX IF NOT EXISTS idx_agent_runs_status
ON agent_runs(status);

CREATE INDEX IF NOT EXISTS idx_agent_runs_due_at
ON agent_runs(due_at);

CREATE TABLE IF NOT EXISTS runtime_events (
    id TEXT PRIMARY KEY,
    agent_name TEXT,
    run_id TEXT,
    event_type TEXT NOT NULL,
    level TEXT NOT NULL CHECK (level IN ('debug', 'info', 'warn', 'error')),
    message TEXT NOT NULL,
    attributes_json TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL,
    FOREIGN KEY (run_id) REFERENCES agent_runs(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_runtime_events_created_at
ON runtime_events(created_at DESC);
