# Research: Immutable Agent Revisions

## Decision: Store revision metadata in settings DB and copied files on disk

**Rationale**: The settings DB already owns applied agent definitions and policy.
It should own revision identity, status, metadata, and lookup, while copied
scripts and fixtures remain normal files under `data/work/<agent-name>/<revision_id>`.
This keeps large or executable content out of SQLite and matches the user's
requested artifact layout.

**Alternatives considered**:
- Store full blobs in SQLite: rejected because it makes script execution and
  inspection awkward and contradicts the requested file artifact model.
- Keep only source paths in SQLite: rejected because it is not immutable or
  self-contained after source deletion.

## Decision: Create revisions idempotently by runtime content digest

**Rationale**: Apply should behave like building an image: changed runtime
content creates a new immutable revision; unchanged runtime content reuses the
latest matching revision. A content digest over prompt, normalized tool
metadata, env material, and copied file checksums provides a deterministic
comparison while keeping the user-facing revision ID as a UUID.

**Alternatives considered**:
- Always create a new revision on every apply: rejected because unchanged apply
  would create noisy duplicate artifacts.
- Use a content hash as the revision ID: rejected because the user requested a
  UUID revision ID on creation.

## Decision: Treat revision artifact creation as staged then finalized

**Rationale**: Apply touches both SQLite and the filesystem. A revision starts
as `pending`, files are copied to a staging directory, checksums are written to
the manifest, and the row becomes `finalized` only after validation succeeds.
Startup recovery can clean old pending directories or mark incomplete revisions
corrupt.

**Alternatives considered**:
- Copy files first without DB state: rejected because crashes leave anonymous
  directories with unclear ownership.
- Finalize DB before copy: rejected because runs could see incomplete artifacts.

## Decision: Copy declared local scripts and declared runtime files only

**Rationale**: Least privilege requires the definition contract to say which
local files are runtime dependencies. `custom_tool` entries represent
agent-provided implementations and are copied when they use relative script
commands or declared file inputs. `host_tool` entries represent host-installed
executables such as `python3`, `node`, `bash`, `gh`, or `curl`; those executables
remain external commands and are not copied.

**Alternatives considered**:
- Copy the whole definition folder: rejected because it can capture undeclared
  secrets or unrelated files.
- Copy nothing and resolve at run time: rejected because source deletion breaks
  immutable execution.

## Decision: Use `custom_tool` and `host_tool` as tool kind names

**Rationale**: `custom_tool` clearly means the agent author provides the tool
implementation and agentd freezes it into the artifact. `host_tool` clearly
means the executable is supplied by the host OS environment. This is more
precise than `process_tool` because both kinds run as processes.

**Alternatives considered**:
- Keep `local_tool`: rejected because it conflates copied agent code with
  host-local system programs.
- Use `process_tool`: rejected because it describes the execution mechanism,
  not the ownership/source of the executable.

## Decision: Rewrite copied local tool commands to revision-local paths

**Rationale**: The runtime manager currently resolves tool paths against the
definition source. Immutable revisions must instead resolve copied
`custom_tool` scripts against the artifact directory. Rewriting at revision
creation gives runs a frozen command contract and avoids consulting removed
source folders.

**Alternatives considered**:
- Rewrite at every run: acceptable but weaker for inspection and corruption
  detection.
- Change the process working directory only: insufficient for commands that
  are stored as source-relative paths.

## Decision: Mask env values in inspection and logs

**Rationale**: Tool env values may include API keys. The daemon needs values at
execution, but users inspecting revision metadata should see variable names and
whether values are present, not raw secrets, unless a future explicit reveal
mode is designed.

**Alternatives considered**:
- Print env values in inspect output: rejected as unsafe default behavior.
- Do not persist env values: rejected because the revision would not be fully
  self-contained for tools that require env values.
