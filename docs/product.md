# Harness Projects Product Model

Harness Projects is a project coordination product for AI coding harnesses.
It targets the same class of long-running project workflows represented by
Claude Code Projects and Cursor Projects, while keeping the project model and
durable state open and portable.

## Core objects

### Project

A long-lived software objective with instructions, constraints, repositories,
acceptance criteria, participants, and project-level memory.

### Goal

The outcome the project is trying to achieve. A goal MAY remain active across
many task runs and agent sessions.

### Task

A bounded unit of work with dependencies, an owner or routing policy, required
context, expected artifacts, and verification criteria.

### Run

One attempt to execute a task. A task MAY have multiple runs after retries,
replanning, or a change of harness.

### Agent

An execution participant that can inspect files, use tools, change code, run
tests, or produce research. An Agent is an abstract role; a provider adapter
maps it to a Claude Code session, Cursor Agent, Codex session, local process, or
another runner.

Durable Agent roles live in `agents/`. A working Agent may also publish a
machine-local presence lease with its actual provider session ID, current
responsibility, modules, and capabilities. Other Agents can discover that
session without treating its self-reported presence as proof of reachability.

### Thread

A durable conversation or execution session that can be resumed. Claude Code
Projects use a Thread-like unit for a full coding session. Harness Projects
keeps this concept provider-neutral.

### Coordinator

The project-level controller. It turns goals into tasks, resolves dependencies,
routes tasks to execution backends, collects results, requests approval, and
decides whether to continue, retry, replan, or stop.

Routing can stop at durable assignment: a task names a harness and provider
session ID, and that session discovers the task through its project inbox.
Directly waking or resuming a provider is an optional later step. Assignment
does not imply the recipient has seen or started the task.

The Coordinator MUST be able to operate without directly editing source code.
Code changes belong to execution Agents. A deployment MAY combine Coordinator
and executor behavior for a single harness when that is how the provider works.

### Context and memory

Project context includes instructions, specifications, repository maps, plans,
decisions, research, handoffs, test knowledge, and artifacts. Memory is the
retrieval and update behavior around that context; it is not only a folder of
unindexed notes.

### Event and trigger

An event is an input such as a user request, pull request, CI result, issue,
Slack message, or schedule tick. A trigger maps matching events to a task,
plan, or Coordinator action.

### Artifact and evidence

Artifacts are outputs such as commits, patches, pull requests, reports, test
results, screenshots, logs, and generated files. Evidence records why an
artifact is considered complete or acceptable.

## Lifecycle

```text
Define goal
    ↓
Plan tasks and dependencies
    ↓
Route tasks to Agents and execution environments
    ↓
Run, observe, and collect artifacts
    ↓
Verify results and resolve conflicts
    ↓
Approve, replan, retry, or complete
    ↓
Persist memory and continue from the next event
```

## Coordinator and execution boundary

The Coordinator owns project-level decisions:

- task decomposition and dependency management;
- agent selection and run placement;
- concurrency limits and conflict avoidance;
- approval gates and stop conditions;
- retry, cancellation, and recovery policy;
- result aggregation and project status.

An execution Agent owns task-level actions:

- repository inspection and code changes;
- tests, builds, and local verification;
- research and technical analysis;
- commits, branches, and pull requests when authorized;
- reporting structured results and blockers.

## Portability layers

```text
Project contract       Portable files and schemas
Coordinator             Task graph, scheduling, approvals, recovery
Execution adapter       Claude Code, Cursor, Codex, local, or remote runner
Repository adapter      Git clone, branch, worktree, PR, and CI operations
Event adapter           User, issue, PR, CI, Slack, webhook, and schedule
Harness view            AGENTS.md, CLAUDE.md, .cursor/rules, skills, APIs
```

Each adapter SHOULD be replaceable without rewriting the project contract.
