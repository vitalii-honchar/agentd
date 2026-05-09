# CLI Contract

## Unified Binary

```bash
agentd --daemon
agentd -d
agentd --deamon
```

Behavior:
- Starts the daemon in the foreground with the same lifecycle previously owned
  by `agentdserver`.
- `--daemon` is canonical.
- `--deamon` is accepted as a compatibility alias.
- Daemon mode combined with client subcommands fails with usage help.

## Run With Structured Input

```bash
agentd run <agent-name> --input-json '{"url":"https://example.com"}'
agentd run <agent-name> --input-file input.json
agentd run <agent-name>:<revision-id> --input-json '{}'
```

Behavior:
- Contracted agents validate JSON input before the run starts.
- Invalid input exits non-zero and does not create a run.
- Existing `--input key=value` remains available for legacy/simple inputs.
- JSON output mode includes the run identifier:

```json
{
  "run_id": "run-uuid",
  "agent_name": "website-snapshot-analyst",
  "revision": "revision-id",
  "status": "running"
}
```

## Run Logs By Run ID

```bash
agentd logs <agent-run-id>
agentd logs <agent-run-id> --tail 100
agentd logs <agent-run-id> --output json
```

Behavior:
- Returns logs for exactly one run.
- Does not accept agent-name lookup.
- Unknown run IDs return a not-found error.
- Agent-name-looking arguments return an actionable error:

```text
logs are retrieved by run identifier; use agentd ps -a or agentd result <agent-name> to find a run_id
```

JSON output:

```json
{
  "run_id": "run-uuid",
  "agent_name": "github-trending-engineering-radar",
  "entries": [
    {
      "timestamp": "2026-05-09T10:00:00Z",
      "run_id": "run-uuid",
      "action": "llm.prompt.send",
      "message": "send LLM prompt to provider",
      "line": "send LLM prompt to provider"
    }
  ]
}
```

## Result Output For Contracted Runs

```bash
agentd result <agent-run-id> --output json
```

For contracted successful runs, `result` is valid JSON encoded according to the
declared `contract.output`.

```json
{
  "run_id": "run-uuid",
  "agent_name": "website-snapshot-analyst",
  "status": "completed",
  "result_format": "json",
  "result": {
    "website_summary": "Example Domain is a minimal placeholder website.",
    "audience": "People testing links or documentation examples.",
    "primary_call_to_action": "None",
    "trust_signals": ["Clear ownership language"],
    "issues": []
  }
}
```

## Codex Provider Diagnostics

For agents with `vendor.name: codex`, setup failures are surfaced as run
failures with actionable errors:

```text
codex provider unavailable: Codex CLI is not installed or not on PATH
codex provider unavailable: run `codex login` before using vendor.name: codex
```
