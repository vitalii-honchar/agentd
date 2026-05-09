# Research: Unified Agentd ReAct Contracts

## Decision: Keep One Daemon Owner And Dispatch Daemon Mode From `agentd`

**Decision**: Make `cmd/agentd` select daemon mode before constructing the CLI
HTTP client. `--daemon`, `-d`, and compatibility alias `--deamon` start the
existing daemon service lifecycle in-process. Client commands remain HTTP
clients of the daemon.

**Rationale**: This keeps distribution to one binary without duplicating
runtime execution logic in the CLI. It also preserves the constitution's
daemon-first runtime rule.

**Alternatives considered**:
- Keep two binaries and only change packaging: rejected because users still
  need to discover and invoke `agentdserver`.
- Make CLI commands execute agents directly when no daemon is running: rejected
  because it splits runtime authority and weakens cleanup/recovery.

## Decision: Contracts Are Optional And Persisted In Revisions

**Decision**: Add optional `contract.input` and `contract.output` schemas to
agent definitions, persist them with the applied agent and immutable revision,
and include them in revision content digests. If omitted, no contract behavior
is applied.

**Rationale**: Contracts must be stable for explicit revision runs, and omitted
contracts must preserve legacy behavior.

**Alternatives considered**:
- Require contracts for every agent immediately: rejected because it breaks
  existing user definitions.
- Store contracts only in source markdown: rejected because immutable revisions
  must keep working after source files change or disappear.

## Decision: Add Runtime JSON Input Support

**Decision**: Keep existing `--input key=value` support for legacy/simple
inputs, and add JSON input surfaces for contracted agents: CLI JSON string,
CLI file input, HTTP JSON object, and public client map/object support.

**Rationale**: JSON Schema supports nested values, arrays, numbers, and
booleans, which cannot be represented safely by string-only key/value input.

**Alternatives considered**:
- Continue only string key/value input: rejected because it cannot satisfy the
  contract requirements.
- Replace all input with JSON only: rejected because it would break existing
  simple CLI workflows.

## Decision: Use A JSON Schema Validation Dependency

**Decision**: Add a focused JSON Schema validation library in the runtime
adapter layer and wrap it behind an agentd contract validator interface.

**Rationale**: Go's standard library does not validate JSON Schema. A narrow
adapter keeps schema behavior replaceable and prevents validation-library types
from leaking into domain/use-case packages.

**Alternatives considered**:
- Hand-roll validation for the subset used by examples: rejected because users
  can provide real JSON Schema documents and need reliable diagnostics.
- Use LLM-only validation: rejected because invalid input must fail before any
  model or tool execution.

## Decision: Finalize Contracted Output From Execution History

**Decision**: The ReAct loop records model messages, tool requests, and safe
observations. A finalization step receives that history and `contract.output`,
asks the selected provider for structured final JSON, validates the result, and
performs bounded repair attempts before failing the run.

**Rationale**: Intermediate ReAct messages are not the final business result.
Finalizing from history keeps the loop flexible while producing reliable
machine-readable output.

**Alternatives considered**:
- Force every model response to match `contract.output`: rejected because
  reasoning/tool steps need a different shape than final results.
- Accept model JSON without validation: rejected because it undermines the
  contract.

## Decision: Adapt go-agent Before Direct Agentd Integration

**Decision**: Reuse go-agent as the ReAct control library only through a
dynamic-schema, provider-neutral adapter that supports external daemon-owned
tool callbacks, history export, cancellation, and structured output. If the
current go-agent module lacks that API, add/adapt it first rather than forcing
agentd into generic/static output types.

**Rationale**: The current go-agent design already has the desired loop,
history, tool limits, and final structured-output pass. Agentd, however, needs
runtime JSON Schemas from markdown, daemon-owned process tools, existing
OpenAI provider plumbing, run logs, and immutable revision state. A narrow
adapter preserves reuse without leaking agentd policy into the library.

**Alternatives considered**:
- Import current generic go-agent APIs directly: rejected because agentd does
  not know output types at compile time.
- Rewrite ReAct logic only inside agentd: rejected unless go-agent cannot be
  adapted, because the feature explicitly prefers go-agent as the control
  library.

**Implementation note**: The implemented adapter uses go-agent's dynamic agent
API with a small provider bridge. Agentd still owns provider selection, tool
execution, revision artifact paths, run logging, and contract validation.
go-agent owns the ReAct loop and tool-call limit. The dynamic agent is given a
small internal `final_text` schema; if the agent definition has
`contract.output`, agentd performs a second provider-backed output finalization
step against the user schema. This is the main deviation from passing the user
schema directly into go-agent and avoids forcing intermediate ReAct turns to
match the final business contract.

## Decision: Represent Model Steps As Provider-Neutral JSON Decisions

**Decision**: The runtime ReAct adapter uses provider-neutral step schemas for
model decisions: continue with a declared tool call, finish, or fail. OpenAI
and Codex providers both satisfy this contract, but use different transport.

**Rationale**: Codex CLI is a process-oriented agent interface rather than a
direct OpenAI tool-call endpoint. A provider-neutral step contract lets agentd
keep tool execution and policy enforcement in the daemon.

**Alternatives considered**:
- Let provider-native tool calls execute tools directly: rejected because it
  would bypass agentd's declared-tool and isolation policy.
- Let Codex CLI operate freely in the repository: rejected because Codex is an
  LLM provider here, not the workload executor.

## Decision: Implement Codex Provider Through Managed CLI Process Execution

**Decision**: Add a `codex` provider that invokes `codex exec` in
non-interactive mode from a run-owned working directory. Use stdin for prompts,
JSON events and final-message file output for capture, and output-schema files
when structured output is needed. Do not extract or store undocumented Codex
tokens.

**Rationale**: The local CLI exposes a supported automation surface and can
reuse the user's existing login/configuration. Token extraction would depend on
private implementation details and create unnecessary secret-handling risk.

**Alternatives considered**:
- Use direct OpenAI API credentials for Codex: rejected because the user's goal
  is to avoid direct API access for this path.
- Extract a Codex developer token: rejected because it is undocumented and
  unsafe to persist or rely on.

## Decision: Change Logs To Run-ID Only

**Decision**: Replace `agentd logs <agent_name> --run <run_id>` and latest-run
fallback with `agentd logs <run_id>`. Add a run-scoped HTTP/client contract and
return an actionable error for agent-name arguments.

**Rationale**: Agent-name log lookup becomes ambiguous after multiple runs.
Run IDs are already the stable identifier for results and active/finished run
listing.

**Alternatives considered**:
- Keep agent-name lookup and require `--run`: rejected because the current CLI
  still allows ambiguous latest-run behavior.
- Support both forever: rejected because users asked to disable the ambiguous
  path.

## Decision: Migrate All Checked-In Examples To Contracts

**Decision**: Add `contract.input` and `contract.output` schemas to every
checked-in example agent. Scheduled examples use an empty-object input schema.
Manual examples model their runtime parameters.

**Rationale**: Examples are the authoring reference. They must demonstrate the
new schema format and structured results.

**Alternatives considered**:
- Add one new contract example only: rejected because most users copy existing
  examples.
- Leave examples prose-only: rejected because it undercuts the contract feature.
