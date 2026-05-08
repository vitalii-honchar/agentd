# Feature Specification: Agent Examples and Results

**Feature Branch**: `002-agent-examples-results`
**Created**: 2026-05-08
**Status**: Draft
**Input**: User description: "Improve the repository examples by deleting the
current weak examples and replacing them with specific real-use-case agents
that demonstrate repeatable automation: cybersecurity subreddit analysis,
Hacker News daily summaries, customer-pain monitoring, Product Hunt launch
monitoring, engineering trend monitoring, open-source issue monitoring,
competitor changelog monitoring, hiring-market signal monitoring, public app
review monitoring, and website screenshot summaries. Scheduled examples should
demonstrate daily monitoring; manual examples should be limited to workflows
that naturally need user input. Add CLI visibility for agent definitions, agent
runs, completed or failed run results, full run details by run ID, useful
timestamped action logs, and support for agents to execute declared
command-line tools stored with their example definition. Result retrieval should
work from Bash, from another local AI agent, and from Go programs that import
the agentd client for same-host integrations. Authorization is out of scope for
now; the daemon should accept only same-host requests. Every example must be
runnable after cloning the repository with zero service configuration from the
user; installing documented local dependencies is acceptable."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Replace Examples with Real Agents (Priority: P1)

A new user explores the repository examples and finds a broad catalog of
realistic scheduled monitoring agents that demonstrate useful repeatable
workflows for security, software engineering, product management, market
research, customer discovery, and news monitoring.

**Why this priority**: The examples are the first practical proof that the
runtime can solve real work. Low-quality examples make the platform hard to
evaluate even if the runtime works.

**Independent Test**: Remove the existing example set, apply each new example
agent, and verify each definition describes a realistic source or input,
expected result, schedule behavior, target user, declared supporting tools, and
a README with setup and run instructions.

**Acceptance Scenarios**:

1. **Given** the repository examples are installed, **When** a user lists the
   examples directory, **Then** only the new concrete real-world examples are
   present, with one folder per example agent.
2. **Given** the cybersecurity example is applied, **When** the daily schedule
   becomes due, **Then** the agent analyzes recent cybersecurity discussion and
   returns a concise assessment of possible vulnerabilities, data leaks, and
   notable risk indicators.
3. **Given** the daily news example is applied, **When** the daily schedule
   becomes due, **Then** the agent reviews recent important technology news and
   returns a short prioritized summary.
4. **Given** the website screenshot example is applied, **When** the user
   manually executes it with a website URL, **Then** the agent captures the
   website state and returns a readable summary of what the site appears to be
   about.
5. **Given** a software engineer monitoring example is applied, **When** its
   daily schedule becomes due, **Then** the agent returns a concrete engineering
   trend, risk, or opportunity summary with evidence and next steps.
6. **Given** a product manager monitoring example is applied, **When** its daily
   schedule becomes due, **Then** the agent returns a concrete product insight
   summary with source references and implications.
7. **Given** a user has freshly cloned the repository and installed documented
   local dependencies, **When** they follow any example README, **Then** the
   example can run without creating external accounts, configuring CI, entering
   API keys for data sources, or connecting private services.

---

### User Story 2 - Discover Definitions and Runs (Priority: P2)

A user applies agent definitions, starts or waits for executions, then uses
Docker-like CLI commands to see which definitions exist and which runs are
active or finished.

**Why this priority**: Without a list of definitions and runs, users cannot
operate or debug scheduled agents after applying them.

**Independent Test**: Apply all examples, start at least one manual run,
let one run finish, and verify definition listing and run listing commands show
the expected names, run identifiers, trigger types, and statuses.

**Acceptance Scenarios**:

1. **Given** one or more agent definitions are applied, **When** the user runs
   the definition listing command, **Then** the CLI shows each applied agent name
   with enough metadata to identify its schedule mode and current state.
2. **Given** one or more agent runs are active, **When** the user runs
   `agentd ps`, **Then** the CLI shows only active runs.
3. **Given** active and finished runs exist, **When** the user runs
   `agentd ps -a`, **Then** the CLI shows active and finished runs, including
   terminal statuses such as `COMPLETED` and `FAILED`.

---

### User Story 3 - Retrieve Run Results for Automation (Priority: P2)

