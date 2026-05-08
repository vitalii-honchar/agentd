# Research: Agent Definition Runtime

## Decision: Go 1.26.2 With Two Thin Binaries

Use one Go module with `cmd/agentd` for the CLI and `cmd/agentdserver` for the
daemon. Each `main.go` stays thin and delegates to an internal package, matching
the `fleeto` pattern.

**Rationale**: Go produces small single-binary tools, handles concurrency well,
and fits local daemon/CLI distribution. Two binaries keep user CLI concerns away
from daemon lifecycle wiring.

**Alternatives considered**:
- Single binary with subcommands for both server and client: simpler packaging,
  but weaker separation between CLI and daemon lifecycle.
- Separate modules: unnecessary until versioning or release cadence diverges.

## Decision: Clean Architecture/DDD Package Boundaries

Use `internal/agentdserver/domain` for entities and invariants, `app` packages
for use cases and ports, and `infra` for adapters. Use `internal/agentd` for
CLI command composition and daemon REST client code.

**Rationale**: This mirrors `fleeto`'s `domain`, `app`, `infra`, `config`, and
service-wiring structure while matching the constitution's Clean Architecture
rule. It keeps REST, SQLite, cron, Markdown parsing, and LLM providers outside
domain logic.

**Alternatives considered**:
- Flat package structure: faster to start but harder to preserve daemon/runtime
  boundaries once scheduling, logs, and provider adapters are added.
- Heavy framework architecture: rejected because the runtime must remain light.

## Decision: Standard `net/http` REST Server

Use standard `net/http` handlers for REST endpoints and JSON models.

**Rationale**: The API surface is small, local-first, and does not need a web
framework. Standard HTTP reduces dependencies and keeps the daemon lightweight.

**Alternatives considered**:
- Gin: familiar from `fleeto`, but unnecessary for a small JSON API and adds
  more dependency surface.
- gRPC: strong contracts, but heavier for a local CLI/server workflow.

## Decision: Cobra CLI With HTTP Client Adapter

Use `spf13/cobra` for `agentd apply`, `execute`, `stop`, `inspect`, `logs`, and
`list`. Commands call an internal REST client; commands do not access SQLite or
runtime state directly.

**Rationale**: Cobra gives predictable command parsing and help output while the
daemon remains the single runtime authority.

**Alternatives considered**:
- Standard `flag`: lighter, but harder to organize Docker-like subcommands.
- Direct local storage access from CLI: rejected by daemon-first principle.

## Decision: SQLite via `modernc.org/sqlite` With Settings DB Plus Per-Agent Runtime DBs

Use one SQLite settings database for Agent Definitions, Agent metadata,
schedules, and access policy. Create a separate SQLite runtime database for each
Agent to store that Agent's Agent Runs, runtime events, and log indexes. Use WAL
mode, embedded migrations, and repository packages modeled after
`fleeto/internal/fleeto/infra/db`.

**Rationale**: SQLite satisfies the local lightweight storage requirement and
does not require a separate service. Splitting runtime data per Agent decouples
concurrent Agent execution writes, so one Agent with heavy or stalled runtime
writes cannot block every other Agent behind a single SQLite writer.
`modernc.org/sqlite` avoids CGO setup, which improves portability for Linux and
macOS.

**Alternatives considered**:
- Postgres/MongoDB: rejected by spec because they are too heavy for developer
  laptops and require external services.
- One SQLite database for all settings and runtime data: simpler, but all Agent
  Run writes contend on one SQLite writer and can block unrelated Agents.
- JSON files only: simpler, but poor fit for concurrent writes, run history,
  recovery reconciliation, and indexed log lookup.

## Decision: Embedded Ordered SQL Migrations Like Fleeto

Store settings schema migrations under
`internal/agentdserver/infra/db/migrations/settings/*.sql` and runtime schema
migrations under `internal/agentdserver/infra/db/migrations/runtime/*.sql`.
Embed them in the DB package with `go:embed`. The DB startup path applies
`*.sql` files in lexicographic order inside transactions, following the same
shape as `fleeto/internal/fleeto/infra/db/migrations/<db-name>/*.sql`.

Initial migration set:
- `settings/001_init.sql`: `agents`, `agent_tools`, `agent_mcp_servers`, and
  baseline indexes.
- `settings/002_agent_policy_indexes.sql`: access-policy lookup indexes.
- `runtime/001_init.sql`: `agent_runs`, `runtime_events`, and baseline indexes.
- `runtime/002_run_logs.sql`: isolated run log reference fields/indexes if
  separated from the initial run table during implementation.
- `runtime/003_runtime_event_indexes.sql`: runtime event indexes for run lookup.

