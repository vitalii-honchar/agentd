---
name: developer-dependency-release-monitor
enabled: true
schedule:
  type: cron
  expression: "45 8 * * *"
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
        "releases_requiring_attention": {
          "type": "array",
          "items": {
            "type": "object",
            "additionalProperties": false,
            "properties": {
              "dependency": { "type": "string" },
              "version": { "type": "string" },
              "url": { "type": "string" },
              "why_it_matters": { "type": "string" },
              "upgrade_priority": {
                "type": "string",
                "enum": ["low", "medium", "high", "urgent"]
              },
              "migration_or_test_impact": { "type": "string" }
            },
            "required": [
              "dependency",
              "version",
              "url",
              "why_it_matters",
              "upgrade_priority",
              "migration_or_test_impact"
            ]
          }
        },
        "routine_releases": {
          "type": "array",
          "items": { "type": "string" }
        },
        "no_action_note": { "type": "string" }
      },
      "required": ["summary", "releases_requiring_attention", "routine_releases", "no_action_note"]
    }
tools:
  - name: fetch_dependency_releases
    kind: custom_tool
    command: tools/fetch_dependency_releases.py
    args: ["--sources", "sources/dependencies.json"]
    timeout: 60s
    network:
      allow:
        - https://api.github.com
        - https://registry.npmjs.org
        - https://pypi.org
access:
  filesystem:
    read: ["fixtures/", "sources/", "tools/"]
    write: [".agentd-work/"]
  network:
    allow:
      - https://api.github.com
      - https://registry.npmjs.org
      - https://pypi.org
---
You are a software engineer monitoring dependency releases that commonly affect application teams.

Use the fetch_dependency_releases tool with the bundled public dependency list. Identify releases that include security fixes, breaking changes, performance improvements, deprecations, or migration work.

Return sections:
- Releases requiring attention
- Why each release matters
- Suggested upgrade priority
- Possible migration or test impact
- No-action note when releases are routine