A user, shell script, another local AI agent, or custom Go integration wants to
use agent output for follow-up processing, so it executes or observes an agent
run, retrieves a compact history of results for one agent, and then inspects a
specific run in full.

**Why this priority**: Agent execution has little value if humans and automation
cannot retrieve completed or failed outputs after the run ends. Result retrieval
is the foundation for using agentd as local AI Agent infrastructure.

**Independent Test**: Complete successful and failed runs for an example agent,
then verify the result commands show a trimmed table by agent name and full run
details by run identifier; also verify a Bash script and a small Go program can
execute an agent, wait for or discover the run, and retrieve the stored result
without reading runtime storage files directly.

**Acceptance Scenarios**:

1. **Given** an agent has multiple finished runs, **When** the user runs
   `agentd result <agent-name>`, **Then** the CLI returns a table containing
   run identifiers, timestamps, statuses, and trimmed result text suitable for a
   normal terminal view.
2. **Given** the result table contains a run identifier, **When** the user runs
   `agentd result <agent-run-id>`, **Then** the CLI returns full run details,
   including the untrimmed result and failure information when applicable.
3. **Given** a run fails, **When** the user retrieves its result, **Then** the
   failure result explains what failed clearly enough for the user to decide the
   next action.
4. **Given** a Bash script or local AI agent runs the CLI, **When** it requests
   machine-readable output for execution, run listing, or result lookup, **Then**
   the command returns stable fields suitable for shell parsing and exits with a
   status that reflects success or failure.
5. **Given** a local Go program imports the public agentd client, **When** it
   executes an agent and retrieves the run result, **Then** it can do so through
   supported client methods without importing internal packages or shelling out
   to the CLI.

---

### User Story 4 - Audit Agent Execution Logs (Priority: P3)

A user investigates why an agent behaved unexpectedly and reads system logs for
that agent that show timestamped runtime actions rather than only the final LLM
response.

**Why this priority**: Useful logs are required to debug definitions, tool
execution, vendor calls, scheduler behavior, and failed runs.

**Independent Test**: Execute an agent that calls at least one declared tool and
verify `agentd logs` shows timestamped action entries for prompt submission,
tool execution start, tool execution result, completion, and failure paths.

**Acceptance Scenarios**:

1. **Given** an agent run sends a prompt to an LLM vendor, **When** the user
   views that run's logs, **Then** the logs include a timestamped action showing
   the prompt submission event without replacing the log stream with the final
   response.
2. **Given** an agent run executes a declared tool, **When** the user views
   logs, **Then** the logs include timestamped action entries for tool start,
   tool completion or failure, and a concise result summary.
3. **Given** multiple agents have logs, **When** a user requests logs for one
   agent, **Then** the CLI shows only that agent's system-level run logs.

---

### User Story 5 - Execute Declared Command-Line Tools (Priority: P3)

An agent definition can declare supporting command-line tools stored next to the
example definition, allowing examples to gather external data or screenshots
without hard-coding tool behavior into the runtime.

**Why this priority**: Real examples need controlled access to external data and
website capture. Tool execution must be explicit so users can understand and
audit what an agent may run.

**Independent Test**: Apply each example with its declared local tools, execute
the manual examples, wait for at least one scheduled example, and verify each
run can invoke only its declared tools and records tool results in logs and run
output.

**Acceptance Scenarios**:

1. **Given** an example agent declares a local command-line tool, **When** the
   agent run requires external data or website capture, **Then** the runtime
   executes that declared tool as a separate process and provides the tool
   result back to the agent.
2. **Given** an agent attempts to use an undeclared tool, **When** the run is
   evaluated, **Then** the runtime denies the request and records an actionable
   failure.
3. **Given** a tool exits unsuccessfully or times out, **When** the run
   completes, **Then** the run is marked failed and the result and logs contain
   the relevant failure summary.

## Required Example Catalog

The repository MUST replace the old examples with these specific example
agents. Each example lives in its own folder, includes one Markdown definition,
includes a README, declares its local tools, and produces a result that can be
retrieved through the result commands. Examples MUST be self-contained: the
default run path may use public unauthenticated sources, bundled public-source
lists, bundled fixture files, or user-provided manual input, but it MUST NOT
require the user to configure external services, CI systems, SaaS integrations,
private accounts, private data, or data-source credentials. Optional API keys
may be documented only as enhancements; they MUST NOT be required for the
example's default successful run. At least eight examples MUST demonstrate
scheduled repeatable monitoring.

