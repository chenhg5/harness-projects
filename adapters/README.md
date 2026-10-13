# Harness Adapters

Adapters connect the portable Harness Projects product to external systems.

Planned adapter classes include:

- **Execution adapters:** Claude Code, Cursor, Codex, local processes, and
  remote runners;
- **Repository adapters:** Git repositories, branches, worktrees, pull
  requests, and CI;
- **Event adapters:** user requests, issues, pull requests, CI, Slack,
  webhooks, and schedules;
- **Harness views:** entry points, skill mounts, rules, and provider-specific
  project configuration.

An adapter may generate integration files or invoke an external system, but it
must not become the only source of project state.
