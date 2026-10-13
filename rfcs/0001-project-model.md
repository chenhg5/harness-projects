# RFC 0001: Harness Project Model

- **Status:** Draft
- **Authors:** Harness Projects contributors
- **Created:** 2026-10-14

## Summary

Define a portable project directory that lets multiple coding harnesses work on
one software task while sharing specifications, state, skills, repositories,
and handoffs.

## Motivation

A coding task can span multiple repositories and multiple agent runtimes. The
task needs a durable context boundary that is independent of a particular
harness or chat session.

## Proposed model

An implementation of the standard MUST provide these canonical areas:

```text
PROJECT.md
spec/
handoffs/
skills/
scripts/
repos/
state/
artifacts/
.harness-project/
```

Harness-specific directories such as `.codex/`, `.claude/`, and `.cursor/`
MUST be adapters or mounts. They MUST NOT become the canonical source of
project state.

### Requirements

#### Requirement: Project identity

A project MUST contain `.harness-project/project.json` with a schema version,
name, and creation timestamp.

#### Requirement: Durable context

A project MUST keep durable requirements, decisions, and architecture under
`spec/`. Each concern SHOULD use a separate Markdown file.

#### Requirement: Append-only handoffs

A project MUST allow multiple handoff records under `handoffs/`. A handoff
MUST NOT overwrite an earlier handoff record.

#### Requirement: Harness portability

A supported harness MUST be able to discover the shared project entry point and
current state without reading a proprietary database.

#### Requirement: Repository separation

Repositories MAY be numerous. Repository worktrees MUST live under `repos/` or
be explicitly mapped there by the project manifest.

### Scenarios

#### Scenario: Start a project

- **GIVEN** a path that does not exist
- **WHEN** an operator runs `hp project init --path <path>`
- **THEN** the CLI creates the canonical directories and harness entry points
- **AND** the CLI returns a machine-readable description of the changes

#### Scenario: Continue with another harness

- **GIVEN** a project with a current plan and at least one handoff
- **WHEN** another harness opens the project directory
- **THEN** it can read `PROJECT.md`, `state/`, `spec/`, and `handoffs/`
- **AND** it does not need the previous harness session

#### Scenario: Validate a project

- **GIVEN** a project with a missing canonical directory
- **WHEN** an operator runs `hp project validate --path <path>`
- **THEN** the command exits non-zero
- **AND** reports the missing path in structured output

## Compatibility

The first version is local-only and file-based. A future implementation MAY
add remote storage or orchestration, but those features MUST preserve the file
contract.

## Open questions

- Should repository mappings allow worktrees outside the project directory?
- Which Cursor adapter form should be normative when a project skill has no
  direct Cursor equivalent?
- Should `project.json` remain JSON, or should a human-edited YAML manifest be
  added alongside it?