**Rationale**: SQL files make schema evolution reviewable, deterministic, and
easy to test. Embedding keeps the daemon a single deployable binary and avoids a
runtime migrations directory dependency.

**Alternatives considered**:
- Auto-create schema from Go structs: less reviewable and harder to migrate.
- External migration tool: unnecessary operational dependency for a local
  daemon.
- One large SQL string in Go: harder to diff and inconsistent with `fleeto`.

## Decision: Markdown Front Matter for Agent Definitions

Use YAML front matter at the top of the Markdown file for machine-readable
properties, and use the Markdown body as the exact Agent prompt.

**Rationale**: This preserves plain Markdown authoring while giving the daemon a
stable schema for name, schedule mode, vendor/model, tools, MCP servers,
enabled state, and access policy.

**Alternatives considered**:
- Kubernetes-style YAML only: easier to parse, but loses the natural prompt body
  and user-requested Markdown format.
- Free-form Markdown headings: human friendly, but less reliable for validation.

## Decision: Cron Adapter With Manual Schedule Mode

Use `robfig/cron/v3` for cron-compatible expressions and represent
`schedule.type = manual` as no automatic schedule entry. Scheduling state is an
Agent field, not a separate entity.

**Rationale**: Cron covers the initial schedule requirement. Manual mode supports
`agentd execute <agent_name>`. Keeping schedule inside Agent avoids a premature
entity until schedules need independent lifecycle or history.

**Alternatives considered**:
- `go-co-op/gocron`: used in `fleeto`, but cron expressions are the core
  requirement here and `robfig/cron/v3` maps directly to that.
- OS cron/launchd/systemd timers: platform-specific and would split runtime
  ownership away from the daemon.

## Decision: Concurrent Runtime Manager With Per-Run Isolation

Create a runtime manager that starts each Agent Run in its own context, work
directory, log sink, environment map, and policy scope. Different Agents run
concurrently by default. Same-Agent overlap defaults to disabled to avoid
surprising duplicate work, but this can later become an Agent policy field.

**Rationale**: The spec defines concurrency as core behavior and isolation as a
Docker-like expectation. Contexts and per-run records give cancellation,
recovery, and failure isolation without introducing real containers in the first
implementation.

**Alternatives considered**:
- Serial scheduler: rejected because concurrent Agents are required.
- Real Docker/container runtime dependency: rejected for local lightweight
  runtime and no external daemon dependency.

## Decision: `slog` Service Logs Plus Isolated Agent Run Logs

Use `slog` for daemon/system logs. Each Agent Run gets a dedicated log file
under the runtime data directory and a persisted log reference in SQLite.

**Rationale**: Service logs explain daemon activity; per-run logs let
`agentd logs <agent_name>` return only the selected Agent's logs even when many
Agents run concurrently.

**Alternatives considered**:
- Single combined log file: rejected because it makes log isolation harder.
- Store all log lines in SQLite: searchable, but higher write amplification for
  concurrent runs and not necessary for the initial logs command.

## Decision: Vendor-Agnostic LLM Provider Port With OpenAI SDK Adapter First

Define an `LLMProvider` port in the application layer and implement OpenAI as
the first provider using `OPENAI_API_KEY` from environment and the official
`github.com/openai/openai-go/v3` SDK. Keep provider request/response details in
`infra/llm/openai`.

**Rationale**: The spec allows multiple vendors, while the current environment
provides an OpenAI key. The official OpenAI Go repository states that the
library provides Go access to the OpenAI REST API, requires Go 1.22+, and uses
the Responses API as the primary model API:
https://github.com/openai/openai-go

The provider port must stay generic enough for future OpenRouter and Anthropic
adapters: input is Agent prompt plus allowed tool/MCP context; output is provider
text, provider request ID, usage metadata when available, and a normalized error.

**Alternatives considered**:
- Direct HTTP calls to OpenAI: rejected because the user explicitly requested
  the official SDK and the SDK reduces hand-written API surface.
- Hard-code OpenAI throughout domain/use cases: rejected because vendor choice
  belongs to Agent Definition metadata and OpenRouter/Anthropic support is
  expected next.
- Add a broad agent framework: rejected as premature complexity for the first
  runtime.

## Decision: Local `.env` Loading for Developer Convenience

Load `.env` during configuration, following the `fleeto` pattern, but treat
environment values as runtime configuration only. Agent Definition files MUST
NOT contain secret values by default.

**Rationale**: The user already added `OPENAI_API_KEY` to `.env`. Loading it
keeps local development simple without making definitions secret-bearing.

**Alternatives considered**:
- Require exported shell variables only: secure but less ergonomic.
- Store secrets in SQLite from `agentd apply`: rejected for initial scope.
