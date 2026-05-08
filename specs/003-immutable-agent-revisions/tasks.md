# Tasks: Immutable Agent Revisions

**Input**: Design documents from `/specs/003-immutable-agent-revisions/`  
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: Included because this feature changes apply semantics, persisted
state, local filesystem artifacts, run selection, tool execution, environment
handling, and logs.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing.

## Phase 1: Setup

- [x] T001 Create settings migration for revision tables in internal/agentdserver/infra/db/migrations/settings/003_agent_revisions.sql
- [x] T002 Add revision, artifact, environment, execution workdir, and tool log domain types in internal/agentdserver/domain/agent.go
- [x] T003 Add custom_tool and host_tool ToolKind constants in internal/agentdserver/domain/agent.go
- [x] T004 Extend agent repository ports for revision save/list/lookup/latest/corruption operations in internal/agentdserver/app/ports.go
- [x] T005 Add revision artifact service file in internal/agentdserver/infra/runtime/revision_artifact.go
- [x] T006 Add env parsing helper file in internal/agentdserver/infra/runtime/env_file.go

## Phase 2: Foundational

- [x] T007 [P] Add migration tests for revision metadata and uniqueness constraints in internal/agentdserver/infra/db/migrations_test.go
- [x] T008 [P] Add repository tests for revision create/list/find/latest in internal/agentdserver/infra/db/repository/agent_repository_test.go
- [x] T009 [P] Add repository tests for revision environment, artifact files, and corruption marking in internal/agentdserver/infra/db/repository/agent_repository_test.go
- [x] T010 Implement revision persistence methods in internal/agentdserver/infra/db/repository/agent_repository.go
- [x] T011 [P] Add parser tests for custom_tool, host_tool, legacy local_tool, environment.variables, and environment.files in internal/agentdserver/infra/definition/parser_test.go
- [x] T012 [P] Add validator tests for custom_tool copy paths and host_tool host commands in internal/agentdserver/app/agent/definition_validator_test.go
- [x] T013 Update definition parser for custom_tool, host_tool, legacy local_tool, and environment metadata in internal/agentdserver/infra/definition/parser.go
- [x] T014 Update definition validator for path-bearing metadata and tool kind validation in internal/agentdserver/app/agent/definition_validator.go
- [x] T015 [P] Add artifact copy tests for custom_tool scripts, declared read files, executable mode, and checksums in internal/agentdserver/infra/runtime/revision_artifact_test.go
- [x] T016 [P] Add artifact validation tests for missing files, symlink policy, and path escape rejection in internal/agentdserver/infra/runtime/revision_artifact_test.go
- [x] T017 Implement artifact staging, copy, manifest, checksum, command rewrite, and finalize behavior in internal/agentdserver/infra/runtime/revision_artifact.go
- [x] T018 [P] Add .env parsing tests for comments, quotes, duplicate keys, invalid lines, and precedence in internal/agentdserver/infra/runtime/env_file_test.go
- [x] T019 Implement .env parsing, environment merge precedence, and masking helpers in internal/agentdserver/infra/runtime/env_file.go
- [x] T020 Update Cybersecurity Reddit Watch definition from local_tool to custom_tool in examples/cybersecurity-reddit-watch/cybersecurity-reddit-watch.md
- [x] T021 Update Hacker News Builder Brief definition from local_tool to custom_tool in examples/hacker-news-builder-brief/hacker-news-builder-brief.md
- [x] T022 Update Reddit Customer Pain Monitor definition from local_tool to custom_tool in examples/reddit-customer-pain-monitor/reddit-customer-pain-monitor.md
- [x] T023 Update Product Hunt Launch Radar definition from local_tool to custom_tool in examples/product-hunt-launch-radar/product-hunt-launch-radar.md
- [x] T024 Update GitHub Trending Engineering Radar definition from local_tool to custom_tool in examples/github-trending-engineering-radar/github-trending-engineering-radar.md
- [x] T025 Update Developer Dependency Release Monitor definition from local_tool to custom_tool in examples/developer-dependency-release-monitor/developer-dependency-release-monitor.md
- [x] T026 Update AI Engineering Hiring Signal Monitor definition from local_tool to custom_tool in examples/ai-engineering-hiring-signal-monitor/ai-engineering-hiring-signal-monitor.md
- [x] T027 Update Website Snapshot Analyst definition from local_tool to custom_tool in examples/website-snapshot-analyst/website-snapshot-analyst.md
- [x] T028 Add host_tool parser fixture for tests in internal/lib/testutil/testutil.go

