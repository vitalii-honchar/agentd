# Contributing to agentd

Thanks for helping improve agentd. This project is early-stage, so focused issues and small pull requests are easiest to review.

## Development Setup

Requirements:

- Go 1.26.2 or newer in the same Go 1.26 release line
- Linux or macOS
- No external database; agentd uses embedded SQLite through `modernc.org/sqlite`

Setup:

```bash
git clone git@github.com:vitalii-honchar/agentd.git
cd agentd
cp .env.example .env
go mod download
go test ./...
```

Only set `OPENAI_API_KEY` when you need to execute OpenAI-backed Agents locally.

## Workflow

- Open an issue before starting large behavior changes.
- Keep pull requests narrowly scoped.
- Add or update tests for behavior changes.
- Run `go test ./...` before opening a pull request.
- Do not commit `.env`, local databases, logs, generated runtime data, API keys, tokens, or private keys.

## Project Structure

- `cmd/agentd`: CLI entrypoint.
- `cmd/agentdserver`: daemon entrypoint.
- `internal/agentd`: CLI application and HTTP client code.
- `internal/agentdserver`: daemon application, domain, infrastructure, runtime, storage, and HTTP code.
- `docs/`: contributor-facing development and operations docs.
- `examples/`: sample Agent Definitions with no secret values.
- `specs/`: Spec Kit artifacts retained for design history.

## Pull Request Checklist

- Tests pass with `go test ./...`.
- User-facing docs are updated when behavior changes.
- Agent Definition examples contain only environment variable names, never secret values.
- New files use the Apache-2.0 project license unless noted otherwise.
