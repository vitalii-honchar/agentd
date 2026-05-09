# Feature Specification: Unified Agentd ReAct Contracts

**Feature Branch**: `004-agentd-react-contracts`  
**Created**: 2026-05-09  
**Status**: Draft  
**Input**: User description: "Merge agentd and agentdserver into one agentd binary with daemon mode, change AI Agent execution to a ReAct loop based on the go-agent library, and add agent input/output JSON-schema contracts."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Distribute One Agentd Binary (Priority: P1)

A local user installs only `agentd`, then uses the same executable either as the command-line client or as the long-running local daemon.

**Why this priority**: Distribution is harder when users must discover, install, and run two binaries for one local platform.

**Independent Test**: Build the project, confirm only `agentd` is required for normal local use, start daemon mode through the new flag, and run existing CLI commands against it.

**Acceptance Scenarios**:

1. **Given** a user has the new `agentd` executable, **When** they run `agentd --daemon`, **Then** the local daemon starts with the same behavior previously provided by `agentdserver`.
2. **Given** a user has the new `agentd` executable, **When** they run `agentd -d`, **Then** the local daemon starts with the same behavior as `agentd --daemon`.
3. **Given** the daemon is already running, **When** the user runs `agentd apply`, `agentd run`, `agentd result`, `agentd logs`, or other client commands, **Then** the commands keep using the daemon API without requiring a second executable.

---

### User Story 2 - Validate Agent Inputs Before Execution (Priority: P1)

An agent author declares an input contract for an AI Agent, and agentd refuses invalid run parameters before any tool or model call starts.

**Why this priority**: Contract validation prevents avoidable model calls, improves automation reliability, and gives users immediate feedback when they pass invalid parameters.

**Independent Test**: Apply an agent with a required input schema, run it with valid and invalid input objects, and verify invalid inputs fail before any run starts.

**Acceptance Scenarios**:

1. **Given** an AI Agent definition declares `contract.input` requiring a valid URL, **When** the user runs the agent with a malformed URL, **Then** agentd rejects the request with a validation error and no AI Agent execution starts.
2. **Given** an AI Agent definition declares required fields in `contract.input`, **When** the user omits one required field, **Then** agentd reports the missing field and does not invoke tools or an LLM.
3. **Given** an AI Agent definition does not declare `contract.input`, **When** the user runs it with the current supported input style, **Then** existing behavior remains available.

---

### User Story 3 - Return Contracted Structured Results (Priority: P1)

An agent author declares an output contract, and each successful contracted run returns a final JSON result that validates against that schema.

**Why this priority**: Plain-text output is hard for scripts, other agents, and applications to consume reliably.

**Independent Test**: Apply an agent with an output schema, run it successfully, and validate the stored final result and machine-readable CLI output against the declared schema.

**Acceptance Scenarios**:

1. **Given** an AI Agent definition declares `contract.output`, **When** the agent completes successfully, **Then** the final result is valid JSON matching the declared output schema.
2. **Given** the agent uses tools or multiple reasoning steps, **When** the final result is produced, **Then** the result is generated from the complete execution history rather than only the original prompt.
3. **Given** the final result cannot be made to match `contract.output`, **When** bounded recovery attempts are exhausted, **Then** the run fails with an output-contract validation error and diagnostic details.

---

### User Story 4 - Execute AI Agents With ReAct Control (Priority: P2)

An AI Agent can reason, choose a declared tool, observe the result, and continue until the task is complete, while a simple non-looping agent can complete after a single model step.

**Why this priority**: Current tool execution runs declared tools ahead of the model, which prevents the agent from choosing tools dynamically and from iterating based on observations.

**Independent Test**: Run one agent whose prompt requires multiple dependent tool calls and another whose prompt needs no tool calls; verify the first iterates and the second does not perform unnecessary tool rounds.

**Acceptance Scenarios**:

1. **Given** an AI Agent prompt requires repeated tool use, **When** the run starts, **Then** the agent follows a reason-action-observation loop until it reaches a final answer or a configured safety limit.
2. **Given** an AI Agent prompt can be answered without tools, **When** the run starts, **Then** the agent may complete without calling any tool.
3. **Given** an AI Agent asks for an undeclared tool or exceeds a tool usage limit, **When** the platform evaluates the next step, **Then** the unsafe action is denied and the run records an actionable failure.

