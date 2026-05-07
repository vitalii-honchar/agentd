# Feature Specification: Agent Definition Runtime

**Feature Branch**: `001-agent-definition-runtime`
**Created**: 2026-05-07
**Status**: Draft
**Input**: User description: "Build an agentd Docker-like environment running as a
system daemon. A lightweight CLI can execute, stop, inspect, and get logs for AI
Agents. AI Agent definitions are plain Markdown files with properties for name,
schedule, vendor, model, tools, MCP servers, and the exact agent prompt. Users
apply definitions with `agentd apply <path_to_file>`; the daemon persists and
schedules execution. Users can also force execution immediately with
`agentd execute <agent_name>`, including agents whose schedule is manual-only.
The service records system logs for daemon activity, and each agent execution
has isolated logs available through `agentd logs <agent_name>`. The runtime must
run many Agents concurrently by design, with each Agent Run isolated from other
runs like a lightweight container workload. It must stay lightweight for
developer laptops, avoid heavy databases or external cloud dependencies except
LLM vendors, and use an Agent-as-Code approach inspired by OpenClaw without
Telegram management."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Apply Agent Definition (Priority: P1)

A developer or product manager creates a Markdown AI Agent definition containing
the agent name, schedule, LLM vendor/model, allowed tools, MCP servers, and the
agent prompt, then applies it to the local agentd daemon.

**Why this priority**: Applying a definition is the core Agent-as-Code workflow;
without it, users cannot register or update agents.

**Independent Test**: Create a valid Markdown definition, run
`agentd apply <path_to_file>`, and verify the daemon records exactly one active
agent with the expected name, configuration summary, and next scheduled run.

**Acceptance Scenarios**:

1. **Given** no agent exists with the definition name, **When** the user applies
   a valid Markdown definition, **Then** the daemon registers the agent and
   reports that it was created.
2. **Given** an agent already exists with the definition name, **When** the user
   applies a changed definition with the same name, **Then** the daemon updates
   the existing agent rather than creating a duplicate.
3. **Given** a definition is missing required properties, **When** the user
   applies it, **Then** the CLI reports validation errors and the daemon leaves
   the previous agent state unchanged.

---

### User Story 2 - Schedule and Execute Agents (Priority: P2)

A user defines whether AI Agents run by schedule or manual execution, applies
their definitions, and expects the daemon service to execute multiple Agents
concurrently when their schedules or explicit execution requests overlap.

**Why this priority**: Scheduled and manual execution are the primary runtime
behaviors after definitions can be applied.

**Independent Test**: Apply multiple definitions with overlapping near-future
schedules and one definition with manual schedule mode. Verify scheduled agents
run concurrently when due, then run `agentd execute <agent_name>` for the manual
agent and verify each Agent Run is recorded with separate status, timestamps,
prompt, vendor/model selection, tool/MCP access summary, and isolated logs.

**Acceptance Scenarios**:

1. **Given** an Agent has an enabled schedule, **When** the schedule
   becomes due, **Then** the daemon starts one agent run using the applied
   definition.
2. **Given** an Agent has manual schedule mode, **When** the user runs
   `agentd execute <agent_name>`, **Then** the daemon starts one agent run
   immediately using the applied definition.
3. **Given** multiple Agents become due or are manually executed at the same
   time, **When** the daemon starts their runs, **Then** each Agent Run proceeds
   concurrently with isolated state, logs, and declared host access.
4. **Given** the daemon restarts before a scheduled run, **When** it starts
   again, **Then** it restores applied definitions and schedules future due runs.
5. **Given** a scheduled or manually executed run fails because a vendor request
   cannot complete,
   **When** the failure is recorded, **Then** the user can inspect the failed run
   and see an actionable failure reason.

---

### User Story 3 - Operate Agents from CLI (Priority: P3)

A user manages Agents from the CLI by listing or inspecting definitions,
executing an agent manually, stopping a running agent, and viewing logs for an
agent or a specific run.

**Why this priority**: Operators need Docker-like visibility and control to use
the daemon confidently on a laptop or workstation.