## Phase 3: User Story 1 - Apply Creates Self-Contained Revision (P1)

- [x] T029 [P] [US1] Add apply use case tests for new revision creation and unchanged idempotency in internal/agentdserver/app/agent/apply_test.go
- [x] T030 [P] [US1] Add apply use case tests for source prompt mutation, source tool mutation, and source deletion isolation in internal/agentdserver/app/agent/apply_test.go
- [x] T031 [P] [US1] Add apply use case tests for environment.variables and environment.files capture in internal/agentdserver/app/agent/apply_test.go
- [x] T032 [US1] Update apply use case to create or reuse immutable revisions in internal/agentdserver/app/agent/apply.go
- [x] T033 [US1] Return revision ID, artifact path, revision status, and unchanged reuse outcome from apply in internal/agentdserver/app/agent/apply.go
- [x] T034 [US1] Persist latest finalized revision metadata on agent save in internal/agentdserver/infra/db/repository/agent_repository.go
- [x] T035 [US1] Add HTTP apply response tests for revision ID, artifact path, revision status, and unchanged reuse in internal/agentdserver/infra/http/apply_handler_test.go
- [x] T036 [US1] Update apply HTTP response mapping for revision metadata in internal/agentdserver/infra/http/apply_handler.go
- [x] T037 [US1] Update apply CLI output for revision metadata in internal/agentd/app/apply.go

## Phase 4: User Story 2 - Run a Specific Revision (P1)

- [x] T038 [P] [US2] Add execute use case tests for latest revision resolution and explicit revision resolution in internal/agentdserver/app/runtime/execute_test.go
- [x] T039 [P] [US2] Add execute use case tests for missing revision, corrupt revision, and run record revision ID in internal/agentdserver/app/runtime/execute_test.go
- [x] T040 [US2] Extend execute request model and use case with revision selector and resolved revision artifact in internal/agentdserver/app/runtime/execute.go
- [x] T041 [US2] Add agentd run command or compatibility alias in internal/agentd/app/execute.go
- [x] T042 [US2] Register agentd run command or compatibility alias in internal/agentd/app/root.go
- [x] T043 [US2] Add HTTP run handler tests for latest and explicit revision execution in internal/agentdserver/infra/http/run_handler_test.go
- [x] T044 [US2] Update run handler request mapping for latest and explicit revision execution in internal/agentdserver/infra/http/run_handler.go
- [x] T045 [US2] Update public client run request types for revision selectors in pkg/agentdclient/runs.go

## Phase 5: User Story 3 - Execute Custom and Host Tools (P2)

- [x] T046 [P] [US3] Add runtime manager tests proving custom_tool commands execute copied artifact files from execution workdirs in internal/agentdserver/infra/runtime/manager_test.go
- [x] T047 [P] [US3] Add runtime manager tests proving host_tool commands invoke host-installed executables without artifact copying in internal/agentdserver/infra/runtime/manager_test.go
- [x] T048 [P] [US3] Add runtime manager tests for missing host_tool executable errors in internal/agentdserver/infra/runtime/manager_test.go
- [x] T049 [US3] Update runtime manager command resolution for custom_tool artifact paths in internal/agentdserver/infra/runtime/manager.go
- [x] T050 [US3] Update runtime manager command resolution for host_tool host executables in internal/agentdserver/infra/runtime/manager.go
- [x] T051 [US3] Validate finalized revision artifacts before custom_tool process start in internal/agentdserver/infra/runtime/manager.go
- [x] T052 [US3] Validate host_tool executables before process start in internal/agentdserver/infra/runtime/manager.go
- [ ] T053 [US3] Build tool process environments from revision env and tool-specific env in internal/agentdserver/infra/runtime/tool_process.go
- [ ] T054 [US3] Stop inheriting undeclared host environment secrets in internal/agentdserver/infra/runtime/tool_process.go

## Phase 6: User Story 4 - Observe Tool Output in Agent and Server Logs (P2)

- [ ] T055 [P] [US4] Add AI Agent log tests for tool stdout, stderr, result summaries, and exit code in internal/agentdserver/infra/runtime/manager_test.go
- [ ] T056 [P] [US4] Add AI Agent log tests for tool timeout state and error message in internal/agentdserver/infra/runtime/manager_test.go
- [ ] T057 [US4] Include tool stdout, stderr, result summaries, exit code, timeout state, and error message in runtime action logs in internal/agentdserver/infra/runtime/manager.go
- [ ] T058 [P] [US4] Add agentdserver structured log tests for stdout, stderr, result summaries, exit state, timeout state, and errors in internal/agentdserver/infra/runtime/manager_test.go
- [ ] T059 [US4] Emit tool execution evidence to agentdserver structured logs in internal/agentdserver/infra/runtime/manager.go
- [ ] T060 [US4] Update logs CLI formatting for tool result, exit, timeout, and error fields in internal/agentd/app/logs.go

