# Contract: Repository Examples

Each example is a self-contained folder under `examples/`.

## Folder Layout

```text
examples/<agent-name>/
├── <agent-name>.md
├── README.md
├── tools/
│   └── <tool files>
└── sources/ or fixtures/
    └── <public source lists or bundled fixtures>
```

## Required Examples

1. `cybersecurity-reddit-watch`: daily scheduled monitor for public
   `r/cybersecurity` posts. No Reddit API credentials required for default run.
2. `hacker-news-builder-brief`: daily scheduled public Hacker News brief.
3. `reddit-customer-pain-monitor`: daily scheduled monitor for bundled public
   subreddit list.
4. `product-hunt-launch-radar`: daily scheduled public Product Hunt launch
   monitor.
5. `github-trending-engineering-radar`: daily scheduled GitHub trend monitor.
6. `developer-dependency-release-monitor`: daily scheduled monitor for bundled
   public release/changelog pages.
7. `ai-engineering-hiring-signal-monitor`: daily scheduled monitor for bundled
   public hiring signal sources.
8. `website-snapshot-analyst`: manual URL screenshot and summary agent.

## README Requirements

Each `README.md` must include:
- What the example does and why it is repeatable or manual.
- Local dependency installation.
- Whether the default run uses public network access or fixtures.
- Apply command.
- Execute command for manual examples.
- Result lookup command by agent name.
- Full result lookup command by run ID.
- Logs command.
- Optional API keys, if any, clearly marked as optional enhancements.

## Zero-Configuration Rule

Default runs must not require:
- External account creation.
- Required API keys.
- CI setup.
- SaaS integrations.
- Private repositories or private data.
- User-specific source configuration before first run.

## Agent Definition Requirements

Each definition declares:
- Unique agent name.
- Daily cron schedule for monitoring examples or `manual` for URL snapshot.
- LLM vendor/model placeholders consistent with runtime conventions.
- Declared tools with command paths inside the example folder.
- Public network access needed by tools.
- Expected result shape.

## Smoke Test Contract

For each example:
1. Install documented local dependencies.
2. Apply the example definition.
3. Execute manual examples or trigger scheduled examples through the runtime
   test hook/manual execute path.
4. Confirm run reaches `COMPLETED` or `FAILED`.
5. Confirm `agentd result <agent-name>` returns a row.
6. Confirm `agentd result <run-id>` returns full output.
7. Confirm `agentd logs <agent-name> --run <run-id>` contains action logs.
