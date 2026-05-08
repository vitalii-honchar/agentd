CREATE TABLE IF NOT EXISTS agent_revisions (
    agent_name TEXT NOT NULL,
    revision_id TEXT NOT NULL,
    content_digest TEXT NOT NULL,
    source_path TEXT NOT NULL,
    artifact_path TEXT NOT NULL,
    environment_json TEXT NOT NULL DEFAULT '[]',
    prompt TEXT NOT NULL,
    vendor_name TEXT NOT NULL,
    vendor_model TEXT NOT NULL,
    schedule_type TEXT NOT NULL CHECK (schedule_type IN ('cron', 'manual')),
    schedule_expression TEXT,
    status TEXT NOT NULL CHECK (status IN ('pending', 'finalized', 'corrupt')),
    created_at TEXT NOT NULL,
    finalized_at TEXT,
    error_message TEXT,
    PRIMARY KEY (agent_name, revision_id),
    FOREIGN KEY (agent_name) REFERENCES agents(name) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_agent_revisions_agent_digest
ON agent_revisions(agent_name, content_digest);

CREATE INDEX IF NOT EXISTS idx_agent_revisions_agent_created
ON agent_revisions(agent_name, created_at);

CREATE INDEX IF NOT EXISTS idx_agent_revisions_status
ON agent_revisions(status);

CREATE TABLE IF NOT EXISTS agent_revision_tools (
    agent_name TEXT NOT NULL,
    revision_id TEXT NOT NULL,
    name TEXT NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('custom_tool', 'host_tool', 'mcp_server')),
    original_command TEXT,
    rewritten_command TEXT,
    host_command TEXT,
    args_json TEXT NOT NULL DEFAULT '[]',
    env_json TEXT NOT NULL DEFAULT '[]',
    timeout_seconds INTEGER NOT NULL DEFAULT 0,
    read_paths_json TEXT NOT NULL DEFAULT '[]',
    write_paths_json TEXT NOT NULL DEFAULT '[]',
    network_allow_json TEXT NOT NULL DEFAULT '[]',
    copied_files_json TEXT NOT NULL DEFAULT '[]',
    created_at TEXT NOT NULL,
    PRIMARY KEY (agent_name, revision_id, name),
    FOREIGN KEY (agent_name, revision_id) REFERENCES agent_revisions(agent_name, revision_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_agent_revision_tools_kind
ON agent_revision_tools(kind);

CREATE TABLE IF NOT EXISTS agent_revision_artifact_files (
    agent_name TEXT NOT NULL,
    revision_id TEXT NOT NULL,
    artifact_relative_path TEXT NOT NULL,
    source_path TEXT NOT NULL,
    sha256 TEXT NOT NULL,
    mode INTEGER NOT NULL,
    size_bytes INTEGER NOT NULL,
    copied_at TEXT NOT NULL,
    PRIMARY KEY (agent_name, revision_id, artifact_relative_path),
    FOREIGN KEY (agent_name, revision_id) REFERENCES agent_revisions(agent_name, revision_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS agent_revision_environment (
    agent_name TEXT NOT NULL,
    revision_id TEXT NOT NULL,
    key TEXT NOT NULL,
    value TEXT NOT NULL,
    source TEXT NOT NULL CHECK (source IN ('literal', 'env_file', 'tool_env')),
    source_path TEXT,
    artifact_relative_path TEXT,
    masked INTEGER NOT NULL DEFAULT 1 CHECK (masked IN (0, 1)),
    created_at TEXT NOT NULL,
    PRIMARY KEY (agent_name, revision_id, key, source),
    FOREIGN KEY (agent_name, revision_id) REFERENCES agent_revisions(agent_name, revision_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_agent_revision_environment_key
ON agent_revision_environment(agent_name, revision_id, key);
