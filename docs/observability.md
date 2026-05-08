# Observability

agentd records two kinds of operational evidence:

- Service logs through `log/slog` for daemon lifecycle and component startup or shutdown.
- Isolated Agent Run logs on disk, referenced by each Agent's runtime SQLite database and exposed through `agentd logs`.

## Service Log Events

Use stable event names in the message or `event` attribute when adding new daemon logs:

- `daemon.starting`
- `daemon.started`
- `daemon.stopping`
- `daemon.stopped`
- `component.start.failed`
- `component.stop.failed`
- `agent.apply.created`
- `agent.apply.updated`
- `agent.apply.unchanged`
- `agent.apply.rejected`
- `agent.apply.failed`
- `agent.run.started`
- `agent.run.completed`
- `agent.run.failed`
- `agent.run.stopping`
- `agent.run.stopped`
- `daemon.recovery.completed`

Common attributes:

- `component`: daemon component name, such as `settings`, `scheduler`, or `http`.
- `agent`: Agent name.
- `run_id`: Agent Run ID.
- `outcome`: apply result such as `created`, `updated`, `unchanged`, `rejected`, or `failed`.
- `revision`: Agent Definition revision hash.
- `source_path`: Agent Definition path supplied by the CLI.
- `status`: Agent or Agent Run status.
- `trigger`: `manual`, `schedule`, or `recovery`.
- `error`: sanitized error value. Do not log API keys, `.env` values, prompts containing secrets, or credential material.

## Run Logs

Each Agent Run writes to one file under `AGENTD_RUN_LOG_DIR/<agent>/<run_id>.log`.
The runtime stores the log path in that Agent's runtime SQLite database.

Operational rules:

- `agentd logs <agent_name>` reads the latest run for that Agent.
- `agentd logs <agent_name> --run <run_id>` reads a specific run.
- Logs are not mixed across Agents.
- Missing or pruned log files return a consistent `not_found` API error.

## Runtime Events

Per-Agent runtime databases include a `runtime_events` table for structured lifecycle and policy events. Use it for events that must survive service log rotation or that need to be correlated with an Agent Run.
