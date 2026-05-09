# Agent Definition Contract

Agent definitions remain markdown files with YAML front matter followed by the
agent prompt. This feature adds optional `contract` metadata and allows
`vendor.name: codex`.

## Contract Field

```yaml
contract:
  input: |
    {
      "type": "object",
      "additionalProperties": false,
      "properties": {},
      "required": []
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
```

Rules:
- `contract` is optional.
- If `contract` is omitted, agentd does not apply input-contract validation or
  output-contract finalization.
- If `contract` is present, `input` and `output` must contain valid JSON Schema
  documents.
- `contract.input` validates runtime input before any run, tool, or model side
  effect.
- `contract.output` validates successful final output.
- Applied revisions persist the resolved contract schemas.

## Empty-Input Scheduled Example Shape

Scheduled examples with no runtime input use an empty-object schema:

```yaml
contract:
  input: |
    {
      "type": "object",
      "additionalProperties": false,
      "properties": {},
      "required": []
    }
  output: |
    {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "executive_summary": { "type": "string" },
        "items": {
          "type": "array",
          "items": {
            "type": "object",
            "additionalProperties": false,
            "properties": {
              "title": { "type": "string" },
              "url": { "type": "string" },
              "reason": { "type": "string" }
            },
            "required": ["title", "url", "reason"]
          }
        }
      },
      "required": ["executive_summary", "items"]
    }
```

Checked-in scheduled examples should still use concrete output schemas. Prefer
domain-specific arrays and fields such as `repositories`,
`important_items`, `high_signal_posts`, `recurring_pains`, or
`releases_requiring_attention` rather than a single unstructured `text` field.
Include a required `summary` or `executive_summary` and a required
`no_action_note` when the agent may legitimately find no high-signal items.

## Manual-Input Example Shape

```yaml
contract:
  input: |
    {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "url": {
          "type": "string",
          "format": "uri",
          "description": "Website URL to screenshot and summarize"
        }
      },
      "required": ["url"]
    }
  output: |
    {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "website_summary": { "type": "string" },
        "audience": { "type": "string" },
        "primary_call_to_action": { "type": "string" },
        "trust_signals": {
          "type": "array",
          "items": { "type": "string" }
        },
        "issues": {
          "type": "array",
          "items": { "type": "string" }
        }
      },
      "required": [
        "website_summary",
        "audience",
        "primary_call_to_action",
        "trust_signals",
        "issues"
      ]
    }
```

Checked-in manual examples should model every user-supplied value in
`contract.input`. Existing legacy `inputs:` metadata can remain for CLI
templating and human descriptions, but `contract.input` is the validation
source of truth before execution starts.

## Codex Provider

```yaml
vendor:
  name: codex
  model: gpt-5.4-mini
```

Rules:
- `codex` is an opt-in provider name.
- The provider uses local Codex CLI non-interactive execution.
- The provider must fail with setup diagnostics when Codex CLI is unavailable
  or unauthenticated.
- The provider must not require undocumented token extraction.

## Compatibility

Legacy definitions without `contract` remain valid:

```yaml
---
name: legacy-agent
enabled: true
schedule:
  type: manual
vendor:
  name: openai
  model: gpt-5.4-mini
---
Return a plain-text answer.
```
