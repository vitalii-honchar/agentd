# Quickstart: Immutable Agent Revisions

## Verify Apply Creates a Self-Contained Revision

1. Create a temporary agent folder with a Markdown definition and a local script
   tool under `tools/`.
2. Run:

   ```bash
   agentd apply /tmp/example-agent/example-agent.md
   ```

3. Confirm output includes a UUID revision and an artifact path under
   `data/work/<agent-name>/<revision_id>`.
4. Delete `/tmp/example-agent`.
5. Run:

   ```bash
   agentd inspect example-agent:<revision-id>
   ```

6. Confirm the prompt, tool declaration, env variable names, copied script, and
   artifact path remain available.

## Verify Explicit Revision Execution

1. Apply the same agent.
2. Change its prompt or script output.
3. Apply it again.
4. Run the old revision:

   ```bash
   agentd run example-agent:<old-revision-id>
   ```

5. Run the latest revision:

   ```bash
   agentd run example-agent
   ```

6. Retrieve both results and confirm each used its own captured prompt/tool
   content.

## Verify Source Folder Deletion

1. Apply an agent whose tool command references `tools/fetch.sh`.
2. Delete the source folder.
3. Run:

   ```bash
   agentd run example-agent:<revision-id>
   agentd logs example-agent --run <run-id>
   ```

4. Confirm the run succeeds and logs show the tool's stdout/stderr summaries.

## Verify Corruption Detection

1. Apply an agent revision.
2. Remove the copied script from the revision artifact directory.
3. Run:

   ```bash
   agentd run example-agent:<revision-id>
   ```

4. Confirm the daemon rejects the run before process start with a corrupted
   revision error that names the missing artifact path.

## Codex Manual Verification: GitHub Trending Example

Codex must perform this manual verification after implementation. Use the
GitHub Trending Engineering Radar example from the repository:

```bash
agentd apply examples/github-trending-engineering-radar/github-trending-engineering-radar.md
agentd run github-trending-engineering-radar
agentd result github-trending-engineering-radar
agentd logs github-trending-engineering-radar --run <run-id>
```

If the CLI command is still temporarily named `execute`, use:

```bash
agentd execute github-trending-engineering-radar
```

Expected result:

- The agent applies successfully and creates or reuses an immutable revision.
- The agent run completes successfully.
- The result contains information returned from GitHub public repository/trend
  sources.
- The run logs include tool execution entries with captured stdout and stderr
  summaries, result summary, exit state, timeout state, and error details if
  any occurred.
- The agentdserver logs include the same tool execution entry, result summary,
  stdout/stderr summaries, exit state, timeout state, and any error details.

Codex verification on 2026-05-08:

- Result: passed after increasing the tool process stdout/stderr summary budget
  to 8000 characters so the LLM receives enough GitHub repository data.
- Applied revision: `e578b123-9140-45d1-921f-903c750526f1`.
- Run ID: `b1a88c07-2d7a-4b30-84b1-f3fa125850d3`.
- Agent result included GitHub-derived repository recommendations including
  `ollama/ollama`, `rustdesk/rustdesk`, `facebook/react`,
  `freeCodeCamp/freeCodeCamp`, `openclaw/openclaw`, `public-apis/public-apis`,
  `golang/go`, and `rust-lang/rust`.
- `agentd logs github-trending-engineering-radar --run <run-id>` showed
  `tool.execute.start`, `tool.execute.complete`, captured GitHub JSON stdout,
  result summary, `exit_code: 0`, and completion events.
- agentdserver structured logs showed `tool.execute.complete` with agent name,
  run ID, revision ID, tool name, `tool_kind=custom_tool`, stdout, result,
  stderr, exit code, and timeout state.

## Automated Verification

Run:

```bash
go test ./...
```

Codex verification on 2026-05-08:

- Command: `go test ./...`
- Result: passed.
- Notes: an initial run exposed a race in the structured runtime log test where
  global logs from another parallel test were matched first; the assertion now
  filters by the target run ID and the full suite passes.
