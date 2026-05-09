# Feature Specification: Immutable Agent Revisions

**Feature Branch**: `002-agent-examples-results`  
**Created**: 2026-05-08  
**Status**: Draft  
**Input**: User description: "After running apply the agentd should resolve all paths declared inside the AI Agent definition and create immutable artifact in the settings database which will contain the AI Agent prompt, all specified tools with script tools copied, and environment vars used by tools. Treat AI Agent Artifacts as Docker images. When we run apply then we create immutable artifact which user can run by specifying agentd run <agent-name>:<revision>. Name AI Agent Artifact as revision and assign uuid on creation. It must be immutable and self contained so if the definition markdown folder and scripts are removed after apply, agentd can still execute the agent. Store immutable revisions in data/work/<agent-name>/<revision_id>; do not store blob data in the settings database if files can be copied to the revision folder. Tool execution for local tools should run from this revision folder, so tool commands may need rewriting. The AI Agent shares the same filesystem as the host OS, but each execution working directory is data/work/<agent_name>/executions/<execution_id>; runs use files from the immutable artifact. Agent definition metadata must explicitly declare which fields contain filesystem paths to resolve and copy. Agent definition metadata should also include environment values or .env file paths that become part of the artifact and are supplied during execution. The spec must cover immutable prompt/tools after source edits or deletion, tool stdout/stderr/results in AI Agent logs, tool stdout/stderr/results/errors in agentdserver logs, and explicit tool kinds for agent-provided tools versus host-installed system tools. Use `custom_tool` for agent-provided implementations copied into the artifact and `host_tool` for host-installed executables; `host_tool` is preferred over `process_tool` because both kinds execute as processes, while only host tools depend on the host OS PATH or allowed absolute executables."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Apply Creates Self-Contained Revision (Priority: P1)

A user applies an AI Agent definition that references prompts, local scripts,
source files, environment metadata, and tool environment variables. Agentd
resolves every declared filesystem path at apply time and creates an immutable
revision artifact that contains all runtime material needed to execute that
exact definition later.

**Why this priority**: This is the core Docker-image-like behavior. An applied
agent cannot be a reliable runtime artifact if it still depends on mutable
definition folders.

**Independent Test**: Apply an agent definition with a relative local script,
a source file, and declared tool environment variables. Delete the original
definition folder, then inspect the persisted revision metadata and copied
artifact directory to verify the prompt, tool declaration, copied script, and
environment data are present without referencing the removed folder.

**Acceptance Scenarios**:

1. **Given** an agent definition references `tools/fetch.py`, **When** the user
   runs `agentd apply path/to/agent.md`, **Then** agentd creates a new revision
   UUID and copies `tools/fetch.py` into the revision artifact.
2. **Given** an agent definition contains relative paths, **When** apply
   succeeds, **Then** every runtime path stored for that revision resolves
   inside the immutable revision directory or to an explicitly allowed external
   non-copied path.
3. **Given** apply has created a revision, **When** the original definition
   folder is removed, **Then** inspecting the revision still shows the prompt,
   tools, tool arguments, and environment declarations needed for execution.
4. **Given** an agent definition includes `environment` values or `.env` file
   paths, **When** apply succeeds, **Then** the resolved environment material is
   captured in the revision artifact and can be supplied to future executions.
5. **Given** the user edits the source markdown prompt or custom tool script
   after apply, **When** the user runs the already-created revision, **Then** the
   run uses the prompt and custom tool implementation captured in that revision,
   not the modified source files.

---

### User Story 2 - Run a Specific Revision (Priority: P1)

A user runs an exact immutable agent revision by specifying
`agentd run <agent-name>:<revision>`, similar to running a Docker image tag or
digest, and receives results produced from the frozen artifact instead of the
current mutable definition.

**Why this priority**: Immutable apply is only useful if users and automation
can choose exactly which revision to execute.

**Independent Test**: Apply one agent twice with different prompt or tool
content to create two revisions. Run each revision explicitly and verify each
run uses the prompt and tool content captured in that revision.

**Acceptance Scenarios**:

1. **Given** two revisions exist for the same agent name, **When** the user runs
   `agentd run example-agent:<old-revision>`, **Then** the daemon executes the
   old revision and records the old revision ID on the run.
2. **Given** an agent name is provided without `:<revision>`, **When** the user
   runs the agent, **Then** the daemon executes the latest active revision for
   that agent.
3. **Given** a requested revision does not exist for the agent, **When** the
   user runs `agentd run example-agent:<missing-revision>`, **Then** the command
   fails with an actionable not-found error and no run is created.

---

### User Story 3 - Execute Custom and Host Tools (Priority: P2)

