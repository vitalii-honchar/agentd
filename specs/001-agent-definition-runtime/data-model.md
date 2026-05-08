# Data Model: Agent Definition Runtime

## Agent Definition

User-authored Markdown file with YAML front matter and Markdown prompt body.

Fields:
- `name`: unique Agent name in the local agentd system; required; stable ID.
- `enabled`: boolean; default `true`.
- `schedule.type`: `cron` or `manual`; required.
- `schedule.expression`: cron expression; required when `schedule.type = cron`;
  omitted for `manual`.
- `vendor.name`: LLM vendor name; required. Initial supported value: `openai`;
  future values include `openrouter` and `anthropic` through provider adapters.
- `vendor.model`: LLM model identifier; required.
- `tools`: list of declared local tool permissions; optional.
- `mcp_servers`: list of declared MCP server permissions; optional.
- `access.filesystem.read`: explicit readable paths; optional.
- `access.filesystem.write`: explicit writable paths; optional.
- `access.network.allow`: explicit network destinations; optional.
- `prompt`: Markdown body after front matter; required and non-empty.

Validation:
- `name` MUST be non-empty, unique, and CLI-safe: lowercase letters, numbers,
  hyphen, underscore, and dot.
- Cron expressions MUST parse before apply succeeds.
- `schedule.type = manual` MUST NOT produce an automatic next run.
- Secrets such as `OPENAI_API_KEY` MUST NOT be required in the Markdown file.
- Tool/MCP/access entries MUST be explicit; missing entries mean denied.
- Provider-specific credentials come from daemon environment/configuration, not
  Agent Definition secrets.

## Agent

Daemon-owned active record created from the latest applied Agent Definition.

Fields:
- `name`: primary key.
- `revision`: deterministic hash of normalized definition content.
- `definition_source_path`: path submitted by CLI for user visibility.
- `definition_markdown`: applied Markdown content.
- `prompt`: normalized prompt content used for runs.
- `enabled`: current enabled state.
- `vendor_name`, `vendor_model`: provider selection.
- `schedule_type`, `schedule_expression`, `next_run_at`: scheduling state.
- `status`: `active`, `disabled`, or `invalid`.
- `last_run_id`: nullable reference to latest Agent Run.
- `last_error`: latest validation/runtime summary, nullable.
- `created_at`, `updated_at`, `applied_at`: timestamps.

Relationships:
- One Agent has many Agent Runs.
- One Agent has many Tool Permissions.
- One Agent has many Runtime Events.

State transitions:
- `missing -> active`: valid first apply.
- `active -> active`: changed valid apply updates revision.
- `active -> disabled`: valid apply with `enabled=false`.
- `active/disabled -> active`: valid apply re-enables or changes schedule.
- `any -> unchanged`: apply identical definition.
- `any -> unchanged with validation error`: invalid apply is rejected and does
  not mutate active Agent state.

## Agent Run

One execution attempt for an Agent.

Fields:
- `id`: UUID.
- `agent_name`: Agent foreign key.
- `agent_revision`: revision used for this run.
- `trigger`: `schedule`, `manual`, or `recovery`.
- `status`: `queued`, `running`, `completed`, `failed`, `stopping`,
  `stopped`, or `interrupted`.
- `started_at`, `completed_at`: timestamps.
- `due_at`: schedule due time for scheduled runs, nullable.
- `work_dir`: isolated local working directory for the run.
- `log_path`: isolated run log file path.
- `provider_request_id`: provider-side request ID, nullable.
- `error_code`, `error_message`: nullable failure details.
- `stop_requested_at`: nullable timestamp.

Relationships:
- Many Agent Runs belong to one Agent.
- Agent Run log files are referenced by `log_path` and served through logs
  use cases.

State transitions:
- `queued -> running`: runtime worker starts.
- `running -> completed`: provider/tool execution succeeds.
- `running -> failed`: provider/tool/runtime error.
- `running -> stopping -> stopped`: stop request cancels context successfully.
- `running -> interrupted`: daemon restart recovery finds a previously active
  run.

Concurrency rules:
- Different Agents run concurrently by default.
- Same-Agent overlap defaults to rejected while a run is active.
- A failed/stopped run MUST NOT alter the state of other active runs.

## Tool Permission

Declared capability available to Agent Runs for an Agent.

