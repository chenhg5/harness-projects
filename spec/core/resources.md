# Project Resources

The local reference implementation stores runtime resources as one JSON file
per resource:

```text
goals/<goal-id>.json
tasks/<task-id>.json
runs/<run-id>.json
.harness-project/repositories/<repository-id>.json
.harness-project/worktrees/<worktree-id>.json
```

## Goals

A goal is a long-lived outcome. It MAY span multiple tasks and runs. Goal
statuses are `draft`, `active`, `completed`, and `archived`.

## Tasks

A task is a bounded unit of work. `dependsOn` contains task IDs that MUST be
complete before the task is dispatchable. Task statuses are `pending`, `ready`,
`in_progress`, `blocked`, `completed`, `failed`, and `cancelled`.

## Runs

A run is one execution attempt for one task. It records the execution adapter,
worktree, status, timestamps, errors, artifacts, and verification references.
Run statuses are `planned`, `running`, `succeeded`, `failed`, `paused`, and
`cancelled`.

The provider-neutral run ID remains stable even when a run is handed from one
harness adapter to another.

## Repositories and worktrees

A repository registration identifies a participating source repository by URL
and/or local path. Registration is metadata only; it does not clone or mutate
the source repository.

A worktree is an isolated checkout created from a registered local Git
repository. `hp worktree create` records the requested branch and base revision,
creates the checkout under `repos/` by default, and marks it `active` after Git
has succeeded. `hp worktree remove` removes the checkout and retains a
`removed` record for auditability.
