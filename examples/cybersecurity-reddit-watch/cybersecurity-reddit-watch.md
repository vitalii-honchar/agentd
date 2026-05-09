---
name: cybersecurity-reddit-watch
enabled: true
schedule:
  type: cron
  expression: "0 8 * * *"
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
        "executive_summary": { "type": "string" },
        "high_signal_posts": {
          "type": "array",
          "items": {
            "type": "object",
            "additionalProperties": false,
            "properties": {
              "title": { "type": "string" },
              "url": { "type": "string" },
              "reason": { "type": "string" },
              "severity": {
                "type": "string",
                "enum": ["low", "medium", "high", "critical"]
              }
            },
            "required": ["title", "url", "reason", "severity"]
          }
        },
        "indicators": {
          "type": "array",
          "items": { "type": "string" }
        },
        "follow_up_actions": {
          "type": "array",
          "items": { "type": "string" }
        },
        "no_action_note": { "type": "string" }
      },
      "required": [
        "executive_summary",
        "high_signal_posts",
        "indicators",
        "follow_up_actions",
        "no_action_note"
      ]
    }
tools:
  - name: fetch_reddit_cybersecurity
    kind: custom_tool
    command: tools/fetch_reddit_cybersecurity.py
    args: ["--subreddit", "cybersecurity", "--limit", "25"]
    timeout: 60s
    network:
      allow:
        - https://www.reddit.com
        - https://oauth.reddit.com
access:
  filesystem:
    read: ["sources/", "tools/"]
    write: [".agentd-work/"]
  network:
    allow:
      - https://www.reddit.com
      - https://oauth.reddit.com
---
You are a cybersecurity intelligence analyst watching public r/cybersecurity posts.

Use the fetch_reddit_cybersecurity tool to collect recent public posts. Identify new vulnerability disclosures, exploit chatter, breach reports, leaked data claims, exposed credential mentions, and unusually urgent defensive guidance.

Return sections:
- Executive summary
- High-signal posts with URL, reason, and severity
- Possible vulnerability or data leak indicators
- Follow-up actions for a security team
- No-action note when nothing meaningful changed
