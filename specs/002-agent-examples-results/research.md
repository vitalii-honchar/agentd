# Research: Agent Examples and Results

## Decision: Store completed and failed run results in the runtime database

**Rationale**: Users, Bash scripts, local AI agents, and Go integrations need
durable access to outputs after a run ends. The existing runtime DB already owns
Agent Run lifecycle state and survives daemon restart, so adding result fields
there keeps result lookup consistent with run status and recovery.

**Alternatives considered**:
- Log-only result retrieval: rejected because logs are optimized for debugging,
  not stable structured output.
- Result files only: rejected because query and table views need indexed
  status, timestamps, and run IDs.

## Decision: Add `agentd ps`, `agentd ps -a`, and `agentd result` as daemon-backed CLI commands

**Rationale**: The requested workflow mirrors local infrastructure operations:
users list active/all runs, then retrieve compact or full output. The CLI
already supports `--output text|json`, so JSON can serve Bash/local-agent
automation while text remains human-readable.

**Alternatives considered**:
- Extend only `inspect`: rejected because run history and result lookup are
  separate operator tasks.
- Require users to read SQLite directly: rejected because it bypasses daemon
  policy and creates unstable automation.

## Decision: Promote the daemon HTTP client into `pkg/agentdclient`

**Rationale**: Go integrations need a supported import path. Moving or wrapping
the existing internal HTTP client avoids adding a new public network API while
giving Go programs typed apply/list/execute/result/log operations.

**Alternatives considered**:
- Tell Go users to shell out to CLI: rejected because it is brittle for
  long-term integrations.
- Expose server internals: rejected because `internal` packages intentionally
  hide daemon implementation details.

## Decision: Enforce same-host client access at the daemon boundary

**Rationale**: The feature explicitly defers auth while allowing local CLI and
Go integrations. Binding/defaulting to loopback plus rejecting non-loopback
remote addresses prevents accidental remote exposure without introducing token
management.

**Alternatives considered**:
- Add API tokens now: rejected as out of scope and unnecessary for same-host use.
- Accept all network clients: rejected because it contradicts local-only safety.

## Decision: Execute tools as declared command-line processes

**Rationale**: Examples need language-flexible data collection and website
capture. A command-line tool contract lets examples use Python or Node locally
while keeping daemon policy simple: only declared commands run, each invocation
has a timeout, working directory, stdout/stderr capture, exit status, and action
logs.

**Alternatives considered**:
- Build all tools into Go: rejected because example data sources evolve and the
  feature asks for flexible tool languages.
- Let agents run arbitrary shell commands: rejected because it violates
  least-privilege and auditability.

## Decision: Example default paths use public unauthenticated sources or bundled seed data

**Rationale**: Examples must work after clone plus documented dependency
installation. Public read-only sources and bundled source lists avoid CI setup,
SaaS integrations, private data, and required API keys while still proving
repeatable scheduled automation.

**Alternatives considered**:
- Examples requiring user-owned products/apps/issues: rejected because many
  users cannot run them immediately.
- Fully offline examples only: rejected because scheduled monitoring over public
  sources is the core agentd demonstration.

## Decision: Keep one manual website snapshot example

**Rationale**: Most examples should demonstrate scheduling, but the platform
also needs manual execution with user input and tool execution. A URL snapshot
example is a clear manual workflow requiring no account setup.

**Alternatives considered**:
- Make all examples scheduled: rejected because it would not demonstrate
  manual run-time input.
- Keep code-review or CI examples: rejected because they are one-off assistant
  workflows and do not show why a daemon should keep running.
