# Tasks: Unified Agentd ReAct Contracts

**Input**: Design documents from `/specs/004-agentd-react-contracts/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: Tests are required by the Agentd Constitution for daemon APIs, agent execution, isolation policy, persistence, recovery, provider process handling, and Linux/macOS behavior. Test tasks are listed before implementation tasks within each user story and should fail before implementation.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel because it touches different files and has no dependency on incomplete tasks in the same phase
- **[Story]**: Maps to a user story from `spec.md`
- Every task includes an exact file path

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Add shared dependencies and test helpers used by multiple stories.

- [X] T001 Add JSON Schema validator and go-agent module dependency to go.mod and go.sum
- [X] T002 [P] Add JSON contract assertion helpers in internal/lib/testutil/testutil.go
- [X] T003 [P] Add reusable HTTP contract fixture loader in internal/lib/testutil/contracts/agent_contract_response.json
- [X] T004 [P] Add CLI command test helper utilities in internal/agentd/app/test_helpers_test.go
- [X] T005 [P] Add process fake helper utilities for provider/process tests in tests/e2e/process_test.go
- [X] T006 Add contract-aware fixture examples directory marker in tests/fixtures/.gitkeep

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Establish shared domain, persistence, validation, and provider boundaries before user-story work.

**CRITICAL**: No user story work should begin until this phase is complete.

- [X] T007 [P] Add contract/domain tests for AgentContract, runtime input, ReAct step, provider metadata, and run result format in internal/agentdserver/domain/agent_test.go
- [X] T008 Add AgentContract, RuntimeInput, ReActStep, provider metadata, and result format fields in internal/agentdserver/domain/agent.go
- [X] T009 [P] Add validation error code tests for contract and provider failures in internal/agentdserver/domain/errors_test.go
- [X] T010 Add contract and provider error codes in internal/agentdserver/domain/errors.go
- [X] T011 [P] Add settings migration tests for nullable contract fields and legacy rows in internal/agentdserver/infra/db/migrations_test.go
- [X] T012 Create additive settings migration for contract/revision metadata in internal/agentdserver/infra/db/migrations/settings/004_agent_contracts.sql
- [X] T013 [P] Add repository tests for saving/loading contracts on agents and revisions in internal/agentdserver/infra/db/repository/agent_repository_test.go
- [X] T014 Persist contract schemas, schema digests, result format, and provider metadata in internal/agentdserver/infra/db/repository/agent_repository.go
- [X] T015 [P] Add runtime contract validator tests for schema compile, input validation, output validation, and diagnostics in internal/agentdserver/infra/runtime/contract_validator_test.go
- [X] T016 Implement JSON Schema contract validator adapter in internal/agentdserver/infra/runtime/contract_validator.go
- [X] T017 [P] Add provider interface compile tests for plain, step, and structured-output requests in internal/agentdserver/app/runtime/provider_test.go
- [X] T018 Extend runtime provider interfaces for ReAct decisions and structured final output in internal/agentdserver/app/runtime/provider.go

**Checkpoint**: Foundation ready; user story implementation can start.

---

## Phase 3: User Story 1 - Distribute One Agentd Binary (Priority: P1)

**Goal**: A user installs only `agentd` and can run either daemon mode or client commands.

**Independent Test**: Build `agentd`, start daemon mode with `--daemon` and `-d`, reject daemon mode combined with a client subcommand, then run a normal client command against the daemon.

### Tests for User Story 1

- [X] T019 [P] [US1] Add daemon flag dispatch tests for `--daemon`, `-d`, and `--deamon` in cmd/agentd/main_test.go
- [X] T020 [P] [US1] Add client-mode regression tests for existing root command behavior in internal/agentd/app/root_test.go
- [X] T021 [P] [US1] Add e2e unified binary startup smoke test in tests/e2e/unified_binary_test.go

### Implementation for User Story 1

- [X] T022 [US1] Add daemon-mode dispatch before HTTP client construction in cmd/agentd/main.go
- [X] T023 [US1] Add reusable daemon runner with signal shutdown in cmd/agentd/daemon.go
- [X] T024 [US1] Update root help and daemon flag validation in internal/agentd/app/root.go
- [X] T025 [US1] Convert cmd/agentdserver/main.go to a deprecated compatibility shim that starts the shared daemon runner from cmd/agentd/daemon.go
- [X] T026 [US1] Update build and usage documentation for one-binary distribution in README.md

**Checkpoint**: User Story 1 is independently functional.

---

## Phase 4: User Story 2 - Validate Agent Inputs Before Execution (Priority: P1)

**Goal**: Agentd validates contracted JSON input before creating a run or invoking tools/models.

**Independent Test**: Apply a contracted manual agent, run valid and invalid JSON inputs, verify invalid input fails before run creation, provider call, or tool execution, and verify legacy agents without contracts still run.

### Tests for User Story 2

- [X] T027 [P] [US2] Add parser tests for optional contract YAML and invalid schema text in internal/agentdserver/infra/definition/parser_test.go
- [X] T028 [P] [US2] Add apply use-case tests for contract validation, legacy no-contract behavior, and revision digest changes in internal/agentdserver/app/agent/apply_test.go
- [X] T029 [P] [US2] Add runtime execute tests proving invalid contracted input does not create a run or call provider in internal/agentdserver/app/runtime/execute_test.go
- [X] T030 [P] [US2] Add HTTP run handler tests for `input` JSON object and `legacy_inputs` in internal/agentdserver/infra/http/run_handler_test.go
- [X] T031 [P] [US2] Add public client tests for structured run input transport in pkg/agentdclient/runs_test.go
- [X] T032 [P] [US2] Add CLI tests for `--input-json`, `--input-file`, and legacy `--input` in internal/agentd/app/execute_test.go
- [X] T033 [P] [US2] Add e2e contract input validation tests in tests/e2e/contracts_test.go

### Implementation for User Story 2

- [X] T034 [US2] Parse `contract.input` and `contract.output` from YAML front matter in internal/agentdserver/infra/definition/parser.go
- [X] T035 [US2] Validate contract schemas during definition validation in internal/agentdserver/app/agent/definition_validator.go
- [X] T036 [US2] Persist contract metadata and include contract digests in apply/revision flow in internal/agentdserver/app/agent/apply.go
- [X] T037 [US2] Include contract schemas in revision artifact digest and finalized revision metadata in internal/agentdserver/infra/runtime/revision_artifact.go
- [X] T038 [US2] Add runtime input model and pre-run validation to execute use case in internal/agentdserver/app/runtime/execute.go
- [X] T039 [US2] Update runtime manager request handling for validated JSON input in internal/agentdserver/infra/runtime/manager.go
- [X] T040 [US2] Update HTTP run request/response models for structured input in internal/agentdserver/infra/http/model/run.go
- [X] T041 [US2] Wire structured input through run handler in internal/agentdserver/infra/http/run_handler.go
- [X] T042 [US2] Add structured input fields and methods to public client types in pkg/agentdclient/types.go
- [X] T043 [US2] Send structured input through public client run methods in pkg/agentdclient/runs.go
- [X] T044 [US2] Add CLI `--input-json` and `--input-file` parsing in internal/agentd/app/execute.go
- [X] T045 [US2] Show contract metadata in inspect/list output without secrets in internal/agentd/app/inspect.go

**Checkpoint**: User Story 2 is independently functional.

---

## Phase 5: User Story 3 - Return Contracted Structured Results (Priority: P1)

**Goal**: Contracted runs produce final JSON validated against `contract.output`.

**Independent Test**: Run a contracted agent successfully and validate the persisted result and JSON CLI output against the declared output schema; force invalid provider output and verify a contract validation failure.

### Tests for User Story 3

- [X] T046 [P] [US3] Add output finalization tests for valid JSON, invalid JSON, and bounded repair attempts in internal/agentdserver/app/runtime/contract_test.go
- [X] T047 [P] [US3] Add runtime manager tests for contracted result persistence and failure status in internal/agentdserver/infra/runtime/manager_test.go
- [X] T048 [P] [US3] Add result handler tests for JSON result format and legacy text result format in internal/agentdserver/infra/http/result_handler_test.go
- [X] T049 [P] [US3] Add CLI result output tests for structured JSON values in internal/agentd/app/result_test.go

### Implementation for User Story 3

- [X] T050 [US3] Implement output finalization use case with schema validation and bounded repair attempts in internal/agentdserver/app/runtime/contract.go
- [X] T051 [US3] Add structured final output request/response flow to runtime manager in internal/agentdserver/infra/runtime/manager.go
- [X] T052 [US3] Persist run result format and contracted output errors in internal/agentdserver/infra/db/repository/run_repository.go
- [X] T053 [US3] Expose JSON result format in HTTP result model in internal/agentdserver/infra/http/model/result.go
- [X] T054 [US3] Decode JSON result values in public client result mapping in pkg/agentdclient/results.go
- [X] T055 [US3] Render contracted JSON result values in CLI output in internal/agentd/app/output.go
- [X] T056 [US3] Emit output finalization and validation events in internal/agentdserver/infra/runtime/manager.go

**Checkpoint**: User Story 3 is independently functional.

---

## Phase 6: User Story 4 - Execute AI Agents With ReAct Control (Priority: P2)

**Goal**: AI Agent execution follows a ReAct loop with daemon-owned tool calls and one-step completion when no tools are needed.

**Independent Test**: Run a multi-step tool-using fixture and a no-tool fixture; verify the first performs dependent model-selected tool calls and the second completes without tool execution.

### Tests for User Story 4

- [X] T057 [P] [US4] Add dynamic-schema ReAct tests in /Users/vitaliihonchar/workspace/go-agent/pkg/goagent/agent/dynamic_agent_test.go
- [X] T058 [P] [US4] Add go-agent external tool callback tests in /Users/vitaliihonchar/workspace/go-agent/pkg/goagent/llm/dynamic_tool_test.go
- [X] T059 [P] [US4] Add agentd ReAct adapter tests for tool-call, final, and fail decisions in internal/agentdserver/app/runtime/react_test.go
- [X] T060 [P] [US4] Add runtime manager ReAct loop tests for multi-step, one-step, tool denied, and tool limit behavior in internal/agentdserver/infra/runtime/manager_test.go
- [X] T061 [P] [US4] Add e2e ReAct behavior tests in tests/e2e/runtime_test.go

### Implementation for User Story 4

- [X] T062 [US4] Implement dynamic-schema agent runner API in /Users/vitaliihonchar/workspace/go-agent/pkg/goagent/agent/dynamic_agent.go
- [X] T063 [US4] Implement provider-neutral dynamic LLM messages and tool callbacks in /Users/vitaliihonchar/workspace/go-agent/pkg/goagent/llm/dynamic_message.go
- [X] T064 [US4] Implement dynamic JSON Schema structured output support in /Users/vitaliihonchar/workspace/go-agent/pkg/goagent/schema/dynamic_schema.go
- [X] T065 [US4] Wire agentd ReAct adapter around go-agent dynamic runner in internal/agentdserver/app/runtime/react.go
- [X] T066 [US4] Replace pre-executed tool behavior with ReAct loop orchestration in internal/agentdserver/infra/runtime/manager.go
- [X] T067 [US4] Convert declared tool execution results into ReAct observations in internal/agentdserver/infra/runtime/tool_process.go
- [X] T068 [US4] Update OpenAI provider to satisfy ReAct step and structured final output interfaces in internal/agentdserver/infra/llm/openai/provider.go
- [X] T069 [US4] Update existing runtime tests that assumed pre-run tool output in internal/agentdserver/infra/runtime/manager_test.go

**Checkpoint**: User Story 4 is independently functional.

---

## Phase 7: User Story 5 - Preserve Existing Operations and Observability (Priority: P3)

**Goal**: Operators can inspect, recover, and debug contracted and legacy runs without exposing secrets.

**Independent Test**: Run contracted and legacy agents, force provider/tool/output failures and daemon restart, then verify terminal states, masked logs, and actionable errors.

### Tests for User Story 5

- [X] T070 [P] [US5] Add runtime event tests for input validation, ReAct steps, output finalization, and failures in internal/agentdserver/app/logs/logs_test.go
- [X] T071 [P] [US5] Add secret masking tests for inspect/log/result output in internal/agentdserver/app/agent/inspect_test.go
- [X] T072 [P] [US5] Add restart recovery tests for active ReAct/provider processes in internal/agentdserver/app/runtime/recovery_test.go
- [X] T073 [P] [US5] Add e2e recovery and observability tests for contracted runs in tests/e2e/recovery_results_test.go

### Implementation for User Story 5

- [X] T074 [US5] Add stable runtime event names for contract, ReAct, provider, and finalization events in internal/agentdserver/domain/agent.go
- [X] T075 [US5] Append structured events throughout contract and ReAct execution in internal/agentdserver/infra/runtime/manager.go
- [X] T076 [US5] Mask contract/provider secret-bearing values in inspect output in internal/agentdserver/app/agent/inspect.go
- [X] T077 [US5] Preserve terminal recovery behavior for interrupted ReAct/provider runs in internal/agentdserver/app/runtime/recovery.go
- [X] T078 [US5] Include actionable error codes for contract/provider failures in internal/agentdserver/infra/http/errors.go
- [X] T079 [US5] Update run log writer/reader handling for ReAct and provider event lines in internal/agentdserver/infra/logs/run_writer.go

**Checkpoint**: User Story 5 is independently functional.

---

## Phase 8: User Story 6 - Read Logs for One Agent Run (Priority: P3)

**Goal**: `agentd logs <run-id>` returns only one run's logs and `agentd logs <agent-name>` is rejected.

**Independent Test**: Run the same agent twice, retrieve logs for each run ID, verify isolation, and verify agent-name lookup fails with actionable guidance.

### Tests for User Story 6

- [X] T080 [P] [US6] Add CLI tests for `agentd logs <run-id>` and rejected `agentd logs <agent-name>` in internal/agentd/app/logs_test.go
- [X] T081 [P] [US6] Add logs use-case tests for run-ID lookup without agent-name fallback in internal/agentdserver/app/logs/logs_test.go
- [X] T082 [P] [US6] Add HTTP handler tests for `/v1/runs/{run_id}/logs` in internal/agentdserver/infra/http/logs_handler_test.go
- [X] T083 [P] [US6] Add public client tests for run-scoped logs in pkg/agentdclient/client_test.go
- [X] T084 [P] [US6] Add e2e test for two-run log isolation in tests/e2e/logs_test.go

### Implementation for User Story 6

- [X] T085 [US6] Change CLI command contract from `logs <agent_name> --run` to `logs <run_id>` in internal/agentd/app/logs.go
- [X] T086 [US6] Implement run-ID-only logs use case lookup across runtime DBs in internal/agentdserver/app/logs/logs.go
- [X] T087 [US6] Add `/v1/runs/{run_id}/logs` route and remove successful agent-name logs route in internal/agentdserver/infra/http/server.go
- [X] T088 [US6] Update logs HTTP handler to read run ID from path and reject agent-name fallback in internal/agentdserver/infra/http/logs_handler.go
- [X] T089 [US6] Update public client Logs query to call run-scoped logs endpoint in pkg/agentdclient/logs.go
- [X] T090 [US6] Update CLI HTTP client mapping for run-scoped logs response in internal/agentd/infra/httpclient/query.go

**Checkpoint**: User Story 6 is independently functional.

---

## Phase 9: User Story 7 - Demonstrate Contracts in Example Agents (Priority: P3)

**Goal**: Every checked-in example agent includes concrete input and output JSON Schemas.

**Independent Test**: Validate every example parses, applies, and has valid `contract.input` and `contract.output`; run one manual-input example and one no-input scheduled example end to end.

### Tests for User Story 7

- [X] T091 [P] [US7] Add example catalog test requiring valid contracts on every checked-in example in internal/agentdserver/infra/definition/example_catalog_test.go
- [X] T092 [P] [US7] Add example smoke test validating contracted output schemas in tests/e2e/examples_smoke_test.go

### Implementation for User Story 7

- [X] T093 [US7] Add contracts to GitHub Trending Engineering Radar example in examples/github-trending-engineering-radar/github-trending-engineering-radar.md
- [X] T094 [US7] Add contracts to Website Snapshot Analyst example in examples/website-snapshot-analyst/website-snapshot-analyst.md
- [X] T095 [US7] Add contracts to Hacker News Builder Brief example in examples/hacker-news-builder-brief/hacker-news-builder-brief.md
- [X] T096 [US7] Add contracts to Cybersecurity Reddit Watch example in examples/cybersecurity-reddit-watch/cybersecurity-reddit-watch.md
- [X] T097 [US7] Add contracts to Reddit Customer Pain Monitor example in examples/reddit-customer-pain-monitor/reddit-customer-pain-monitor.md
- [X] T098 [US7] Add contracts to Product Hunt Launch Radar example in examples/product-hunt-launch-radar/product-hunt-launch-radar.md
- [X] T099 [US7] Add contracts to Developer Dependency Release Monitor example in examples/developer-dependency-release-monitor/developer-dependency-release-monitor.md
- [X] T100 [US7] Add contracts to AI Engineering Hiring Signal Monitor example in examples/ai-engineering-hiring-signal-monitor/ai-engineering-hiring-signal-monitor.md
- [X] T101 [US7] Update scheduled example README files with contract-aware result expectations in examples/github-trending-engineering-radar/README.md, examples/hacker-news-builder-brief/README.md, examples/cybersecurity-reddit-watch/README.md, and examples/reddit-customer-pain-monitor/README.md
- [X] T102 [US7] Update remaining example README files with contract-aware result expectations in examples/website-snapshot-analyst/README.md, examples/product-hunt-launch-radar/README.md, examples/developer-dependency-release-monitor/README.md, and examples/ai-engineering-hiring-signal-monitor/README.md
- [X] T103 [US7] Update example contract guidance in specs/004-agentd-react-contracts/contracts/agent-definition.md

**Checkpoint**: User Story 7 is independently functional.

---

## Phase 10: User Story 8 - Use Codex as an LLM Provider (Priority: P3)

**Goal**: Agents can opt into `vendor.name: codex` and use local Codex CLI as the model provider.

**Independent Test**: Run a contracted Codex-backed fixture with authenticated Codex CLI, verify no direct OpenAI API key is required, then simulate missing CLI/auth/process failures and verify actionable errors.

### Tests for User Story 8

- [X] T104 [P] [US8] Add Codex config tests for command path, model/profile, and timeout settings in internal/agentdserver/config/config_test.go
- [X] T105 [P] [US8] Add Codex provider fake-process tests for success, missing CLI, unauthenticated output, malformed events, timeout, and cancellation in internal/agentdserver/infra/llm/codex/provider_test.go
- [X] T106 [P] [US8] Add service registration tests for OpenAI and Codex providers in internal/agentdserver/service_test.go
- [X] T107 [P] [US8] Add Codex contracted-output runtime test in internal/agentdserver/infra/runtime/manager_test.go
- [X] T108 [P] [US8] Add Codex provider e2e fixture test in tests/e2e/codex_provider_test.go

### Implementation for User Story 8

- [X] T109 [US8] Add Codex provider config fields and environment loading in internal/agentdserver/config/config.go
- [X] T110 [US8] Implement non-interactive Codex CLI provider process adapter in internal/agentdserver/infra/llm/codex/provider.go
- [X] T111 [US8] Add process-group cancellation helpers for Codex provider on Linux/macOS in internal/agentdserver/infra/llm/codex/process_unix.go
- [X] T112 [US8] Add bounded stdout/stderr and JSON event parsing for Codex provider in internal/agentdserver/infra/llm/codex/provider.go
- [X] T113 [US8] Write temporary output schema and final-message files for Codex structured output in internal/agentdserver/infra/llm/codex/provider.go
- [X] T114 [US8] Register Codex provider alongside OpenAI provider in internal/agentdserver/service.go
- [X] T115 [US8] Add Codex provider fixture agent definition in tests/fixtures/codex-provider-agent.md
- [X] T116 [US8] Document Codex provider setup and failure modes in specs/004-agentd-react-contracts/quickstart.md

**Checkpoint**: User Story 8 is independently functional.

---

## Final Phase: Polish & Cross-Cutting Concerns

**Purpose**: Final verification, cleanup, and cross-feature documentation.

- [X] T117 [P] Run gofmt on changed Go files under cmd/, internal/, pkg/, tests/, and /Users/vitaliihonchar/workspace/go-agent/pkg/
- [X] T118 [P] Update generated API/client contract fixtures in internal/lib/testutil/contracts/run_result_response.json
- [X] T119 [P] Update generated API/client contract fixtures in internal/lib/testutil/contracts/run_list_response.json
- [X] T120 [P] Update generated API/client contract fixtures in internal/lib/testutil/contracts/error_response.json
- [X] T121 Run full automated verification and record any gaps in specs/004-agentd-react-contracts/quickstart.md
- [X] T122 Run manual quickstart verification for unified binary, contracts, run logs, examples, and Codex provider in specs/004-agentd-react-contracts/quickstart.md
- [X] T123 Update implementation notes for any go-agent adapter deviations in specs/004-agentd-react-contracts/research.md
- [X] T124 Final review for secret masking, Linux/macOS process cleanup, and daemon recovery in specs/004-agentd-react-contracts/plan.md

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies.
- **Foundational (Phase 2)**: Depends on Setup; blocks all user stories.
- **US1, US2, US6**: Can start after Foundational and are the best first implementation slices.
- **US3**: Depends on US2 contract parsing/persistence and runtime input/result shape.
- **US4**: Depends on Foundational provider interfaces and benefits from US2/US3 contract surfaces.
- **US5**: Depends on US2, US3, and US4 for full observability/recovery coverage.
- **US7**: Depends on US2 and US3 so examples can apply and validate output contracts.
- **US8**: Depends on Foundational provider interfaces and US3 structured output finalization.
- **Polish**: Depends on all desired stories.

### User Story Dependencies

- **US1 (P1)**: Independent after Foundational.
- **US2 (P1)**: Independent after Foundational.
- **US3 (P1)**: Depends on US2.
- **US4 (P2)**: Depends on Foundational; should integrate with US2/US3 before final validation.
- **US5 (P3)**: Depends on US2, US3, and US4 for complete behavior.
- **US6 (P3)**: Independent after Foundational, but final e2e tests benefit from US5 events.
- **US7 (P3)**: Depends on US2 and US3.
- **US8 (P3)**: Depends on Foundational and US3.

### Within Each User Story

- Write and run the story tests first; confirm they fail before implementation.
- Implement domain/model changes before use cases.
- Implement use cases before HTTP/client/CLI surfaces.
- Implement provider/process adapters before e2e verification.
- Complete each story's checkpoint before moving to lower-priority stories.

---

## Parallel Opportunities

- Setup tasks T002-T006 can run in parallel.
- Foundational tests T007, T009, T011, T013, T015, and T017 can run in parallel.
- US1 tests T019-T021 can run in parallel.
- US2 tests T027-T033 can run in parallel.
- US3 tests T046-T049 can run in parallel.
- US4 go-agent tests T057-T058 can run in parallel with agentd tests T059-T061.
- US5 tests T070-T073 can run in parallel.
- US6 tests T080-T084 can run in parallel.
- US7 example file updates T093-T100 can run in parallel after schema patterns are agreed.
- US8 tests T104-T108 can run in parallel.
- Polish fixture updates T118-T120 can run in parallel.

## Parallel Example: User Story 2

```text
Task T027: Add parser tests in internal/agentdserver/infra/definition/parser_test.go
Task T029: Add runtime execute tests in internal/agentdserver/app/runtime/execute_test.go
Task T030: Add HTTP run handler tests in internal/agentdserver/infra/http/run_handler_test.go
Task T032: Add CLI input JSON tests in internal/agentd/app/execute_test.go
```

## Parallel Example: User Story 7

```text
Task T093: Update examples/github-trending-engineering-radar/github-trending-engineering-radar.md
Task T094: Update examples/website-snapshot-analyst/website-snapshot-analyst.md
Task T095: Update examples/hacker-news-builder-brief/hacker-news-builder-brief.md
Task T096: Update examples/cybersecurity-reddit-watch/cybersecurity-reddit-watch.md
```

---

## Implementation Strategy

### MVP First

1. Complete Phase 1 and Phase 2.
2. Complete US1 for one-binary distribution.
3. Complete US2 for optional contract parsing and input validation.
4. Complete US6 for unambiguous run-scoped logs.
5. Stop and validate these slices before ReAct/provider changes.

### Incremental Delivery

1. US1: one binary starts daemon mode and preserves client commands.
2. US2: contracted input validation before execution.
3. US3: contracted output finalization.
4. US4: ReAct execution control.
5. US6: run-ID-only logs if not already shipped in the MVP slice.
6. US5: complete observability/recovery polish around the execution model.
7. US7: migrate all examples to contracts.
8. US8: add Codex provider.

### Parallel Team Strategy

After Foundational completes:
- Developer A: US1 unified binary and documentation.
- Developer B: US2 contracts/input validation.
- Developer C: US6 run-scoped logs.
- Developer D: US4 go-agent dynamic adapter work in `/Users/vitaliihonchar/workspace/go-agent`.

After US2/US3 are stable:
- Example migration (US7) can run in parallel by example directory.
- Codex provider (US8) can run in parallel with observability/recovery (US5).

## Notes

- [P] tasks touch different files and can run in parallel when their phase dependencies are met.
- [US] labels map tasks to user stories from `spec.md`.
- Tasks that touch `/Users/vitaliihonchar/workspace/go-agent` are required because the plan chooses a go-agent dynamic runner adapter before direct agentd integration.
- All tests listed in story phases should be written and observed failing before implementation.
