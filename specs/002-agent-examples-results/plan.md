# Implementation Plan: Agent Examples and Results

**Branch**: `002-agent-examples-results` | **Date**: 2026-05-08 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/002-agent-examples-results/spec.md`

**Note**: This template is filled in by the `/speckit-plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Extend the existing local `agentd` daemon and CLI so users can operate applied
agents like scheduled infrastructure: list definitions, list active/all runs,
retrieve completed or failed run results by agent or run ID, inspect scoped
action logs, and use the same daemon client from Bash, local AI agents, and Go
integrations. Add daemon-owned local tool process execution for declared
example tools, then replace the old weak examples with eight self-contained
example folders: seven daily public-source monitoring agents plus one manual
website snapshot agent.

The implementation builds on the current `001-agent-definition-runtime`
architecture. Runtime state remains daemon-owned, persisted in SQLite, exposed
through local REST endpoints, formatted by the CLI, and wrapped by a new public
Go client package. Example tools run as child processes from per-run isolated
work directories with declared filesystem/network access, structured action
events, timeouts, and stored results.

## Technical Context

**Language/Version**: Go 1.26.2 for daemon, CLI, public Go client, and tests;
example tools may use Python 3 and Node.js only where their README documents
local dependency installation.
**Primary Dependencies**: Existing `spf13/cobra`, standard `net/http`,
standard `log/slog`, `modernc.org/sqlite`, `robfig/cron/v3`, `gopkg.in/yaml.v3`,
`joho/godotenv`, and `google/uuid`; new public client package wraps the existing
daemon HTTP contract; example tools use command-line programs declared by each
Agent Definition.
**Storage**: Existing settings SQLite database for Agent Definitions and tools;
existing per-Agent runtime SQLite database extended with stored run results and
tool execution events; per-run log files remain on disk.
**Testing**: `go test ./...`; focused unit tests for result formatting, run
result persistence, public Go client, same-host middleware, tool execution, and
definition parsing; HTTP contract tests; CLI integration tests; example smoke
tests for fresh-clone default paths.
**Target Platform**: Linux and macOS daemon plus local CLI and importable Go
client.
**Project Type**: daemon-service plus CLI plus public Go client package plus
repository examples.
**Performance Goals**: Existing daemon goals remain: at least 25 applied Agents
and 5 concurrent Agent Runs on a developer laptop; `list`, `ps`, `result`, and
`logs` commands return within 1 second for normal local state; run result tables
remain readable in an 80-column terminal.
**Constraints**: No external cloud dependency except configured LLM vendors;
examples require no dedicated infrastructure, no required API keys, no CI/SaaS
setup, and no private data; daemon accepts client requests only from the same
host; no user authentication in this feature; no heavy framework or separate DB.
**Scale/Scope**: Single-user local daemon for developer laptops/workstations;
seven scheduled monitoring examples plus one manual URL-analysis example;
remote API access, multi-user auth, web UI, and marketplace packaging are out
of scope.
**Daemon/Agent Impact**: Adds daemon-owned run listing, result lookup, result
storage, local tool process execution, same-host request enforcement, and
structured per-run action events. CLI remains a thin local client.
**Isolation Policy**: Tool processes execute only when declared by the Agent
Definition, from a per-run work directory, with explicit env construction,
documented timeouts, scoped filesystem paths, public network access declared by
example, no inherited secret-bearing files, and audit log entries for every
tool invocation.
**State & Recovery**: Run records gain stored result fields for successful and
failed executions; tool executions and action logs are persisted or indexed so
finished runs survive daemon restart; active tool processes are cancelled or
marked interrupted on restart according to existing recovery policy.
**Observability**: Per-run logs and runtime events include stable action names
for prompt submission, tool start, tool completion/failure, result persistence,
run completion, and run failure; CLI and Go client expose actionable error
codes and stable statuses.
**Architecture/Complexity**: Keep existing Clean Architecture boundaries:
domain types in `internal/agentdserver/domain`, use cases in
`internal/agentdserver/app/*`, adapters in `internal/agentdserver/infra/*`, CLI
in `internal/agentd/app`, and public importable client in `pkg/agentdclient`.
No extra abstraction beyond public-client boundary and process-tool adapter
needed by current requirements.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Daemon-first runtime**: PASS. Run listing, results, tool execution,
  logging, same-host checks, and scheduling remain daemon-owned. CLI and public
  Go client delegate to daemon REST endpoints.
