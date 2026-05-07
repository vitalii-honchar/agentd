<!--
Sync Impact Report
Version change: 1.0.0 -> 1.1.0
Modified principles:
- I. Daemon-First Agent Runtime
- II. Isolation and Least Privilege
- III. Linux and macOS Portability
- IV. Durable State and Recovery
- V. Observable and Tested Operations
Added principles:
- VI. Simplicity and Clean Architecture
Added sections:
- None
Removed sections:
- None
Templates requiring updates:
- ✅ .specify/templates/plan-template.md
- ✅ .specify/templates/spec-template.md
- ✅ .specify/templates/tasks-template.md
- ✅ .specify/templates/commands/*.md (directory not present)
Follow-up TODOs:
- None
-->
# Agentd Constitution

## Core Principles

### I. Daemon-First Agent Runtime
Agentd MUST be designed as a long-running system daemon that manages AI agent
workloads through explicit lifecycle operations: create, start, observe, stop,
restart, and remove. CLI, API, and UI surfaces MUST delegate runtime authority
to the daemon instead of duplicating execution logic. Agent execution behavior
MUST be described as a stable control-plane contract before implementation.

Rationale: container-like agent execution needs one accountable runtime owner so
resource cleanup, policy enforcement, and user-facing state remain consistent.

### II. Isolation and Least Privilege
Agent workloads MUST run with the narrowest practical host access. Filesystem
mounts, environment variables, network access, credentials, process privileges,
and device access MUST be explicit in the workload configuration. Host root,
unbounded filesystem access, inherited secrets, and unrestricted networking
MUST NOT be defaults. Any elevated capability MUST be documented with the
specific user value, risk, and audit signal.

Rationale: executing AI agents can run untrusted or semi-trusted commands, so
host safety is a core product requirement rather than an optional hardening step.

### III. Linux and macOS Portability
Linux and macOS are the primary supported operating systems. Features that touch
process management, sandboxing, networking, filesystems, permissions, service
installation, or signals MUST define expected behavior on both platforms.
Platform-specific code MUST be isolated behind small adapters with tests or
documented verification. If parity is not technically possible, the plan MUST
state the user-visible difference and the fallback behavior.

Rationale: the daemon must be dependable on developer laptops and production
Linux hosts without hiding platform differences until release time.

### IV. Durable State and Recovery
Daemon state that affects running or recoverable work MUST be persisted through
well-defined schemas. Operations that create, mutate, or delete workloads MUST be
idempotent or include explicit conflict handling. The daemon MUST support
graceful shutdown, restart recovery, orphan process detection, and cleanup of
temporary resources. Schema changes MUST include migration and rollback notes
when existing state can be affected.

Rationale: a runtime service loses trust if restart, crash, or repeated commands
leave users with invisible work, stale locks, or unmanaged host processes.

### V. Observable and Tested Operations
Daemon lifecycle events, agent lifecycle events, policy decisions, resource
limits, security-relevant actions, and failures MUST emit structured logs or
events with stable identifiers. Features that affect daemon control APIs,
agent execution, isolation policy, persisted state, or cross-platform behavior
MUST include automated tests and a quickstart or manual verification path.
Failures MUST return actionable errors that identify the failed operation and
the remediation path when one exists.

Rationale: operators and developers need enough evidence to debug agent runs,
enforce policy, and verify regressions before releasing runtime changes.

### VI. Simplicity and Clean Architecture
Implementations MUST choose the simplest design that satisfies current
requirements and preserves clear change boundaries. Premature optimization,
speculative abstractions, framework additions, and cross-cutting indirection
MUST NOT be introduced without a measured bottleneck, a concrete near-term use
case, or a documented reduction in complexity.

Code MUST follow SOLID design, Clean Code readability, and Clean Architecture
dependency direction: domain and runtime policy MUST remain independent from
transport, persistence, platform adapters, and presentation details. Modules
MUST expose small interfaces, keep responsibilities explicit, and make invalid
states hard to represent.

Rationale: a daemon runtime needs predictable boundaries more than clever
generalization. Simple code is easier to audit, port across Linux and macOS,
secure, test, and recover after operational failures.

## Runtime Constraints

Agentd is a local-first daemon service for container-like AI agent execution.
The minimum supported host targets are Linux and macOS. Runtime features MUST
define their process model, resource ownership, cleanup path, and host access
policy before implementation.

The daemon MUST keep runtime policy centralized. Workload definitions MUST be
data-driven and reviewable so access to files, network, credentials, and host
capabilities can be audited. Background work MUST be cancellable, bounded by
configured limits, and observable without attaching a debugger.

## Development Workflow

Every feature plan MUST pass the Constitution Check before research begins and
again after design. The check MUST cover daemon lifecycle impact, isolation
policy, Linux/macOS behavior, persisted state and recovery, observability,
required verification, simplicity, and clean architecture boundaries.

Specifications MUST describe user-visible behavior and failure modes without
depending on a specific implementation. Tasks MUST include tests or verification
steps for every constitution-sensitive behavior: daemon APIs, agent execution,
sandboxing, persistence, recovery, and platform-specific adapters.

## Governance

This constitution supersedes conflicting project practices and templates.
Amendments require a documented rationale, an explicit semantic version bump,
and updates to affected templates or runtime guidance in the same change.

Versioning policy:
- MAJOR: incompatible changes to principles or governance requirements.
- MINOR: new principles, new sections, or materially expanded requirements.
- PATCH: clarifications and wording changes that do not alter obligations.

Compliance review is required for every feature plan and implementation review.
Any approved deviation MUST be recorded in the plan's Complexity Tracking table
with the reason, rejected simpler alternative, and mitigation.

**Version**: 1.1.0 | **Ratified**: 2026-05-07 | **Last Amended**: 2026-05-07
