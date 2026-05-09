# Data Model: Unified Agentd ReAct Contracts

## Agent Definition

Represents the parsed source markdown for an AI Agent.

**Fields**
- `name`: stable agent name.
- `enabled`: whether the agent may run.
- `schedule`: manual or cron schedule.
- `vendor`: model provider name and model/config selection.
- `environment`: declared environment variables and files.
- `inputs`: legacy input declarations retained for compatibility.
- `contract`: optional Agent Contract.
- `tools`: declared custom/host tools.
- `mcp_servers`: declared MCP server permissions if present.
- `access`: declared filesystem/network access policy.
- `prompt`: agent behavior prompt.
- `source_path`, `raw_markdown`: source metadata.

**Validation**
- Existing name, schedule, vendor, and prompt validation still applies.
- If `contract` is omitted, no contract validation is applied.
- If `contract` is present, both schemas must be valid JSON Schema documents.
- Scheduled agents with required contracted input fields are invalid unless the
  schema accepts the automatic scheduled input object.

## Agent Contract

Optional input/output schema pair for one agent definition and revision.

**Fields**
- `input_schema_raw`: raw JSON Schema string from `contract.input`.
- `output_schema_raw`: raw JSON Schema string from `contract.output`.
- `input_schema_digest`: digest of canonicalized input schema.
- `output_schema_digest`: digest of canonicalized output schema.
- `created_from_revision`: revision ID when persisted into immutable artifact.

**Relationships**
- Belongs to Agent Definition.
- Copied into Agent Revision.
- Used by Agent Run for input validation and output finalization.

**Validation**
- Raw schema text must parse as JSON.
- Schema must compile through the contract validator.
- Output schema must describe a JSON value the final result can validate
  against; examples should use object schemas.

## Agent Revision

Immutable applied artifact for one definition version.

**New/Changed Fields**
- `contract_input_schema_raw`: nullable.
- `contract_output_schema_raw`: nullable.
- `contract_digest`: nullable; included in content digest when contract exists.
- `provider_name`, `provider_model`: existing vendor fields remain authoritative.

**State Transitions**
- `pending` -> `finalized`: only after schema validation and artifact copy pass.
- `pending` -> `corrupt`: crash or validation/copy failure.
- `finalized`: immutable; contract cannot change in-place.

## Runtime Input

JSON object supplied for one run.

**Fields**
- `raw_json`: original JSON object used for contracted validation.
- `legacy_inputs`: string map retained for existing `--input key=value` flows.
- `source`: CLI flag, input file, public client, schedule, or internal test.

**Validation**
- Contracted agents validate `raw_json` against `contract.input` before run
  creation or side effects.
- Legacy agents may continue using string map expansion.
- Invalid contracted input fails without tool/model invocation.

## AI Agent Run

One execution attempt for an agent revision.

**New/Changed Fields**
- `input_json`: optional validated runtime input.
- `contract_input_schema_digest`: copied from revision when present.
- `contract_output_schema_digest`: copied from revision when present.
- `provider_name`: selected LLM provider.
- `provider_request_id`: existing direct provider ID or local provider process
  identifier when useful.
- `result`: final plain text or JSON string.
- `result_format`: text or json.
- `error_code`, `error_message`: includes contract/provider/log errors.

**State Transitions**
- `queued` -> `running`: run accepted after input validation.
- `running` -> `completed`: final output is available and valid when required.
- `running` -> `failed`: provider/tool/schema/finalization failure.
- `running` -> `stopping` -> `stopped`: user stop request.
- `running` -> `interrupted`: daemon restart or unrecovered process loss.

## ReAct Step

One model-controlled execution iteration.

**Fields**
- `step_index`: monotonic step number.
- `run_id`, `agent_name`, `revision_id`.
- `model_message`: safe model content for diagnostics.
- `decision`: tool_call, final, or fail.
- `tool_name`: set for tool_call.
- `tool_args_json`: arguments selected by model.
- `observation_summary`: safe tool result summary.
- `started_at`, `completed_at`.
- `error_message`: failure details if the step failed.

**Validation**
- `tool_name` must refer to a declared tool.
- Tool arguments must be valid JSON and satisfy any tool argument schema used by
  the ReAct adapter.
- Step count must stay under configured loop limits.

## Tool Invocation

Daemon-owned execution of one declared tool requested by a ReAct step.

**Fields**
- Existing tool permission fields: name, kind, command, args, env, timeout,
  read/write paths, network allow list.
- `call_id`: model/provider tool request ID or generated step call ID.
- `stdout_summary`, `stderr_summary`, `result_summary`, `exit_code`,
  `timed_out`, `error_message`.

**Validation**
- Must be declared in revision.
- `custom_tool` executes from copied revision artifact.
- `host_tool` validates explicit host executable.
- Access remains bounded by declared policy.

## Output Finalization

Final structured-output pass for contracted agents.

**Fields**
- `run_id`.
- `history_ref`: complete safe execution history.
- `output_schema_raw`.
- `attempt`: finalization/repair attempt number.
- `output_json`: candidate final JSON.
- `validation_errors`: schema validation errors.

**State Transitions**
- `pending` -> `valid`: output validates and run can complete.
- `pending` -> `retrying`: output invalid but repair attempts remain.
- `pending` -> `failed`: attempts exhausted or provider failed.

## LLM Provider

Selectable model execution backend.

**Fields**
- `name`: `openai`, `codex`, or future provider.
- `model`: provider model/profile selector.
- `config`: provider-specific non-secret settings.
- `requires_direct_api_key`: true for direct API provider; false for Codex CLI
  provider.

**Validation**
- Provider name must be registered.
- Missing provider setup fails with an actionable error.

## Codex Provider Invocation

One managed local Codex CLI provider process.

**Fields**
- `run_id`.
- `command_path`: resolved `codex` executable.
- `args`: non-interactive arguments.
- `work_dir`: run-owned provider working directory.
- `prompt_file` or stdin payload metadata.
- `output_schema_file`: optional schema file for structured output.
- `last_message_file`: captured final response file.
- `json_events_summary`: bounded summary of JSONL events.
- `exit_code`, `stderr_summary`, `timed_out`, `started_at`, `completed_at`.

**Validation**
- CLI must be available and runnable.
- Invocation must be non-interactive.
- Process must stop on context cancellation or timeout.
- Provider must not read or persist undocumented Codex tokens.

## Agent Run Log

Ordered diagnostic entries for exactly one run.

**Fields**
- `run_id`.
- `agent_name`.
- `revision_id`.
- `timestamp`.
- `action`.
- `message`.
- `line`.

**Validation**
- Lookup by run ID only.
- Unknown run ID returns not found.
- Agent-name lookup is rejected and never falls back to latest run.

## Unified Agentd Runtime

Process mode selected by the `agentd` executable.

**Fields**
- `mode`: daemon or client.
- `daemon_flags`: `--daemon`, `-d`, compatibility `--deamon`.
- `client_args`: existing command arguments.

**Validation**
- Daemon mode with client subcommands is rejected with actionable help.
- Daemon startup handles SIGINT/SIGTERM on Linux and macOS.