A revision can declare `custom_tool` entries for agent-provided scripts that
are copied into the immutable artifact, and `host_tool` entries for host-
installed executables such as `curl`, `gh`, `python3`, or `node`. Custom tools
execute from copied artifact files rather than from the original source folder,
while host tools execute the allowed host program. The execution process uses
`data/work/<agent_name>/executions/<execution_id>` as its working directory. The
AI Agent shares the same filesystem namespace as the host OS, but its
reproducible runtime inputs come from the immutable revision artifact.

**Why this priority**: Tools are the main mutable dependency in current agent
definitions. Agent-provided tools must be frozen with the revision, while
host-installed tools must be explicit so users can distinguish copied code from
programs already available on the host.

**Independent Test**: Apply an agent with one `custom_tool` script and one
`host_tool` command. Modify or delete the original custom script, then run the
created revision and verify the process working directory is
`data/work/<agent_name>/executions/<execution_id>`, the custom script is loaded
from `data/work/<agent-name>/<revision_id>`, and the host tool command resolves
from the host environment without being copied into the artifact.

**Acceptance Scenarios**:

1. **Given** a `custom_tool` command points at a relative script path, **When**
   the revision runs, **Then** the process working directory is an execution
   directory and the process command points at the copied script inside the
   revision directory.
2. **Given** the original custom tool script is modified after apply, **When** an existing
   revision runs, **Then** the run uses the copied script content from apply
   time, not the modified source script.
3. **Given** a `host_tool` command is declared as `gh` or `curl`, **When** the
   revision runs, **Then** agentd executes the host-installed program and does
   not copy that executable into the revision artifact.
4. **Given** a tool kind is invalid or ambiguous, **When** apply validates the
   definition, **Then** agentd rejects it and tells the user to use
   `custom_tool` for copied agent code or `host_tool` for host-installed
   executables.

---

### User Story 4 - Observe Tool Output in Agent and Server Logs (Priority: P2)

A user can diagnose tool behavior from both AI Agent logs and agentdserver logs.
For every tool execution, logs expose stdout, stderr, result summary, exit
status, timeout state, and error details when a tool fails.

**Why this priority**: Immutable artifacts make runs reproducible, but users
still need clear evidence for what each tool did and why a run succeeded or
failed.

**Independent Test**: Run an agent with one successful tool and one failing
tool. Verify `agentd logs` shows tool stdout, stderr, result summary, exit
status, and error details for the run, and verify agentdserver structured logs
contain the same tool execution evidence with agent name, run ID, revision ID,
tool name, and stable event names.

**Acceptance Scenarios**:

1. **Given** a tool succeeds and writes stdout or stderr, **When** the user runs
   `agentd logs <agent-name> --run <run-id>`, **Then** the AI Agent logs include
   the tool stdout, stderr, result summary, exit code, and completion event.
2. **Given** a tool fails, exits non-zero, or times out, **When** the user reads
   AI Agent logs, **Then** the logs include stdout, stderr, result summary,
   error message, exit code, timeout state, and failure event.
3. **Given** a tool succeeds or fails, **When** the user reads agentdserver
   logs, **Then** the server logs include the tool stdout, stderr, result
   summary, error details when present, agent name, run ID, revision ID, tool
   name, and stable event name.

---

### User Story 5 - Audit and Retain Revisions (Priority: P3)

A user can list or inspect available immutable revisions for an agent, identify
which revision is latest, and understand what was captured at apply time without
opening runtime database internals.

**Why this priority**: Operators need traceability to understand why a run
behaved differently across revisions.

**Independent Test**: Apply multiple revisions, list revisions for the agent,
inspect one revision, and verify the output includes revision ID, creation time,
source path, prompt digest or prompt text as allowed by output format, tool
metadata, and artifact path.

**Acceptance Scenarios**:

1. **Given** multiple revisions exist, **When** the user lists agent revisions,
   **Then** the CLI shows revision IDs, creation timestamps, latest marker,
   source path, and status.
2. **Given** a revision exists, **When** the user inspects it, **Then** the CLI
   shows captured prompt, declared tools, environment variable names, and the
   artifact directory path.
3. **Given** a revision artifact is missing from disk, **When** the user runs or
   inspects that revision, **Then** agentd reports a corrupted revision error
   that names the missing artifact path and remediation.

### Edge Cases

- The same definition is applied repeatedly without changes.
- The same definition is applied after only non-runtime metadata changes.
- The execution working directory already exists for a run ID.
- A relative script path points outside the definition directory through `..`.
- A `custom_tool` command is a PATH-resolved binary such as `python3` or `node`.
- A `host_tool` command points to a relative source file that should have been a
  `custom_tool`.
- A `host_tool` command is not installed on the host at run time.
- A `custom_tool` command is a relative script path with interpreter arguments.
- A legacy definition still uses `kind: local_tool`.
- A declared path does not exist during apply.
- A path-like string appears in prompt text or another non-path metadata field.
- A copied script loses executable bits, symlinks, or platform-specific line
  endings during artifact creation.
