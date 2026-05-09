# AI Engineering Hiring Signal Monitor

Daily scheduled agent for product managers and engineering leaders that watches public AI engineering hiring sources for repeated skill and tooling signals. This is repeatable because hiring conversations and job-market signals shift frequently.

## Install

Install Python 3.10+. The default tool uses the standard library and bundled public source lists.

## Run

This example is zero configuration. API keys are not required; optional authenticated source access can be added later as an enhancement.

```sh
agentd apply examples/ai-engineering-hiring-signal-monitor/ai-engineering-hiring-signal-monitor.md
agentd execute ai-engineering-hiring-signal-monitor
agentd result <agent-name>
agentd result <run-id>
agentd logs <run-id>
```

Use `agentd result ai-engineering-hiring-signal-monitor` to compare hiring themes over time.
Run results are finalized as JSON matching the example contract, including `summary`, `strong_hiring_signals`, and product/platform opportunities.
