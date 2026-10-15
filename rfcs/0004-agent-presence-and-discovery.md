# RFC 0004: Agent Participation and Session Discovery

- **Status:** Draft
- **Authors:** Harness Projects contributors
- **Created:** 2026-10-15

## Summary

An agent that begins work in a Harness Project registers its current provider
session so other agents and the Coordinator can discover who is working, where,
and on what. Registration includes the harness type, the actual provider
session ID, responsibility, modules, and capabilities. Participation is a
durable project fact. A separate local delivery lease helps the Coordinator
decide whether a CLI resume is safe to attempt.

## Motivation

The `agents/` directory describes durable roles, while `runs/` records
execution attempts. Neither answers whether a Codex, Claude Code, Cursor, or
other agent session has participated in the project. Discovery is a
prerequisite for choosing a recipient for a handoff or future dispatch.

## Scope and non-goals

This RFC defines project participation and a local delivery hint. It does not
define agent-to-agent messaging or claim a provider process is running.
Dispatch still follows RFC 0003 and must select a session explicitly.

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

Each registration has a stable ID derived from the canonical harness
name and provider session ID. Its record includes:

- `harness` and `sessionId` for provider identity;
- `worktree` and optional `taskId` for project placement;
- `focus`, `modules`, and `capabilities` for self-described responsibility;
- `registeredAt` and `updatedAt` for participation history.

Registration is idempotent for the same harness and session ID: a later call
updates the description while preserving the original registration
time. A session used in two projects has a separate registration in each
project. A run ID is not a registration ID or a provider session ID.

### Discovery and lifecycle

`hp agent discover --project .` returns all registered participants. It MAY
filter by harness, module, or capability. There is no project-level online,
offline, or expiration state. `--include-stale` is accepted for compatibility
but has no effect on durable discovery.

Agents MAY refresh a machine-local delivery hint when they want the Coordinator
to attempt direct CLI delivery:

```sh
hp agent heartbeat --project . --id <registration-id> --availability available
```

`hp agent unregister` ends that local delivery hint and retains the durable
participation record. The reference CLI uses a two-hour default hint and
permits one minute to seven days. `hp agent prune` removes old local hints
only. Neither command deletes participant history.

### Storage and routing boundary

Durable participant records live under `agents/sessions/` and are tracked by
project Git. Machine-local delivery hints live under
`.harness-project/agent-sessions/` and are ignored by Git. Roles, worklogs,
handoffs, tasks, and runs keep their existing meanings. The CLI is the write
interface for both records.

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
- **THEN** the registration shows its session ID and focus

### Expired local delivery hint

- **GIVEN** a Codex agent registered and its local delivery hint expires
- **WHEN** another agent discovers project participants
- **THEN** the Codex session still appears
- **AND** direct CLI delivery requires a refreshed local hint or falls back to inbox

### Harness does not expose an ID

- **GIVEN** a harness cannot provide its actual provider session ID
- **WHEN** its entry prompt asks it to register
- **THEN** it reports the limitation rather than creating a false identity
- **AND** project work can continue through durable project files

## Compatibility and migration

Existing Harness Projects remain valid without a participant directory.
Registering the first session creates `agents/sessions/` on demand. Old local
delivery hints do not automatically become durable participation records;
the actual session must register again. Existing role documents and run records
remain unchanged.

## Open questions

- How should a remote Coordinator verify direct delivery across machines?
- Which harnesses can expose their current session ID to an in-session agent?
- Should provider transports advertise verified routing capabilities in a
  separate registry instead of on the presence record?