---

### User Story 5 - Preserve Existing Operations and Observability (Priority: P3)

An operator can inspect definitions, revisions, runs, logs, and failures for both contracted and legacy agents without losing current daemon recovery and isolation behavior.

**Why this priority**: The new execution model must not make local daemon behavior harder to diagnose or less safe.

**Independent Test**: Apply and run existing examples, contracted examples, failing examples, and daemon restart scenarios; verify logs, results, statuses, and revision metadata remain useful.

**Acceptance Scenarios**:

1. **Given** an existing agent definition has no `contract`, **When** it is applied and run, **Then** it remains valid and returns the current result format.
2. **Given** an applied agent has a `contract`, **When** the user inspects the agent or revision, **Then** the input and output contract metadata is visible without exposing secret values.
3. **Given** the daemon restarts during an AI Agent run, **When** recovery completes, **Then** the run reaches a clear terminal state and persisted history remains consistent.

---

### User Story 6 - Read Logs for One Agent Run (Priority: P3)

A user debugs one execution and reads logs by run identifier, so the output contains only events for that specific run rather than a mixed stream for the whole agent.

**Why this priority**: Agent-name log lookup is ambiguous once an agent has multiple runs; users need execution logs for the exact run they are investigating.

**Independent Test**: Run the same agent at least twice, retrieve logs for each run identifier, and verify each command returns only the events for the selected run. Also verify `agentd logs <agent-name>` is no longer accepted as a successful lookup.

**Acceptance Scenarios**:

1. **Given** an agent has two completed runs, **When** the user runs `agentd logs <agent-run-id>` for the first run, **Then** the CLI returns only logs for the first run.
2. **Given** an agent has two completed runs, **When** the user runs `agentd logs <agent-run-id>` for the second run, **Then** the CLI returns only logs for the second run.
3. **Given** a user runs `agentd logs <agent-name>`, **When** the argument is an agent name rather than a run identifier, **Then** the CLI rejects the request and explains that logs are retrieved by run identifier.

---

### User Story 7 - Demonstrate Contracts in Example Agents (Priority: P3)

A new user reviews or runs the checked-in example agents and sees concrete input and output contracts for each example instead of only prose result instructions.

**Why this priority**: Examples are the main reference for agent authors, so they must demonstrate the new contract format and make structured output expectations clear.

**Independent Test**: Inspect every checked-in example agent definition, confirm it declares `contract.input` and `contract.output` with valid JSON Schemas, apply every example, and run representative examples to validate contracted results.

**Acceptance Scenarios**:

1. **Given** a checked-in example agent has no runtime parameters, **When** the example is updated, **Then** it declares an input contract that accepts an empty JSON object.
2. **Given** a checked-in example agent requires runtime parameters, **When** the example is updated, **Then** its input contract declares those parameters and validation rules.
3. **Given** a checked-in example agent describes prose output sections today, **When** the example is updated, **Then** its output contract captures the expected structured result as JSON Schema.

---

### User Story 8 - Use Codex as an LLM Provider (Priority: P3)

A user configures an AI Agent to use Codex as the model provider so agentd can run model requests through the user's existing Codex CLI setup instead of requiring direct API-key based access for every agent run.

**Why this priority**: A Codex-backed provider can reduce platform usage cost and reuse the user's existing local Codex authentication and model configuration.

**Independent Test**: Configure a test agent with the `codex` provider, run it through agentd, and verify the run completes through the local Codex CLI path with captured output, errors, cancellation, timeout, and logs.

**Acceptance Scenarios**:

1. **Given** Codex CLI is installed and authenticated for the user, **When** an AI Agent declares `vendor.name: codex`, **Then** agentd executes the model request through Codex and records the result as the agent run output.
2. **Given** Codex CLI is not installed or not authenticated, **When** an AI Agent declares `vendor.name: codex`, **Then** agentd fails the run with an actionable setup error instead of falling back silently to another provider.
3. **Given** an AI Agent declares `contract.output`, **When** it runs through the Codex provider, **Then** the final result still validates against the output contract.

### Edge Cases