1. **Cybersecurity Reddit Watch**: Daily scheduled security analyst agent that
   reviews recent public read-only posts from `r/cybersecurity` without
   requiring Reddit API credentials, flags possible new vulnerabilities, active
   exploitation claims, data leak reports, affected products, confidence, and
   source links.
2. **Hacker News Builder Brief**: Daily scheduled engineering and product news
   agent that reviews important Hacker News stories from the last day and
   returns the top items with one-sentence summaries, why each item matters, and
   links.
3. **Reddit Customer Pain Monitor**: Daily scheduled product manager agent that
   reviews a bundled list of public subreddits such as SaaS, startups, and small
   business communities, then summarizes repeated complaints, feature requests,
   buying signals, workaround patterns, and urgent customer pains.
4. **Product Hunt Launch Radar**: Daily scheduled market research agent that
   reviews public Product Hunt launches and returns the most interesting new
   products, categories, target users, positioning, traction signals, and product
   opportunities.
5. **GitHub Trending Engineering Radar**: Daily scheduled software engineer
   agent that reviews public trending repositories or a bundled list of public
   repository discovery pages, then summarizes notable developer tools,
   libraries, infrastructure projects, adoption signals, and why engineers might
   care.
6. **Open Source Issue Pain Monitor**: Daily scheduled software engineer and
   product agent that reviews public issues from a bundled list of popular
   repositories and summarizes recurring bugs, confusing user experiences,
   missing features, support burden, and tool opportunities.
7. **Competitor Changelog Monitor**: Daily scheduled product manager agent that
   watches a bundled list of public unauthenticated competitor or product
   changelog pages and returns detected positioning, packaging, pricing, or
   feature changes with product implications.
8. **Hiring Market Signal Monitor**: Daily scheduled strategy agent that reviews
   public hiring discussion pages or bundled public job-search pages and
   summarizes recurring roles, required skills, demand signals, and emerging
   market trends relevant to builders.
9. **Public App Review Theme Monitor**: Daily scheduled product manager agent
   that reviews public app review pages when available or bundled public review
   fixtures by default, then returns recurring complaints, praised features,
   sentiment shifts, and roadmap opportunities.
10. **Website Snapshot Analyst**: Manual research agent that accepts a website
   URL, captures the visible page state, and summarizes the product, audience,
   key claims, calls to action, obvious trust signals, and competitive
   positioning.

### Edge Cases

- A user applies an example before installing documented local dependencies
  required by that example's tools.
- A user runs an example from a fresh clone without any external service
  accounts, API keys, CI systems, SaaS integrations, or private data.
- A README mentions optional API keys, but the user follows the default path
  without providing any keys.
- An external source is unavailable, rate-limited, returns malformed data, or
  returns no recent items.
- The cybersecurity source includes rumors, duplicates, jokes, advertisements,
  or content that is not evidence of a vulnerability or leak.
- The daily news source includes many low-value stories or no high-value stories
  within the daily window.
- The website screenshot example receives a missing, malformed, unreachable, or
  private-network URL.
- A website blocks capture, requires authentication, loads slowly, or renders
  mostly empty content.
- A manual execution is requested without required user input.
- The daily scheduled agent is manually executed even though it is primarily
  scheduled.
- A public-source monitor receives an empty page, changed page structure,
  malformed response, duplicate items, or content that cannot be confidently
  categorized.
- A product monitoring example receives noisy comments, jokes, advertisements,
  spam, or off-topic discussion that should not be treated as customer pain.
- A competitor website changes layout without changing meaningful product
  information.
- A scheduled example finds no meaningful new items since its previous run.
- `agentd ps` is run when no active runs exist, and `agentd ps -a` is run when
  no runs have ever occurred.
- `agentd result <agent-name>` is run for an unknown agent or an agent with no
  finished runs.
- `agentd result <agent-run-id>` is run for an unknown, active, completed, or
  failed run.
