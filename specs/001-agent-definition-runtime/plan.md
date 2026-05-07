# Implementation Plan: Agent Definition Runtime

**Branch**: `001-agent-definition-runtime` | **Date**: 2026-05-07 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/001-agent-definition-runtime/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Build `agentd` as a Docker-like local runtime for AI Agents defined as Markdown
files. The user-facing CLI submits definitions to a long-running daemon over
local REST endpoints; the daemon validates definitions, stores Agent state in
SQLite, schedules cron/manual execution, runs many isolated Agent Runs
concurrently, records service and per-run logs with `slog`, and exposes inspect,
stop, and logs operations back to the CLI.

The implementation follows the existing `fleeto` style: thin `cmd/*/main.go`
entrypoints, a service wiring root, domain types independent from transport and
persistence, application use cases, infrastructure adapters, embedded SQLite
migrations, and integration tests around the runtime boundary.

## Technical Context

**Language/Version**: Go 1.26.2
**Primary Dependencies**: `spf13/cobra` for CLI, standard `net/http` for REST,
standard `log/slog` for logs, `modernc.org/sqlite` for embedded SQLite,
`robfig/cron/v3` for cron-compatible schedules, `gopkg.in/yaml.v3` for Markdown
front matter, `joho/godotenv` for local `.env` loading, `google/uuid` for run
identifiers, `github.com/openai/openai-go/v3` for the first official OpenAI
provider adapter
**Storage**: One settings SQLite database with WAL mode for Agent Definitions,
Agent metadata, schedule fields, and access policy; one runtime SQLite database
per Agent for Agent Runs, runtime events, and log indexes; per-Agent-Run log
files on local disk for isolated execution logs; embedded SQL migrations under
`internal/agentdserver/infra/db/migrations/settings/*.sql` and
`internal/agentdserver/infra/db/migrations/runtime/*.sql`
**Testing**: `go test ./...`, focused application/domain unit tests, repository
tests with temporary SQLite files, HTTP contract tests, CLI/server integration
tests, concurrency/isolation tests, restart-recovery tests
**Target Platform**: Linux and macOS daemon plus local CLI
**Project Type**: daemon-service plus CLI in one Go module
**Performance Goals**: Support at least 25 applied Agents and at least 5
concurrent Agent Runs on a developer laptop; apply/inspect/log commands return
within 1 second for normal local state; daemon restart restores Agent schedules
within 5 seconds for 25 Agents
**Constraints**: No external cloud dependency except configured LLM vendors;
SQLite only for local persistence; `.env` may provide `OPENAI_API_KEY` but
secrets MUST NOT be written into Agent Definition files or logs; no heavy
frameworks or separate database service
**Scale/Scope**: Single-user local daemon for developer laptops/workstations;
multi-user auth, remote fleet management, and Telegram/chat management are out
of scope for this feature
**Daemon/Agent Impact**: Introduces daemon-owned apply, execute, stop, inspect,
logs, schedule, run recovery, and concurrent Agent Run lifecycle control
**Isolation Policy**: Per-run work directory, explicit environment construction,
declared filesystem/network/tool/MCP access, isolated per-run log writer,
context cancellation, no inherited host access by default
**State & Recovery**: Persist Agent definitions, revisions, schedules, and
access policy in settings DB; persist each Agent's run records, runtime events,
and log references in that Agent's runtime DB; mark active runs interrupted on
daemon restart; recompute next scheduled runs from Agent schedule fields
**Observability**: `slog` service logs with stable event names; runtime events
table; isolated Agent Run logs retrievable with `agentd logs <agent_name>` and
by run ID
**Architecture/Complexity**: Clean Architecture/DDD boundaries modeled after
`fleeto`: `domain` has entities and invariants, `app` has use cases and ports,
`infra` has HTTP, SQLite, Markdown, scheduler, provider, and runtime adapters;
LLM execution is vendor-agnostic through a provider port with OpenAI as the
first adapter; no abstraction beyond current runtime seams

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Daemon-first runtime**: PASS. All CLI commands call daemon REST endpoints;
  apply, schedule, execute, stop, recovery, and logging authority live in
  `agentdserver`.