- A declared environment variable references a host secret that is unavailable
  at apply or run time.
- A declared `.env` file contains comments, quoted values, duplicate keys, or
  invalid lines.
- A secret exists in the daemon host environment but is not declared in the
  agent definition metadata.
- The daemon crashes after creating the artifact directory but before updating
  settings state.
- The daemon crashes after updating settings state but before fully copying the
  artifact directory.
- The original definition folder is deleted after apply.
- Two apply requests for the same agent run concurrently.
- Linux and macOS differ in executable permission, symlink, or process behavior.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST create an immutable revision record whenever
  `agentd apply` accepts an agent definition whose runtime content differs from
  the latest revision for that agent.
- **FR-002**: Each revision MUST have a generated UUID identifier and MUST be
  addressable as `<agent-name>:<revision-id>` for execution.
- **FR-003**: System MUST preserve the applied agent prompt in the immutable
  revision.
- **FR-004**: System MUST preserve all declared tools and MCP server metadata
  needed to execute the revision.
- **FR-005**: System MUST support `custom_tool` for agent-provided tool
  implementations copied into the revision artifact and `host_tool` for
  host-installed executables that are not copied.
- **FR-006**: System MUST treat `local_tool` as a legacy compatibility alias
  only during migration and MUST emit new examples, docs, API responses, and
  inspection output with `custom_tool` or `host_tool`.
- **FR-007**: System MUST copy declared `custom_tool` script implementations
  and declared local runtime files into `data/work/<agent-name>/<revision_id>`.
- **FR-008**: System MUST create each run working directory at
  `data/work/<agent_name>/executions/<execution_id>` and run tools from that
  directory while loading copied runtime files from the immutable revision
  artifact.
- **FR-009**: System MUST rewrite copied `custom_tool` commands so execution
  uses paths inside the revision artifact, while keeping `host_tool` commands as
  host executable names or explicitly allowed absolute host paths.
- **FR-010**: System MUST reject `custom_tool` definitions whose implementation
  path cannot be copied into the revision artifact.
- **FR-011**: System MUST reject `host_tool` definitions that use source-relative
  implementation paths or otherwise look like agent-provided code.
- **FR-012**: System MUST preserve declared environment variable names and
  resolved values required by tools according to the agent definition contract.
- **FR-013**: System MUST support an agent definition metadata field named
  `environment` that can contain literal key-value variables and paths to `.env`
  files to resolve and copy into the revision artifact.
- **FR-014**: System MUST parse captured `.env` files at apply time and store
  their resolved environment entries with the revision so execution does not
  depend on the original `.env` file.
- **FR-015**: System MUST NOT inherit undeclared host environment variables into
  agent tool execution, except for minimal OS process variables explicitly
  allowed by daemon policy.
- **FR-016**: System MUST define these agent definition metadata fields as
  filesystem path-bearing fields resolved during apply: `tools[].command` when
  it belongs to a `custom_tool`, relative path entries inside `tools[].args`
  only when marked by the definition contract as file inputs,
  `tools[].read_paths`, `tools[].write_paths`, `access.filesystem.read`,
  `access.filesystem.write`, and `environment.files`.
- **FR-017**: System MUST ignore path-looking strings in non-path metadata
  fields such as prompt text, input descriptions, network allow lists, vendor
  names, model names, and environment variable values unless a field is
  explicitly declared as a filesystem path field.
- **FR-018**: System MUST NOT rely on the original definition markdown file,
  source folder, script folder, or environment file after apply succeeds.
- **FR-019**: System MUST run `agentd run <agent-name>:<revision-id>` against
  the requested revision and record that revision ID on the run.
- **FR-020**: System MUST run `agentd run <agent-name>` against the latest
  active revision for that agent.
- **FR-021**: System MUST reject attempts to mutate an existing revision's
  prompt, copied files, tool definitions, or environment material after the
  revision is finalized.
- **FR-022**: System MUST detect and report missing or corrupted revision
  artifacts before starting a run.
- **FR-023**: System MUST make apply idempotent for unchanged runtime content,
  returning the existing latest revision instead of creating duplicate
  revisions.
- **FR-024**: System MUST log revision creation, path resolution decisions, tool
  command rewriting, environment capture source names, artifact validation
  failures, and revision-specific run start events with stable event names.
- **FR-025**: System MUST include captured tool stdout, stderr, result summary,
  exit code, timeout state, and error details in AI Agent logs.
- **FR-026**: System MUST include captured tool stdout, stderr, result summary,
  exit code, timeout state, and error details in agentdserver structured logs.
- **FR-027**: System MUST define required filesystem, network, environment,
  credential, and privilege access for revision creation and execution.
