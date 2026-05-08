# Agent Definition Contract: Immutable Revision Inputs

Agent definitions continue using the existing Markdown/YAML format. At apply
time, agentd classifies runtime dependencies as follows.

## Captured Always

- Agent name
- Enabled flag
- Schedule
- Vendor
- Prompt
- Inputs
- Tool declarations
- MCP server declarations
- Access policy
- Environment declarations

## Path-Bearing Metadata Fields

Agentd resolves and freezes only these front matter fields as filesystem paths:

- `tools[].kind`: must be `custom_tool` for agent-provided implementations or
  `host_tool` for host-installed executables. `local_tool` is a legacy
  compatibility alias for `custom_tool` during migration only.
- `tools[].command`: copied only for `custom_tool` when the value is a relative
  local script path such as `tools/fetch.py`; `host_tool` commands such as
  `curl`, `gh`, `python3`, or allowed absolute host paths remain external.
- `tools[].args`: copied only for argument values that the definition contract
  marks as file inputs. Plain strings, URLs, flags, and input placeholders are
  not treated as filesystem paths by guessing.
- `tools[].read_paths`: copied when they are runtime input files or directories.
- `tools[].write_paths`: resolved for policy and rewritten to the execution
  working directory when they are relative run-output paths.
- `access.filesystem.read`: copied when the paths are runtime input files or
  directories.
- `access.filesystem.write`: resolved for policy and rewritten to the execution
  working directory when relative.
- `environment.files`: copied and parsed as `.env` files during apply.

Path-like strings in prompt text, input descriptions, network allow lists,
vendor names, model names, environment variable values, and schedule
expressions are not filesystem paths.

## Copied When Declared

- `custom_tool` relative local script commands such as `tools/fetch.py`.
- Relative local files declared by path-bearing metadata fields when the
  definition contract marks them as runtime dependencies.
- Example source lists and fixture files declared by the agent definition.
- `.env` files listed in `environment.files`.

## Not Copied

- `host_tool` PATH-resolved binaries such as `python3`, `node`, `bash`, `curl`,
  `gh`, or `npx`.
- Undeclared files in the definition directory.
- Git metadata, README files, and documentation unless declared as runtime
  inputs.
- Secret-bearing files that are not explicitly listed in `environment.files`,
  including private keys, credential stores, and certificates with private
  material.

## Environment and Secrets

Agent definitions may declare process environment material:

```yaml
environment:
  variables:
    REDDIT_CLIENT_ID: "public-or-secret-value"
    REDDIT_USER_AGENT: "agentd-example/1.0"
  files:
    - .env
    - secrets/reddit.env
```

Rules:

- `environment.variables` is a key-value map captured into the revision.
- `environment.files` is a list of `.env` file paths resolved relative to the
  definition markdown directory, copied into the revision artifact, and parsed
  at apply time.
- Captured environment entries are supplied to local tools during execution.
- Declared literal variables override values parsed from declared `.env` files.
- Undeclared host environment variables are not inherited by default, except
  minimal OS process variables explicitly allowed by daemon policy.
- Default inspection and logs show environment variable names and masked values
  only.
- `.env` files are allowed to contain secrets because they are explicitly
  declared runtime material, but they must remain ignored by Git and must not be
  printed raw in logs.

Tool-level `tools[].env` remains supported for tool-specific entries. If both
global `environment` and `tools[].env` define the same key, the tool-level value
wins for that tool execution.

## Tool Kind Examples

Agent-provided implementation copied into the artifact:

```yaml
tools:
  - name: fetch_github_trending
    kind: custom_tool
    command: tools/fetch_github_trending.py
    args: ["--languages", "sources/languages.txt"]
```

Host-installed executable not copied into the artifact:

```yaml
tools:
  - name: github_api
    kind: host_tool
    command: gh
    args: ["api", "search/repositories", "-f", "q=stars:>10000"]
```

Naming decision:

- `custom_tool` means the agent author provides the implementation and agentd
  copies it into the immutable revision.
- `host_tool` means the tool is supplied by the host OS environment and agentd
  validates/invokes it without copying it.
- `host_tool` is preferred over `process_tool` because both custom and host
  tools run as processes; the meaningful distinction is artifact-owned code
  versus host-owned executable.

## Path Resolution

Relative paths are resolved from the definition markdown directory during
apply. Finalized revision metadata stores rewritten artifact-local paths for
copied files.

## Execution Working Directory

The AI Agent process shares the same filesystem namespace as the host OS. Agentd
does not create a full filesystem sandbox in this feature. For each run, agentd
sets the process working directory to:

```text
data/work/<agent_name>/executions/<execution_id>
```

Local tools load copied scripts and declared read-only runtime inputs from the
immutable revision artifact. Relative write paths target the execution working
directory so generated files are per-run outputs rather than revision mutations.
