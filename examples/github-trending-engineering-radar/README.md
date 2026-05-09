# GitHub Trending Engineering Radar

Daily scheduled software-engineering agent that scans public GitHub trend signals for repositories worth evaluating. This is repeatable because repository momentum and new tools change daily.

## Install

Install Python 3.10+. The bundled `custom_tool` uses the standard library and public GitHub endpoints, with a checked-in fixture fallback when live GitHub search is unavailable.

## Run

This example is zero configuration. API keys are not required.

```sh
agentd apply examples/github-trending-engineering-radar/github-trending-engineering-radar.md
agentd run github-trending-engineering-radar
agentd result <agent-name>
agentd result <run-id>
agentd logs <run-id>
```

Use `agentd result github-trending-engineering-radar` for the compact daily history.
Run results are finalized as JSON matching the example contract, including `summary`, `repositories`, and `no_action_note`.

`agentd apply` creates an immutable revision under `data/work/github-trending-engineering-radar/<revision_id>`. The `custom_tool` script plus declared `sources/` and `fixtures/` files are copied into that revision, so an explicit run like `agentd run github-trending-engineering-radar:<revision_id>` keeps working even if this example folder is later edited or removed.
