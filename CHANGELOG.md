# Changelog

All notable changes to Harness Projects will be documented here.

## Unreleased

- Added `hp coordinator dispatch` for explicit-session CLI delivery to Codex,
  Claude Code, and Cursor, with durable task assignment, safe inbox fallback,
  local session locking, run outcomes, and bounded diagnostic logs.

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