- **Least-privilege isolation**: PASS. Agent Definitions declare host access;
  runtime constructs explicit per-run environment and work/log directories and
  denies undeclared access by policy.
- **Linux/macOS portability**: PASS. Process, signal, filesystem, service path,
  and cancellation behavior sit behind platform/runtime adapters and are tested
  on both target OS families where supported.
- **Durable recovery**: PASS. SQLite persists Agent settings and per-Agent run
  state; daemon startup opens settings DB, opens each Agent runtime DB,
  reconciles active runs, writes interruption events, and restores schedules.
- **Observable tested operations**: PASS. Service events use `slog`, runtime
  events are persisted, each Agent Run has isolated logs, and quickstart plus
  automated tests cover apply/execute/stop/logs/restart/concurrency.
- **Simplicity and clean architecture**: PASS. The design uses the standard
  library for HTTP/logging, SQLite for local state, small DDD packages, and
  provider/runtime ports only where isolation or external services require a
  boundary. OpenAI SDK usage stays in `infra/llm/openai` behind the vendor
  provider port.

**Post-design re-check**: PASS. `research.md`, `data-model.md`, contracts, and
`quickstart.md` preserve the gates above. No constitution violations require
complexity tracking.

## Project Structure

### Documentation (this feature)

```text
specs/001-agent-definition-runtime/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── agent-definition.md
│   ├── cli.md
│   └── openapi.yaml
└── tasks.md
```

### Source Code (repository root)

```text
cmd/
├── agentd/
│   └── main.go                 # CLI entrypoint
└── agentdserver/
    └── main.go                 # daemon entrypoint

internal/
├── agentd/
│   ├── app/                    # CLI command construction and output policies
│   ├── config/                 # CLI env/defaults for server URL and output
│   └── infra/httpclient/       # REST client for daemon API
├── agentdserver/
│   ├── app/
│   │   ├── agent/              # apply, inspect, list use cases
│   │   ├── logs/               # log lookup/read use cases
│   │   ├── runtime/            # execute, stop, recovery orchestration
│   │   └── scheduling/         # schedule reconciliation/use cases
│   ├── config/                 # env loading, validation, slog config
│   ├── domain/                 # Agent Definition, Agent, Agent Run, events
│   ├── infra/
│   │   ├── db/                 # SQLite DB wrapper, migrations, repositories
│   │   │   └── migrations/
│   │   │       ├── settings/
│   │   │       │   ├── 001_init.sql
│   │   │       │   └── 002_agent_policy_indexes.sql
│   │   │       └── runtime/
│   │   │           ├── 001_init.sql
│   │   │           ├── 002_run_logs.sql
│   │   │           └── 003_runtime_event_indexes.sql
│   │   ├── definition/         # Markdown front matter parser/validator
│   │   ├── http/               # REST server, handlers, JSON models
│   │   ├── llm/                # vendor-agnostic provider port adapters; OpenAI first
│   │   ├── logs/               # service/run log sinks and readers
│   │   ├── runtime/            # concurrent isolated run manager
│   │   └── scheduler/          # cron/manual scheduling adapter
│   └── service.go              # daemon wiring root, like fleeto.NewFleeto()
└── lib/
    ├── validator/              # shared validation helper if needed
    └── testutil/               # integration fixtures

tests/
└── e2e/                        # optional black-box CLI/server tests
```

**Structure Decision**: Use one Go module with two binaries. Keep CLI and server
separated under `internal/agentd` and `internal/agentdserver`; keep server domain
and use cases independent from HTTP, SQLite, scheduler, LLM provider, and
runtime adapters. OpenAI is the first `infra/llm` adapter through the official
Go SDK; future OpenRouter or Anthropic adapters plug into the same provider
port. This follows `fleeto`'s service-wiring pattern while avoiding web UI/MQTT
components that are not needed here.

## Complexity Tracking

No constitution gate violations.
