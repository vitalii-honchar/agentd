---
name: codex-provider-agent
enabled: true
schedule:
  type: manual
vendor:
  name: codex
  model: gpt-5.4-mini
contract:
  input: |
    {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "topic": { "type": "string" }
      },
      "required": ["topic"]
    }
  output: |
    {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "summary": { "type": "string" }
      },
      "required": ["summary"]
    }
tools: []
mcp_servers: []
access:
  filesystem:
    read: []
    write: []
  network:
    allow: []
---
Summarize the provided topic in one short sentence.
