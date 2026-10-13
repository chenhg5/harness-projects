# RFC 0002: Project Runtime and Coordination Layer

- **Status:** Draft
- **Authors:** Harness Projects contributors
- **Created:** 2026-10-14

## Summary

Extend Harness Projects from a portable project file contract into a runtime
that can coordinate long-running software work across Claude Code Projects,
Cursor Projects, Codex, and other coding harnesses.

## Motivation

The durable project directory is necessary for interoperability but does not
create a Projects product on its own. A useful open implementation must also
manage goals, tasks, execution runs, shared context, repository isolation,
events, verification, and human control.

## Scope

This RFC defines the provider-neutral domain model and adapter boundaries. It
does not require a hosted service or a single orchestration algorithm.

## Requirements

### Requirement: Long-lived goals

The runtime MUST represent a goal separately from an individual task or run. A
goal MAY remain active after a run completes or fails.

### Requirement: Task graph

The runtime MUST represent task dependencies explicitly. A Coordinator MUST NOT
dispatch a task whose required dependencies are incomplete unless an explicit
override is recorded.

### Requirement: Resumable runs

Each execution attempt MUST have an identifiable run. A run MUST record its
execution adapter, repository/worktree mapping, status, inputs, outputs, and
verification evidence.

### Requirement: Adapter boundary

The core runtime MUST interact with Claude Code, Cursor, Codex, repositories,
and event systems through adapters. Provider-specific identifiers MAY be stored
as adapter metadata but MUST NOT replace the provider-neutral run identity.

### Requirement: Shared context

The runtime MUST make relevant project instructions, specifications, decisions,
handoffs, and prior artifacts available to a run. Context retrieval SHOULD be
scoped to the task and SHOULD expose its source paths.

### Requirement: Human control

The runtime MUST support inspecting status and MUST provide a way to pause,
cancel, retry, or redirect a run. High-risk actions SHOULD support explicit
approval gates.

### Requirement: Event safety

An event trigger MUST identify the event source, matching rule, resulting action,
and deduplication behavior. Replaying an event MUST NOT create unbounded
duplicate runs.

### Requirement: Verification

The runtime MUST distinguish an Agent-reported result from verified evidence.
Project completion SHOULD require configured acceptance checks or an explicit
human override.

## Scenarios

### Scenario: Parallel repository work

- **GIVEN** a project with frontend and backend repositories
- **WHEN** the Coordinator creates independent tasks for each repository
- **THEN** each task receives an isolated worktree or execution environment
- **AND** the Coordinator records the dependency needed before integration

### Scenario: Agent handoff

- **GIVEN** a run stops after producing research and a partial patch
- **WHEN** another harness starts a new run for the same task
- **THEN** the new run receives the prior handoff, artifacts, commits, and
  verification evidence
- **AND** the task retains one provider-neutral identity

### Scenario: Failed verification

- **GIVEN** an Agent reports a task complete
- **WHEN** the configured test or acceptance check fails
- **THEN** the Coordinator marks the run as verification_failed
- **AND** chooses a configured retry, replan, or human approval action

### Scenario: Event deduplication

- **GIVEN** a pull request event is delivered twice
- **WHEN** both events match the same trigger
- **THEN** the runtime reuses or links to the existing event invocation
- **AND** does not create duplicate work without an explicit policy

## Initial implementation order

1. Add project goals and task/run metadata to the local file model.
2. Add repository and worktree mappings.
3. Add a local Coordinator with explicit task dependencies.
4. Add execution adapter interfaces and a local process adapter.
5. Add Claude Code, Cursor, and Codex adapters.
6. Add verification, human control, and event adapters.

## Open questions

- Should task and run state use JSON files, SQLite, or a pluggable store?
- How should a Coordinator request approval from a non-interactive Agent?
- Which provider capabilities can be safely normalized across harnesses?
