# Cybersecurity Reddit Watch

Daily scheduled agent that reviews public posts from `https://www.reddit.com/r/cybersecurity/` and summarizes vulnerability disclosures, exploit chatter, and possible data leak signals. This is repeatable because new subreddit posts arrive continuously and security teams need a compact daily signal.

## Install

Install Python 3.10+ and the local dependency:

```sh
python3 -m pip install praw
```

The default tool can also fall back to Reddit's public JSON endpoint when PRAW credentials are absent.

## Run

This example is zero configuration for the default public-read path. API keys are optional enhancements only; set Reddit PRAW environment variables if you want authenticated higher-rate reads.

```sh
agentd apply examples/cybersecurity-reddit-watch/cybersecurity-reddit-watch.md
agentd execute cybersecurity-reddit-watch
agentd result <agent-name>
agentd result <run-id>
agentd logs cybersecurity-reddit-watch --run <run-id>
```

Use `agentd result cybersecurity-reddit-watch` for the compact history table.
