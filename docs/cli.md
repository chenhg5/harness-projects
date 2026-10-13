# `hp` CLI

`hp` is the reference CLI for creating and maintaining Harness Projects. It
uses a noun-verb command tree so agents can discover commands incrementally.

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
```

Commands that write files support `--dry-run`. A successful dry run exits with
code `10`. Non-TTY output defaults to JSON. JSON data goes to stdout and
machine-readable errors go to stderr.

The first release intentionally keeps repository and worktree operations out
of the CLI until the project manifest and safety rules are stable.
