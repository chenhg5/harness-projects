# Skill Mount Layout

Canonical skill content MUST live under `skills/`:

```text
skills/
├── shared/<skill-name>/
├── codex/<skill-name>/
├── claude-code/<skill-name>/
└── cursor/<skill-name>/
```

Harness directories MAY contain symlinks or generated adapter files. A mount
MUST point back to canonical content and MUST be safe to regenerate.

`hp skill sync` maps targets to these discovery directories:

| Target | Directory |
| --- | --- |
| `codex` | `.codex/skills/` |
| `claude` | `.claude/skills/` |
| `cursor` | `.cursor/skills/` |
| `agents` | `.agents/skills/` |

For compatibility with existing projects, a directory below `skills/` that
contains `SKILL.md` outside the four reserved scope directories is treated as
shared. Discovery stops at the first `SKILL.md`, so a bundled upstream copy
inside another skill is not mounted separately.

Shared skills SHOULD be exposed to every compatible harness. A harness-specific
skill MUST only be mounted where its declared adapter is supported.

Synchronization MUST be idempotent, MUST use relative symlinks, and MUST refuse
to overwrite non-symlink entries. Pruning MUST only remove symlinks whose target
is inside the canonical project `skills/` directory.
