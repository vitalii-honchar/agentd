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