**Independent Test**: Apply an agent, execute it manually with
`agentd execute <agent_name>`, inspect it, stop it, and retrieve isolated agent
logs with `agentd logs <agent_name>`; each command returns the expected current
state without requiring external management tools.

**Acceptance Scenarios**:

1. **Given** an agent is applied, **When** the user inspects it, **Then** the CLI
   shows definition metadata, current state, last run, next scheduled run, and
   recent errors if any.
2. **Given** an agent is idle, **When** the user runs
   `agentd execute <agent_name>`, **Then** the daemon creates a run immediately
   and reports the run identifier.
3. **Given** an agent run is active, **When** the user stops it, **Then** the
   daemon requests cancellation, records the final state, and exposes the
   outcome in inspect and logs.
4. **Given** an agent has one or more runs, **When** the user runs
   `agentd logs <agent_name>`, **Then** the CLI shows logs for that agent without
   mixing logs from other agents.

---

### Edge Cases

- Two different Markdown files declare the same unique agent name.
- A schedule is malformed, unsupported, disabled, or would run too frequently
  for a laptop-friendly runtime.
- An Agent Definition uses manual schedule mode and therefore has no next
  automatic scheduled run.
- A user runs `agentd execute <agent_name>` for a missing, disabled, invalid, or
  already-running agent.
- The definition file is removed or changed after being applied.
- The daemon restarts while an apply, manual start, scheduled run, stop, or log
  retrieval is in progress.
- Linux and macOS expose different process, signal, filesystem, or credential
  behaviors for agent execution.
- A vendor/model is unknown, unavailable, rate-limited, or missing credentials.
- A listed tool or MCP server is unavailable at run time.
- Logs grow large enough to affect local disk usage.
- Multiple agents run at the same time and produce logs concurrently.
- One concurrent Agent Run fails, stalls, or is stopped while other Agent Runs
  continue.
- Concurrent Agent Runs request overlapping tools, MCP servers, files, network
  access, or credentials.
- The user requests logs for an agent with no runs, an unknown agent, or a run
  whose logs were already pruned by retention policy.
- A prompt contains Markdown content that could be confused with definition
  metadata.
- Host access that is not listed in the definition is requested during a run.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST allow users to define an AI Agent in a plain Markdown
  file with required metadata properties and prompt content.
- **FR-002**: System MUST require every Agent Definition applied to the daemon
  to include a unique agent name within the current local agentd system.
- **FR-003**: System MUST support metadata for schedule, LLM vendor, LLM model,
  allowed tools, MCP servers, and enabled/disabled state.
- **FR-004**: System MUST support cron-compatible schedules initially while
  preserving a user-visible schedule field that can support manual execution
  mode and additional schedule forms later.
- **FR-005**: System MUST validate definitions before applying them and report
  all detected validation errors without changing the active agent state.
- **FR-006**: System MUST let users apply a definition with
  `agentd apply <path_to_file>` and receive a clear created, updated, unchanged,
  or rejected outcome.
- **FR-007**: System MUST persist applied Agent Definitions, Agent state, run
  history, and Agent schedule fields locally so they survive daemon restarts.
- **FR-008**: System MUST execute enabled agents when their schedules become due.
- **FR-009**: System MUST prevent duplicate scheduled executions for the same
  due time after daemon restart or repeated apply operations.
- **FR-010**: System MUST allow users to manually execute an Agent with
  `agentd execute <agent_name>`.
- **FR-011**: System MUST run multiple Agent Runs concurrently when schedules or
  manual execution requests overlap.
- **FR-012**: System MUST isolate each concurrent Agent Run from other Agent
  Runs for runtime state, logs, declared host access, and failure handling.
- **FR-013**: System MUST allow users to stop a running agent and expose whether
  the run stopped, completed first, or failed to stop.
- **FR-014**: System MUST allow users to inspect an Agent and view
  definition metadata, state, last run, next scheduled run, and recent failures.
- **FR-015**: System MUST allow users to retrieve isolated logs for an Agent
  with `agentd logs <agent_name>`.
