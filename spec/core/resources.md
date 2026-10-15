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

A goal is a long-lived outcome. It MAY span multiple tasks and runs. An active
goal MUST have a description, an observable outcome, and one or more `doneWhen`
conditions. Goal statuses are `draft`, `active`, `completed`, and `archived`.
A goal MUST NOT become `completed` until it has evidence references. A project
MAY choose OKR, milestones, experiments, metrics, or a custom method. If it
chooses OKR, a key result SHOULD define a metric, direction, target, current
value, and evidence; that extension is not required by the core protocol.

## Tasks

A task is a bounded execution unit. It MUST describe its deliverables and
acceptance criteria before execution in a production project.
`dependsOn` contains task IDs that MUST be complete before the task is
ready to start. An optional `assignment` names a canonical harness and actual
provider `sessionId`; this pair identifies the session that can discover the
task through its inbox. Assigning a task MUST NOT imply that the session was
woken or that work started. Replaced and removed assignments SHOULD remain in
`assignmentHistory` with their end times. Local presence lease expiry MUST NOT
erase a durable task assignment. A blocked task MUST state why it is blocked.
A completed task MUST include a result or evidence reference. Task statuses are `pending`,
`ready`, `in_progress`, `blocked`, `completed`, `failed`, and `cancelled`.

## Runs

A run is one execution attempt for one task. It records the execution adapter,
harness and agent, objective, worktree, command, commit, changed files, status,
timestamps, errors, artifacts, and verification references. A succeeded run
MUST include verification evidence. Run statuses are `planned`, `running`,
`succeeded`, `failed`, `paused`, and `cancelled`.

The provider-neutral run ID remains stable even when a run is handed from one
harness adapter to another.

## State snapshot

`state/project-state.json` is a materialized control-plane snapshot. It MUST
identify the current phase, health, progress, next actions, and the source
records from which it was derived. It is updated after meaningful goal, task,
run, or verification changes. It is a navigation aid, not a replacement for
the underlying records.

## Repositories and worktrees

A repository registration identifies a participating source repository by URL
and/or local path. Registration is metadata only; it does not clone or mutate
the source repository.

A worktree is an isolated checkout created from a registered local Git
repository. `hp worktree create` records the requested branch and base revision,
creates the checkout under `repos/` by default, and marks it `active` after Git
has succeeded. `hp worktree remove` removes the checkout and retains a
`removed` record for auditability.
