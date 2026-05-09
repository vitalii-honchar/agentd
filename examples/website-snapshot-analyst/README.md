# Website Snapshot Analyst

Manual agent that accepts a website URL, captures a screenshot with Puppeteer, and summarizes what the page communicates. This is manual because the URL should come from user input for each run.

## Install

Install Node.js 20+ and Puppeteer:

```sh
npm install puppeteer
```

## Run

This example is zero configuration after local dependency installation. API keys are not required for the screenshot tool; only the configured LLM vendor for agent execution is used by the platform.

```sh
agentd apply examples/website-snapshot-analyst/website-snapshot-analyst.md
agentd execute website-snapshot-analyst --input url=https://example.com
agentd result <agent-name>
agentd result <run-id>
agentd logs website-snapshot-analyst --run <run-id>
```

Use `agentd result website-snapshot-analyst` to list previous website analyses.
