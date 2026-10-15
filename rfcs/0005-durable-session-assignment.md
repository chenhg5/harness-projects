# RFC 0005: Durable Task Assignment and Session Inbox

- **Status:** Draft
- **Authors:** Harness Projects contributors
- **Created:** 2026-10-15

## Summary

Assigning a task to an agent session is a durable routing decision. It does not
require a message transport or start the provider. The agent can later find its
work by querying the project with its own harness name and provider session ID.
Direct CLI dispatch is an optional execution path governed by RFC 0003.

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
those later stages. The CLI MUST NOT resume or launch a provider as a side
effect of task assignment or inbox retrieval. Only a future explicit dispatch
command may do so, with the safeguards in RFC 0003.

## Workflow

```sh
hp agent discover --project . --module api
hp task assign --project . --id implement-api --registration-id <id>

# In the assigned provider session, now or later:
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