## Phase 7: User Story 5 - Audit and Retain Revisions (P3)

- [ ] T061 [P] [US5] Add revision list use case tests for revision metadata and latest marker in internal/agentdserver/app/agent/inspect_test.go
- [ ] T062 [P] [US5] Add revision inspect use case tests for tool kinds, rewritten commands, host commands, copied files, and masked env in internal/agentdserver/app/agent/inspect_test.go
- [ ] T063 [US5] Implement revision list and inspect use cases in internal/agentdserver/app/agent/revision.go
- [ ] T064 [P] [US5] Add HTTP contract tests for revision list and inspect in internal/agentdserver/infra/http/inspect_handler_test.go
- [ ] T065 [US5] Add revision list and inspect HTTP routes in internal/agentdserver/infra/http/server.go
- [ ] T066 [US5] Add revision list and inspect HTTP handlers in internal/agentdserver/infra/http/inspect_handler.go
- [ ] T067 [US5] Add CLI revision list output in internal/agentd/app/list.go
- [ ] T068 [US5] Add CLI revision inspect output in internal/agentd/app/inspect.go

## Phase 8: Recovery, Docs, and Verification

- [ ] T069 Add daemon startup recovery for pending/corrupt revision artifacts in internal/agentdserver/service.go
- [ ] T070 Add daemon startup cleanup for stale execution directories in internal/agentdserver/service.go
- [ ] T071 [P] Add e2e test applying an agent, mutating source prompt/tool, deleting source folder, and running explicit revision in tests/e2e/apply_test.go
- [ ] T072 [P] Add e2e test proving declared .env values survive source deletion and undeclared host env secrets are absent in tests/e2e/runtime_test.go
- [ ] T073 Update OpenAPI revision tool kinds and revision fields in specs/003-immutable-agent-revisions/contracts/openapi.yaml
- [ ] T074 Update CLI contract and agent definition contract docs in specs/003-immutable-agent-revisions/contracts/cli.md and specs/003-immutable-agent-revisions/contracts/agent-definition.md
- [ ] T075 Update example README docs for custom_tool terminology in examples/github-trending-engineering-radar/README.md
- [ ] T076 Run `go test ./...` and record any failures in specs/003-immutable-agent-revisions/quickstart.md
- [ ] T077 Manually verify with Codex by applying and running GitHub Trending Engineering Radar and record results in specs/003-immutable-agent-revisions/quickstart.md
- [ ] T078 Manually launch agentdserver, use agentd to apply and execute examples/github-trending-engineering-radar/github-trending-engineering-radar.md, verify the run result is present, verify tool stdout/stderr logs are present, verify agentdserver logs contain tool execution evidence, and record that the flow works without errors in specs/003-immutable-agent-revisions/quickstart.md

## Dependencies

- Phase 1 must complete before all other phases.
- Phase 2 blocks all user stories because revisions, parser rules, artifact copy, and environment capture are shared.
- US1 and US2 are the MVP path and must complete before US3 can execute revision-owned tools.
- US4 depends on US3 tool execution evidence.
- US5 depends on persisted revision metadata from US1 and tool kind metadata from US3.
- Phase 8 follows all user stories.

## Parallel Execution Examples

- Phase 2: T007, T008, T011, T012, T015, T016, and T018 can be written in parallel after T001-T006.
- US1: T029, T030, T031, and T035 can be written in parallel after T010-T019.
- US2: T038, T039, and T043 can be written in parallel after US1 persistence is available.
- US3: T046, T047, and T048 can be written in parallel because custom_tool, host_tool, and missing executable cases are independent.
- US4: T055, T056, and T058 can be written in parallel because AI Agent logs and agentdserver logs are separate assertions.
- US5: T061, T062, and T064 can be written in parallel before route/CLI implementation.

## Implementation Strategy

Implement MVP first: T001-T045 plus T071 should prove that apply creates a self-contained immutable revision and explicit revision execution works after source mutation/deletion. Then complete tool kind separation with T046-T054, observability with T055-T060, revision inspection with T061-T068, and final verification with T069-T078.