- **FR-016**: System MUST keep logs from each Agent Run isolated from other
  Agent Runs while preserving the Agent name and run identity for lookup.
- **FR-017**: System MUST record service-level logs for daemon activity,
  including apply, scheduling, start, stop, completion, failure, and daemon
  restart recovery.
- **FR-018**: System MUST define required filesystem, network, environment,
  credential, and privilege access for any agent execution behavior.
- **FR-019**: System MUST deny host access that is not declared by the applied
  definition or granted by the daemon policy.
- **FR-020**: System MUST define Linux and macOS behavior for daemon lifecycle,
  process control, filesystem access, credentials, cancellation, and log access.
- **FR-021**: System MUST define restart, cancellation, cleanup, and recovery
  behavior for work that can outlive a single CLI request.
- **FR-022**: System MUST operate without requiring any external cloud service
  except the LLM vendors selected by applied definitions.
- **FR-023**: System MUST keep local runtime storage lightweight and suitable for
  developer laptops without requiring a separate database service.
- **FR-024**: System MUST avoid requiring new abstractions, dependencies, or
  optimizations unless they are justified by current requirements or measured
  bottlenecks.

### Key Entities *(include if feature involves data)*

- **Agent Definition**: A user-authored Markdown document that declares a unique
  name, schedule mode, vendor/model selection, tools, MCP servers, enabled state,
  and exact prompt content.
- **Agent**: The daemon's active record created from an Agent Definition,
  including current revision, validation status, operational state, schedule
  mode, next scheduled run when applicable, and future schedule metadata.
- **Agent Run**: One execution attempt for an Agent, including trigger source,
  timestamps, status, output summary, errors, isolation boundary, and isolated
  log references.
- **Tool Permission**: A declared capability that an Agent Run may use,
  including local tools and MCP server access.
- **Runtime Event**: A structured service-level record of daemon, apply,
  scheduling, lifecycle, policy, and failure activity.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A first-time user can create a minimal valid Markdown agent
  definition, apply it, inspect it, and view its next scheduled run in under
  10 minutes using local documentation.
- **SC-002**: Applying an unchanged definition reports an unchanged outcome and
  does not create duplicate agents or duplicate scheduled runs in 100% of test
  cases.
- **SC-003**: After daemon restart, 100% of applied enabled agents are restored
  with their latest definition state and future schedule information.
- **SC-004**: Users can identify the reason for a rejected definition or failed
  run from CLI output, service logs, or isolated agent logs without opening
  daemon internals in 95% of
  validation and runtime failure cases.
- **SC-005**: On a typical developer laptop, the daemon remains usable for at
  least 25 Agents without requiring an external storage or management
  service.
- **SC-006**: Manual execute, stop, inspect, and logs commands complete with a
  clear user-visible outcome for idle, running, completed, failed, disabled, and
  missing agent states.
- **SC-007**: An agent configured for manual schedule mode never runs
  automatically and starts only after an explicit user execution request in 100%
  of schedule behavior tests.
- **SC-008**: When at least five Agents run concurrently, each Agent Run remains
  independently inspectable, stoppable, and logged without mixing state or logs
  in 100% of concurrency isolation tests.
- **SC-009**: When one concurrent Agent Run fails or is stopped, other active
  Agent Runs continue to a normal terminal state in 100% of failure isolation
  tests.

## Assumptions

- Markdown definition metadata uses a clear properties section at the top of the
  file, and the remaining Markdown body can hold the exact agent prompt.
- Cron-compatible expressions and manual-only execution are the first supported
  schedule modes; later formats can be added without changing the definition's
  core identity model.
- The first release targets a single-user local daemon on a developer laptop or
  workstation, with multi-user authorization out of scope.
- The daemon stores runtime metadata locally and does not require a separately
  managed storage service.
- Concurrent Agent execution is a default runtime behavior, and concurrency
  limits can be introduced as local policy without changing the Agent Definition
  model.
- LLM vendor credentials are supplied by the local user or host environment and
  are not embedded in plain Markdown definitions by default.
- Telegram or chat-based agent management is out of scope; the management model
  is CLI plus Agent-as-Code definitions.
