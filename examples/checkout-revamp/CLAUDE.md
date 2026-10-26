# Claude Code project instructions

Follow `AGENTS.md` and read `PROJECT.md` for the shared project context. The canonical project files are outside `.claude/`.

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