- **FR-028**: System MUST define Linux and macOS behavior for copying files,
  preserving executable permissions, resolving symlinks, parsing `.env` files,
  and running copied tools.
- **FR-029**: System MUST define restart, cancellation, cleanup, and recovery
  behavior for partially-created revision artifacts and partially-created
  execution working directories.
- **FR-030**: System MUST avoid storing copied script, fixture, or `.env` file
  blob data in the settings database when the revision artifact directory can
  contain it durably.

### Key Entities *(include if feature involves data)*

- **Agent Revision**: Immutable runtime artifact for one applied agent version.
  Key attributes include agent name, revision UUID, runtime content digest,
  source path, prompt, vendor, schedule, tool metadata, environment material,
  artifact path, status, created timestamp, and finalized timestamp.
- **Revision Artifact Directory**: Durable filesystem directory at
  `data/work/<agent-name>/<revision_id>` containing copied local scripts,
  copied runtime files, manifest metadata, and checksums.
- **Revision Tool**: Frozen tool declaration associated with an agent revision.
  A `custom_tool` is an agent-provided implementation copied into the artifact.
  A `host_tool` is a host-installed executable invoked from the host. Attributes
  include original command, rewritten command where applicable, args, env,
  timeout, declared access, host/copy mode, and copied file references.
- **Revision Environment Variable**: Captured tool environment entry used by a
  revision. Values are available to tool execution; CLI inspection masks values
  by default and shows names.
- **Environment File**: Declared `.env` file path in agent definition metadata.
  It is resolved and copied during apply, parsed into revision environment
  entries, and no longer read from the source folder during execution.
- **Agent Run**: Existing run record extended so every run references the exact
  revision used for execution and a working directory under
  `data/work/<agent_name>/executions/<execution_id>`.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can apply an agent, delete the original definition folder,
  and still successfully run the applied revision.
- **SC-002**: A user can apply two different revisions of one agent and
  explicitly run each revision with `agentd run <agent-name>:<revision-id>`.
- **SC-003**: All `custom_tool` script executions for immutable revisions use
  `data/work/<agent_name>/executions/<execution_id>` as the process working
  directory and load copied scripts or files from the immutable revision
  artifact.
- **SC-004**: All `host_tool` executions use an explicitly declared
  host-installed executable and do not copy that executable into the revision
  artifact.
- **SC-005**: Reapplying unchanged runtime content returns the existing latest
  revision without creating a duplicate artifact.
- **SC-006**: Missing copied tool files are detected before process start and
  reported as corrupted revision errors.
- **SC-007**: Tool stdout, stderr, result summaries, and errors are visible
  through `agentd logs` for successful and failed tool executions.
- **SC-008**: Tool stdout, stderr, result summaries, and errors are visible in
  agentdserver logs for successful and failed tool executions.
- **SC-009**: Revision creation, listing, inspection, and execution behavior is
  covered by automated tests on Linux/macOS-compatible paths.
- **SC-010**: Declared environment values and declared `.env` files are
  available during execution after the original definition folder is deleted,
  while undeclared host environment secrets are absent.
- **SC-011**: Codex manually applies and runs
  `examples/github-trending-engineering-radar/github-trending-engineering-radar.md`;
  the agent successfully returns a GitHub-derived result, `agentd logs`
  includes the GitHub tool stdout and stderr summaries for the run, and
  agentdserver logs include the tool result summary and error details if any
  occurred.

## Assumptions

- Agent revision UUIDs are generated by the daemon at apply time.
- `data/work/<agent-name>/<revision_id>` is rooted under the existing daemon
  data directory, not the current shell working directory.
- `data/work/<agent_name>/executions/<execution_id>` is the working directory
  for one run and may contain generated output, temporary files, and tool write
  targets.
- The latest revision for `agentd run <agent-name>` means the newest finalized
  active revision created by apply.
- Environment values may contain secrets. Logs and default CLI inspection show
  variable names and masked values, not raw secrets.
- Declared `environment.variables` values take precedence over values parsed
  from declared `environment.files` when the same key appears in both places.
- `custom_tool` replaces the previous `local_tool` vocabulary for
  agent-provided tool implementations. `host_tool` is the chosen name for
  system-wide executables because it clearly communicates that the executable
  comes from the host OS, unlike `process_tool`, which could describe both
  custom and host tools.
- PATH-resolved interpreter or binary commands such as `python3`, `node`, and
  `bash` are represented as `host_tool` commands and are not copied; script
  paths passed as command or arguments are copied only when declared by the
  agent definition contract for a `custom_tool`.
- Full filesystem sandboxing is out of scope: the AI Agent process shares the
  host OS filesystem namespace, while agentd controls the process working
  directory, immutable artifact inputs, declared environment, and audit logs.
