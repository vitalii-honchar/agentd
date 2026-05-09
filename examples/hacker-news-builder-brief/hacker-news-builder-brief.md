---
name: hacker-news-builder-brief
enabled: true
schedule:
  type: cron
  expression: "0 7 * * *"
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
        "important_items": {
          "type": "array",
          "items": {
            "type": "object",
            "additionalProperties": false,
            "properties": {
              "title": { "type": "string" },
              "url": { "type": "string" },
              "why_it_matters": { "type": "string" }
            },
            "required": ["title", "url", "why_it_matters"]
          }
        },
        "engineering_implications": {
          "type": "array",
          "items": { "type": "string" }
        },
        "tools_or_libraries": {
          "type": "array",
          "items": { "type": "string" }
        },
        "risks_or_security_notes": {
          "type": "array",
          "items": { "type": "string" }
        },
        "skip_list": {
          "type": "array",
          "items": { "type": "string" }
        }
      },
      "required": [
        "summary",
        "important_items",
        "engineering_implications",
        "tools_or_libraries",
        "risks_or_security_notes",
        "skip_list"
      ]
    }
tools:
  - name: fetch_hacker_news
    kind: custom_tool
    command: tools/fetch_hacker_news.py
    args: ["--limit", "30"]
    timeout: 45s
    network:
      allow:
        - https://hacker-news.firebaseio.com
access:
  filesystem:
    read: ["fixtures/", "sources/", "tools/"]
    write: [".agentd-work/"]
  network:
    allow:
      - https://hacker-news.firebaseio.com
---
You are a senior software engineer preparing a daily builder brief from Hacker News.

Use the fetch_hacker_news tool to read current public top stories through the official Hacker News Firebase API. Identify stories that matter to engineers building products: language/runtime releases, databases, infrastructure, security, AI engineering, developer tools, browsers, and operational lessons.

Return sections:
- Five most important items with URL and why they matter
- Engineering implications
- Tools or libraries worth trying
- Risks or security notes
- Short "skip" list for noisy items
