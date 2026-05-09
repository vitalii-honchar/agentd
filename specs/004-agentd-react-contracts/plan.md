# Implementation Plan: Unified Agentd ReAct Contracts

**Branch**: `004-agentd-react-contracts` | **Date**: 2026-05-09 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/004-agentd-react-contracts/spec.md`

## Summary

Ship one `agentd` binary that can either run client commands or start the
local daemon, then replace the current prompt-plus-preexecuted-tools behavior
with a daemon-owned ReAct execution controller. Agent definitions gain an
optional `contract` block with `input` and `output` JSON Schemas. Contracted
inputs are validated before a run starts; contracted outputs are finalized from
the complete execution history and validated before a run completes. Existing
definitions without contracts stay valid and bypass contract validation.

The ReAct controller will use go-agent behavior as the target runtime model,
but the current go-agent public API needs a dynamic-schema adapter before it can
be imported directly. The plan is to adapt or add a go-agent dynamic runner API
that agentd can call while preserving daemon-owned tools, revision artifacts,
logs, cancellation, and provider adapters. Agentd also gains a `codex` provider
implemented through managed non-interactive Codex CLI process execution, not
undocumented token extraction.

## Technical Context

**Language/Version**: Go 1.26.2 for daemon, CLI, public Go client, and tests.  
**Primary Dependencies**: Existing `spf13/cobra`, standard `net/http`,
standard `log/slog`, `modernc.org/sqlite`, `robfig/cron/v3`, `gopkg.in/yaml.v3`,
`joho/godotenv`, `google/uuid`, and `github.com/openai/openai-go/v3`.
Add a JSON Schema validation dependency for runtime input/output validation.
Use go-agent as the ReAct control library after adding or selecting a
dynamic-schema runtime API; keep provider adapters in agentd. Use local Codex
CLI via `codex exec` for the `codex` provider.  
**Storage**: Settings SQLite extends agent and revision metadata with optional
contract schemas and provider details. Runtime SQLite keeps run status, result,
provider request ID/process ID where useful, and event rows. Run log files stay
under the daemon run-log directory and are addressed by run ID. Temporary
contract schema files for provider calls live in per-run work directories.  
**Testing**: `go test ./...`; focused tests for definition parsing, contract
schema validation, contracted input validation before run creation, contracted
output finalization, ReAct loop behavior, one-step completion, tool denial,
tool limits, immutable revision contract persistence, unified binary daemon
mode, Codex provider process handling, and run-ID-only log lookup. Manual
verification must apply and run representative examples, including one
manual-input contracted example and one scheduled empty-input example.  
**Target Platform**: Linux and macOS daemon plus local CLI.  
**Project Type**: daemon-service plus CLI plus public Go client package.  
**Performance Goals**: Contract schema validation and run input validation add
no noticeable delay for normal local definitions; one-step agents avoid
unnecessary ReAct turns; Codex provider startup overhead is accepted only for
agents explicitly configured with `vendor.name: codex`.  
**Constraints**: Contracts are optional. No direct Codex token extraction. No
agent-name log fallback after run-ID log lookup ships. No arbitrary Codex CLI
workspace access beyond the run-owned working directory. Secret values and
provider credentials must not appear in default logs, inspect output, or
contract validation errors.  
**Scale/Scope**: Single-user local daemon; dozens of applied revisions per
agent; one active run per agent remains the current default unless changed by a
future feature; all checked-in examples are migrated to contracts.  
**Daemon/Agent Impact**: `cmd/agentd` owns both CLI and daemon launch. Runtime
execution moves from pre-running declared tools to daemon-controlled ReAct
steps: model asks for a declared tool, daemon validates and executes it,
observation is appended to history, and loop continues until final output or a
bounded stop condition.  
**Isolation Policy**: Tool execution remains daemon-owned and uses existing
revision-local `custom_tool` and host-validated `host_tool` policies. Codex CLI
runs as a provider process in a run-owned work directory with non-interactive
execution, bounded stdout/stderr capture, cancellation through process-group
termination, and no inherited undeclared secrets beyond explicit provider
configuration.  
**State & Recovery**: Applied finalized revisions persist resolved contract
schemas and provider configuration. Active runs interrupted by daemon shutdown
are recovered to terminal states. Codex provider child processes and tool
processes are stopped on cancellation/restart cleanup. Schema migrations must
be additive and preserve legacy rows with null contract fields.  
**Observability**: Structured runtime events cover input validation, ReAct step
start/end, model request/response, requested tool, tool observation, loop stop
reason, output finalization, schema validation failure, Codex process start/exit
state, run-ID log lookup, and invalid agent-name log lookup.  
**Architecture/Complexity**: Keep Clean Architecture boundaries: domain types
in `internal/agentdserver/domain`; use cases in `internal/agentdserver/app/*`;
DB/filesystem/process/provider adapters in `internal/agentdserver/infra/*`; CLI
parsing in `internal/agentd/app`; public client transport in `pkg/agentdclient`.
Add small interfaces for dynamic ReAct model calls and contract validation
because they isolate volatile provider/schema behavior from runtime policy.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Daemon-first runtime**: PASS. The daemon remains the only owner of ReAct
  state, tool execution, validation, run logs, cancellation, recovery, and
  provider process lifecycle. CLI commands delegate to daemon APIs except when
  launching daemon mode.
- **Least-privilege isolation**: PASS. Contract validation reduces unnecessary
  execution. Codex CLI is treated as a bounded provider process with explicit
  working directory, timeout, cancellation, and no token scraping.
- **Linux/macOS portability**: PASS. The plan covers unified binary signal
  handling, Codex/process execution, process-group cleanup, file paths, and
  run logs on both supported targets.
- **Durable recovery**: PASS. Contract metadata is persisted in settings DB and
  revisions; runtime runs/logs remain persisted; interrupted provider/tool
  processes transition to terminal states during recovery.
- **Observable tested operations**: PASS. The plan includes structured events,
  actionable validation errors, run-scoped logs, Codex process diagnostics, and
  automated plus manual verification.
- **Simplicity and clean architecture**: PASS. The feature adds focused
  adapters and domain objects around existing package boundaries instead of a
  second runtime or broad registry subsystem.

**Post-design re-check**: PASS. Research, data model, contracts, and quickstart
preserve daemon ownership, explicit host access, durable state, observable
events, cross-platform process handling, and narrow interfaces for schema,
ReAct, and provider behavior.

## Project Structure

### Documentation (this feature)

```text
specs/004-agentd-react-contracts/
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
cmd/
├── agentd/
│   └── main.go              # dispatch daemon mode or CLI mode
└── agentdserver/
    └── main.go              # remove or keep as deprecated compatibility shim

internal/
├── agentd/
│   ├── app/
│   │   ├── root.go          # daemon flag registration and command wiring
│   │   ├── execute.go       # JSON input support and run ID visibility
│   │   ├── logs.go          # logs <run-id>, reject agent-name lookup
│   │   ├── ps.go            # run IDs remain copyable
│   │   └── result.go        # contracted JSON result display
│   └── infra/httpclient/
│       ├── query.go         # run-scoped logs client calls
│       └── runs.go          # structured run input transport
├── agentdserver/
│   ├── app/
│   │   ├── agent/
│   │   │   ├── apply.go
│   │   │   └── definition_validator.go
│   │   ├── logs/
│   │   │   └── logs.go      # read by run ID only
│   │   └── runtime/
│   │       ├── contract.go  # input/output contract validation use cases
│   │       ├── execute.go
│   │       ├── provider.go  # dynamic model request/response contracts
│   │       └── react.go     # daemon-owned go-agent adapter boundary
│   ├── config/
│   │   └── config.go        # Codex provider command/model/timeout config
│   ├── domain/
│   │   └── agent.go         # contracts, ReAct steps, provider metadata
│   └── infra/
│       ├── db/
│       │   ├── migrations/settings/004_agent_contracts.sql
│       │   └── repository/agent_repository.go
│       ├── definition/
│       │   └── parser.go    # contract YAML parsing and validation
│       ├── http/
│       │   ├── logs_handler.go
│       │   ├── run_handler.go
│       │   └── model/
│       ├── llm/
│       │   ├── codex/
│       │   │   └── provider.go
│       │   └── openai/
│       │       └── provider.go
│       └── runtime/
│           ├── contract_validator.go
│           ├── manager.go
│           ├── react_runner.go
│           └── tool_process.go
pkg/
└── agentdclient/
    ├── client.go
    ├── logs.go              # /v1/runs/{run_id}/logs
    └── runs.go              # JSON input support
examples/
└── */*.md                   # add contract.input and contract.output
tests/
├── e2e/
│   ├── contracts_test.go
│   ├── logs_test.go
│   └── unified_binary_test.go
└── lib/testutil/
```

**Structure Decision**: Keep the current daemon/CLI/public-client layout. Move
new behavior into existing bounded packages with small additions: contract
validation, ReAct runner adapter, Codex provider adapter, run-ID logs, and
unified binary dispatch.

## Implementation Strategy

### MVP Slice

Deliver the single binary, optional contract parsing/persistence, contracted
input validation, and run-ID log lookup first. This produces immediate user
value and reduces risk before replacing the execution loop.

### Phase Order

1. Add contract domain types, parser support, validation dependency, repository
   persistence, revision digest/persistence inclusion, inspect/list output, and
   compatibility behavior for omitted contracts.
2. Add structured run input transport: CLI `--input-json`/file support, public
   client support, HTTP request model changes, and pre-run validation before run
   creation or side effects.
3. Replace `agentd logs <agent_name> --run <id>` with `agentd logs <run_id>`
   and add `/v1/runs/{run_id}/logs`; remove successful latest-run-by-agent log
   behavior.
4. Unify binary dispatch: `agentd --daemon`, `agentd -d`, and compatibility
   `agentd --deamon`; keep or deprecate `cmd/agentdserver` only as an explicit
   compatibility shim if needed for one release.
5. Adapt go-agent for agentd usage: dynamic JSON Schema output, external
   daemon-owned tool callback, provider-neutral model interface, cancellation,
   history export, and bounded loop controls. Integrate through
   `internal/agentdserver/app/runtime` interfaces.
6. Replace pre-executed tool behavior with the daemon-owned ReAct loop:
   model step, tool request validation, tool execution, observation append,
   loop stop, and finalization.
7. Implement contracted output finalization from full history and validate the
   final JSON against `contract.output`.
8. Add the Codex provider adapter around non-interactive `codex exec`, using
   schema files when structured output is needed, bounded output capture,
   process-group cancellation, setup diagnostics, and run events.
9. Migrate all checked-in examples to contracts and update README guidance.
10. Complete recovery, e2e coverage, cross-platform process tests, and manual
    verification.

### Traceability

| Spec Coverage | Implementation Focus |
|---------------|----------------------|
| US1, FR-001..FR-005, FR-027, FR-028, SC-001, SC-009 | Unified `agentd` daemon/client binary, help/docs, signal handling, compatibility |
| US2, FR-006..FR-011, FR-023..FR-026, SC-002, SC-007 | Contract schema parsing, persistence, input validation, structured input transport |
| US3, FR-012..FR-014, SC-003 | Output finalization, output schema validation, result persistence |
| US4, FR-015..FR-020, SC-004, SC-005 | go-agent dynamic adapter, ReAct loop, tool request/observation control |
| US5, FR-021, FR-022, SC-008, SC-010 | Observability, masking, recovery, restart/cancel behavior |
| US6, FR-034..FR-039, SC-014, SC-015 | Run-ID-only log lookup across CLI, HTTP, client, and tests |
| US7, FR-029..FR-033, SC-011..SC-013 | Example contract migration and documentation |
| US8, FR-040..FR-046, SC-016..SC-018 | Codex CLI provider process adapter and provider diagnostics |

## Complexity Tracking

No constitution gate violations.

## Manual Verification

After automated tests pass, run:

```bash
go test ./...
go build ./cmd/agentd
./agentd --daemon
```

In another shell:

```bash
./agentd apply examples/github-trending-engineering-radar/github-trending-engineering-radar.md
run_id=$(./agentd run github-trending-engineering-radar --output json | jq -r .run_id)
./agentd result "$run_id" --output json
./agentd logs "$run_id"
./agentd logs github-trending-engineering-radar
```

Expected result:
- `agentd --daemon` starts the daemon without `agentdserver`.
- The scheduled no-input example applies and runs with an empty-object input
  contract.
- The result validates against the example output contract.
- `agentd logs "$run_id"` returns only that run's events.
- `agentd logs github-trending-engineering-radar` fails with an actionable
  "logs require an agent run ID" message.

Then verify a manual-input contracted agent:

```bash
./agentd apply examples/website-snapshot-analyst/website-snapshot-analyst.md
run_id=$(./agentd run website-snapshot-analyst --input-json '{"url":"https://example.com"}' --output json | jq -r .run_id)
./agentd result "$run_id" --output json
./agentd logs "$run_id"
```

Finally verify Codex provider behavior with a small contracted fixture agent:

```bash
codex exec --help
./agentd apply tests/fixtures/codex-provider-agent.md
run_id=$(./agentd run codex-provider-agent --input-json '{"topic":"agentd"}' --output json | jq -r .run_id)
./agentd result "$run_id" --output json
./agentd logs "$run_id"
```

Expected result:
- The Codex-backed run uses local Codex CLI authentication/configuration.
- No OpenAI API key is required for the Codex-backed provider path.
- Provider process start, exit state, stderr summary, and final output are
  visible in run-scoped logs.

## Final Review Notes

- Secret masking: inspect/list responses expose contract schema digests, not
  raw schema bodies. Environment values remain masked in inspect output.
- Linux/macOS process cleanup: Codex CLI execution is bounded by timeout and
  uses process-group termination on Darwin/Linux so child processes are killed
  on cancellation.
- Daemon recovery: startup recovery still marks active interrupted runs
  terminal, and ReAct/Codex runs persist provider errors and contract
  finalization failures as run failures with run-scoped logs.
- Remaining manual gap: live OpenAI and real authenticated Codex CLI quickstart
  runs were not executed in this session; automated e2e coverage uses fake
  providers/CLI for deterministic verification.
