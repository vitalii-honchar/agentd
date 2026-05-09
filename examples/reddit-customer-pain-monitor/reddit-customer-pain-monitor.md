---
name: reddit-customer-pain-monitor
enabled: true
schedule:
  type: cron
  expression: "30 7 * * *"
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
        "recurring_pains": {
          "type": "array",
          "items": {
            "type": "object",
            "additionalProperties": false,
            "properties": {
              "pain": { "type": "string" },
              "audience": { "type": "string" },
              "urgency": { "type": "string" },
              "evidence_urls": {
                "type": "array",
                "items": { "type": "string" }
              }
            },
            "required": ["pain", "audience", "urgency", "evidence_urls"]
          }
        },
        "opportunity_hypotheses": {
          "type": "array",
          "items": { "type": "string" }
        },
        "audience_notes": {
          "type": "array",
          "items": { "type": "string" }
        },
        "no_action_note": { "type": "string" }
      },
      "required": [
        "summary",
        "recurring_pains",
        "opportunity_hypotheses",
        "audience_notes",
        "no_action_note"
      ]
    }
tools:
  - name: fetch_reddit_pain_posts
    kind: custom_tool
    command: tools/fetch_reddit_pain_posts.py
    args: ["--sources", "sources/subreddits.txt", "--limit", "40"]
    timeout: 75s
    network:
      allow:
        - https://www.reddit.com
access:
  filesystem:
    read: ["fixtures/", "sources/", "tools/"]
    write: [".agentd-work/"]
  network:
    allow:
      - https://www.reddit.com
---
You are a product manager looking for repeated public customer pains in Reddit discussions.

Use the fetch_reddit_pain_posts tool to collect recent public posts from the bundled subreddit list. Focus on pains that can inform roadmap, positioning, onboarding, support automation, or new product opportunities.

Return sections:
- Top recurring pains
- Evidence posts with URL and quoted short context
- Product opportunity hypotheses
- Urgency and audience notes
- No-action note when the day is low signal
