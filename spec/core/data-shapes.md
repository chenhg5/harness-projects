# Data shapes

Harness Projects uses two complementary shapes. They must not be confused.

## Documents

Use Markdown when an Agent or human needs to understand meaning, rationale,
scope, constraints, decisions, or evidence. Documents are the natural shape for
project briefs, goals, tasks, specifications, handoffs, work logs, and
verification notes. A document MAY start with YAML frontmatter for stable
metadata such as `id`, `status`, `owner`, `createdAt`, and `updatedAt`.

The body is the source of semantic truth. It MUST explain the problem, desired
outcome, completion conditions, and evidence needed to call the work complete.

## Relations and indexes

Use structured data when the CLI or Coordinator needs exact filtering, joins,
dependency traversal, status transitions, or reproducible machine output.
Goals-to-tasks, task dependencies, runs-to-tasks, repository registrations,
worktree state, and the materialized project state are relational/index data.

JSON files are indexes and execution records. They MUST NOT be the only place
where an Agent can learn what a goal or task means. A JSON index MAY contain an
optional method projection such as OKR key results, but the core protocol does
not require a particular management method.

The v1 reference CLI keeps JSON resource files as its CRUD compatibility layer
and does not silently parse arbitrary Markdown edits. When a project keeps both
forms for the same resource, the project MUST either update them together or
declare which one is authoritative in the document frontmatter. A future
two-way projection command can remove that manual synchronization step without
changing the portable file contract.

## Goal contract

Every active goal MUST state an outcome and one or more `doneWhen` conditions.
Every completed goal MUST include inspectable evidence. `method` is optional;
values such as `okr`, `milestone`, `experiment`, and `custom` are extensions.
