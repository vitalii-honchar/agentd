# Contract: agentd CLI Extensions

The CLI remains a thin client. It validates local arguments, reads files when
needed, sends requests to the local daemon, and formats responses.

## Global Options

- `--server URL`: daemon URL; default from `AGENTD_SERVER_URL`, then
  `http://127.0.0.1:18080`.
- `--output text|json`: output format; default `text`. JSON is the stable
  machine-readable format for Bash scripts and local AI agents.

## Commands

### `agentd list`

Lists applied Agent Definitions.

Text columns:
- `NAME`
- `ENABLED`
- `STATUS`
- `SCHEDULE`
- `NEXT RUN`
- `LAST RUN`

JSON shape:

```json
{
  "agents": [
    {
      "name": "hacker-news-builder-brief",
      "enabled": true,
      "status": "active",
      "schedule_type": "cron",
      "next_run_at": "2026-05-09T08:00:00Z",
      "last_run_status": "completed"
    }
  ]
}
```

### `agentd ps [-a]`

Lists Agent Runs. Without `-a`, returns active runs only. With `-a`, returns
active and terminal runs.

Text columns:
- `RUN ID`
- `AGENT`
- `STATUS`
- `TRIGGER`
- `STARTED`
- `COMPLETED`

JSON shape:

```json
{
  "runs": [
    {
      "run_id": "3c8590e6-7eb5-4db8-a90f-4e0df9f72b05",
      "agent_name": "cybersecurity-reddit-watch",
      "status": "completed",
      "trigger": "schedule",
      "started_at": "2026-05-08T08:00:00Z",
      "completed_at": "2026-05-08T08:01:12Z"
    }
  ]
}
```

Exit codes:
- `0`: command succeeded.
- non-zero: daemon unavailable or malformed arguments.

### `agentd result <agent-name>`

Returns terminal run results for one agent as a compact table.

Text columns:
- `RUN ID`
- `STATUS`
- `COMPLETED`
- `RESULT`

JSON shape:

```json
{
  "agent_name": "hacker-news-builder-brief",
  "results": [
    {
      "run_id": "3c8590e6-7eb5-4db8-a90f-4e0df9f72b05",
      "status": "completed",
      "completed_at": "2026-05-08T08:01:12Z",
      "result_summary": "Top stories: SQLite on servers, Go release notes..."
    }
  ]
}
```

Exit codes:
- `0`: command succeeded, even when no terminal runs exist.
- `2`: agent not found.
- `10`: daemon communication failure.

### `agentd result <agent-run-id>`

Returns full details for one run.

JSON shape:

```json
{
  "run_id": "3c8590e6-7eb5-4db8-a90f-4e0df9f72b05",
  "agent_name": "hacker-news-builder-brief",
  "status": "completed",
  "trigger": "schedule",
  "started_at": "2026-05-08T08:00:00Z",
  "completed_at": "2026-05-08T08:01:12Z",
  "result": "Full untrimmed result text...",
  "failure": null
}
```

Exit codes:
- `0`: terminal result found.
- `3`: run not found.
- `4`: run exists but has no final result yet.
- `5`: run failed and the failed-run result was returned.
- `10`: daemon communication failure.

### `agentd execute <agent-name> [--input key=value]...`

Requests immediate execution. Manual examples that need run-time input, such as
`website-snapshot-analyst`, use `--input`.

JSON shape:

```json
{
  "run_id": "3c8590e6-7eb5-4db8-a90f-4e0df9f72b05",
  "agent_name": "website-snapshot-analyst",
  "status": "queued"
}
```

### `agentd logs <agent-name> [--run <run-id>] [--tail N]`

Returns scoped run action logs. Logs must contain runtime actions, not only the
final LLM response.
