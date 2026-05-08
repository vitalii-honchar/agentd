# GitHub Trending Engineering Radar

Daily scheduled software-engineering agent that scans public GitHub trend signals for repositories worth evaluating. This is repeatable because repository momentum and new tools change daily.

## Install

Install Python 3.10+. The default tool uses the standard library and public GitHub endpoints.

## Run

This example is zero configuration. API keys are not required; `GITHUB_TOKEN` can be used as an optional enhancement for higher rate limits.

```sh
agentd apply examples/github-trending-engineering-radar/github-trending-engineering-radar.md
agentd execute github-trending-engineering-radar
agentd result <agent-name>
agentd result <run-id>
agentd logs github-trending-engineering-radar --run <run-id>
```

Use `agentd result github-trending-engineering-radar` for the compact daily history.
