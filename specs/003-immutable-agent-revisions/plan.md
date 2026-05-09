# Implementation Plan: Immutable Agent Revisions

**Branch**: `002-agent-examples-results` | **Date**: 2026-05-08 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/003-immutable-agent-revisions/spec.md`

## Summary

Create Docker-image-like immutable Agent Revisions during `agentd apply`.
Apply resolves runtime paths, copies local scripts and declared local files into
`data/work/<agent-name>/<revision_id>`, persists revision metadata in the
settings database, and makes `agentd run <agent-name>:<revision>` execute that
self-contained artifact even if the original definition folder is later
removed. Agent-provided tools use `custom_tool` and are copied into revisions;
host-installed executables use `host_tool` and are invoked from the host
environment. AI Agent logs and agentdserver logs expose captured tool stdout,
stderr, result summaries, exit state, and errors.

## Technical Context

**Language/Version**: Go 1.26.2 for daemon, CLI, public Go client, and tests.  
**Primary Dependencies**: Existing `spf13/cobra`, standard `net/http`,
standard `log/slog`, `modernc.org/sqlite`, `robfig/cron/v3`, `gopkg.in/yaml.v3`,
`joho/godotenv`, and `google/uuid`.  
**Storage**: Settings SQLite gains revision metadata tables; copied artifact
files live under the daemon data directory at `data/work/<agent-name>/<revision_id>`.
Per-run execution directories live at
`data/work/<agent_name>/executions/<execution_id>`. Runtime DB run records
continue storing the executed revision ID and work directory.  
**Testing**: `go test ./...`; focused tests for apply revision creation,
artifact copying, path rewriting, idempotency, run revision resolution, CLI
`agentd run <agent>:<revision>`, corruption detection, and tool stdout/stderr
logs in both AI Agent logs and agentdserver logs. Focused tests must cover
`custom_tool` artifact copying, `host_tool` host executable validation, and
legacy `local_tool` migration handling. Codex must also perform manual
verification by applying and running
`examples/github-trending-engineering-radar/github-trending-engineering-radar.md`;
expected result is a successful GitHub-derived agent result with tool stdout
and stderr summaries visible in `agentd logs`.  
**Target Platform**: Linux and macOS daemon plus local CLI.  
**Project Type**: daemon-service plus CLI plus public Go client package.  
**Performance Goals**: Apply completes within 1 second for normal local
definitions with small scripts and fixture files; run startup validates revision
artifacts without noticeable delay for current examples.  
**Constraints**: No external storage service; no blob storage in settings DB for
copied scripts; environment values may be secret and must not be printed raw in
default logs or CLI inspection.  
**Scale/Scope**: Single-user local daemon; dozens of revisions per agent;
large artifact garbage collection, remote registries, signing, and sharing are
out of scope.  
**Daemon/Agent Impact**: Apply now creates immutable revision artifacts; run
resolution can target latest revision or explicit `<agent>:<revision>`; tool
execution uses revision-local commands for `custom_tool` scripts and host
commands for `host_tool` executables.  
**Isolation Policy**: Revision creation copies only declared local runtime
files for `custom_tool` and declared environment files. Tool execution receives
declared env and runs from daemon-owned execution directories while reading
revision-owned immutable material. `host_tool` entries can invoke only explicit
host-installed commands or allowed absolute host paths. Host root access,
unrestricted secret inheritance, and undeclared local file dependencies remain
denied by policy.  
**State & Recovery**: Revision rows use pending/finalized/corrupt status so
crashes during copy can be recovered or reported. Finalized revisions are
immutable; partial directories are cleaned or marked corrupt on daemon startup.  
**Observability**: Structured events cover revision creation, unchanged apply,
path resolution, artifact copy, command rewrite, corruption detection, explicit
revision run start, and tool stdout/stderr/result/error summaries in both AI
Agent logs and agentdserver logs.  
**Architecture/Complexity**: Keep Clean Architecture boundaries: domain types in
`internal/agentdserver/domain`, apply/run use cases in
`internal/agentdserver/app/*`, DB/filesystem adapters in
`internal/agentdserver/infra/*`, and CLI parsing in `internal/agentd/app`.
Add a small revision artifact service because apply must coordinate settings
metadata and filesystem copies atomically enough for recovery.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Daemon-first runtime**: PASS. Apply, revision creation, revision selection,
  artifact validation, and execution remain daemon-owned.
- **Least-privilege isolation**: PASS. The feature narrows runtime dependency
  access by copying declared files and avoiding undeclared source-folder reads
  after apply.
- **Linux/macOS portability**: PASS. The plan covers permission preservation,
  symlink handling, path resolution, and process execution on both supported
  platforms.
- **Durable recovery**: PASS. Revision status and artifact directories are
  persisted with crash recovery rules for partial artifacts.
- **Observable tested operations**: PASS. Revision lifecycle events, corruption
  errors, explicit run selection, and tool stdout/stderr logs have test and
  quickstart coverage.
- **Simplicity and clean architecture**: PASS. The design extends existing
  repositories and runtime adapters with one focused artifact service.

**Post-design re-check**: PASS. Research, data model, contracts, and quickstart
preserve daemon ownership, explicit declared access, durable state, observable
events, and local-only execution without new heavyweight dependencies.

## Project Structure

### Documentation (this feature)

```text
specs/003-immutable-agent-revisions/
├── spec.md
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
internal/
├── agentd/
│   ├── app/
│   │   ├── execute.go        # parse <agent>:<revision> and pass revision selector
│   │   ├── inspect.go        # expose revision metadata and masked env
│   │   ├── list.go           # include latest revision metadata
│   │   └── logs.go           # expose tool stdout/stderr/result/error entries
│   └── infra/httpclient/
│       ├── agents.go         # revision list/inspect client calls if needed
│       └── runs.go           # explicit revision execution request
├── agentdserver/
│   ├── app/
│   │   ├── agent/
│   │   │   ├── apply.go      # create or reuse immutable revision on apply
│   │   │   └── revision.go   # revision list/inspect use cases
│   │   └── runtime/
│   │       ├── execute.go    # resolve latest or explicit revision before run
│   │       └── provider.go   # execute request carries selected revision
│   ├── domain/
│   │   └── agent.go          # revisions, env, workdir, tool kind/log types
│   └── infra/
│       ├── db/
│       │   ├── migrations/settings/003_agent_revisions.sql
│       │   └── repository/agent_repository.go
│       ├── definition/
│       │   └── parser.go     # environment, path fields, custom/host kinds
│       ├── http/
│       │   ├── apply_handler.go
│       │   ├── inspect_handler.go
│       │   └── run_handler.go
│       └── runtime/
│           ├── revision_artifact.go
│           ├── manager.go    # revision execution, workdirs, structured logs
│           ├── tool_process.go
│           └── env_file.go   # .env parsing and masking helpers if split
examples/
└── */*.md                    # migrate local_tool examples to custom_tool
tests/
└── e2e/
    └── apply_test.go         # source deletion/mutation and revision run tests
