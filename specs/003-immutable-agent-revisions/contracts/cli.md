# CLI Contract: Immutable Agent Revisions

## Apply

Command:

```bash
agentd apply path/to/agent.md
```

Behavior:
- Creates a finalized immutable revision when runtime content changed.
- Returns the existing latest revision when runtime content is unchanged.
- Output includes agent name, outcome, revision ID, and artifact path.

Text output example:

```text
APPLIED example-agent
OUTCOME created
REVISION 6f1918f2-37ef-4f33-a5cc-8a8475e4f0fe
ARTIFACT data/work/example-agent/6f1918f2-37ef-4f33-a5cc-8a8475e4f0fe
```

## Run Latest Revision

Command:

```bash
agentd run example-agent
```

Behavior:
- Resolves `example-agent` to the latest finalized revision.
- Creates a run with `agent_revision` set to that revision ID.

## Run Explicit Revision

Command:

```bash
agentd run example-agent:6f1918f2-37ef-4f33-a5cc-8a8475e4f0fe
```

Behavior:
- Runs the requested finalized revision.
- Fails with not-found if the revision ID does not belong to the agent.
- Fails with corrupted-revision if required artifact files are missing.

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
