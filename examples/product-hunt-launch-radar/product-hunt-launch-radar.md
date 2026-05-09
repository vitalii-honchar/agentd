---
name: product-hunt-launch-radar
enabled: true
schedule:
  type: cron
  expression: "0 9 * * *"
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
        "interesting_launches": {
          "type": "array",
          "items": {
            "type": "object",
            "additionalProperties": false,
            "properties": {
              "name": { "type": "string" },
              "url": { "type": "string" },
              "why_it_matters": { "type": "string" },
              "category": { "type": "string" },
              "positioning_signal": { "type": "string" }
            },
            "required": ["name", "url", "why_it_matters", "category", "positioning_signal"]
          }
        },
        "positioning_patterns": {
          "type": "array",
          "items": { "type": "string" }
        },
        "product_or_competitive_notes": {
          "type": "array",
          "items": { "type": "string" }
        },
        "no_action_note": { "type": "string" }
      },
      "required": [
        "summary",
        "interesting_launches",
        "positioning_patterns",
        "product_or_competitive_notes",
        "no_action_note"
      ]
    }
tools:
  - name: fetch_product_hunt_launches
    kind: custom_tool
    command: tools/fetch_product_hunt_launches.py
    args: ["--fixture", "fixtures/product_hunt_sample.json"]
    timeout: 45s
    network:
      allow:
        - https://www.producthunt.com
access:
  filesystem:
    read: ["fixtures/", "sources/", "tools/"]
    write: [".agentd-work/"]
  network:
    allow:
      - https://www.producthunt.com
---
You are a product strategist reviewing Product Hunt launches for useful market signals.

Use the fetch_product_hunt_launches tool. Identify launches with clear buyer pain, unusual positioning, strong developer or AI workflows, interesting pricing/package clues, and crowded categories.

Return sections:
- Most interesting launches
- Why each matters
- Category and positioning patterns
- Product ideas or competitive watch notes
- No-action note when launches are low signal
