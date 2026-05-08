# Product Hunt Launch Radar

Daily scheduled product agent that reviews public Product Hunt launch data or the bundled fixture and highlights launches worth a product manager's attention. This is repeatable because new launches appear every day.

## Install

Install Python 3.10+. The default run uses a bundled fixture, so no package installation is required.

## Run

This example is zero configuration by default. API keys are not required; a future Product Hunt API token can be treated as an optional enhancement only.

```sh
agentd apply examples/product-hunt-launch-radar/product-hunt-launch-radar.md
agentd execute product-hunt-launch-radar
agentd result <agent-name>
agentd result <run-id>
agentd logs product-hunt-launch-radar --run <run-id>
```

Use `agentd result product-hunt-launch-radar` to review the daily launch history.
