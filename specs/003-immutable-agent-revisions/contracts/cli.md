# CLI Contract: Immutable Agent Revisions

## Apply

Command:

```bash
agentd apply path/to/agent.md
```

Behavior:
- Creates a finalized immutable revision when runtime content changed.
- Returns the existing latest revision when runtime content is unchanged.
- Output includes agent name, outcome, revision ID, artifact path, revision
  status, and whether the revision was reused.

Text output example:

```text
APPLIED example-agent
OUTCOME created
REVISION 6f1918f2-37ef-4f33-a5cc-8a8475e4f0fe
ARTIFACT data/work/example-agent/6f1918f2-37ef-4f33-a5cc-8a8475e4f0fe
STATUS finalized
REUSED false
```

## Run Latest Revision

Command:

```bash
agentd run example-agent
```

Behavior:
- Resolves `example-agent` to the latest finalized revision.
- Creates a run with `agent_revision` set to that revision ID.
- Text output is `<status> <agent_name> <run_id>`.

## Run Explicit Revision

Command:

```bash
agentd run example-agent:6f1918f2-37ef-4f33-a5cc-8a8475e4f0fe
```

Behavior:
- Runs the requested finalized revision.
- Fails with not-found if the revision ID does not belong to the agent.
- Fails with corrupted-revision if required artifact files are missing.
- `agentd execute <agent-name[:revision]>` remains available as the
  compatibility command path and uses the same selector semantics.

## List Revisions

Command:

```bash
agentd revisions example-agent
```

Text columns:
- `REVISION`
- `STATUS`
- `CREATED`
- `LATEST`
- `SOURCE`
- `ARTIFACT`

The text output is tab-separated without a header. JSON output returns
`{"revisions":[...]}`.

## Inspect Revision

Command:

```bash
agentd inspect example-agent:6f1918f2-37ef-4f33-a5cc-8a8475e4f0fe
```

Behavior:
- Shows captured prompt, tool metadata, copied files, env variable names, and
  artifact path.
- Shows each tool kind as `custom_tool`, `host_tool`, or `mcp_server`.
- Shows rewritten artifact-local commands for `custom_tool` and host executable
  commands for `host_tool`.
- Masks environment values by default.
- JSON output returns `{"revision":{...}}` with `tools`, `artifact_files`, and
  `environment` arrays.

## Logs

Command:

```bash
agentd logs example-agent --run <run-id>
```

Behavior:
- Tool completion and failure action entries include captured stdout, stderr,
  result summaries, exit state, timeout state, and errors.
- The same tool execution evidence is emitted to agentdserver structured logs
  with agent name, run ID, revision ID, tool name, and event name.
- Text log lines include run ID when available so tool output can be correlated
  with a specific revision execution.
