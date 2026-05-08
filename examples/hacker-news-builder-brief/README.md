# Hacker News Builder Brief

Daily scheduled agent that reads public Hacker News top stories through `https://github.com/HackerNews/API` and summarizes the most important news for software builders. This is repeatable because the top-story set changes every day.

## Install

Install Python 3.10+. No Python packages are required for the default tool; it uses the standard library.

## Run

This example is zero configuration and uses public network access only. API keys are not required and there are no optional API keys for the default workflow.

```sh
agentd apply examples/hacker-news-builder-brief/hacker-news-builder-brief.md
agentd result <agent-name>
agentd result <run-id>
agentd logs hacker-news-builder-brief --run <run-id>
```

Use `agentd execute hacker-news-builder-brief` to run it manually during local testing even though the normal schedule is daily.
