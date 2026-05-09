---
name: ai-engineering-hiring-signal-monitor
enabled: true
schedule:
  type: cron
  expression: "20 9 * * *"
vendor:
  name: openai
  model: gpt-5.4-mini
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
        "summary": { "type": "string" },
        "strong_hiring_signals": {
          "type": "array",
          "items": {
            "type": "object",
            "additionalProperties": false,
            "properties": {
              "signal": { "type": "string" },
              "why_it_matters": { "type": "string" },
              "evidence_links": {
                "type": "array",
                "items": { "type": "string" }
              }
            },
            "required": ["signal", "why_it_matters", "evidence_links"]
          }
        },
        "repeated_skills_and_tools": {
          "type": "array",
          "items": { "type": "string" }
        },
        "product_or_platform_opportunities": {
          "type": "array",
          "items": { "type": "string" }
        },
        "no_action_note": { "type": "string" }
      },
      "required": [
        "summary",
        "strong_hiring_signals",
        "repeated_skills_and_tools",
        "product_or_platform_opportunities",
        "no_action_note"
      ]
    }
tools:
  - name: fetch_ai_hiring_signals
    kind: custom_tool
    command: tools/fetch_ai_hiring_signals.py
    args: ["--sources", "sources/hiring_sources.json"]
    timeout: 60s
    network:
      allow:
        - https://news.ycombinator.com
        - https://www.reddit.com
access:
  filesystem:
    read: ["fixtures/", "sources/", "tools/"]
    write: [".agentd-work/"]
  network:
    allow:
      - https://news.ycombinator.com
      - https://www.reddit.com
---
You are a product-minded engineering leader monitoring public AI engineering hiring signals.

Use the fetch_ai_hiring_signals tool to inspect bundled public sources. Identify recurring skills, infrastructure choices, evaluation needs, data/security concerns, and workflow gaps that appear in AI engineering hiring conversations.

Return sections:
- Strong hiring signals
- Skills and tools appearing repeatedly
- Product or platform opportunities
- Evidence links
- No-action note when the signal is weak
