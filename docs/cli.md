# `hp` CLI

`hp` is the reference CLI for creating and maintaining Harness Projects. It
uses a noun-verb command tree so agents can discover commands incrementally.
The CLI is the local control plane for the future Coordinator and adapters.

## Build

```bash
go build -o ./bin/hp ./cmd/hp
```

## Commands

```bash
hp project init --path ./my-project --name "My Project"
hp project status --path ./my-project --format json
hp project validate --path ./my-project --format json
hp file write --project ./my-project --section spec \
  --path architecture/api.md --from ./api.md --dry-run
hp file read --project ./my-project --section state \
  --path current-plan.md --format text
hp schema list --format json
hp schema get --command "file write" --format json

hp goal create --project ./my-project --id modernize-api \
  --title "Modernize API" --description "Reduce failed releases" \
  --outcome "API releases meet the reliability target" \
  --done-when "reliability target is met,verification is recorded" \
  --key-results '[{"id":"reliability","metric":"successful requests","direction":"increase","target":99,"status":"planned"}]' \
  --status active
hp task create --project ./my-project --id inventory-api \
  --title "Inventory API" --goal-id modernize-api \
  --deliverables "API endpoints" --acceptance "integration tests" --priority high
hp task ready --project ./my-project --format json
hp run create --project ./my-project --id run-001 --task-id inventory-api
hp run update --project ./my-project --id run-001 --status running
hp state validate --project ./my-project
hp repository add --project ./my-project --id api --path ../api --default-branch main
hp worktree create --project ./my-project --id api-task --repository-id api
hp worktree list --project ./my-project --status active
hp worktree remove --project ./my-project --id api-task
hp context add --project ./my-project --id api-decision \
  --kind decision --title "API boundary" --from ./decision.md
hp context search --project ./my-project --query "API boundary"
hp handoff create --project ./my-project --id api-work \
  --from codex --to claude --objective "Build API" \
  --completed "Added routes" --next "Add tests"
hp handoff latest --project ./my-project --format text
hp worklog create --project ./my-project --id session-001 --agent codex \
  --objective "Implement memory" --completed "Added search" \
  --remaining "Add docs" --verification "go test ./..." --next "Review"
hp worklog latest --project ./my-project --format text
hp cleanup plan --project ./my-project --older-than 30d
hp cleanup apply --project ./my-project --older-than 30d
hp coordinator plan --project ./my-project --format json
```

Commands that write files support `--dry-run`. A successful dry run exits with
code `10`. Non-TTY output defaults to JSON. JSON data goes to stdout and
machine-readable errors go to stderr.

Work logs are chronological session records. Handoffs are responsibility
transfers and may include branch, commit, and blocker details. Cleanup only
selects transient sections (`worklogs`, `artifacts`, and `context/imports`);
the default action archives them under `archive/` and never selects durable
project history.
