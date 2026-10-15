# RFC 0004: Agent Session Presence and Discovery

- **Status:** Draft
- **Authors:** Harness Projects contributors
- **Created:** 2026-10-15

## Summary

An agent that begins work in a Harness Project registers its current provider
session so other agents and the Coordinator can discover who is working, where,
and on what. Registration includes the harness type, the actual provider
session ID, responsibility, modules, capabilities, and a bounded presence
lease. The reference CLI provides registration, discovery, renewal, and exit
commands.

## Motivation

The `agents/` directory describes durable roles, while `runs/` records
execution attempts. Neither answers whether a Codex, Claude Code, Cursor, or
other agent session is currently participating in the project. Discovery is a
prerequisite for choosing a recipient for a handoff or future dispatch.

## Scope and non-goals

This RFC defines local project presence. It does not define agent-to-agent
messaging, prove that a provider process is still running, or grant permission
to resume a session. Dispatch still follows RFC 0003 and must select a session
explicitly. A future remote registry may synchronize presence across machines.

## Proposed contract

### Registration

When an agent starts meaningful work, the project's harness entry point or
prompt SHOULD instruct it to discover its actual provider session ID and run:

```sh
hp agent register --project . --harness codex --session-id <actual-id> \
  --modules cli,coordinator --capabilities go,testing \
  --focus "Implement session discovery"
```

The agent MUST NOT invent a session ID when its harness does not expose one.
It should report that limitation and continue to use the project files for
handoffs. `cc` is a CLI alias for `claude-code`; the stored harness name is
`claude-code`. Other harness names remain extensible.

Each registration has a stable local ID derived from the canonical harness
name and provider session ID. Its record includes:

- `harness` and `sessionId` for provider identity;
- `worktree` and optional `taskId` for project placement;
- `focus`, `modules`, and `capabilities` for self-described responsibility;
- `availability` (`available`, `busy`, or `paused`);
- `registeredAt`, `lastSeenAt`, and `expiresAt` for a bounded lease.

Registration is idempotent for the same harness and session ID: a later call
updates the description and lease while preserving the original registration
time. A session used in two projects has a separate registration in each
project. A run ID is not a registration ID or a provider session ID.

### Discovery and lifecycle

`hp agent discover --project .` returns unexpired registrations. It MAY filter
by harness, module, or capability. `--include-stale` exposes expired records
for inspection. Discovery MUST report the declared availability and whether
the lease is live; it MUST NOT claim that a live lease proves a reachable
process.

Agents SHOULD renew their lease after meaningful work and when waiting for
further tasks:

```sh
hp agent heartbeat --project . --id <registration-id> --availability available
```

On leaving a project, an agent SHOULD unregister. Expired registrations are
hidden from default discovery even if an agent fails to unregister. The
reference CLI uses a two-hour default lease and permits one minute to seven
days. An interrupted agent can renew an expired record using its ID. The CLI
also provides `hp agent prune` to remove records that have stayed expired for
seven days by default; a dry run previews the removals.

### Storage and routing boundary

Local presence records live under `.harness-project/agent-sessions/` and are
machine-local transient state. They are ignored by Git; durable roles remain
in `agents/`, and durable work outcomes remain in `worklogs/`, `handoffs/`,
tasks, and runs. The registry records are JSON so agents can inspect them, but
the CLI is the write interface.

Registration makes an agent discoverable, not automatically reachable. A task
may be durably assigned to its session without sending a message, as defined
in RFC 0005. Before attempting direct CLI delivery, the Coordinator must know
a supported transport and verify that the session can safely receive work. A
CLI adapter must not resume a session that may already be active elsewhere.

## Scenarios

### Find an agent for a module

- **GIVEN** a Claude Code agent registers with module `canvas` and capability
  `typescript`
- **WHEN** another agent discovers sessions filtered by `canvas`
- **THEN** the registration shows its session ID, focus, and availability

### Stale registration

- **GIVEN** a Codex agent stops without unregistering
- **WHEN** its lease expires
- **THEN** default discovery omits it
- **AND** `--include-stale` still allows diagnosis

### Harness does not expose an ID

- **GIVEN** a harness cannot provide its actual provider session ID
- **WHEN** its entry prompt asks it to register
- **THEN** it reports the limitation rather than creating a false identity
- **AND** project work can continue through durable project files

## Compatibility and migration

Existing Harness Projects remain valid without a presence directory.
Registering the first session creates the local directory on demand. Existing
`agents/` role documents and run records remain unchanged.

## Open questions

- How should a remote registry reconcile leases from multiple machines?
- Which harnesses can expose their current session ID to an in-session agent?
- Should provider transports advertise verified routing capabilities in a
  separate registry instead of on the presence record?
