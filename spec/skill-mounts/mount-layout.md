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

Shared skills SHOULD be exposed to every compatible harness. A harness-specific
skill MUST only be mounted where its declared adapter is supported.
