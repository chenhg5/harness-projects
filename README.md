# Harness Projects

An open-source Projects platform for Claude Code, Cursor, Codex, and other AI
coding harnesses.

Harness Projects is an open, interoperable counterpart to the product ideas
behind Claude Code Projects and Cursor Projects. It is more than a directory
standard: it is a project coordination layer for long-running software work
across repositories, worktrees, agents, coding harnesses, and development
sessions.

The project directory remains the durable source of context, while the runtime
will provide coordination, task execution, shared memory, event triggers,
verification, and harness adapters.

## Product goals

- Define a long-lived project goal, constraints, instructions, and acceptance
  criteria.
- Coordinate dependent and parallel tasks through a shared task graph.
- Run work through Claude Code, Cursor, Codex, and other execution backends.
- Preserve project memory, decisions, research, plans, handoffs, and evidence.
- Isolate repository worktrees and execution environments for each run.
- React to human requests, pull requests, CI results, issue events, and
  schedules.
- Let humans inspect, pause, redirect, approve, retry, and stop work.
- Keep the project model portable so a harness can be replaced without losing
  the project.

## What the product contains

```text
Project
├── Goal and instructions
├── Coordinator
│   ├── Task graph and dependencies
│   ├── Agent and Thread lifecycle
│   ├── Scheduling and routing
│   └── Human approval points
├── Shared context and memory
├── Execution backends
│   ├── Claude Code
│   ├── Cursor
│   ├── Codex
│   └── Generic local or remote runners
├── Repository and worktree manager
├── Events and triggers
├── Verification and artifact collection
└── Harness adapters
```

The first release is local-first and file-based. The architecture leaves room
for remote runners and hosted event services without making them mandatory.

## Boundaries

- Harness Projects does not replace Claude Code, Cursor, Codex, or another
  execution harness. It coordinates them through adapters.
- It does not require one model, vendor, repository host, or execution
  environment.
- It does not hide the durable project state in a proprietary database.
- A project may use only one harness; interoperability is an available
  capability, not a requirement for every run.

## Project layout

A Harness Project is organized around the project, not around one repository:

```text
my-project/
├── PROJECT.md
├── spec/
├── goals/
├── tasks/
├── runs/
├── agents/
├── context/
├── handoffs/
├── events/
├── verification/
├── skills/
├── scripts/
├── repos/
├── state/
├── artifacts/
├── integrations/
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

## Capability status

The repository is in the foundation phase. The current CLI can create and
validate the project file structure and safely read or write project files. The
runtime capabilities are being specified next:

1. Project goals, instructions, memory, and acceptance criteria;
2. Tasks, dependencies, runs, agents, and handoffs;
3. Repository/worktree and execution backend adapters;
4. Coordinator scheduling, retries, approvals, and result aggregation;
5. Events, subscriptions, schedules, and continuous project operation.

See [docs/product.md](docs/product.md) and
[docs/capability-map.md](docs/capability-map.md). Backward-compatible changes
will be documented in `CHANGELOG.md`.

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
