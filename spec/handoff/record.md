# Handoff Record

Each handoff MUST be a new Markdown file under `handoffs/`. Recommended names
use a monotonically increasing number and a short slug:

```text
0001-codex-to-claude.md
0002-claude-to-cursor.md
```

Each record SHOULD include:

```markdown
# Handoff: <short title>

- From: <harness or agent>
- To: <harness or agent>
- Created: <RFC 3339 timestamp>
- Project: <project name>

## Objective
## Completed
## In progress
## Next actions
## Repositories and commits
## Verification
## Blockers and risks
```