- A script needs result data without fragile table parsing.
- A Go integration imports the public client from a version of agentd that does
  not match the running daemon exactly.
- A program running on another host attempts to call the daemon.
- A result is too large for a readable table and must be trimmed without hiding
  the full result from detailed lookup.
- The daemon restarts while a tool process, scheduled run, manual run, result
  lookup, or log lookup is in progress.
- Multiple runs of the same agent finish close together and must remain
  distinguishable by timestamp and run identifier.
- A tool writes sensitive values to output or logs; the system must avoid
  exposing undeclared secrets.
- Logs grow large enough that terminal output must remain readable.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST remove the previous repository examples and replace
  them with the complete Required Example Catalog from this specification.
- **FR-002**: The cybersecurity example MUST run on a daily schedule and return a
  concise assessment of possible vulnerabilities, data leaks, and notable risk
  indicators from recent cybersecurity discussion.
- **FR-003**: The daily news example MUST run automatically once per day and
  return a short prioritized summary of the most important recent technology
  news.
- **FR-004**: The website screenshot example MUST be manually executable,
  accept a user-provided website URL at run time, capture the website state, and
  return a readable website summary.
- **FR-005**: Each example definition MUST declare its schedule behavior,
  expected inputs, expected outputs, and any supporting command-line tools it may
  execute.
- **FR-006**: Each example folder MUST include a README that explains what the
  example does, what local dependencies to install, how to apply it, how to run
  it, and how to retrieve its results and logs.
- **FR-007**: Every example MUST run from a freshly cloned repository after
  installing documented local dependencies, without requiring external account
  creation, service credentials, CI setup, SaaS integrations, private data, or
  user-specific remote configuration.
- **FR-008**: Examples MAY use public unauthenticated sources, bundled fixtures,
  current repository files, or explicit manual user input.
- **FR-009**: Optional API keys MAY be documented for richer behavior, but every
  example MUST have a default successful run path that does not require those
  keys.
- **FR-010**: The cybersecurity Reddit example MUST use a public read-only path
  for its default run and MUST NOT require Reddit API credentials.
- **FR-011**: The Hacker News example MUST use the public read-only Hacker News
  data source for its default run and MUST NOT require API credentials.
- **FR-012**: The website screenshot example MUST require only a user-provided
  URL and documented local browser/tool dependencies for its default run.
- **FR-013**: The competitor monitoring example MUST ship with a bundled list of
  public unauthenticated pages so users do not need to configure competitors
  before the first run.
- **FR-014**: Product and market monitoring examples MUST ship with bundled
  public-source lists or sample fixtures needed for the default run so users do
  not need to choose sources before first use.
- **FR-015**: The example catalog MUST include Reddit customer pain monitoring,
  Product Hunt launch monitoring, GitHub engineering trend monitoring,
  open-source issue pain monitoring, competitor changelog monitoring, hiring
  market signal monitoring, and public app review theme monitoring.
- **FR-016**: At least eight examples MUST run automatically on a daily schedule
  by default to demonstrate repeatable agentd automation.
- **FR-017**: Manual examples MUST be limited to workflows that naturally
  require user-provided input, such as website URL analysis.
- **FR-018**: The software engineering examples MUST include Hacker News builder
  brief, GitHub trending engineering radar, and open-source issue pain monitor
  workflows with concrete source references and output sections.
- **FR-019**: The product manager examples MUST include Reddit customer pain
  monitor, Product Hunt launch radar, competitor changelog monitor, and public
  app review theme monitor workflows with concrete source references and
  decision-oriented output sections.
- **FR-020**: System MUST provide a CLI command to list applied agent
  definitions by agent name, schedule mode, enabled state, and current state.
- **FR-021**: System MUST provide `agentd ps` to list active agent runs only.
- **FR-022**: System MUST provide `agentd ps -a` to list active and finished
  agent runs.
- **FR-023**: Run listings MUST include run identifier, agent name, trigger
  source, start time, completion time when available, and current or terminal
  status.
- **FR-024**: Agent runs MUST reach a terminal status of `COMPLETED` when they
  finish successfully and `FAILED` when they do not finish successfully.
- **FR-025**: Agent runs MUST store a result for both completed and failed
  executions.
