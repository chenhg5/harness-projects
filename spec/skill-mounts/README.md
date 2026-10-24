# Skill Mount Specification

Canonical skills live under the project `skills/` directory. Harness-specific
directories expose those skills through generated links or adapter files.

Synchronize mounts with:

```bash
hp skill sync --project . --targets codex,claude,cursor,agents --dry-run
hp skill sync --project . --targets codex,claude,cursor,agents
```

The command is idempotent and refuses to overwrite non-symlink entries. Use
`--prune` only when stale project-owned links should be removed.
