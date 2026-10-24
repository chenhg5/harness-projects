# Skills

Skills are reusable instructions and workflows that help an agent operate on a
Harness Project.

```text
skills/
├── shared/       Skills supported by multiple harnesses
├── codex/        Codex-specific skills
├── claude-code/  Claude Code-specific skills
└── cursor/       Cursor-specific skills
```
# Skills

Portable and harness-specific instructions that help agents use Harness Projects.

- [`harness-projects/SKILL.md`](harness-projects/SKILL.md): shared workflow for Claude Code, Cursor, Codex, and other harnesses.
- Generated projects also mount a small project-context skill through `.codex/skills/` and `.claude/skills/`.
- `hp skill sync` exposes canonical project skills through `.codex/skills/`,
  `.claude/skills/`, `.cursor/skills/`, and `.agents/skills/`.
