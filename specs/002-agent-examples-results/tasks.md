# Tasks: Agent Examples and Results

**Input**: Design documents from `/specs/002-agent-examples-results/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: Included because this feature changes daemon control APIs, run
persistence, tool execution, isolation policy, observability, public client
contracts, and example smoke verification.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1, US2, US3, US4, US5)
- Include exact file paths in descriptions

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Prepare package structure, shared fixtures, and baseline contracts.

- [x] T001 Create public client package skeleton in pkg/agentdclient/{client.go,types.go,agents.go,runs.go,logs.go}
- [x] T002 Create result app package skeleton in internal/agentdserver/app/result/doc.go
- [x] T003 Create CLI command skeleton files in internal/agentd/app/{ps.go,result.go}
- [x] T004 Create example directory skeletons under examples/{cybersecurity-reddit-watch,hacker-news-builder-brief,reddit-customer-pain-monitor,product-hunt-launch-radar,github-trending-engineering-radar,developer-dependency-release-monitor,ai-engineering-hiring-signal-monitor,website-snapshot-analyst}/
- [x] T005 [P] Create example shared test fixture directory in internal/lib/testutil/examples/
- [x] T006 [P] Add contract fixture JSON files for run list and result responses in internal/lib/testutil/contracts/

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Add shared persistence, domain, daemon route, client, and output foundations required by all stories.

**CRITICAL**: No user story work should start until this phase is complete.

- [x] T007 Add run result fields and ToolExecution domain types in internal/agentdserver/domain/agent.go
- [x] T008 Add runtime migration for run result columns in internal/agentdserver/infra/db/migrations/runtime/004_run_results.sql
- [x] T009 Add runtime migration for tool executions in internal/agentdserver/infra/db/migrations/runtime/005_tool_executions.sql
- [x] T010 Update migration coverage for result/tool tables in internal/agentdserver/infra/db/migrations_test.go
- [x] T011 Extend run repository interfaces for run listing, result persistence, and result lookup in internal/agentdserver/app/ports.go
- [x] T012 Implement run result and run listing repository methods in internal/agentdserver/infra/db/repository/run_repository.go
- [x] T013 [P] Add repository tests for terminal run result persistence in internal/agentdserver/infra/db/repository/run_repository_test.go
- [x] T014 [P] Add repository tests for active/all run listing in internal/agentdserver/infra/db/repository/run_repository_test.go
- [x] T015 Add same-host request middleware in internal/agentdserver/infra/http/server.go
- [x] T016 [P] Add same-host middleware tests in internal/agentdserver/infra/http/server_test.go
- [x] T017 Implement stable daemon error code mapping for result/run errors in internal/agentdserver/infra/http/errors.go
- [x] T018 Move or wrap internal HTTP client primitives into pkg/agentdclient/client.go
- [x] T019 Define public client request/response/error types in pkg/agentdclient/types.go
- [x] T020 [P] Add public client unit tests for error decoding in pkg/agentdclient/client_test.go
- [x] T021 Extend CLI output support for table trimming and JSON stability in internal/agentd/app/output.go
- [x] T022 [P] Add CLI output tests for trimmed result table formatting in internal/agentd/app/output_test.go

**Checkpoint**: Persistence, same-host boundary, public client base, and output primitives are ready.

---

## Phase 3: User Story 1 - Replace Examples with Real Agents (Priority: P1) MVP

**Goal**: Replace the weak flat examples with eight self-contained example folders: seven daily monitoring agents and one manual website snapshot agent.

**Independent Test**: From a fresh clone after documented local dependency installation, apply each example definition, verify folder README/tool/source layout, execute examples through smoke paths, and retrieve at least one run result/log per example.

### Tests for User Story 1

- [x] T023 [P] [US1] Add example catalog validation test in internal/agentdserver/infra/definition/example_catalog_test.go
- [x] T024 [P] [US1] Add example README checklist test in internal/agentdserver/infra/definition/example_readme_test.go
- [x] T025 [P] [US1] Add definition parser fixture tests for local tools, inputs, and network access in internal/agentdserver/infra/definition/parser_test.go
- [x] T026 [P] [US1] Add example smoke test harness in tests/e2e/examples_smoke_test.go

### Implementation for User Story 1

- [x] T027 [US1] Delete old flat examples in examples/{ai-product-research.md,daily-pr-review.md,release-notes-helper.md}
- [x] T028 [P] [US1] Create cybersecurity-reddit-watch definition and README in examples/cybersecurity-reddit-watch/
- [x] T029 [P] [US1] Create hacker-news-builder-brief definition and README in examples/hacker-news-builder-brief/
- [x] T030 [P] [US1] Create reddit-customer-pain-monitor definition and README in examples/reddit-customer-pain-monitor/
- [x] T031 [P] [US1] Create product-hunt-launch-radar definition and README in examples/product-hunt-launch-radar/
- [x] T032 [P] [US1] Create github-trending-engineering-radar definition and README in examples/github-trending-engineering-radar/
- [x] T033 [P] [US1] Create developer-dependency-release-monitor definition and README in examples/developer-dependency-release-monitor/
- [x] T034 [P] [US1] Create ai-engineering-hiring-signal-monitor definition and README in examples/ai-engineering-hiring-signal-monitor/
- [x] T035 [P] [US1] Create website-snapshot-analyst definition and README in examples/website-snapshot-analyst/
- [x] T036 [P] [US1] Add public source lists for scheduled examples in examples/*/sources/
- [x] T037 [P] [US1] Add zero-configuration fixture files where required in examples/*/fixtures/
- [x] T038 [P] [US1] Add local fetch/screenshot tool scripts in examples/*/tools/
- [x] T039 [US1] Update Markdown definition parser for inputs/tool timeout/network metadata in internal/agentdserver/infra/definition/parser.go
- [x] T040 [US1] Update definition validation for example-local tool paths and no required secrets in internal/agentdserver/app/agent/definition_validator.go
- [x] T041 [US1] Document example catalog usage in specs/002-agent-examples-results/quickstart.md

**Checkpoint**: The new examples are present, self-contained, parseable, and smoke-testable.

---

## Phase 4: User Story 2 - Discover Definitions and Runs (Priority: P2)

**Goal**: Let users list applied definitions and view active or all Agent Runs with Docker-like CLI commands.

**Independent Test**: Apply examples, start a manual run, let one run finish, then verify `agentd list`, `agentd ps`, and `agentd ps -a` show expected names, run IDs, statuses, triggers, and timestamps.

### Tests for User Story 2

- [x] T042 [P] [US2] Add HTTP contract tests for GET /v1/runs active/all in internal/agentdserver/infra/http/run_query_handler_test.go
- [ ] T043 [P] [US2] Add CLI tests for agentd ps and ps -a in internal/agentd/app/ps_test.go
- [ ] T044 [P] [US2] Add public client tests for ListRuns in pkg/agentdclient/runs_test.go

### Implementation for User Story 2

- [ ] T045 [US2] Implement run list use case in internal/agentdserver/app/result/list_runs.go
- [ ] T046 [US2] Add GET /v1/runs handler in internal/agentdserver/infra/http/run_query_handler.go
- [ ] T047 [US2] Register GET /v1/runs route in internal/agentdserver/infra/http/server.go
- [ ] T048 [US2] Implement ListRuns in pkg/agentdclient/runs.go
- [ ] T049 [US2] Wire CLI QueryClient to public client run listing in internal/agentd/infra/httpclient/runs.go
- [ ] T050 [US2] Implement agentd ps command and -a flag in internal/agentd/app/ps.go
- [ ] T051 [US2] Register ps command in internal/agentd/app/root.go
- [ ] T052 [US2] Update CLI contract documentation in specs/002-agent-examples-results/contracts/cli.md

**Checkpoint**: Users can discover active and finished runs independently of result retrieval.

---

## Phase 5: User Story 3 - Retrieve Run Results for Automation (Priority: P2)

**Goal**: Let humans, Bash scripts, local AI agents, and Go integrations retrieve compact result history by agent and full run details by run ID.

**Independent Test**: Complete successful and failed runs, verify `agentd result <agent-name>` and `agentd result <run-id>` in text/JSON, then verify a Bash script and Go client can execute and retrieve results without reading storage files.

### Tests for User Story 3

- [ ] T053 [P] [US3] Add repository tests for result lookup by agent and run ID in internal/agentdserver/infra/db/repository/run_repository_test.go
- [ ] T054 [P] [US3] Add result use case tests in internal/agentdserver/app/result/result_test.go
- [ ] T055 [P] [US3] Add HTTP contract tests for GET /v1/agents/{name}/results and GET /v1/runs/{run_id}/result in internal/agentdserver/infra/http/result_handler_test.go
- [ ] T056 [P] [US3] Add CLI result command tests for text and JSON output in internal/agentd/app/result_test.go
- [ ] T057 [P] [US3] Add public client result tests in pkg/agentdclient/results_test.go
- [ ] T058 [P] [US3] Add Bash automation scenario to tests/e2e/result_automation_test.go

### Implementation for User Story 3

- [ ] T059 [US3] Persist successful and failed run results from runtime completion in internal/agentdserver/app/runtime/manager.go
- [ ] T060 [US3] Implement result summary generation in internal/agentdserver/app/result/summary.go
- [ ] T061 [US3] Implement ResultsByAgent and ResultByRunID use cases in internal/agentdserver/app/result/result.go
- [ ] T062 [US3] Add result HTTP handlers in internal/agentdserver/infra/http/result_handler.go
- [ ] T063 [US3] Register result routes in internal/agentdserver/infra/http/server.go
- [ ] T064 [US3] Implement ResultsByAgent and ResultByRunID in pkg/agentdclient/runs.go
- [ ] T065 [US3] Implement agentd result command dispatch for agent name vs UUID in internal/agentd/app/result.go
- [ ] T066 [US3] Add CLI exit-code mapping for missing agent, missing run, active run, failed run, and daemon unavailable in internal/agentd/app/result.go
- [ ] T067 [US3] Update OpenAPI results contract in specs/002-agent-examples-results/contracts/openapi.yaml
- [ ] T068 [US3] Update public Go client contract in specs/002-agent-examples-results/contracts/public-go-client.md

**Checkpoint**: Results are durable, scriptable, and available through CLI and Go client.

---

## Phase 6: User Story 4 - Audit Agent Execution Logs (Priority: P3)

**Goal**: Show scoped system-level action logs for each agent run instead of only final LLM responses.

**Independent Test**: Execute an agent with at least one tool and verify `agentd logs <agent-name> --run <run-id>` shows timestamped prompt/tool/result/run action entries scoped to that run only.

### Tests for User Story 4

- [ ] T069 [P] [US4] Add runtime event repository tests for action lookup by run in internal/agentdserver/infra/db/repository/run_repository_test.go
- [ ] T070 [P] [US4] Add logs use case tests for scoped action logs in internal/agentdserver/app/logs/logs_test.go
- [ ] T071 [P] [US4] Add CLI logs regression test for action logs in internal/agentd/app/logs_test.go

### Implementation for User Story 4

- [ ] T072 [US4] Add stable run action constants in internal/agentdserver/domain/agent.go
- [ ] T073 [US4] Emit llm.prompt.send and provider response action events in internal/agentdserver/app/runtime/manager.go
- [ ] T074 [US4] Emit run.result.persisted, run.complete, and run.fail events in internal/agentdserver/app/runtime/manager.go
- [ ] T075 [US4] Update log reader to merge or expose runtime action events by run in internal/agentdserver/infra/logs/reader.go
- [ ] T076 [US4] Update logs HTTP response mapping for action log fields in internal/agentdserver/infra/http/logs_handler.go
- [ ] T077 [US4] Update logs CLI formatting for timestamp/action/message in internal/agentd/app/logs.go

**Checkpoint**: Logs explain what the daemon did for one run and remain scoped to that run.

---

## Phase 7: User Story 5 - Execute Declared Command-Line Tools (Priority: P3)

**Goal**: Let agent runs execute declared local command-line tools as separate processes with timeout, scoped access, result capture, and audit logs.

**Independent Test**: Apply an example with declared tools, execute it, verify declared tool execution succeeds, undeclared tools are denied, timeout/non-zero exits fail the run with result/log evidence, and other runs remain isolated.

### Tests for User Story 5

- [ ] T078 [P] [US5] Add tool declaration validation tests in internal/agentdserver/app/agent/definition_validator_test.go
- [ ] T079 [P] [US5] Add process tool adapter tests for success, non-zero exit, timeout, and env scoping in internal/agentdserver/infra/runtime/tool_process_test.go
- [ ] T080 [P] [US5] Add runtime manager tests for declared tool execution in internal/agentdserver/app/runtime/execute_test.go
- [ ] T081 [P] [US5] Add isolation test for undeclared tool denial in internal/agentdserver/infra/runtime/manager_test.go
- [ ] T082 [P] [US5] Add Linux/macOS platform verification tests for process cancellation in internal/agentdserver/infra/runtime/platform_test.go

### Implementation for User Story 5

- [ ] T083 [US5] Add ToolExecutor port to internal/agentdserver/app/runtime/provider.go
- [ ] T084 [US5] Implement command-line process tool adapter in internal/agentdserver/infra/runtime/tool_process.go
- [ ] T085 [US5] Persist tool execution records in internal/agentdserver/infra/db/repository/run_repository.go
- [ ] T086 [US5] Wire ToolExecutor into service construction in internal/agentdserver/service.go
- [ ] T087 [US5] Enforce declared-tool lookup before execution in internal/agentdserver/app/runtime/manager.go
- [ ] T088 [US5] Add timeout, stdout/stderr summary, and non-zero exit handling in internal/agentdserver/infra/runtime/tool_process.go
- [ ] T089 [US5] Emit tool.execute.start, tool.execute.complete, and tool.execute.fail events in internal/agentdserver/app/runtime/manager.go
- [ ] T090 [US5] Update example tools to follow CLI stdin/stdout/exit contract in examples/*/tools/

**Checkpoint**: Declared local tools are auditable, bounded, and integrated into run results.

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Final verification, documentation consistency, and cleanup.

- [ ] T091 [P] Update root README or docs with examples/results overview in README.md
- [ ] T092 [P] Update AGENTS.md context if implementation plan path changes in AGENTS.md
- [ ] T093 [P] Run OpenAPI/schema consistency review against specs/002-agent-examples-results/contracts/openapi.yaml
- [ ] T094 [P] Run public Go client import smoke test in pkg/agentdclient/client_test.go
- [ ] T095 Run example catalog smoke tests from specs/002-agent-examples-results/quickstart.md
- [ ] T096 Run restart recovery verification for persisted run results and active tool interruption in tests/e2e/recovery_results_test.go
- [ ] T097 Run Linux/macOS parity checklist for tool execution and screenshot dependencies in specs/002-agent-examples-results/quickstart.md
- [ ] T098 Run full `go test ./...`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies.
- **Foundational (Phase 2)**: Depends on Setup and blocks all user stories.
- **US1 (Phase 3)**: Depends on Foundational; MVP deliverable.
- **US2 (Phase 4)**: Depends on Foundational; can run in parallel with US1 after shared storage/client base.
- **US3 (Phase 5)**: Depends on Foundational and benefits from US2 run listing, but result retrieval is independently testable by run ID.
- **US4 (Phase 6)**: Depends on Foundational; benefits from US3 result persistence but can be tested with existing run logs.
- **US5 (Phase 7)**: Depends on Foundational; integrates with US1 examples and US4 logs.
- **Polish (Phase 8)**: Depends on selected user stories being complete.

### User Story Dependencies

- **US1**: Primary MVP for replacing examples.
- **US2**: Can proceed after Foundational; no dependency on US1 implementation beyond having any applied agents.
- **US3**: Can proceed after Foundational; should use generated or fixture runs for independent tests.
- **US4**: Can proceed after Foundational; validates logs for any run.
- **US5**: Can proceed after Foundational; final integration with example tools requires US1 examples.

### Parallel Opportunities

- Setup skeleton tasks T005-T006 can run in parallel.
- Repository, HTTP, public client, and CLI tests in Phase 2 can run in parallel after domain/schema decisions.
- Example folder tasks T028-T038 can run in parallel.
- US2 HTTP, CLI, and public client tests T042-T044 can run in parallel.
- US3 repository, use case, HTTP, CLI, public client, and Bash tests T053-T058 can run in parallel.
- US4 logs tests T069-T071 can run in parallel.
- US5 validation, process adapter, runtime, isolation, and platform tests T078-T082 can run in parallel.

---

## Parallel Example: User Story 1

```text
Task: "T028 Create cybersecurity-reddit-watch definition and README in examples/cybersecurity-reddit-watch/"
Task: "T029 Create hacker-news-builder-brief definition and README in examples/hacker-news-builder-brief/"
Task: "T030 Create reddit-customer-pain-monitor definition and README in examples/reddit-customer-pain-monitor/"
Task: "T031 Create product-hunt-launch-radar definition and README in examples/product-hunt-launch-radar/"
Task: "T032 Create github-trending-engineering-radar definition and README in examples/github-trending-engineering-radar/"
Task: "T033 Create developer-dependency-release-monitor definition and README in examples/developer-dependency-release-monitor/"
Task: "T034 Create ai-engineering-hiring-signal-monitor definition and README in examples/ai-engineering-hiring-signal-monitor/"
Task: "T035 Create website-snapshot-analyst definition and README in examples/website-snapshot-analyst/"
```

## Parallel Example: User Story 3

```text
Task: "T055 Add HTTP contract tests for result endpoints in internal/agentdserver/infra/http/result_handler_test.go"
Task: "T056 Add CLI result command tests for text and JSON output in internal/agentd/app/result_test.go"
Task: "T057 Add public client result tests in pkg/agentdclient/results_test.go"
Task: "T058 Add Bash automation scenario to tests/e2e/result_automation_test.go"
```

## Implementation Strategy

### MVP First

Complete Phases 1-3 to replace examples and prove the repository now contains
self-contained real agent definitions.

### Infrastructure Increment

Complete Phases 4-5 to make runs and results discoverable by humans, Bash, and
Go integrations.

### Observability and Tools

Complete Phases 6-7 to make execution auditable and enable flexible declared
tool processes.

### Final Verification

Complete Phase 8 and validate `quickstart.md`, example smoke tests, recovery,
and `go test ./...`.