└── lib/testutil/
```

**Structure Decision**: Keep current daemon/CLI/public-client package layout.
Add revision-specific behavior beside existing agent apply and runtime code
instead of creating a separate registry subsystem.

## Implementation Strategy

### MVP Slice

Deliver User Story 1 and User Story 2 first: finalized immutable revisions,
idempotent apply, artifact copy, environment capture, and explicit revision
execution. This proves the Docker-image-like contract before broadening tool
execution and inspection surfaces.

### Phase Order

1. Add domain and persistence primitives for revisions, revision tools,
   revision environment, artifact files, execution directories, and tool log
   evidence.
2. Update definition parsing and validation for `custom_tool`, `host_tool`,
   legacy `local_tool`, path-bearing metadata, and `environment`.
3. Implement artifact staging/finalization, checksums, command rewriting, `.env`
   parsing, and crash-safe status transitions.
4. Wire apply to create or reuse revisions based on runtime content digest.
5. Wire run execution to resolve latest or explicit revisions, create execution
   working directories, validate artifacts, and run from frozen material.
6. Split tool execution behavior: `custom_tool` uses copied artifact commands;
   `host_tool` validates and invokes host-installed commands.
7. Emit tool stdout, stderr, result, exit, timeout, and error evidence to both
   AI Agent logs and agentdserver structured logs.
8. Add revision list/inspect endpoints and CLI output with masked environment
   values and tool kind details.
9. Complete recovery, e2e tests, example migration, and Codex manual
   verification with GitHub Trending Engineering Radar.

### Traceability

| Spec Coverage | Implementation Tasks |
|---------------|----------------------|
| US1, FR-001..FR-004, FR-007, FR-012..FR-018, FR-021..FR-024, FR-027..FR-030 | T001-T037, T069-T074 |
| US2, FR-002, FR-018..FR-020, SC-002 | T038-T045, T071 |
| US3, FR-005..FR-011, FR-016, SC-003, SC-004 | T011-T028, T046-T054, T073-T074 |
| US4, FR-024..FR-026, SC-007, SC-008 | T055-T060, T077 |
| US5, revision inspection/listing, corrupted artifact reporting | T061-T068 |
| Environment and secrets, FR-012..FR-015, SC-010 | T002, T007, T011-T019, T031, T053-T054, T072 |
| GitHub Trending manual verification, SC-011 | T077-T078 |

## Complexity Tracking

No constitution gate violations.

## Manual Verification

After implementation and automated tests pass, Codex MUST run an end-to-end
manual verification using the repository example:

```bash
agentd apply examples/github-trending-engineering-radar/github-trending-engineering-radar.md
agentd run github-trending-engineering-radar
agentd result github-trending-engineering-radar
agentd logs github-trending-engineering-radar --run <run-id>
```

If the CLI compatibility command is still named `execute` during the migration,
use `agentd execute github-trending-engineering-radar` for the run step and
record that compatibility path in the verification notes.

Expected result:
- The apply command creates or reuses an immutable revision.
- The run starts and completes successfully.
- The final result contains data derived from GitHub public trend/repository
  signals.
- `agentd logs` for the run includes the local tool execution action and the
  captured stdout, stderr, result summary, exit state, timeout state, and error
  details from the GitHub trending tool.
- agentdserver logs for the run include the tool execution action, result
  summary, stdout/stderr summaries, exit state, timeout state, and errors if any
  occurred.
