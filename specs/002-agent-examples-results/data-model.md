# Data Model: Agent Examples and Results

## Agent Definition

Represents an applied Markdown definition.

**Fields**:
- `name`: unique local agent name.
- `enabled`: whether scheduler/manual execution is allowed.
- `schedule_type`: `cron` or `manual`.
- `schedule_expression`: required for `cron`, omitted for `manual`.
- `vendor_name`, `vendor_model`: configured LLM provider/model.
- `prompt`: agent instructions.
- `tools`: declared local command-line tools.
- `access`: declared filesystem/network permissions.
- `source_path`, `definition_markdown`, `revision`.

**Validation**:
- Name follows existing lowercase name rules.
- Scheduled monitoring examples use daily cron schedules by default.
- Manual examples use `manual` schedule and declare required run-time inputs in
  README and definition metadata.
- Tool declarations must reference files inside the example folder or other
  explicitly allowed paths.

## Agent Run

Represents one execution attempt.

**Fields**:
- `id`: UUID.
- `agent_name`, `agent_revision`.
- `trigger`: `schedule`, `manual`, or `recovery`.
- `status`: `queued`, `running`, `completed`, `failed`, `stopping`, `stopped`,
  or `interrupted`.
- `due_at`, `started_at`, `completed_at`.
- `work_dir`, `log_path`.
- `result`: full successful or failed run output.
- `result_summary`: trimmed text for tables.
- `error_code`, `error_message`.
- `provider_request_id`.

**State transitions**:
- `queued -> running -> completed`
- `queued -> running -> failed`
- `running -> stopping -> stopped`
- `queued|running -> interrupted` during recovery

**Validation**:
- Terminal runs must have `completed_at`.
- `completed` and `failed` runs must store a result.
- Result summaries must be generated from the full result and must not replace
  the full result.

## Run Result

Represents retrievable output for automation and users.

**Fields**:
- `run_id`, `agent_name`.
- `status`.
- `trigger`.
- `started_at`, `completed_at`.
- `result`.
- `result_summary`.
- `failure`: optional code/message/details.

**Validation**:
- Result lookup by agent name returns terminal runs for that agent.
- Result lookup by run ID returns full details for one run.
- Active runs return a clear no-final-result response.

## Tool Declaration

Represents a local command-line program an agent may invoke.

**Fields**:
- `name`.
- `command`.
- `args`.
- `env`.
- `read_paths`, `write_paths`.
- `network_allow`.
- `timeout`.

**Validation**:
- Tool command must be declared before execution.
- Example tool paths must stay within the example folder unless explicitly
  documented.
- Secret-bearing env/files are not inherited by default.

## Tool Execution

Represents one process invocation during a run.

**Fields**:
- `id`.
- `run_id`, `agent_name`, `tool_name`.
- `command_summary`.
- `started_at`, `completed_at`.
- `exit_code`.
- `timed_out`.
- `stdout_summary`, `stderr_summary`.
- `error_message`.

**Validation**:
- Every invocation emits tool-start and terminal tool action logs.
- Non-zero exit or timeout marks the tool execution failed and contributes to a
  failed Agent Run unless the agent definition explicitly treats it as optional.

## Run Log Entry

Represents a scoped action event for a run.

**Fields**:
- `timestamp`.
- `agent_name`, `run_id`.
- `action`: stable action name such as `llm.prompt.send`,
  `tool.execute.start`, `tool.execute.complete`, `run.result.persisted`,
  `run.complete`, or `run.fail`.
- `level`.
- `message`.
- `attributes`.

**Validation**:
- Logs are scoped by agent/run and must not mix unrelated agents.
- Logs summarize tool results without dumping secret values.

## Public Client

Represents the importable Go integration surface.

**Fields/operations**:
- Client config: `ServerURL`, timeout, optional HTTP client.
- Agent operations: apply, list, inspect.
- Run operations: execute, list active/all, stop, result by agent, result by run.
- Log operations: logs by agent/run.

**Validation**:
- Uses same REST contract as CLI.
- Does not expose internal daemon packages.
- Returns typed errors with stable daemon error codes.

## Example Agent Folder

Represents one repository example.

**Fields/files**:
- `<agent-name>.md`: Agent Definition.
- `README.md`: install, apply, run, result, and logs instructions.
- `tools/`: declared CLI tools.
- `fixtures/` or `sources/`: bundled public-source lists or sample fixtures.

**Validation**:
- Default run path requires no external account, required API key, CI setup,
  SaaS integration, private data, or user-specific remote configuration.
- README lists local dependencies and optional API keys only as enhancements.
