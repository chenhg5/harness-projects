# Capability Map

The table describes the capability class we are targeting based on the
Projects workflows summarized in this repository. It is a product target, not
a claim that every feature is implemented today.

| Capability | Current `hp` | Target in Harness Projects | Provider-neutral abstraction |
| --- | --- | --- | --- |
| Long-lived project goal | File scaffold | Goal, constraints, acceptance criteria, instructions | `Project` and `Goal` |
| Shared project context | Markdown files | Indexed context, retrieval, freshness, and memory updates | `ContextStore` |
| Task decomposition | None | Task graph with dependencies and parallel branches | `TaskGraph` |
| Coordinator | None | Plan, route, observe, aggregate, replan | `Coordinator` |
| Execution sessions | None | Resumable runs and provider threads | `Run` and `Thread` |
| Claude Code integration | Generated entry files only | Thread/session adapter | `ExecutionAdapter` |
| Cursor integration | Generated rule file only | Project and Agent adapter | `ExecutionAdapter` |
| Codex integration | Generated skill mount only | Session and skill adapter | `ExecutionAdapter` |
| Repository isolation | Directory placeholder | Clone, branch, worktree, environment mapping | `RepositoryAdapter` |
| Verification | CLI structure validation | Tests, CI, review, acceptance gates, evidence | `Verifier` |
| Handoffs | Manual Markdown records | Structured resumable transfer between runs | `HandoffStore` |
| Events and schedules | None | PR, CI, issue, Slack, webhook, and timer triggers | `EventAdapter` |
| Human intervention | Shell control only | Approve, pause, cancel, redirect, retry, and inspect | `ControlPlane` |
| Remote execution | None | Local, cloud, and self-hosted runners | `RunnerAdapter` |

## Product alignment

Claude Code Projects and Cursor Projects demonstrate two useful execution
shapes:

- a project-level Coordinator that routes work to durable coding Threads;
- a Coordinator that separates planning from execution Agents and can react to
  external events.

Harness Projects should support both shapes through the same task, run, context,
and adapter model. A provider adapter decides whether a run becomes a Thread,
an Agent, a local process, or a remote job.
