# Local Development

## Requirements

- Go 1.26.2
- Linux or macOS
- SQLite support through `modernc.org/sqlite` (no external database service)
- `OPENAI_API_KEY` in your shell or local `.env` when using the OpenAI provider

Do not commit `.env`, `.env.*`, API keys, tokens, private keys, or generated credential files.

## Configuration

The daemon and CLI load defaults from environment variables. A local `.env` file is supported for development convenience.

Daemon variables:

- `AGENTD_SERVER_HOST`, default `127.0.0.1`
- `AGENTD_SERVER_PORT`, default `18080`
- `AGENTD_DATA_DIR`, default `./data`
- `AGENTD_SETTINGS_DB_PATH`, default `./data/agentd-settings.db`
- `AGENTD_RUNTIME_DB_DIR`, default `./data/agents`
- `AGENTD_RUN_LOG_DIR`, default `./data/logs`
- `AGENTD_SQLITE_MAX_CONNS`, default `4`
- `OPENAI_API_KEY`, required only when executing Agents through OpenAI

CLI variables:

- `AGENTD_SERVER_URL`, default `http://127.0.0.1:18080`
- `AGENTD_OUTPUT`, default `text`; use `json` for structured command output
- `AGENTD_REQUEST_TIMEOUT`, default `10s`

## Common Commands

Run all tests:

```bash
go test ./...
```

Start the daemon:

```bash
go run ./cmd/agentdserver
```

Apply an Agent Definition:

```bash
go run ./cmd/agentd apply examples/release-notes-helper.md
```

Execute an Agent manually:

```bash
go run ./cmd/agentd execute release-notes-helper
```

Read isolated logs:

```bash
go run ./cmd/agentd logs release-notes-helper
```

## Storage Layout

By default, local runtime state is written under `./data`:

```text
data/
├── agentd-settings.db
├── agents/
│   └── <agent>.db
├── logs/
│   └── <agent>/<run_id>.log
└── work/
    └── <agent>/<run_id>/
```

The settings database stores applied Agent Definitions and schedule metadata.
Each Agent has a separate runtime database for Agent Runs and runtime events.
Run logs remain separate files so concurrent Agents do not contend on one log table.
