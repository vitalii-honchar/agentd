# Tasks: Agent Definition Runtime

**Input**: Design documents from `/specs/001-agent-definition-runtime/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: Tests are required for daemon control APIs, Agent Definition parsing,
SQLite persistence, scheduling, concurrent Agent Runs, isolation, restart
recovery, logs, and CLI/server integration.

**Organization**: Tasks are grouped by user story to enable independent
implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3)
- Every task includes an exact file path

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Initialize Go module, binary entrypoints, and project skeleton.

- [X] T001 Initialize Go module `agentd` with Go 1.26.2 in go.mod
- [X] T002 Add dependencies `spf13/cobra`, `modernc.org/sqlite`, `robfig/cron/v3`, `gopkg.in/yaml.v3`, `joho/godotenv`, `google/uuid`, and `github.com/openai/openai-go/v3` in go.mod
- [X] T003 Create CLI entrypoint skeleton in cmd/agentd/main.go
- [X] T004 Create daemon entrypoint skeleton in cmd/agentdserver/main.go
- [X] T005 Create server package directory skeleton under internal/agentdserver/
- [X] T006 Create CLI package directory skeleton under internal/agentd/
- [X] T007 [P] Create shared test utility package skeleton in internal/lib/testutil/testutil.go
- [X] T008 [P] Create validation helper package skeleton in internal/lib/validator/validator.go
- [X] T009 [P] Add sample manual Agent Definition in examples/release-notes-helper.md
- [X] T010 [P] Add sample cron Agent Definition in examples/daily-pr-review.md

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core architecture, configuration, storage, and runtime interfaces
that MUST be complete before user stories.

**Critical**: No user story work begins until this phase is complete.

- [X] T011 Define daemon configuration fields and defaults in internal/agentdserver/config/config.go
- [X] T012 Add daemon `.env` loading and config validation tests in internal/agentdserver/config/config_test.go
- [X] T013 Configure `slog` service logger in internal/agentdserver/config/logger.go
- [X] T014 [P] Define CLI configuration fields and defaults in internal/agentd/config/config.go
- [X] T015 [P] Add CLI configuration tests in internal/agentd/config/config_test.go
- [X] T016 Define domain errors in internal/agentdserver/domain/errors.go
- [X] T017 Define Agent Definition, Agent, Agent Run, Tool Permission, and Runtime Event domain types in internal/agentdserver/domain/agent.go
- [X] T018 Add domain validation and state transition tests in internal/agentdserver/domain/agent_test.go
- [X] T019 Create SQLite DB wrapper with embedded migration support in internal/agentdserver/infra/db/db.go
- [X] T020 Add SQLite DB wrapper migration tests in internal/agentdserver/infra/db/db_test.go
- [X] T021 Create settings SQLite migration `001_init.sql` in internal/agentdserver/infra/db/migrations/settings/001_init.sql
- [X] T022 Create settings SQLite migration `002_agent_policy_indexes.sql` in internal/agentdserver/infra/db/migrations/settings/002_agent_policy_indexes.sql
- [X] T023 Create runtime SQLite migration `001_init.sql` in internal/agentdserver/infra/db/migrations/runtime/001_init.sql
- [X] T024 Create runtime SQLite migration `002_run_logs.sql` in internal/agentdserver/infra/db/migrations/runtime/002_run_logs.sql
- [X] T025 Create runtime SQLite migration `003_runtime_event_indexes.sql` in internal/agentdserver/infra/db/migrations/runtime/003_runtime_event_indexes.sql
- [X] T026 Add migration schema tests for settings/runtime DBs in internal/agentdserver/infra/db/migrations_test.go
- [X] T027 Define repository interfaces for Agents, Agent Runs, Runtime Events, and logs in internal/agentdserver/app/ports.go
- [X] T028 Implement settings DB Agent repository skeleton in internal/agentdserver/infra/db/repository/agent_repository.go
- [X] T029 Implement per-Agent runtime DB manager skeleton in internal/agentdserver/infra/db/repository/runtime_db_manager.go
- [X] T030 Implement runtime DB run/event repository skeleton in internal/agentdserver/infra/db/repository/run_repository.go
- [X] T031 Add repository integration test fixtures in internal/agentdserver/infra/db/repository/repository_test.go
- [X] T032 Define vendor-agnostic LLM provider port in internal/agentdserver/app/runtime/provider.go
- [X] T033 Implement fake LLM provider for tests in internal/agentdserver/app/runtime/fake_provider_test.go
- [X] T034 Implement OpenAI provider adapter skeleton using `openai-go` in internal/agentdserver/infra/llm/openai/provider.go
- [X] T035 Add OpenAI provider configuration tests without real API calls in internal/agentdserver/infra/llm/openai/provider_test.go
- [X] T036 Define runtime manager interfaces for execute, stop, recovery, and active-run tracking in internal/agentdserver/app/runtime/manager.go
- [X] T037 Define scheduler adapter interface for cron/manual schedules in internal/agentdserver/app/scheduling/scheduler.go
- [X] T038 Create REST server skeleton with health endpoint in internal/agentdserver/infra/http/server.go
- [X] T039 Add REST server health test in internal/agentdserver/infra/http/server_test.go
- [X] T040 Create daemon service wiring root in internal/agentdserver/service.go
- [X] T041 Add daemon service startup/shutdown wiring test in internal/agentdserver/service_test.go
- [X] T042 Create CLI root command and output policy in internal/agentd/app/root.go
- [X] T043 Create CLI HTTP client skeleton in internal/agentd/infra/httpclient/client.go

**Checkpoint**: Foundational architecture is ready for story implementation.

---

## Phase 3: User Story 1 - Apply Agent Definition (Priority: P1)

**Goal**: Users can apply Markdown Agent Definitions and get created, updated,
unchanged, or rejected outcomes without mutating existing state on validation
failure.

**Independent Test**: Apply valid, changed, unchanged, and invalid Markdown
definitions and verify Agent settings plus schedule summary.

### Tests for User Story 1

- [X] T044 [P] [US1] Add Agent Definition parser contract tests in internal/agentdserver/infra/definition/parser_test.go
- [X] T045 [P] [US1] Add apply use case tests for created/updated/unchanged/rejected outcomes in internal/agentdserver/app/agent/apply_test.go
- [X] T046 [P] [US1] Add settings repository tests for Agent and permission persistence in internal/agentdserver/infra/db/repository/agent_repository_test.go
- [X] T047 [P] [US1] Add REST contract tests for `POST /v1/agents/apply` in internal/agentdserver/infra/http/apply_handler_test.go
- [X] T048 [P] [US1] Add CLI apply command tests in internal/agentd/app/apply_test.go

### Implementation for User Story 1

- [X] T049 [US1] Implement Markdown front matter parser in internal/agentdserver/infra/definition/parser.go
- [X] T050 [US1] Implement Agent Definition validation and normalization in internal/agentdserver/app/agent/definition_validator.go
- [X] T051 [US1] Implement settings Agent repository create/update/read methods in internal/agentdserver/infra/db/repository/agent_repository.go
- [X] T052 [US1] Implement per-Agent runtime DB creation after first apply in internal/agentdserver/infra/db/repository/runtime_db_manager.go
- [X] T053 [US1] Implement apply use case in internal/agentdserver/app/agent/apply.go
- [X] T054 [US1] Implement apply HTTP request/response models in internal/agentdserver/infra/http/model/agent.go
- [X] T055 [US1] Implement `POST /v1/agents/apply` handler in internal/agentdserver/infra/http/apply_handler.go
- [X] T056 [US1] Register apply handler in internal/agentdserver/infra/http/server.go
- [X] T057 [US1] Implement CLI HTTP client apply method in internal/agentd/infra/httpclient/agents.go
- [X] T058 [US1] Implement `agentd apply <path_to_file>` command in internal/agentd/app/apply.go
- [X] T059 [US1] Wire apply command into CLI root in internal/agentd/app/root.go
- [X] T060 [US1] Add end-to-end apply smoke test in tests/e2e/apply_test.go

**Checkpoint**: US1 is independently functional and testable.

---

## Phase 4: User Story 2 - Schedule and Execute Agents (Priority: P2)

**Goal**: The daemon runs enabled Agents by cron/manual triggers, supports many
concurrent isolated Agent Runs, records run state/log references, and recovers
after restart.

**Independent Test**: Apply overlapping cron/manual Agents, execute one
manually, verify concurrent isolated runs, stop one failed/stalled run, and
restart daemon to mark active runs interrupted and restore schedules.

### Tests for User Story 2

- [X] T061 [P] [US2] Add scheduler adapter tests for cron and manual schedule modes in internal/agentdserver/infra/scheduler/scheduler_test.go
- [X] T062 [P] [US2] Add runtime manager concurrency and isolation tests in internal/agentdserver/infra/runtime/manager_test.go
- [X] T063 [P] [US2] Add runtime DB Agent Run repository tests in internal/agentdserver/infra/db/repository/run_repository_test.go
- [X] T064 [P] [US2] Add execute use case tests for manual run, disabled Agent, unknown Agent, and same-Agent overlap rejection in internal/agentdserver/app/runtime/execute_test.go
- [X] T065 [P] [US2] Add stop use case tests for cancellation outcomes in internal/agentdserver/app/runtime/stop_test.go
- [X] T066 [P] [US2] Add recovery tests for interrupted active runs in internal/agentdserver/app/runtime/recovery_test.go
- [X] T067 [P] [US2] Add REST contract tests for execute and stop endpoints in internal/agentdserver/infra/http/run_handler_test.go
- [X] T068 [P] [US2] Add CLI execute command tests in internal/agentd/app/execute_test.go

### Implementation for User Story 2

- [X] T069 [US2] Implement cron/manual scheduler adapter in internal/agentdserver/infra/scheduler/scheduler.go
- [X] T070 [US2] Implement schedule reconciliation use case in internal/agentdserver/app/scheduling/reconcile.go
- [X] T071 [US2] Implement per-Agent runtime DB Agent Run create/update/query methods in internal/agentdserver/infra/db/repository/run_repository.go
- [X] T072 [US2] Implement runtime event repository methods in internal/agentdserver/infra/db/repository/event_repository.go
- [X] T073 [US2] Implement per-run work directory and environment builder in internal/agentdserver/infra/runtime/isolation.go
- [X] T074 [US2] Implement isolated per-run log writer creation in internal/agentdserver/infra/logs/run_writer.go
- [X] T075 [US2] Implement concurrent runtime manager with active run registry in internal/agentdserver/infra/runtime/manager.go
- [X] T076 [US2] Implement execute use case in internal/agentdserver/app/runtime/execute.go
- [X] T077 [US2] Implement stop use case in internal/agentdserver/app/runtime/stop.go
- [X] T078 [US2] Implement daemon restart recovery use case in internal/agentdserver/app/runtime/recovery.go
- [X] T079 [US2] Implement OpenAI provider execution call behind provider port in internal/agentdserver/infra/llm/openai/provider.go
- [X] T080 [US2] Implement `POST /v1/agents/{name}/runs` handler in internal/agentdserver/infra/http/run_handler.go
- [X] T081 [US2] Implement `POST /v1/agents/{name}/runs/{run_id}/stop` handler in internal/agentdserver/infra/http/stop_handler.go
- [X] T082 [US2] Register run and stop handlers in internal/agentdserver/infra/http/server.go
- [X] T083 [US2] Wire scheduler, runtime manager, provider, and recovery into daemon service in internal/agentdserver/service.go
- [X] T084 [US2] Implement CLI HTTP client execute and stop methods in internal/agentd/infra/httpclient/runs.go
- [X] T085 [US2] Implement `agentd execute <agent_name>` command in internal/agentd/app/execute.go
- [X] T086 [US2] Implement `agentd stop <agent_name> [--run <run_id>]` command in internal/agentd/app/stop.go
- [X] T087 [US2] Wire execute and stop commands into CLI root in internal/agentd/app/root.go
- [X] T088 [US2] Add end-to-end concurrency and recovery test in tests/e2e/runtime_test.go

**Checkpoint**: US2 is independently functional and testable.

---

## Phase 5: User Story 3 - Operate Agents from CLI (Priority: P3)

**Goal**: Users can list, inspect, stop, and read isolated logs for Agents and
Agent Runs from the CLI with Docker-like visibility.

**Independent Test**: Apply an Agent, execute it, inspect it, stop it, list all
Agents, and retrieve logs with no cross-Agent log mixing.

### Tests for User Story 3

- [X] T089 [P] [US3] Add inspect/list use case tests in internal/agentdserver/app/agent/inspect_test.go
- [X] T090 [P] [US3] Add logs use case tests for latest run, run ID, no logs, and pruned logs in internal/agentdserver/app/logs/logs_test.go
- [X] T091 [P] [US3] Add REST contract tests for list, inspect, and logs endpoints in internal/agentdserver/infra/http/query_handler_test.go
- [X] T092 [P] [US3] Add CLI inspect/list/logs command tests in internal/agentd/app/query_test.go
- [X] T093 [P] [US3] Add log isolation integration test for concurrent Agents in tests/e2e/logs_test.go

### Implementation for User Story 3

- [X] T094 [US3] Implement inspect and list use cases in internal/agentdserver/app/agent/inspect.go
- [X] T095 [US3] Implement run log reader in internal/agentdserver/infra/logs/reader.go
- [X] T096 [US3] Implement logs use case in internal/agentdserver/app/logs/logs.go
- [X] T097 [US3] Implement `GET /v1/agents` list handler in internal/agentdserver/infra/http/list_handler.go
- [X] T098 [US3] Implement `GET /v1/agents/{name}` inspect handler in internal/agentdserver/infra/http/inspect_handler.go
- [X] T099 [US3] Implement `GET /v1/agents/{name}/logs` logs handler in internal/agentdserver/infra/http/logs_handler.go
- [X] T100 [US3] Register list, inspect, and logs handlers in internal/agentdserver/infra/http/server.go
- [X] T101 [US3] Implement CLI HTTP client list, inspect, and logs methods in internal/agentd/infra/httpclient/query.go
- [X] T102 [US3] Implement `agentd list` command in internal/agentd/app/list.go
- [X] T103 [US3] Implement `agentd inspect <agent_name>` command in internal/agentd/app/inspect.go
- [X] T104 [US3] Implement `agentd logs <agent_name> [--run <run_id>] [--tail N]` command in internal/agentd/app/logs.go
- [X] T105 [US3] Wire list, inspect, and logs commands into CLI root in internal/agentd/app/root.go
- [X] T106 [US3] Add end-to-end CLI operations test in tests/e2e/cli_operations_test.go

**Checkpoint**: US3 is independently functional and testable.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, hardening, verification, and release readiness.

- [X] T107 [P] Add Linux/macOS runtime path and signal behavior tests in internal/agentdserver/infra/runtime/platform_test.go
- [X] T108 [P] Add API error response consistency tests in internal/agentdserver/infra/http/errors_test.go
- [X] T109 [P] Add CLI JSON/text output snapshot tests in internal/agentd/app/output_test.go
- [X] T110 [P] Add benchmark for five concurrent Agent Runs writing to separate runtime DBs in internal/agentdserver/infra/runtime/concurrency_benchmark_test.go
- [X] T111 Add service-level slog event names and attributes documentation in docs/observability.md
- [X] T112 Add local development guide with `.env`, `OPENAI_API_KEY`, and data directory defaults in docs/development.md
- [X] T113 Update quickstart examples in specs/001-agent-definition-runtime/quickstart.md after implementation paths are confirmed
- [X] T114 Run `go test ./...` and record any follow-up fixes in specs/001-agent-definition-runtime/tasks.md (passed 2026-05-07; no follow-up fixes required)
- [ ] T115 Run quickstart apply/execute/logs/recovery validation from specs/001-agent-definition-runtime/quickstart.md
- [ ] T116 Verify Git history has no committed `.env` files or OpenAI API keys, and rewrite history before release if any are found
- [ ] T117 Add ai-product-research example Agent Definition and local script-tool support plan in examples/ai-product-research.md and specs/001-agent-definition-runtime/quickstart.md: model the Product Hunt research workflow, include a Playwright screenshot tool declaration, document required env vars without committing secrets, and verify the runtime can represent script tools with explicit command, args, env allow-list, read/write paths, and network allow-list

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies.
- **Foundational (Phase 2)**: Depends on Setup; blocks all user stories.
- **US1 Apply Agent Definition (Phase 3)**: Depends on Foundational.
- **US2 Schedule and Execute Agents (Phase 4)**: Depends on Foundational and uses Agents created by US1.
- **US3 Operate Agents from CLI (Phase 5)**: Depends on Foundational and benefits from US1/US2 for meaningful data.
- **Polish (Phase 6)**: Depends on desired user stories being complete.

### User Story Dependencies

- **US1**: MVP. Provides Agent Definition apply and persistence.
- **US2**: Requires applied Agents from US1, then adds scheduling, execution, concurrency, stop, and recovery.
- **US3**: Requires applied/running Agents to inspect/log; can start after US1 for list/inspect and after US2 for stop/logs.

### Within Each User Story

- Tests MUST be written and fail before implementation.
- Domain and repository behavior before use cases.
- Use cases before HTTP handlers.
- HTTP client before CLI commands.
- End-to-end tests after CLI/server wiring.

---

## Parallel Opportunities

- Setup tasks T007-T010 can run in parallel.
- Foundational domain/config/DB skeleton tasks T014-T018 can run in parallel after T001-T006.
- Migration files T021-T025 can be created in parallel.
- US1 tests T044-T048 can run in parallel.
- US2 tests T061-T068 can run in parallel.
- US3 tests T089-T093 can run in parallel.
- Polish tests/docs T107-T113 can run in parallel after user stories complete.

## Parallel Example: User Story 1

```bash
Task: "T044 [P] [US1] Add Agent Definition parser contract tests in internal/agentdserver/infra/definition/parser_test.go"
Task: "T045 [P] [US1] Add apply use case tests for created/updated/unchanged/rejected outcomes in internal/agentdserver/app/agent/apply_test.go"
Task: "T047 [P] [US1] Add REST contract tests for POST /v1/agents/apply in internal/agentdserver/infra/http/apply_handler_test.go"
Task: "T048 [P] [US1] Add CLI apply command tests in internal/agentd/app/apply_test.go"
```

## Parallel Example: User Story 2

```bash
Task: "T061 [P] [US2] Add scheduler adapter tests for cron and manual schedule modes in internal/agentdserver/infra/scheduler/scheduler_test.go"
Task: "T062 [P] [US2] Add runtime manager concurrency and isolation tests in internal/agentdserver/infra/runtime/manager_test.go"
Task: "T064 [P] [US2] Add execute use case tests for manual run, disabled Agent, unknown Agent, and same-Agent overlap rejection in internal/agentdserver/app/runtime/execute_test.go"
Task: "T067 [P] [US2] Add REST contract tests for execute and stop endpoints in internal/agentdserver/infra/http/run_handler_test.go"
```

## Parallel Example: User Story 3

```bash
Task: "T089 [P] [US3] Add inspect/list use case tests in internal/agentdserver/app/agent/inspect_test.go"
Task: "T090 [P] [US3] Add logs use case tests for latest run, run ID, no logs, and pruned logs in internal/agentdserver/app/logs/logs_test.go"
Task: "T091 [P] [US3] Add REST contract tests for list, inspect, and logs endpoints in internal/agentdserver/infra/http/query_handler_test.go"
Task: "T092 [P] [US3] Add CLI inspect/list/logs command tests in internal/agentd/app/query_test.go"
```

---

## Implementation Strategy

### MVP First (US1 Only)

1. Complete Phase 1 Setup.
2. Complete Phase 2 Foundational architecture.
3. Complete Phase 3 US1 apply flow.
4. Validate `agentd apply <path_to_file>` for created, updated, unchanged, and rejected outcomes.

### Incremental Delivery

1. US1 delivers Agent-as-Code apply and persistent Agent records.
2. US2 adds daemon-owned execution, scheduling, concurrency, stop, and recovery.
3. US3 adds operational visibility with list, inspect, and isolated logs.
4. Polish validates Linux/macOS, observability, quickstart, and full test suite.

### Team Parallel Strategy

1. One stream owns domain/config/DB foundations.
2. One stream owns REST/CLI contracts and client/server wiring.
3. One stream owns runtime/scheduler/provider/log isolation.
4. Integrate at story checkpoints with end-to-end tests.
