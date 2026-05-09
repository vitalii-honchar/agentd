---
name: website-snapshot-analyst
enabled: true
schedule:
  type: manual
vendor:
  name: openai
  model: gpt-5.4-mini
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
        "audience_and_value": { "type": "string" },
        "important_visible_content": {
          "type": "array",
          "items": { "type": "string" }
        },
        "ux_or_trust_issues": {
          "type": "array",
          "items": { "type": "string" }
        },
        "follow_up_questions": {
          "type": "array",
          "items": { "type": "string" }
        }
      },
      "required": [
        "website_summary",
        "audience_and_value",
        "important_visible_content",
        "ux_or_trust_issues",
        "follow_up_questions"
      ]
    }
inputs:
  - name: url
    required: true
    description: Website URL to screenshot and summarize
tools:
  - name: capture_website
    kind: custom_tool
    command: tools/capture_website.js
    args: ["--url", "{{inputs.url}}", "--output", ".agentd-work/screenshot.png"]
    timeout: 90s
    network:
      allow:
        - "*"
access:
  filesystem:
    read: ["fixtures/", "tools/", ".agentd-work/"]
    write: [".agentd-work/"]
  network:
    allow:
      - "*"
---
You are a website analyst helping a user understand a provided public URL.

Use the capture_website tool with the user-provided url input. Inspect the captured page metadata and screenshot output. Summarize what the website is, who it appears to serve, the primary call to action, trust signals, possible usability issues, and anything surprising.

Return sections:
- Website summary
- Audience and value proposition
- Important visible content
- UX or trust issues
- Follow-up questions for deeper analysis
