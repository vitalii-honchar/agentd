# Reddit Customer Pain Monitor

Daily scheduled product-management agent that reviews a bundled list of public subreddits for repeated complaints, unmet needs, and buying-intent language. This is repeatable because new public discussions appear daily and product teams can use the summary for discovery.

## Install

Install Python 3.10+. No required packages are needed for the default public JSON fetcher.

## Run

This example is zero configuration. API keys are not required; optional Reddit credentials may be added later only to raise rate limits.

```sh
agentd apply examples/reddit-customer-pain-monitor/reddit-customer-pain-monitor.md
agentd execute reddit-customer-pain-monitor
agentd result <agent-name>
agentd result <run-id>
agentd logs <run-id>
```

Use `agentd result reddit-customer-pain-monitor` to compare daily pain themes over time.
Run results are finalized as JSON matching the example contract, including `summary`, `recurring_pains`, and `opportunity_hypotheses`.