- **FR-026**: System MUST provide `agentd result <agent-name>` to show a table
  of all finished runs for the specified agent, including run identifier,
  timestamp, terminal status, and trimmed result text.
- **FR-027**: System MUST provide `agentd result <agent-run-id>` to show full
  run details for one run, including untrimmed result text, terminal status,
  trigger source, timestamps, and failure information when present.
- **FR-028**: Result, run listing, and execution commands MUST provide
  machine-readable output suitable for Bash scripts and local AI agents.
- **FR-029**: Machine-readable command output MUST include stable field names
  for run identifier, agent name, status, timestamps, result, and failure
  summary when those fields are available.
- **FR-030**: Command exit statuses MUST distinguish successful command
  execution from missing agents, missing runs, active runs with no final result,
  failed agent runs, and daemon communication failures.
- **FR-031**: System MUST provide a public Go client package that custom local
  integrations can import to apply or list definitions, execute agents, list
  runs, and retrieve run results without importing internal packages.
- **FR-032**: The public Go client MUST reuse the same daemon communication
  behavior as the CLI so CLI and Go integrations observe consistent statuses,
  results, and errors.
- **FR-033**: The daemon MUST accept client requests only from the same host
  where it is running for this feature.
- **FR-034**: The system MUST NOT require user authentication or authorization
  for same-host CLI or public-client requests in this feature.
- **FR-035**: Trimmed result tables MUST preserve enough text for users to
  identify the output while keeping terminal table formatting readable.
- **FR-036**: System MUST record system-level logs for each agent run as
  timestamped action entries rather than replacing logs with only the final LLM
  response.
- **FR-037**: Agent run logs MUST include timestamped entries for prompt
  submission, tool execution start, tool execution completion or failure, run
  completion, and run failure when those events occur.
- **FR-038**: `agentd logs` MUST show logs scoped to the selected agent or run
  without mixing entries from unrelated agents.
- **FR-039**: System MUST allow an agent definition to declare local
  command-line tools stored alongside the agent definition.
- **FR-040**: System MUST execute declared command-line tools as separate
  processes and provide their results back to the agent run.
- **FR-041**: System MUST deny execution of undeclared tools and record an
  actionable failure result and log entry.
- **FR-042**: System MUST record tool process exit status, timeout, and concise
  output summary in the agent run logs.
- **FR-043**: System MUST support manual execution with user-provided inputs for
  agents that require run-time parameters.
- **FR-044**: System MUST preserve applied definitions, run records, run
  statuses, results, and run logs across daemon restarts.
- **FR-045**: System MUST keep the examples understandable enough that a new
  user can apply, execute, inspect, read logs, and retrieve results without
  editing runtime internals.
- **FR-046**: System MUST define required filesystem, network, environment,
  credential, and privilege access for any example tool execution behavior.
- **FR-047**: System MUST define Linux and macOS behavior for daemon, process,
  filesystem, networking, permission, and screenshot-capture differences.
- **FR-048**: System MUST define restart, cancellation, cleanup, and recovery
  behavior for tool processes and runs that can outlive a single CLI request.
- **FR-049**: System MUST avoid requiring new abstractions, dependencies, or
  optimizations unless they are justified by current requirements or measured
  bottlenecks.

### Key Entities *(include if feature involves data)*

- **Example Agent**: A repository-provided Agent Definition and companion files
  that demonstrate one realistic workflow, including schedule behavior,
  required user inputs, declared tools, and expected result shape.
- **Example README**: The per-example guide that explains the example's purpose,
  local dependencies, default zero-configuration run path, optional enhancements,
  apply command, run command, result lookup, and log lookup.
- **Agent Definition**: An applied Markdown definition with a unique name,
  schedule mode, enabled state, prompt, input expectations, and declared tool
  permissions.
- **Agent Run**: One execution attempt for an agent, including a unique run
  identifier, agent name, trigger source, timestamps, current or terminal
  status, stored result, and links to scoped logs.
- **Run Result**: The stored outcome of an Agent Run, containing either the
  successful output or the failed-run explanation needed for follow-up
  processing.
- **Public Client**: A supported importable client for local Go programs that
  need to communicate with the agentd daemon using the same operations exposed
  by the CLI.
