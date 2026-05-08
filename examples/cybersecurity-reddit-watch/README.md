# Cybersecurity Reddit Watch

Daily scheduled agent that reviews public posts from `https://www.reddit.com/r/cybersecurity/` and summarizes vulnerability disclosures, exploit chatter, and possible data leak signals. This is repeatable because new subreddit posts arrive continuously and security teams need a compact daily signal.

## Install

Install Python 3.10+.

```sh
python3 --version
```

The default tool reads Reddit's public JSON endpoint and requires no Reddit
configuration. PRAW is optional if you want Reddit API read-only access through
an app client:

```sh
python3 -m pip install praw
```

## Run

This example is zero configuration for the default public-read path. API keys
for a Reddit app are optional enhancements only; they are not Reddit account
login credentials and the tool forces PRAW read-only mode. Set
`REDDIT_CLIENT_ID`, `REDDIT_CLIENT_SECRET`, and optionally `REDDIT_USER_AGENT`
in your shell or local ignored `.env` file if you want PRAW read-only reads.

```sh
agentd apply examples/cybersecurity-reddit-watch/cybersecurity-reddit-watch.md
agentd execute cybersecurity-reddit-watch
agentd result <agent-name>
agentd result <run-id>
agentd logs cybersecurity-reddit-watch --run <run-id>
```

Use `agentd result cybersecurity-reddit-watch` for the compact history table.
