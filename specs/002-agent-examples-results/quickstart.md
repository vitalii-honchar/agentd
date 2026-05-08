# Quickstart: Agent Examples and Results

This quickstart verifies the planned feature from a fresh clone after local
dependencies are installed.

## 1. Build and start the daemon

```bash
go test ./...
go build ./cmd/agentd ./cmd/agentdserver
agentdserver
```

The daemon listens on `127.0.0.1` by default and rejects non-same-host requests.

## 2. Apply examples

```bash
agentd apply examples/cybersecurity-reddit-watch/cybersecurity-reddit-watch.md
agentd apply examples/hacker-news-builder-brief/hacker-news-builder-brief.md
agentd apply examples/reddit-customer-pain-monitor/reddit-customer-pain-monitor.md
agentd apply examples/product-hunt-launch-radar/product-hunt-launch-radar.md
agentd apply examples/github-trending-engineering-radar/github-trending-engineering-radar.md
agentd apply examples/developer-dependency-release-monitor/developer-dependency-release-monitor.md
agentd apply examples/ai-engineering-hiring-signal-monitor/ai-engineering-hiring-signal-monitor.md
agentd apply examples/website-snapshot-analyst/website-snapshot-analyst.md
agentd list
```

Expected:
- Each applied example is listed.
- Scheduled examples show daily cron schedule metadata.
- Website snapshot shows manual schedule mode.

## 3. Run a manual example

```bash
agentd execute website-snapshot-analyst --input url=https://example.com --output json
agentd ps
agentd ps -a
```

Expected:
- `execute` returns a run ID.
- `ps` shows the active run while it is running.
- `ps -a` shows the run after completion or failure.

## 4. Retrieve results

```bash
agentd result website-snapshot-analyst
agentd result <run-id>
agentd result <run-id> --output json
```

Expected:
- Agent-name lookup returns a readable table with trimmed result text.
- Run-ID lookup returns the full untrimmed result or failed-run explanation.
- JSON output includes stable fields for automation.

## 5. Inspect scoped logs

```bash
agentd logs website-snapshot-analyst --run <run-id>
```

Expected action names include prompt submission, tool start, tool terminal
outcome, result persistence, and run terminal outcome.

## 6. Verify Bash automation

```bash
run_id=$(agentd execute website-snapshot-analyst --input url=https://example.com --output json | jq -r .run_id)
agentd result "$run_id" --output json | jq -r .result
```

Expected:
- Script can capture run ID and retrieve the final result without parsing a
  human table.

## 7. Verify Go integration

Create a small local Go program using `pkg/agentdclient`:

```go
client := agentdclient.New(agentdclient.Config{ServerURL: "http://127.0.0.1:18080"})
run, err := client.Execute(ctx, "website-snapshot-analyst", map[string]string{"url": "https://example.com"})
result, err := client.ResultByRunID(ctx, run.RunID)
```

Expected:
- The program imports only `pkg/agentdclient`.
- It can execute an agent and retrieve result data through typed methods.

## 8. Example smoke test rule

For every example folder:

```bash
agentd apply examples/<agent-name>/<agent-name>.md
agentd execute <agent-name> --output json   # scheduled examples may use manual execute for smoke tests
agentd result <agent-name>
agentd logs <agent-name> --tail 100
```

Expected:
- No required API keys.
- No external account setup.
- No CI/SaaS/private data setup.
- README documents any local dependencies and optional enhancements.

## 9. Example catalog dependency notes

- `cybersecurity-reddit-watch`: Python 3.10+ and optional `praw`; default
  public JSON fallback requires no Reddit credentials.
- `hacker-news-builder-brief`: Python 3.10+ only; uses the public Hacker News
  Firebase API.
- `reddit-customer-pain-monitor`: Python 3.10+ only; uses bundled subreddit
  sources and public Reddit JSON.
- `product-hunt-launch-radar`: Python 3.10+ only; default smoke path uses a
  bundled Product Hunt fixture.
- `github-trending-engineering-radar`: Python 3.10+ only; optional
  `GITHUB_TOKEN` can raise public API rate limits.
- `developer-dependency-release-monitor`: Python 3.10+ only; uses public npm,
  PyPI, and GitHub release endpoints with fixtures as fallback.
- `ai-engineering-hiring-signal-monitor`: Python 3.10+ only; default smoke path
  uses bundled public-source fixtures.
- `website-snapshot-analyst`: Node.js 20+ and `npm install puppeteer` for live
  screenshots; fixture fallback supports parser/catalog smoke tests.

## 10. Linux/macOS parity checklist

Run this checklist on Linux and macOS before release:

```bash
go test ./internal/agentdserver/infra/runtime -run TestProcessToolExecutorCancelsProcessGroupOnUnix
go test ./tests/e2e -run TestManagerRecoveryInterruptsActiveToolProcess
node --check examples/website-snapshot-analyst/tools/capture_website.js
python3 -m py_compile examples/*/tools/*.py
```

Expected:
- Tool timeout or recovery cancels the full process group, including child
  processes spawned by shell scripts.
- Tool stdout is captured as result context and stderr is captured as diagnostic
  context without mixing with the final LLM result.
- Website screenshot tooling is optional for parser/catalog smoke tests, but a
  live screenshot run requires Node.js 20+ and local Puppeteer dependencies.
- Python examples use only Python 3 standard library on the default path, except
  optional `praw` for authenticated Reddit reads.
