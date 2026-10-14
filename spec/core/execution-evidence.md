# Execution evidence

Harness Projects treats completion as an evidence-backed state transition.

## Goal completion

An agent MUST NOT mark a goal `completed` because its title or acceptance text
looks satisfied. The goal record MUST contain evidence references such as
verification files, release URLs, review records, metrics, or other inspectable
outputs. If the project chooses OKR as its method, key results are additional
evidence and are not a universal protocol requirement.

## Task completion

A task record MUST name the deliverable, acceptance criteria, and result. The
completion update MUST include a verification or evidence reference. If work is
blocked, the record MUST explain the blocker and identify the next action.

## Run success

A run marked `succeeded` MUST identify what was verified and where the evidence
can be inspected. Commits, changed files, logs, and approvals SHOULD be
recorded when applicable.