- A user passes both daemon mode and a client subcommand in the same invocation.
- A user uses the misspelled long flag `--deamon` from earlier notes.
- A contracted input schema is not valid JSON Schema.
- A contracted output schema is not valid JSON Schema.
- Input is valid JSON but does not match `contract.input`.
- Input requires arrays, numbers, booleans, or nested objects rather than only string key-value pairs.
- An output schema is valid but impossible for the agent to satisfy from available evidence.
- A checked-in example has no runtime input and should validate an empty input object rather than inventing artificial parameters.
- A user requests logs with an agent name after the agent has multiple runs.
- A user requests logs with an unknown, active, completed, failed, stopped, or interrupted run identifier.
- A run identifier and an agent name have similar-looking values.
- Codex CLI is missing, exits unsuccessfully, is not logged in, asks for interactive input, emits malformed machine-readable events, or writes only partial output.
- A Codex-backed run is cancelled, stopped, times out, or leaves a child process running.
- A Codex-backed run needs structured output and the provider cannot satisfy the requested output schema.
- A ReAct loop reaches maximum iterations, maximum tool calls, timeout, cancellation, or stop request.
- A tool call requests undeclared host, filesystem, network, environment, credential, or privilege access.
- A simple agent produces a final answer on the first model call and should not be forced into additional loop steps.
- Existing applied revisions were created before contracts existed.
- The reusable go-agent controller cannot be imported directly without changes.
- Linux and macOS differ in signal handling, process lifetime, path permissions, or daemon startup behavior.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST provide a single `agentd` executable that can run both client commands and local daemon mode.
- **FR-002**: System MUST start daemon mode when `agentd` is launched with `--daemon` or `-d`.
- **FR-003**: System SHOULD accept `--deamon` as a compatibility alias for daemon mode and document `--daemon` as the canonical spelling.
- **FR-004**: Existing client commands MUST keep their current names, arguments, output modes, and exit-code behavior unless a change is explicitly covered by this feature.
- **FR-005**: System MUST make local distribution, install instructions, help text, and quickstarts describe `agentd` as the only required binary for normal local use.
- **FR-006**: AI Agent definitions MUST support an optional `contract` field with `input` and `output` subfields whose values contain JSON Schema documents.
- **FR-007**: System MUST validate declared contract schemas when an agent definition is applied, and MUST reject invalid schemas with field-specific diagnostics.
- **FR-008**: System MUST preserve existing behavior for definitions that omit `contract`; when no contract is specified, input-contract validation and output-contract finalization MUST NOT be applied.
- **FR-009**: System MUST treat the runtime input for contracted agents as a JSON object and validate it against `contract.input` before creating or starting an AI Agent execution.
- **FR-010**: System MUST reject invalid contracted input before invoking any tool, model provider, or scheduled execution side effect.
- **FR-011**: Users MUST be able to provide contracted input values that include strings, numbers, booleans, arrays, and nested objects.
- **FR-012**: System MUST generate the final output for a contracted agent as JSON that validates against `contract.output`.
- **FR-013**: System MUST create contracted final output from the complete AI Agent execution history, including model messages and tool observations that are safe to include.
- **FR-014**: System MUST fail a run with a clear output-contract validation error when valid final JSON cannot be produced after bounded recovery attempts.
- **FR-015**: AI Agent execution MUST follow a ReAct-style loop in which the agent can request declared tools, observe results, and continue until done or stopped.
- **FR-016**: AI Agent execution MUST also support one-iteration completion when the prompt and model response indicate no further tool use is needed.
- **FR-017**: Declared tools MUST be invoked only when selected by the AI Agent during execution, not automatically pre-run before the first model response.
- **FR-018**: System MUST enforce tool declaration, tool usage limits, maximum loop iterations, cancellation, timeout, stop, filesystem, network, environment, credential, and privilege policies during ReAct execution.
- **FR-019**: The ReAct execution controller SHOULD reuse the existing go-agent ReAct behavior when it satisfies agentd compatibility requirements for runtime JSON schemas, daemon-owned tools, cancellation, logging, provider integration, and structured final output.
- **FR-020**: If direct go-agent reuse does not satisfy those requirements, the implementation MUST either adapt the library first or preserve equivalent ReAct behavior inside agentd with a documented compatibility reason.
- **FR-021**: Run logs and daemon logs MUST include observable events for input validation, model requests, tool requests, tool observations, loop termination, output finalization, validation failures, run completion, and run failure.
- **FR-022**: Logs and inspect output MUST mask secrets and avoid exposing undeclared environment values.
- **FR-023**: Applied immutable revisions MUST include the resolved contract metadata so an explicit revision run uses the same input and output rules even if the source definition later changes.
- **FR-024**: A contract change MUST create or select a distinct applied revision according to the existing immutable-revision rules.
- **FR-025**: Scheduled agents with `contract.input` MUST have a valid default or scheduled input source before they can run automatically.
- **FR-026**: Result retrieval and machine-readable output MUST preserve stable fields for run identifiers, statuses, errors, and final results for both contracted and legacy runs.
- **FR-027**: System MUST define Linux and macOS behavior for daemon startup, signal handling, process cleanup, file permissions, and recovery.
- **FR-028**: System MUST document and test migration behavior for existing users who previously ran `agentdserver` directly.
- **FR-029**: All checked-in example agent definitions MUST be updated to include `contract.input` and `contract.output` with valid JSON Schemas.
- **FR-030**: Example agents with no runtime parameters MUST declare an input schema that accepts an empty JSON object and does not require artificial parameters.
- **FR-031**: Example agents with runtime parameters MUST declare input schemas that match the parameters users pass through the CLI or client API.
- **FR-032**: Example output schemas MUST be specific to each example's expected result and SHOULD replace prose-only result instructions with structured fields that scripts can consume.
- **FR-033**: Example documentation MUST show the contract-aware input and output shape where that helps users understand how to run or consume the example.
- **FR-034**: System MUST provide `agentd logs <agent-run-id>` to retrieve logs for one specific agent run.
- **FR-035**: System MUST disable successful `agentd logs <agent-name>` lookup and return an actionable error telling the user to provide a run identifier.
- **FR-036**: Run-scoped log output MUST include only events belonging to the selected run identifier.
- **FR-037**: Run-scoped log output MUST work for completed, failed, stopped, interrupted, and currently active runs whenever logs have been recorded for that run.
- **FR-038**: Run, result, and listing commands MUST expose run identifiers clearly enough that users can copy them into `agentd logs <agent-run-id>`.
- **FR-039**: Unknown run identifiers MUST produce a distinct not-found error and MUST NOT fall back to agent-name log lookup.
- **FR-040**: System MUST support `codex` as an additional AI Agent vendor/provider name.
- **FR-041**: The Codex provider MUST run model requests through the installed Codex CLI in non-interactive mode by default, reusing the user's existing Codex authentication and configuration.
- **FR-042**: The Codex provider MUST NOT require extracting, storing, or depending on undocumented Codex developer tokens for normal operation.
- **FR-043**: The Codex provider MUST capture final output, provider errors, exit state, cancellation, timeout, and relevant diagnostic events in the same run result and log surfaces as other providers.
- **FR-044**: The Codex provider MUST support contracted final output for agents with `contract.output`.
- **FR-045**: The Codex provider MUST fail with actionable setup or execution errors when the Codex CLI is missing, unauthenticated, incompatible, or unable to produce a usable final response.
- **FR-046**: The Codex provider MUST be bounded by the same run cancellation, stop, timeout, isolation, logging, and secret-masking policies as other model providers.