Fields:
- `agent_name`: Agent foreign key.
- `kind`: `local_tool` or `mcp_server`.
- `name`: logical permission name.
- `command`: executable command for MCP/local tool, nullable.
- `args`: argument list, nullable.
- `env`: explicit environment variable names allowed for this tool.
- `read_paths`, `write_paths`: explicit filesystem paths allowed.
- `network_allow`: explicit destinations allowed.

Validation:
- Tool names MUST be unique within an Agent Definition.
- Environment variables are allow-listed by name; values come from daemon
  runtime environment.
- Paths are normalized before use.

## Runtime Event

Structured service-level event for daemon and runtime activity.

Fields:
- `id`: UUID or monotonic generated ID.
- `agent_name`: nullable.
- `run_id`: nullable.
- `event_type`: stable string such as `agent.apply.created`,
  `agent.run.started`, `agent.run.failed`, `daemon.recovery.completed`.
- `level`: `debug`, `info`, `warn`, `error`.
- `message`: human-readable summary.
- `attributes_json`: JSON object for structured details.
- `created_at`: timestamp.

## SQLite Schema Sketch

The daemon uses multiple SQLite databases:

- **Settings DB**: one durable database for all Agent Definitions, Agent
  metadata, schedules, and access policy.
- **Runtime DBs**: one SQLite database per Agent for that Agent's Agent Runs,
  runtime events, and log references.

This split keeps Agent Definition reads/writes stable and prevents one busy or
stalled Agent from blocking write-heavy runtime data for all other Agents.

Settings DB tables:
- `agents`
- `agent_tools`
- `agent_mcp_servers`

Per-Agent runtime DB tables:
- `agent_runs`
- `runtime_events`

Migration layout:

```text
internal/agentdserver/infra/db/
├── db.go
├── migrations/
│   ├── settings/
│   │   ├── 001_init.sql
│   │   └── 002_agent_policy_indexes.sql
│   └── runtime/
│       ├── 001_init.sql
│       ├── 002_run_logs.sql
│       └── 003_runtime_event_indexes.sql
└── repository/
```

`settings/001_init.sql` responsibilities:
- Create `agents` with unique `name`, revision, definition source/content,
  vendor/model, schedule fields, status, last run/error fields, and timestamps.
- Create `agent_tools` and `agent_mcp_servers` with `ON DELETE CASCADE` back to
  `agents`.
- Create baseline indexes for Agent lookup, schedule lookup, and enabled Agent
  discovery.

`settings/002_agent_policy_indexes.sql` responsibilities:
- Add indexes for tool, MCP server, and access-policy lookups by Agent name.

`runtime/001_init.sql` responsibilities:
- Create `agent_runs` with run identity, Agent name/revision, trigger, status,
  due/start/completion timestamps, work directory, log path, provider request
  ID, stop request timestamp, and error fields.
- Create `runtime_events` for the owning Agent.
- Create baseline indexes for run lookup by time, active run lookup, and
  scheduled run de-duplication.

`runtime/002_run_logs.sql` responsibilities:
- Add or normalize log reference fields if implementation separates run log
  metadata from `agent_runs`.
- Add indexes needed by `agentd logs <agent_name>` and `agentd logs --run`.

`runtime/003_runtime_event_indexes.sql` responsibilities:
- Add indexes for `run_id`, `event_type`, and `created_at DESC`.

Settings DB indexes:
- `agents(name)`
- `agents(enabled, schedule_type, next_run_at)`
- `agent_tools(agent_name)`
- `agent_mcp_servers(agent_name)`

Per-Agent runtime DB indexes:
- `agent_runs(started_at DESC)`
- `agent_runs(status)`
- `agent_runs(due_at)`
- `runtime_events(run_id, created_at DESC)`
- `runtime_events(event_type, created_at DESC)`

Persistence rules:
- Apply updates Agent and permission rows in one settings DB transaction.
- The daemon creates or opens the Agent runtime DB after a successful first
  apply and runs `runtime/*.sql` migrations before scheduling or execution.
- Scheduled run creation uses a uniqueness guard on `(agent_name, due_at)` when
  `due_at` is present. Since each runtime DB belongs to one Agent, this guard is
  local to that Agent's database.
- Startup recovery enumerates Agents from settings DB, opens each Agent runtime
  DB, updates active runs to `interrupted`, then schedules future due runs.
