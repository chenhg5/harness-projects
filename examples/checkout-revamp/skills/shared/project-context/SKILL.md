---
name: project-context
description: Read the portable context of a Harness Project at the start of a session and record progress before stopping. Use when working in a directory that contains .harness-project/.
---

# Project context

1. Run `hp brief` (from anywhere inside the project) and read it: goal, plan, next actions, ready tasks, and the remaining work from the latest work log and handoff.
2. Read the files the brief points to, relevant `spec/` files, and each repository in `repos/` you will edit.
3. Register the session with `hp agent register --project . --focus <responsibility>`; the session ID is detected from `HP_SESSION_ID` (session hook) or `CODEX_THREAD_ID`. Do not invent an ID.
4. Check task readiness and dependencies with `hp task ready --project .`, then mark your task `in_progress`.
5. Before stopping, write a work log with objective, completed work, remaining work, verification, and next action; update task status and `hp state update --next-actions` when the plan changed. Write a handoff when transferring responsibility.
6. Advanced: find peers with `hp agent discover`, assigned work with `hp task inbox --project .`, and use `hp coordinator dispatch` for explicit CLI delivery. Inspect an uncertain run before any `--retry`.

7. Check `hp coordinator get --project .` for the current Coordinator. A Coordinator may configure `hp wakeup` with an interval or cron schedule and run the foreground scheduler; the role is explicit and transferable. Inspect `hp wakeup status` and delivery logs before retrying an uncertain wakeup.

<!-- hp:managed-workflow v2 -->
## Reliable lifecycle

1. Run `hp brief` and inspect the task inbox before editing.
2. Continue relevant in-progress work from the brief, or use `hp task ready` for unstarted tasks. Mark the chosen task with `hp task update --status in_progress`.
3. Use focused `hp context search` queries for structured and ordinary notes; every whitespace-separated term must match.
4. Run `hp layout list --project <project>` before creating a top-level directory. Use a registered location or add an intentional project-specific directory with `hp layout add`.
5. End meaningful work with `hp finish`; use `--status in_progress` for partial work. Create a handoff in addition only when responsibility moves.
6. Run `hp project doctor --path <project>` and repair reported drift. Do not commit, branch, push, or open a pull request unless explicitly requested.
7. Use `hp project upgrade --path <project>` after hp upgrades to refresh managed instructions, hooks, and all supported skill mounts.
<!-- /hp:managed-workflow -->
