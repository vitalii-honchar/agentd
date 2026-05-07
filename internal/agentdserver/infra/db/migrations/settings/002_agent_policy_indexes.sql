CREATE INDEX IF NOT EXISTS idx_agent_tools_kind
ON agent_tools(kind);

CREATE INDEX IF NOT EXISTS idx_agent_tools_command
ON agent_tools(command)
WHERE command IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_agent_mcp_servers_command
ON agent_mcp_servers(command);

CREATE INDEX IF NOT EXISTS idx_agents_vendor
ON agents(vendor_name, vendor_model);

CREATE INDEX IF NOT EXISTS idx_agents_status
ON agents(status);
