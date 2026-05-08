CREATE INDEX IF NOT EXISTS idx_runtime_events_run_created
ON runtime_events(run_id, created_at DESC)
WHERE run_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_runtime_events_type_created
ON runtime_events(event_type, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_runtime_events_agent_created
ON runtime_events(agent_name, created_at DESC)
WHERE agent_name IS NOT NULL;