### Key Entities

- **Agent Definition**: Markdown-backed AI Agent configuration that may include schedule, vendor, tools, access rules, prompt, and optional contract metadata.
- **Agent Contract**: Optional input and output schema pair associated with an AI Agent definition and stored in immutable revisions.
- **LLM Provider**: A selectable model execution backend used by an AI Agent run, such as direct API access or a local CLI-backed provider.
- **Codex Provider**: The LLM Provider that executes model requests through the local Codex CLI and maps CLI output, errors, and lifecycle events into agentd run results.
- **Input Contract**: JSON Schema used to validate user-provided runtime parameters before an AI Agent execution starts.
- **Output Contract**: JSON Schema used to validate the final structured result from an AI Agent execution.
- **AI Agent Run**: One execution attempt, including input validation outcome, loop history, tool observations, finalization outcome, status, result, errors, logs, and revision reference.
- **Agent Run Log**: The ordered diagnostic events and messages recorded for exactly one AI Agent Run and retrieved by run identifier.
- **ReAct Step**: One model-driven iteration containing the agent's next action decision, optional tool request, observation, and termination decision.
- **Tool Invocation**: A declared custom or host tool call requested by the AI Agent during the ReAct loop and governed by existing isolation policy.
- **Example Agent**: A checked-in reference agent definition that demonstrates expected contract syntax, validation behavior, and structured result consumption.
- **Unified Agentd Runtime**: The single executable mode that either serves daemon behavior or dispatches client commands.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can install one local executable and complete daemon startup, apply, run, result lookup, and log lookup without installing or invoking `agentdserver`.
- **SC-002**: 100% of invalid contracted input cases in automated tests are rejected before any tool or model provider is invoked.
- **SC-003**: 100% of successful contracted-run tests produce final JSON that validates against the declared output contract.
- **SC-004**: A multi-step tool-using test agent performs at least two dependent tool calls chosen during execution, proving tools are not merely pre-run.
- **SC-005**: A non-tool contracted test agent completes without any tool invocation in 100% of relevant tests.
- **SC-006**: 100% of legacy agent definitions outside the migrated examples continue to apply and run successfully or fail with the same user-visible category of error as before when they omit `contract`.
- **SC-007**: Contract validation errors identify the invalid field or schema path well enough that a user can correct common input mistakes in under 30 seconds.
- **SC-008**: Daemon stop, cancellation, timeout, and restart tests leave no active run stuck outside a terminal state.
- **SC-009**: Distribution documentation and quickstarts contain no required `agentdserver` invocation for normal local usage.
- **SC-010**: Inspect, result, and log commands show enough information to diagnose contracted output failures without exposing secret environment values.
- **SC-011**: 100% of checked-in example agent definitions include valid input and output contracts.
- **SC-012**: 100% of checked-in example agent definitions apply successfully after contract migration.
- **SC-013**: At least one manual-input example and one scheduled no-input example are run end-to-end and produce final output that validates against their declared output contracts.
- **SC-014**: 100% of run-scoped log tests return only events for the requested run identifier.
- **SC-015**: 100% of `agentd logs <agent-name>` tests fail with an actionable message directing users to provide a run identifier.
- **SC-016**: A Codex-backed test agent completes successfully without direct API-key provider configuration when Codex CLI is installed and authenticated.
- **SC-017**: 100% of Codex provider setup-failure tests produce actionable errors for missing CLI, missing authentication, or incompatible CLI behavior.
- **SC-018**: 100% of Codex-backed contracted-output tests produce final JSON that validates against the declared output contract or fail with an output-contract validation error.

