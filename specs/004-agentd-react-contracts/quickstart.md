# Quickstart: Unified Agentd ReAct Contracts

## Build

```bash
go test ./...
go build ./cmd/agentd
```

Only `agentd` is required for normal local use.

## Start Daemon Mode

```bash
./agentd --daemon
```

Compatibility aliases:

```bash
./agentd -d
./agentd --deamon
```

## Apply And Run A Scheduled Contracted Example

```bash
./agentd apply examples/github-trending-engineering-radar/github-trending-engineering-radar.md
run_id=$(./agentd run github-trending-engineering-radar --output json | jq -r .run_id)
./agentd result "$run_id" --output json
./agentd logs "$run_id"
```

Expected:
- The example applies with `contract.input` accepting `{}`.
- The run result is JSON that validates against `contract.output`.
- Logs contain only events for `$run_id`.

This command must fail because logs are run-scoped:

```bash
./agentd logs github-trending-engineering-radar
```

Expected:

```text
logs require an agent run ID
```

## Apply And Run A Manual Contracted Example

```bash
./agentd apply examples/website-snapshot-analyst/website-snapshot-analyst.md
run_id=$(./agentd run website-snapshot-analyst --input-json '{"url":"https://example.com"}' --output json | jq -r .run_id)
./agentd result "$run_id" --output json
./agentd logs "$run_id"
```

Invalid input fails before the run starts:

```bash
./agentd run website-snapshot-analyst --input-json '{"url":"not a url"}'
```

Expected:
- Non-zero exit.
- Actionable contract validation error.
- No model request or tool execution.

## Verify Legacy Optional Contract Behavior

Apply a legacy fixture with no `contract` block:

```bash
./agentd apply tests/fixtures/legacy-no-contract-agent.md
run_id=$(./agentd run legacy-no-contract-agent --output json | jq -r .run_id)
./agentd result "$run_id"
```

Expected:
- Agent remains valid.
- Input-contract validation is not applied.
- Output-contract finalization is not applied.

## Verify Codex Provider

Confirm Codex CLI is installed:

```bash
codex exec --help
```

Optional provider configuration:

```bash
export AGENTD_CODEX_COMMAND=codex
export AGENTD_CODEX_MODEL=gpt-5.4-mini
export AGENTD_CODEX_PROFILE=agentd
export AGENTD_CODEX_TIMEOUT=10m
```

`AGENTD_CODEX_MODEL` is a provider default; an agent's `vendor.model` still
selects the model when the env var is empty. `AGENTD_CODEX_PROFILE` is only
passed when set. Agentd uses local Codex CLI auth and does not extract or store
Codex tokens.

Apply and run a small Codex-backed fixture:

```bash
./agentd apply tests/fixtures/codex-provider-agent.md
run_id=$(./agentd run codex-provider-agent --input-json '{"topic":"agentd"}' --output json | jq -r .run_id)
./agentd result "$run_id" --output json
./agentd logs "$run_id"
```

Expected:
- The run uses local Codex CLI authentication/configuration through
  `codex exec --json`.
- No direct OpenAI API key is required for this provider path.
- The result validates against `contract.output`.
- Run logs include provider failures when setup/auth/process execution fails
  and output finalization events when the run succeeds.

If Codex is not logged in:

```bash
./agentd run codex-provider-agent --input-json '{"topic":"agentd"}'
```

Expected:
- Run fails with an actionable message such as `run codex login before using
  vendor.name: codex`.

If the CLI is unavailable or a process times out:
- Missing CLI fails with `codex CLI not found`.
- Timeout fails with `codex CLI timed out`.
- Non-auth stderr is truncated before being returned in the run error.

## Regression Checklist

```bash
go test ./...
go test ./internal/agentd/app -run 'Logs|Execute|Run'
go test ./internal/agentdserver/app/runtime -run 'Contract|React|Codex'
go test ./internal/agentdserver/app/logs -run Logs
go test ./internal/agentdserver/infra/definition -run Contract
go test ./internal/agentdserver/infra/llm/codex
go test ./tests/e2e -run 'Contract|Logs|UnifiedBinary|CodexProvider'
```

Verification record:
- 2026-05-09: `go test ./...` passed.
- During verification, `tests/e2e/cli_operations_test.go` still used the old
  `agentd logs <agent-name>` command shape; it was updated to
  `agentd logs <run-id>` and the full suite was rerun successfully.
- 2026-05-09: `go build -o /tmp/agentd-verify ./cmd/agentd` passed.
- 2026-05-09: `/tmp/agentd-verify --help` showed the unified `agentd` command,
  `--daemon`, `--deamon`, and the run-scoped `logs <run_id>` command text.
- Provider-backed manual quickstart runs were not executed against live OpenAI
  or real Codex CLI credentials in this session; equivalent contracted
  examples, run logs, and Codex provider behavior were covered by automated e2e
  tests with fake providers/CLI.
