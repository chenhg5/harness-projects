---
name: harness-projects
description: Work in a portable Harness Project using hp for shared goals, tasks, memory, work logs, handoffs, cleanup, and skill synchronization across Claude Code, Cursor, Codex, and Agent Skills-compatible harnesses.
---

# Harness Projects

Use the Harness Project directory as the shared coordination context for a task that may involve more than one repository or coding harness.

## Start every session

1. Locate the project root by finding `PROJECT.md` and `.harness-project/project.json`.
2. Read `AGENTS.md`, `PROJECT.md`, `state/current-plan.md`, the active goal, the relevant task, and the latest work log. Read the latest handoff when responsibility has moved between agents.
3. Run `hp project validate --path <project>` before changing project files; run `hp project doctor --path <project>` when the brief reports drift or before finalizing shared records.
4. Inspect only the repository worktrees under `repos/` that the task needs. The project Git tracks coordination state; nested repositories under `repos/` keep their own Git history.
5. Register your actual provider session with `hp agent register`, then inspect
   `hp task inbox --project <project> --harness <name> --session-id <actual-id>`.
   Never invent a session ID. Registration records participation in
   `agents/sessions/` and remains discoverable; it does not indicate whether
   a provider process is currently running. Pass `--worktree` when executing
   outside the project directory and retain the returned registration ID for
   `hp agent unregister --project <project> --id <registration-id>`.

## Use the CLI

```bash
hp goal list --project <project>
hp task ready --project <project>
hp task update --project <project> --id <task-id> --status in_progress
hp context search --project <project> --query "relevant terms"
hp worklog latest --project <project> --format text
hp handoff latest --project <project> --format text
hp project validate --path <project>
hp project doctor --path <project>
hp project upgrade --path <project> --dry-run
hp skill sync --project <project> --targets codex,claude,cursor,agents --dry-run
hp coordinator dispatch --project <project> --task-id <task-id> \
  --harness <name> --session-id <actual-id> --worktree <project-relative-path> \
  --dry-run
```

Use `hp file write` for project sections instead of writing through an arbitrary path. Use `--dry-run` before mutations when the change is broad or destructive.
For direct delivery, omit `--dry-run` only after checking the exact target
session and worktree. Preflight failure leaves the task in its session inbox.
If a provider process started but its result is uncertain, inspect the run
and local dispatch logs before an intentional `--retry`.
`hp agent heartbeat` and `hp agent unregister` manage only a machine-local
hint used for direct CLI delivery. They do not erase participation. Both use
the durable registration ID returned by `hp agent register`.

## Record progress

After each meaningful agent session, use `hp finish`. It updates the task,
appends the work log, and reconciles task-backed state together. Use
`--status in_progress` when the session made progress but the task remains open:

```bash
hp finish --project <project> --task-id <task-id> --id <session-id> \
  --agent <agent> --harness <claude|cursor|codex|other> \
  --objective "..." --completed "..." --remaining "..." \
  --verification "..." --next "..."
```

Use `hp worklog create` only when no task record exists. `hp finish` reconciles
task-backed fields; update goal status, project phase, health, and project-level
next actions separately when they changed. Run `hp project doctor` before
closeout; it detects stale state, task/state disagreement, missing evidence,
malformed logs, skill mount drift, and repository/worktree drift.

Create a handoff only when another agent or harness will take ownership. A
handoff names the source and destination and preserves repository branches,
commits, blockers, and next actions. It supplements `hp finish`; it never
replaces the regular closeout work log.

Do not create commits, branches, pushes, or pull requests unless the user or
the repository workflow explicitly requests them.

## Keep projects bounded

Generated outputs, imported transcripts, and old work logs must not grow forever. Preview stale transient files with:

```bash
hp cleanup plan --project <project> --older-than 30d
```

The default action archives eligible files under `archive/`. Deletion is explicit and irreversible:

```bash
hp cleanup apply --project <project> --older-than 30d --action delete --force
```

Never use cleanup to remove specifications, goals, tasks, runs, verification, or handoffs. Those are durable project history.

## Harness adapters

The canonical context lives in project files, not in one vendor's private memory. `hp context search` searches structured memories and ordinary Markdown/text notes under `context/`. Every whitespace-separated query term must match, so retry with fewer or more specific terms when a broad query returns nothing. Canonical skills live under `skills/`; preview and apply their harness mounts with `hp skill sync --dry-run` and `hp skill sync`, or use `hp project upgrade` after installing a new hp version. Claude Code reads `AGENTS.md`/`CLAUDE.md` and `.claude/skills/`, Codex reads `.codex/skills/`, Cursor can use `.cursor/rules/` and `.cursor/skills/`, and compatible agents can use `.agents/skills/`. When an adapter offers a private project memory feature, treat it as a cache or convenience and record durable decisions in `spec/`, `context/`, `worklogs/`, or `handoffs/`.