## Assumptions

- The canonical daemon flag is `--daemon`; `--deamon` is treated as a compatibility alias because it appeared in the original request.
- Existing legacy agent definitions remain supported when they do not declare `contract`, and contract behavior is opt-in for any definition outside the updated examples.
- Checked-in examples should all opt in to contracts so they serve as current authoring references.
- Run identifiers are the only supported selector for `agentd logs` after this feature; users can find them from run creation, run listing, or result lookup output.
- The Codex provider should prefer managed non-interactive Codex CLI process execution over private token extraction because the CLI exposes a local automation surface and token extraction would depend on undocumented behavior.
- The Codex provider is opt-in through agent definition vendor configuration; OpenAI direct API access remains available for agents that explicitly choose it.
- The preferred contracted-output design is a dedicated finalization step that receives the AI Agent history and declared output schema, instead of forcing every intermediate model response to match the final schema.
- It makes sense to reuse go-agent's ReAct behavior conceptually because it already provides looping, tool limits, history, and structured-output finalization.
- Direct go-agent import is not assumed to be ready without validation because the current library is generic/static-schema oriented, uses a different OpenAI SDK surface than agentd, and does not yet expose agentd's daemon-owned tool, revision, logging, and dynamic-schema needs as first-class runtime contracts.
- Planning may choose either to adapt go-agent into a reusable dependency or to port the necessary behavior while preserving go-agent compatibility goals; the user-visible requirement is the ReAct and contract behavior described above.
- Existing daemon storage, immutable revisions, result retrieval, and run logging remain the source of truth for agentd operations.
