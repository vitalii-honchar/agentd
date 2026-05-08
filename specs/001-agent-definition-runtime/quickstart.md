# Quickstart: Agent Definition Runtime

This quickstart validates the planned Agent-as-Code workflow on a local
developer machine.

## Prerequisites

- Go 1.26.2
- Linux or macOS
- `OPENAI_API_KEY` available in `.env` or the shell environment when executing
  OpenAI-backed Agents

Optional local overrides:

```bash
export AGENTD_DATA_DIR=./data
export AGENTD_SERVER_URL=http://127.0.0.1:18080
```

## 1. Start the daemon

```bash
go run ./cmd/agentdserver
```

Expected:
- The daemon starts on `127.0.0.1:18080` by default.
- Service logs are emitted through `slog`.
- The settings SQLite database is created locally if missing.
- Each applied Agent gets its own runtime SQLite database for run/event/log
  metadata.
- Runtime state defaults to `./data`.

## 2. Create a manual Agent Definition

Create `examples/release-notes-helper.md`:

```markdown
---
name: release-notes-helper
enabled: true
schedule:
  type: manual
vendor:
  name: openai
  model: gpt-5
tools: []
mcp_servers: []
access:
  filesystem:
    read: []
    write: []
  network:
    allow: ["api.openai.com"]
---
You summarize recent project changes into concise release notes.
```

## 3. Apply the definition

```bash
go run ./cmd/agentd apply examples/release-notes-helper.md
```

Expected:
- First apply returns `created`.
- Re-applying unchanged content returns `unchanged`.
- `agentd inspect release-notes-helper` shows schedule mode `manual` and no
  automatic next run.

List and inspect the applied Agent:

```bash
go run ./cmd/agentd list
go run ./cmd/agentd inspect release-notes-helper
```

## 4. Execute manually

```bash
go run ./cmd/agentd execute release-notes-helper
```

Expected:
- CLI prints an Agent Run ID.
- `agentd inspect release-notes-helper` shows the latest run state.
- The run has an isolated work directory and isolated log file.
- If `OPENAI_API_KEY` is missing or invalid, the run is recorded as failed with
  provider error details.

## 5. Read isolated logs

```bash
go run ./cmd/agentd logs release-notes-helper
```

Expected:
- Only logs for `release-notes-helper` are shown.
- Logs from other Agents are not mixed into this output.

Read a specific run if needed:

```bash
go run ./cmd/agentd logs release-notes-helper --run <run_id> --tail 100
```

## 6. Validate concurrent Agents

Apply at least five Agent Definitions with overlapping cron schedules or execute
five different manual Agents in quick succession:

```bash
go run ./cmd/agentd execute agent-a
go run ./cmd/agentd execute agent-b
go run ./cmd/agentd execute agent-c
go run ./cmd/agentd execute agent-d
go run ./cmd/agentd execute agent-e
```

Expected:
- Runs proceed concurrently.
- Each Agent Run remains independently inspectable and stoppable.
- Logs remain isolated per Agent and run.

## 7. Validate restart recovery

Start one or more long-running Agent Runs, stop the daemon process, then start it
again.

Expected:
- Previously active runs are marked `interrupted`.
- Future schedules are restored.
- Service logs include daemon recovery events.

## 8. Storage layout

Expected default files:

```text
data/
├── agentd-settings.db
├── agents/
│   └── release-notes-helper.db
├── logs/
│   └── release-notes-helper/<run_id>.log
└── work/
    └── release-notes-helper/<run_id>/
```

## 9. Run automated verification

```bash
go test ./...
```

Expected:
- Unit, repository, REST, CLI/server integration, concurrency, and recovery tests
  pass.

## 10. Product research example

`examples/ai-product-research.md` models the Python Agent from
`/path/to/ai-product-research`.

The example declares:

- A manual schedule, so execution is controlled with `agentd execute ai-product-research`.
- OpenAI as the initial LLM vendor.
- A local `uv` script tool for the Python workflow.
- A Playwright/Chromium setup tool for website screenshot support.
- Environment variable names required by the script, without secret values.
- Explicit read/write paths and network destinations, including broad HTTPS
  egress for arbitrary product websites.

Before applying it, review the filesystem paths and install the Python project
dependencies in the source project:

```bash
cd /path/to/ai-product-research
uv sync
uv run python -m playwright install chromium
```

Then apply the definition:

```bash
go run ./cmd/agentd apply examples/ai-product-research.md
```

The current runtime persists and validates the script-tool declaration as part
of the Agent Definition. Executing local script tools from an Agent Run is the
next implementation step after the initial LLM-only runtime path.
