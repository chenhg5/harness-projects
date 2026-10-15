# RFC 0005: Durable Task Assignment and Session Inbox

- **Status:** Draft
- **Authors:** Harness Projects contributors
- **Created:** 2026-10-15

## Summary

Assigning a task to an agent session is a durable routing decision. It does not
require a message transport or start the provider. The agent can later find its
work by querying the project with its own harness name and provider session ID.
The intended Coordinator flow first records this assignment, then attempts
direct CLI delivery to that exact session when an adapter can do so safely.
If direct delivery is unavailable before sending work, the assignment remains
in the inbox for voluntary discovery. RFC 0003 defines the direct path.

## Contract

`hp task assign` writes `assignment` in the task resource. The object contains
canonical `harness`, actual `sessionId`, `assignedAt`, and an optional
`registrationId` for provenance. The pair `(harness, sessionId)` is the
routing key. It is distinct from a provider-neutral `agentId`, task ID, run ID,
and transient registration ID. A live registration can supply the routing key;
the caller can also provide the pair explicitly for an offline session.

The task's assignment is stored with the project and can be synchronized with
its Git history. Local presence leases are ignored by Git. Expiration or
removal of a presence record MUST NOT remove an assignment. A new assignment
to the same routing key is idempotent; changing it requires `--replace` and
moves the previous assignment into `assignmentHistory`. `hp task unassign`
also preserves the old assignment in history with `endedAt`.

`hp task inbox --harness <name> --session-id <id>` returns only tasks addressed
to that exact routing key. Each result includes dependency readiness, missing
dependencies, and incomplete dependencies. The default view includes active
tasks, including blocked and in-progress tasks; `--ready-only` filters to
tasks whose dependencies are complete and whose status allows work to begin.
`--all` includes completed and cancelled assignments for audit.

Assignment does not mean that the agent saw, accepted, or started the task.
The task status, run records, work logs, and verification evidence communicate
those later stages. `hp task assign` and `hp task inbox` MUST NOT resume or
launch a provider as a side effect. The separate `hp coordinator dispatch`
command is the direct delivery path and follows RFC 0003. Assignment is its
durable fallback, not a replacement for CLI delivery.

## Delivery and fallback

The Coordinator uses the following order for an explicitly chosen
existing provider session:

1. Persist the task assignment to `(harness, sessionId)`.
2. Check dependency readiness, adapter support, worktree ownership, and whether
   that exact session can safely receive a CLI prompt.
3. If safe, deliver the task through the provider adapter and record the run.
4. If delivery cannot begin, retain the assignment and report that the task is
   waiting in the session inbox, with the reason. Do not substitute a different
   session or claim that delivery succeeded.

A failed preflight can safely downgrade to inbox discovery. Once a CLI call may
have sent a prompt, an unclear outcome MUST be recorded as uncertain delivery
and surfaced for reconciliation; the Coordinator MUST NOT blindly retry and
create duplicate work. If the request selected `--session new`, there is no
known session ID for an inbox fallback until the provider returns one, so a
failed launch MUST be reported rather than silently queued to an invented ID.

## Workflow

```sh
hp agent discover --project . --module api
hp coordinator dispatch --project . --task-id implement-api \
  --harness claude-code --session-id <actual-id> --worktree ./repos/api

# If direct delivery cannot start, in the assigned provider session later:
hp agent register --project . --harness claude-code --session-id <actual-id>
hp task inbox --project . --harness claude-code --session-id <actual-id>
```

An agent that has no accessible session ID should not invent one. It can still
use the normal project files and handoffs, but cannot use a session inbox until
the provider exposes an addressable ID.

## Limitations

The reference implementation uses project files as a shared queue. Another
machine sees assignments only after project changes are synchronized. Inbox
lookup proves a matching ID was supplied, not that the caller controls the
provider session. Concurrent task file edits still require normal Git merge
handling; this RFC does not define a distributed claim lock or reliable wakeup.
The reference CLI implements assignment, inbox discovery, direct adapters for
Codex, Claude Code, and Cursor, and preflight downgrade to the inbox. A
provider process that starts but does not return a clear outcome is recorded
as uncertain and requires human reconciliation before `--retry`.
