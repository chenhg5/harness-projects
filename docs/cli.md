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
  --title "Modernize API" --status active
hp task create --project ./my-project --id inventory-api \
  --title "Inventory API" --goal-id modernize-api
hp task ready --project ./my-project --format json
hp run create --project ./my-project --id run-001 --task-id inventory-api
hp run update --project ./my-project --id run-001 --status running
hp coordinator plan --project ./my-project --format json
```

The planned command surface is:

```text
hp project create|inspect|update|validate
hp goal create|list|get|update
hp task create|list|get|update|ready
hp run create|list|get|update
hp agent list|inspect|start|stop
hp context search|add|forget
hp repository add|list|remove
hp worktree create|list|remove
hp event subscribe|list|test
hp handoff create|list|show
hp artifact list|inspect
hp coordinator plan
```

These commands are intentionally planned as resources and verbs. They will be
added only when their file and adapter contracts are specified.

Commands that write files support `--dry-run`. A successful dry run exits with
code `10`. Non-TTY output defaults to JSON. JSON data goes to stdout and
machine-readable errors go to stderr.

The first release intentionally keeps repository and worktree operations out
of the CLI until the project manifest and safety rules are stable.
