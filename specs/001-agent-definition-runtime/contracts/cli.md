# Contract: agentd CLI

The CLI is a thin client. It validates local command arguments, reads files when
needed, sends REST requests to `agentdserver`, and formats responses.

## Global Options

- `--server URL`: daemon URL; default from `AGENTD_SERVER_URL`, then
  `http://127.0.0.1:18080`.
- `--output text|json`: output format; default `text`.

## Commands

### `agentd apply <path_to_file>`

Reads a Markdown Agent Definition and submits it to the daemon.

Success outcomes:
- `created`
- `updated`
- `unchanged`

Failure outcomes:
- validation errors with field paths
- daemon unavailable
- file unreadable

### `agentd execute <agent_name>`

Requests an immediate Agent Run for an applied Agent. Works for manual and cron
Agents.

Success output:
- run ID
- Agent name
- initial status

Failure outcomes:
- unknown Agent
- disabled/invalid Agent
- same Agent already running
- daemon unavailable

### `agentd stop <agent_name> [--run <run_id>]`

Requests cancellation. Without `--run`, stops the current active run for the
Agent when one exists.

### `agentd inspect <agent_name>`

Shows Agent metadata, current state, revision, schedule mode, next scheduled
run, last run, and recent failure summary.

### `agentd logs <agent_name> [--run <run_id>] [--tail N]`

Shows isolated logs for an Agent. Without `--run`, returns recent logs for the
latest run or recent runs according to daemon policy.

### `agentd list`

Lists applied Agents with enabled state, status, schedule mode, next run, and
last run status.
