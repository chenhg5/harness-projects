# Installing `hp`

`hp` is the reference CLI for Harness Projects. It creates and maintains the portable project layer shared by Claude Code, Cursor, Codex, and other coding harnesses.

## Release binary

On macOS or Linux:

```bash
curl -fsSL https://raw.githubusercontent.com/chenhg5/harness-projects/main/install.sh | sh
hp --version
```

To install a specific release, set `HP_VERSION` (for example `v0.0.6`). The installer downloads a platform archive and verifies its SHA-256 checksum.

## Go toolchain

```bash
go install github.com/chenhg5/harness-projects/cmd/hp@v0.0.6
```

## Build from source

```bash
git clone https://github.com/chenhg5/harness-projects
cd harness-projects
make build
./bin/hp project init --path ./my-project --name "My Project"
```

Run `make check` before a change is submitted. Maintainers build the release archives with `make release-all VERSION=vX.Y.Z`, then publish the contents of `dist/` and `checksums.txt` under the matching Git tag.

## Start a project

```bash
hp project init --path ./my-project --name "My Project"
hp project validate --path ./my-project
hp worklog latest --project ./my-project --format text
```

The generated project is its own Git repository. Any source repositories or linked worktrees under `repos/` remain independent nested Git repositories and are ignored by the project repository.
