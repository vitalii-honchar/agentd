# Developer Dependency Release Monitor

Daily scheduled engineering agent that watches a bundled list of public dependency release sources and summarizes updates worth an engineer's attention. This is repeatable because dependencies release new versions and advisories continuously.

## Install

Install Python 3.10+. The default tool uses the standard library and public package/repository endpoints.

## Run

This example is zero configuration and does not need access to your codebase. API keys are not required; `GITHUB_TOKEN` is an optional enhancement for GitHub rate limits.

```sh
agentd apply examples/developer-dependency-release-monitor/developer-dependency-release-monitor.md
agentd execute developer-dependency-release-monitor
agentd result <agent-name>
agentd result <run-id>
agentd logs developer-dependency-release-monitor --run <run-id>
```

Use `agentd result developer-dependency-release-monitor` to review release history across days.
