CREATE TABLE IF NOT EXISTS agents (
    name TEXT PRIMARY KEY,
    revision TEXT NOT NULL,
    definition_source_path TEXT NOT NULL,
    definition_markdown TEXT NOT NULL,
    prompt TEXT NOT NULL,
    enabled INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0, 1)),
    vendor_name TEXT NOT NULL,
    vendor_model TEXT NOT NULL,
    schedule_type TEXT NOT NULL CHECK (schedule_type IN ('cron', 'manual')),
    schedule_expression TEXT,
    next_run_at TEXT,
    status TEXT NOT NULL CHECK (status IN ('active', 'disabled', 'invalid')),
    last_run_id TEXT,
    last_error TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    applied_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_agents_enabled_schedule
ON agents(enabled, schedule_type, next_run_at);

CREATE TABLE IF NOT EXISTS agent_tools (
    agent_name TEXT NOT NULL,
    name TEXT NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('local_tool', 'mcp_server')),
    command TEXT,
    args_json TEXT NOT NULL DEFAULT '[]',
    env_json TEXT NOT NULL DEFAULT '[]',
    read_paths_json TEXT NOT NULL DEFAULT '[]',
    write_paths_json TEXT NOT NULL DEFAULT '[]',
    network_allow_json TEXT NOT NULL DEFAULT '[]',
    created_at TEXT NOT NULL,
    PRIMARY KEY (agent_name, name),
    FOREIGN KEY (agent_name) REFERENCES agents(name) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_agent_tools_agent_name
ON agent_tools(agent_name);

CREATE TABLE IF NOT EXISTS agent_mcp_servers (
    agent_name TEXT NOT NULL,
    name TEXT NOT NULL,
    command TEXT NOT NULL,
    args_json TEXT NOT NULL DEFAULT '[]',
    env_json TEXT NOT NULL DEFAULT '[]',
    created_at TEXT NOT NULL,
    PRIMARY KEY (agent_name, name),
    FOREIGN KEY (agent_name) REFERENCES agents(name) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_agent_mcp_servers_agent_name
ON agent_mcp_servers(agent_name);