- **Least-privilege isolation**: PASS. Examples declare required public network
  and local tool access; tool execution is explicit, audited, timeout-bound, and
  denied when undeclared.
- **Linux/macOS portability**: PASS. Process execution, cancellation, browser
  screenshot setup, path handling, and localhost binding are planned for both
  Linux and macOS with README verification.
- **Durable recovery**: PASS. Run results, terminal statuses, tool events, and
  log references are persisted in runtime DB/log files and survive daemon
  restart.
- **Observable tested operations**: PASS. CLI commands, HTTP endpoints, public
  client calls, tool actions, and examples have automated or quickstart
  verification paths.
- **Simplicity and clean architecture**: PASS. The plan extends existing
  packages and promotes only the daemon HTTP client to `pkg/agentdclient` for a
  concrete integration requirement.

**Post-design re-check**: PASS. `research.md`, `data-model.md`, contracts, and
`quickstart.md` preserve daemon ownership, explicit tool access, durable run
results, observable action logs, and local-only access. No constitution
violations require complexity tracking.

## Project Structure

### Documentation (this feature)

```text
specs/002-agent-examples-results/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── agent-definition.md
│   ├── cli.md
│   ├── examples.md
│   ├── openapi.yaml
│   └── public-go-client.md
└── tasks.md                  # created by /speckit-tasks, not this command
```

### Source Code (repository root)

```text
cmd/
├── agentd/
│   └── main.go
└── agentdserver/
    └── main.go

pkg/
└── agentdclient/
    ├── client.go             # public Go client and options
    ├── agents.go             # apply/list/inspect operations
    ├── runs.go               # execute/list/result/stop operations
    ├── logs.go               # log lookup operations
    └── types.go              # stable public request/response types

internal/
├── agentd/
│   ├── app/
│   │   ├── list.go           # `agentd list` / optional `ls` alias
│   │   ├── ps.go             # active/all run listing
│   │   ├── result.go         # result table/detail output
│   │   └── output.go         # text/json formatting and exit policy
│   └── infra/httpclient/     # thin wrappers or migration shim to pkg client
├── agentdserver/
│   ├── app/
│   │   ├── result/           # list/get run result use cases
│   │   ├── runtime/          # result persistence and tool orchestration
│   │   └── logs/             # scoped action log lookup
│   ├── domain/               # AgentRun result fields and ToolExecution types
│   └── infra/
│       ├── db/
│       │   └── migrations/runtime/
│       │       ├── 004_run_results.sql
│       │       └── 005_tool_executions.sql
│       ├── http/             # ps/result routes and same-host middleware
│       └── runtime/          # command-line tool process adapter
└── lib/testutil/

examples/
├── cybersecurity-reddit-watch/
├── hacker-news-builder-brief/
├── reddit-customer-pain-monitor/
├── product-hunt-launch-radar/
├── github-trending-engineering-radar/
├── developer-dependency-release-monitor/
├── ai-engineering-hiring-signal-monitor/
└── website-snapshot-analyst/
```

**Structure Decision**: Keep daemon, CLI, and runtime internals in the existing
`internal/agentd*` package layout. Add `pkg/agentdclient` because user-owned Go
integrations must import a supported client without reaching into `internal`.
Replace flat example files with one folder per example containing the
definition, README, tools, and fixtures/source lists.

## Complexity Tracking

No constitution gate violations.
