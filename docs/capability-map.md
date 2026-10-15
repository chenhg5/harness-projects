# Capability Map

The table describes the capability class we are targeting based on the
Projects workflows summarized in this repository. It is a product target, not
a claim that every feature is implemented today.

| Capability | Current `hp` | Target in Harness Projects | Provider-neutral abstraction |
| --- | --- | --- | --- |
| Outcome-oriented project goal | Goal JSON index plus optional Markdown document, doneWhen conditions, and evidence | Goal, constraints, optional method data, instructions, completion gates | `Project` and `Goal` |
| Shared project context | Context files, searchable memory records, and handoffs | Indexed context, retrieval, freshness, and memory updates | `ContextStore` |
| Task decomposition | Task JSON resources and dependency readiness | Task graph with dependencies and parallel branches | `TaskGraph` |
| Coordinator | Local readiness plan | Plan, route, observe, aggregate, replan | `Coordinator` |
| Agent discovery | Local session registration, lease renewal, capability filters, and stale pruning | Discover reachable agents across machines and supported transports | `AgentPresence` |
| Execution sessions | Run JSON resources | Resumable runs and provider threads | `Run` and `Thread` |
| Claude Code integration | Generated entry files only | Thread/session adapter | `ExecutionAdapter` |
| Cursor integration | Generated rule file only | Project and Agent adapter | `ExecutionAdapter` |
| Codex integration | Generated skill mount only | Session and skill adapter | `ExecutionAdapter` |
| Repository isolation | Repository registration and local Git worktree lifecycle | Clone, remote branch, worktree, environment mapping | `RepositoryAdapter` |
| Verification | Run/task evidence fields and project state validation | Tests, CI, review, acceptance gates, evidence aggregation | `Verifier` |
| Handoffs | Manual Markdown records | Structured resumable transfer between runs | `HandoffStore` |
| State snapshot | Structured state with source references and CLI validation | Derived control-plane state, freshness, health, and risk rollups | `StateStore` |
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