- **Machine-Readable Output**: A stable CLI output mode intended for Bash
  scripts, local AI agents, and other automation to parse without relying on
  human-oriented tables.
- **Same-Host Request**: A daemon request that originates from the same machine
  where agentd is running and is accepted without user authentication for this
  feature.
- **Run Log Entry**: A timestamped system-level event for an Agent Run,
  including action name, concise details, and relevant status.
- **Declared Tool**: A command-line program referenced by an Agent Definition
  that the runtime may execute as a separate process during a run.
- **Tool Execution**: One invocation of a declared tool, including input
  summary, start time, finish time, exit outcome, timeout state, and output
  summary.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A new user can apply all repository examples, list their
  applied definitions, and identify each example's purpose and schedule mode in
  under 15 minutes using repository documentation.
- **SC-002**: At least eight examples are scheduled monitoring agents that
  produce stored completed or failed results retrievable from the CLI in 100% of
  local smoke tests.
- **SC-003**: Each daily monitoring example runs no more than once per scheduled
  day without manual intervention in 100% of scheduler tests.
- **SC-004**: `agentd ps` shows only active runs, and `agentd ps -a` shows both
  active and finished runs with correct statuses in 100% of run-listing tests.
- **SC-005**: Every completed or failed run stores a result that can be retrieved
  by agent name and by run identifier in 100% of result retrieval tests.
- **SC-006**: A Bash script can execute an example agent, capture the run
  identifier, wait for or discover terminal status, and retrieve the final
  result in 100% of automation tests.
- **SC-007**: A local Go integration can execute an example agent and retrieve
  its result through the public client in 100% of client integration tests.
- **SC-008**: Requests from another host are rejected in 100% of local-boundary
  tests.
- **SC-009**: Result tables remain readable in an 80-column terminal while full
  untrimmed results remain available by run identifier in 100% of formatting
  tests.
- **SC-010**: Agent logs show timestamped action entries for prompt submission,
  tool execution, and terminal run outcome in 100% of runs that reach those
  events.
- **SC-011**: A user can diagnose the reason for a failed example run from
  result output or scoped logs without opening runtime storage files in at least
  95% of tested failure cases.
- **SC-012**: After daemon restart, 100% of previously finished runs keep their
  statuses, stored results, and retrievable logs.
- **SC-013**: At least three examples clearly target software engineers and at
  least two examples clearly target product managers, with concrete inputs and
  output sections verified during example review.
- **SC-014**: Each example result includes at least one actionable next step,
  source reference, or explicit "no action needed" conclusion in 100% of
  successful example runs.
- **SC-015**: Every example completes its documented default run path from a
  fresh clone after installing local dependencies, with no external service
  configuration, in 100% of example smoke tests.
- **SC-016**: Every example folder includes a README with dependency
  installation, apply, run, result, and logs instructions in 100% of example
  documentation checks.
- **SC-017**: Optional API keys, when documented, are not required for the
  default successful run in 100% of example smoke tests.
- **SC-018**: Every scheduled example answers why it should run repeatedly by
  reporting new items, changed trends, recurring pain, or a no-change conclusion
  in 100% of successful runs.

## Assumptions

- The feature extends the existing local single-user daemon and CLI model from
  the Agent Definition Runtime feature.
- The definition listing command may be exposed as either `agentd ls` or
  `agentd list`; the user-visible behavior is the same.
- Machine-readable CLI output can use a single documented format selected by a
  flag or output option; the exact format is a planning decision.
- The public Go client can be created by moving or wrapping the existing daemon
  client into an importable package, provided internal server implementation
  details remain private.
- Same-host access is sufficient for the current platform; remote access,
  multi-user authorization, tokens, and role-based permissions are out of scope.
- Example agents may require public network access for public sources, but the
  default documented path must not require external-service credentials.
- Optional API keys may improve rate limits or richer source access, but missing
  optional keys should not prevent the default example run.
- Result retention follows the existing local runtime retention policy unless a
  later planning phase defines a stricter limit.
- The website screenshot example may restrict unsafe or private-network URLs by
  default to protect the local machine.
- Command-line tools are intentionally limited to explicitly declared programs
  located with the example definition for this feature.
- Example tools may access public websites, user-provided files, local
  repository data, or user-provided text exports according to each definition's
  declared permissions.
