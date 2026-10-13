# Harness Projects

An open project standard for interoperable AI coding harnesses.

Harness Projects defines a portable structure for software work that may span
multiple repositories, worktrees, coding agents, and development sessions.
The project directory is the shared source of context. Individual harnesses
such as Claude Code, Cursor, Codex, and other tools consume that context
through small, generated adapters.

## Goals

- Keep project context portable across coding harnesses.
- Treat a multi-repository task as one coherent project.
- Preserve plans, specifications, decisions, and handoffs as durable files.
- Keep harness-specific integration code separate from the project model.
- Make the standard usable locally without a hosted service.

## Non-goals

- Replacing coding harnesses or agent runtimes.
- Defining a single model, vendor, or orchestration strategy.
- Hiding project state in a proprietary database.
- Requiring every project to use the same repository hosting provider.

## Project layout

A Harness Project is organized around the project, not around one repository:

```text
my-project/
├── PROJECT.md
├── spec/
├── handoffs/
├── skills/
├── scripts/
├── repos/
├── state/
├── artifacts/
├── .codex/
├── .claude/
├── .cursor/
└── .harness-project/
```

The canonical project files live outside harness-specific directories. Adapter
files and skill mounts make those files discoverable by each supported harness.

## Repository structure

```text
spec/       The standard and its versioned specifications
rfcs/       Proposed standard changes and review records
schemas/    Machine-readable schemas
adapters/   Harness-specific entry points and mappings
skills/     Shared and harness-specific skills
templates/  Starter files for generated projects
examples/   Complete example projects
```

## Status

This repository is in the design stage. The first milestone will define the
project manifest, directory contract, handoff format, and skill mount rules.
Backward-compatible changes will be documented in `CHANGELOG.md`.

## CLI

The reference CLI is written in Go and is called `hp`:

```bash
go build -o ./bin/hp ./cmd/hp
./bin/hp project init --path ./my-project --name "My Project"
./bin/hp project validate --path ./my-project --format json
```

See [docs/cli.md](docs/cli.md) for the command contract.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for the contribution workflow. Design
proposals should explain the problem, the compatibility impact, and how a
harness can adopt the change without becoming a required dependency.

## License

This project is licensed under the Apache License 2.0. See [LICENSE](LICENSE).
