# Changelog

All notable changes to Harness Projects will be documented here.

## Unreleased

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
- Added local agent session registration and discovery with capability filters,
  expiring leases, stale pruning, and project entry-point instructions.
