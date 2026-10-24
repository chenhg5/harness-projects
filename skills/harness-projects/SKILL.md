---
name: harness-projects
description: Work in a portable Harness Project using hp for shared goals, tasks, memory, work logs, handoffs, cleanup, and skill synchronization across Claude Code, Cursor, Codex, and Agent Skills-compatible harnesses.
---

# Harness Projects

Use the Harness Project directory as the shared coordination context for a task that may involve more than one repository or coding harness.

## Start every session

1. Locate the project root by finding `PROJECT.md` and `.harness-project/project.json`.
2. Read `AGENTS.md`, `PROJECT.md`, `state/current-plan.md`, the active goal, the relevant task, and the latest work log. Read the latest handoff when responsibility has moved between agents.
3. Run `hp project validate --path <project>` before changing project files.
4. Inspect only the repository worktrees under `repos/` that the task needs. The project Git tracks coordination state; nested repositories under `repos/` keep their own Git history.
5. Register your actual provider session with `hp agent register`, then inspect
   `hp task inbox --project <project> --harness <name> --session-id <actual-id>`.
   Never invent a session ID. Registration records participation in
   `agents/sessions/` and remains discoverable; it does not indicate whether
   a provider process is currently running.

## Use the CLI

```bash
hp goal list --project <project>
hp task ready --project <project>
hp context search --project <project> --query "relevant terms"
hp worklog latest --project <project> --format text
hp handoff latest --project <project> --format text
hp project validate --path <project>
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
hint used for direct CLI delivery. They do not erase participation.

## Record progress

After each meaningful agent session, append a work log. It is a chronological activity record and must include what the session set out to do, what it completed, what remains, verification, and the next action:

```bash
hp worklog create --project <project> --id <session-id> \
  --agent <agent> --harness <claude|cursor|codex|other> \
  --objective "..." --completed "..." --remaining "..." \
  --verification "..." --next "..."
```

Create a handoff when another agent or harness will take ownership. A handoff names the source and destination and preserves repository branches, commits, blockers, and next actions. Do not replace regular work logs with handoffs.

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

The canonical context lives in project files, not in one vendor's private memory. Canonical skills live under `skills/`; preview and apply their harness mounts with `hp skill sync --dry-run` and `hp skill sync`. Claude Code reads `AGENTS.md`/`CLAUDE.md` and `.claude/skills/`, Codex reads `.codex/skills/`, Cursor can use `.cursor/rules/` and `.cursor/skills/`, and compatible agents can use `.agents/skills/`. When an adapter offers a private project memory feature, treat it as a cache or convenience and record durable decisions in `spec/`, `context/`, `worklogs/`, or `handoffs/`.
