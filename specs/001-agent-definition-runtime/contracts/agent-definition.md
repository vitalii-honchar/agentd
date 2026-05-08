# Contract: Agent Definition Markdown

Agent Definitions are Markdown files with YAML front matter followed by the
exact prompt body. The front matter is the machine-readable runtime contract;
the Markdown body is passed to the selected LLM provider as the Agent prompt.

## Minimal Manual Agent

```markdown
---
name: release-notes-helper
enabled: true
schedule:
  type: manual
vendor:
  name: openai
  model: gpt-5
tools: []
mcp_servers: []
access:
  filesystem:
    read: []
    write: []
  network:
    allow: []
---
You are a release notes assistant.

Summarize the latest changes into concise release notes for developers.
```

## Cron Agent

```markdown
---
name: daily-pr-review
enabled: true
schedule:
  type: cron
  expression: "0 9 * * MON-FRI"
vendor:
  name: openai
  model: gpt-5
tools:
  - name: git
    kind: local_tool
    command: git
    args: ["status", "--short"]
    env: []
    read_paths: ["."]
    write_paths: []
mcp_servers:
  - name: github
    command: github-mcp-server
    args: []
    env: ["GITHUB_TOKEN"]
access:
  filesystem:
    read: ["."]
    write: []
  network:
    allow: ["api.openai.com", "api.github.com"]
---
Review open pull requests and identify issues that need attention.
```

## Validation Rules

- `name` is required and unique in the local daemon.
- `schedule.type` is required and must be `manual` or `cron`.
- `schedule.expression` is required for `cron` and omitted for `manual`.
- `vendor.name` and `vendor.model` are required.
- `vendor.name` is provider-agnostic. Initial supported value is `openai`; future
  values such as `openrouter` or `anthropic` must use the same Agent Definition
  shape unless a provider requires explicitly documented extensions.
- Missing `tools`, `mcp_servers`, or access lists mean no access.
- The Markdown body after front matter is required and becomes the prompt.
- Secrets are referenced by environment variable name and are never embedded as
  literal values.
