# Changelog

All notable changes to Harness Projects will be documented here.

## Unreleased

- Add an explicit, transferable Coordinator role for registered sessions with
  assignment history and role-aware discovery.
- Add an optional interval or cron wakeup schedule for the current Coordinator,
  a foreground scheduler, exact-session CLI delivery, and local retry state.
- Stop automatic retries when a wakeup may have reached the provider but its
  outcome cannot be confirmed.

## v0.0.5 - 2026-10-15

- Persist agent session participation in project Git under `agents/sessions/`.
  Discovery and task assignment keep working after a local delivery hint expires.
- Keep machine-local leases only for safe direct CLI delivery. Ending or pruning
  a hint no longer removes a participant; unsafe delivery still falls back to
  the task inbox.
- Update project templates, the RFCs, and English and Chinese documentation to
  distinguish durable participation from direct-delivery readiness.

## v0.0.4 - 2026-10-15

- Added `hp coordinator dispatch` for explicit-session CLI delivery to Codex,
  Claude Code, and Cursor, with durable task assignment, safe inbox fallback,
  local session locking, run outcomes, and bounded diagnostic logs.
- Rebuilt the English README around a quick start, capability overview, and
  cross-harness workflow; added a Simplified Chinese README.

## v0.0.3 - 2026-10-15

- Added a local project dashboard with section navigation and English and
  Simplified Chinese language switching. Fixed status badge sizing.
- Added local agent session registration, discovery, lease renewal, capability
  filters, and stale pruning.
- Added durable task assignment to an explicit provider session, a session
  inbox with dependency readiness, and reassignment history.
- Specified direct CLI delivery with an inbox fallback when delivery cannot
  begin safely. Provider CLI delivery remains planned; this release does not
  invoke another coding agent.

## v0.0.2 - 2026-10-14

- Added evidence-backed completion rules for goals, tasks, and runs.
- Made goal management method-neutral while retaining optional OKR data.
- Clarified the v1 boundary between explanatory documents and structured
  indexes.

## v0.0.1 - 2026-10-14

- Created the repository and initial project structure.
- Added the Go `hp` CLI with project initialization, validation, file read/write,
  dry-run support, structured output, and command schema introspection.
- Added the first RFC-style project model and normative specification drafts.
- Expanded the product definition from a file standard to a cross-harness
  Projects coordination layer aligned with Claude Code Projects and Cursor
  Projects.
- Expanded `hp project init` with first-class goals, tasks, runs, Agents,
  context, events, verification, and integration directories.
- Added local goal, task, dependency-readiness, and run persistence with JSON
  schemas and agent-friendly CLI commands.
- Added the first local Coordinator plan that separates dispatchable tasks from
  tasks waiting on dependencies.
- Added portable work logs for every agent session, distinct from handoffs.
- Added bounded project cleanup with safe archival and explicit forced deletion.
- Added the cross-harness `harness-projects` skill plus release installation,
  multi-platform Makefile archives, checksum generation, and `INSTALL.md`.
