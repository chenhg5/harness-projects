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

- Define outcome-oriented goals with measurable key results, owners, dates, and
  evidence-backed completion.
- Coordinate dependent and parallel tasks through a shared task graph.
- Run work through Claude Code, Cursor, Codex, and other execution backends.
- Preserve project memory, decisions, research, plans, work logs, handoffs, and
  inspectable evidence.
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

The first release is local-first and file-based. Goals use an Objective plus
measurable Key Results model; tasks and runs carry their own delivery and
verification evidence. The architecture leaves room
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
├── worklogs/
├── archive/
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
validate the project file structure, safely read or write project files, persist
goals, tasks, runs, portable memory records, and handoffs, compute a local
Coordinator plan, register repositories, and create or remove local Git
worktrees. The next runtime capabilities are:

1. Project goals, instructions, measurable key results, memory, and evidence;
2. Repository clone, remote branch, and execution backend adapters;
3. Harness session adapters for Claude Code, Cursor, Codex, and other providers;
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
./bin/hp worklog create --project ./my-project --id session-001 --agent codex \
  --objective 'Implement a feature' --completed 'Added the code' \
  --remaining 'Add review' --next 'Run tests'
./bin/hp cleanup plan --project ./my-project --older-than 30d
```

See [INSTALL.md](INSTALL.md) for release installation and [docs/cli.md](docs/cli.md) for the command contract.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for the contribution workflow. Design
proposals should explain the problem, the compatibility impact, and how a
harness can adopt the change without becoming a required dependency.

## License

This project is licensed under the Apache License 2.0. See [LICENSE](LICENSE).
