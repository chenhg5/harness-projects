# RFC 0006: Coordinator role and scheduled wakeup

- **Status:** Implemented (development branch)
- **Authors:** Harness Projects contributors
- **Created:** 2026-10-15

## Summary

Add a transferable Coordinator role and an optional local wakeup schedule.

## Problem

A durable participant registration says a session has worked on a project, but
does not identify who currently coordinates it. The project also needs an
optional way to revisit its plan and task inbox on a schedule. These are
different facts: participation is history; Coordinator is a transferable role;
the scheduler is a local process.

## Portable records

- `agents/coordinator.json` stores one current registration ID and ended terms.
  Reassignment requires an explicit `--replace`. Clearing the role preserves
  participant records.
- `agents/coordinator-wakeup.json` stores one project schedule: `enabled`,
  either `every` or five-field `cron`, `timeZone` for cron, a project-relative
  `worktree`, and a project-relative `promptFile`. The schedule follows the
  current role, not a hard-coded provider session.
- `agents/coordinator-wakeup.md` is the human-editable routine. Agents should
  read the current goal, plan, inbox, and latest work logs before acting.

Both JSON records are project Git state. `artifacts/wakeup/state.json` and
`artifacts/dispatch/*.log` are machine-local execution state, ignored by Git.
The local state names the target registration and config revision, next due
time, last attempt, outcome, and log paths. A handoff or schedule edit resets
the next due time rather than replaying missed wakeups.

## Execution

`hp wakeup start` holds one local scheduler lock and stays in the foreground.
Intervals are measured after a completed attempt; cron is evaluated in the
configured time zone. Each attempt is serialized with direct dispatch to the
same session. `hp wakeup once` makes a manual attempt, including when the
schedule is disabled. A dry run reads the routine but starts no provider.

The provider adapter must address the exact existing session. Codex queues a
message by thread ID. Claude Code checks the provider's active-session list
before resuming. Cursor needs a fresh local delivery hint because an
equivalent active-session probe is not available here. Unsupported providers,
missing CLIs, active Claude sessions, and missing Cursor hints remain `queued`
for a later attempt. An outcome that may have reached the provider but cannot
be confirmed is `uncertain`; the scheduler stops automatic retries until an
operator inspects logs and explicitly runs `hp wakeup once --retry`.

The scheduler does not mark tasks complete. Agents must verify deliverables and
write work logs through the ordinary project flow. Version one allows one
Coordinator schedule per project; additional independent schedules need a
separate proposal so their ownership and retry rules stay explicit.
