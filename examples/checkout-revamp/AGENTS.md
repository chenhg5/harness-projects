# Agent instructions

This directory is a Harness Project: the shared memory for every coding agent working on it. Chat history is not shared between sessions or tools, so these files are the source of truth.

## Start of every session

Run `hp brief` and read it. It shows the goal, plan, next actions, ready tasks, and what the last session left unfinished. Then read the files it points to before changing code. Code repositories live in `repos/` and keep their own Git history.

## While working

- Pick a ready task and mark it with `hp task update --project . --id <task> --status in_progress`.
- Record durable decisions in `spec/` and research in `context/`.
- Register this session with `hp agent register --project . --focus <what you are doing>`. The session ID is detected from the session hook (`HP_SESSION_ID`) or Codex (`CODEX_THREAD_ID`); never invent one. Pass `--harness` and `--session-id` only when detection fails.

## Before you stop

- `hp worklog create --project . --id <slug> --agent <you> --objective ... --completed ... --remaining ... --next ...`
- `hp task update` for task status, and `hp state update --next-actions ...` when the plan changed.
- `hp handoff create` when another agent or person continues the work.
- Commit the project files.

## Advanced: coordination

Use `hp agent discover --project .` to find other agents and `hp task inbox --project .` for work assigned to this session. A Coordinator can use `hp coordinator dispatch --project . --task-id <id> --harness <name> --session-id <id> --worktree <path>` for explicit CLI delivery; preflight failure leaves the assignment in the inbox. Inspect an uncertain run before any `--retry`. Participation remains registered; `hp agent heartbeat` refreshes the local delivery hint and `hp agent unregister` ends it. Use `hp cleanup plan` to keep transient files bounded.

The current Coordinator is an explicit project role, not a capability label or an online status. Check `hp coordinator get --project .` before acting as Coordinator. Assign a registered session with `hp coordinator assign --project . --registration-id <id>`; use `--replace` for a handoff. If `hp wakeup status --project .` shows an enabled schedule, the foreground `hp wakeup start --project .` process delivers the routine in `agents/coordinator-wakeup.md` to the current session. Inspect uncertain delivery logs before any manual `hp wakeup once --retry`.

<!-- hp:managed-workflow v2 -->
## Managed hp workflow

- Start with `hp brief`; use `hp project doctor --path .` when the brief reports drift.
- Continue relevant in-progress work from the brief, or use `hp task ready` for unstarted tasks; start one with `hp task update --status in_progress`.
- Keep durable research in `context/`; `hp context search` searches structured and ordinary notes and requires every whitespace-separated query term to match.
- Before creating a top-level directory, run `hp layout list --project .`. Use an existing registered location, or document an intentional project-specific directory with `hp layout add`.
- Use `hp finish` at the end of meaningful work, including `--status in_progress` for a partial iteration. It updates the task, writes the work log, and reconciles task-backed state.
- Use `hp handoff create` only when responsibility moves. A handoff supplements `hp finish`; it does not replace it.
- Run `hp project doctor --path .` before closeout. Do not commit, branch, push, or open a pull request unless the user or repository workflow explicitly requests it.
- Run `hp project upgrade --path .` after updating hp; it refreshes this managed block, hooks, and skill mounts without replacing project-specific instructions.
<!-- /hp:managed-workflow -->
