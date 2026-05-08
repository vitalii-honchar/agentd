# Contract: Agent Definition Extensions

This feature keeps Markdown Agent Definitions as the source of truth and adds
fields needed by runnable examples and local tools.

## Front Matter Fields

```yaml
name: hacker-news-builder-brief
enabled: true
schedule:
  type: cron
  expression: "0 8 * * *"
vendor:
  name: openai
  model: gpt-5.4-mini
inputs:
  - name: url
    required: true
    description: Website URL to capture
tools:
  - name: fetch_hacker_news
    kind: local_tool
    command: tools/fetch_hacker_news.py
    timeout: 60s
    network:
      allow:
        - https://hacker-news.firebaseio.com
access:
  filesystem:
    read:
      - sources/
      - fixtures/
    write:
      - .agentd-work/
```

## Validation Rules

- `inputs` are optional for scheduled examples and required only for manual
  workflows that naturally need user input.
- `tools[].command` must resolve relative to the definition folder unless an
  absolute path is explicitly allowed by policy.
- `tools[].timeout` defaults to daemon policy when omitted.
- `tools[].network.allow` must list public network destinations used by the
  tool.
- Secret-bearing files and environment variables are not inherited by default.

## Prompt Body

The Markdown body describes:
- Role and objective.
- Source/tool usage instructions.
- Output sections.
- Required no-change/no-action conclusion when no meaningful new items exist.
