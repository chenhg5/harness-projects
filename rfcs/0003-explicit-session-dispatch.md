# RFC 0003: Explicit Session Selection for CLI Dispatch

- **Status:** Draft
- **Authors:** Harness Projects contributors
- **Created:** 2026-10-15

## Summary

When Harness Projects dispatches work through a coding harness CLI, the caller
must select the target conversation explicitly. A provider session is not a
Harness Projects run: several runs may use one session, and one task may have
runs in different harnesses. The Coordinator records the requested session,
the effective session reported by the provider, and the exact worktree used.

## Motivation

The current `hp run create` records a planned execution but does not launch a
coding agent or bind that run to a provider conversation. Launching a CLI
without a session choice can silently create a fresh conversation or continue
the latest conversation in a directory. Either outcome can lose the intended
context or mix unrelated work.

Existing implementations provide useful patterns: bot-connect runs Claude Code
and Codex CLIs with explicit resume IDs and reads the resulting ID from their
JSON event streams; harness-bot persists a conversation-to-session binding;
cc-connect validates project ownership of a saved session and guards against
concurrent use. Harness Projects needs those capabilities while retaining its
provider-neutral task and run records.

## Scope

This RFC covers local CLI execution through provider adapters. It does not
define remote/cloud dispatch, messaging bridges, or a common transcript format.
Task assignment and voluntary inbox discovery are defined in RFC 0005 and do
not require this execution transport. They are the durable fallback when the
selected session cannot safely be called through its provider CLI.
It does not imply that one provider's session can be resumed by another
provider. Cross-harness continuation uses project goals, tasks, work logs,
handoffs, artifacts, and verification evidence.

## Proposed contract

### Explicit selection

`hp dispatch` MUST require `--session`. The value MUST be either `new` or a
specific provider session ID. A future named binding MAY also be accepted if
it resolves to exactly one provider, working directory, and session ID before
launch. Empty values, `latest`, implicit `continue`, and interactive pickers
MUST NOT be used by unattended dispatch.

```sh
hp dispatch --project . --task-id implement-api --harness codex \
  --worktree ./repos/api/feature-x --session new

hp dispatch --project . --task-id implement-api --harness codex \
  --worktree ./repos/api/feature-x --session 0192-example-provider-id
```

`--session new` is an explicit request for a fresh conversation. An existing ID
is an explicit request to resume that conversation. The CLI MUST NOT change
one request into the other as an automatic fallback. If an ID cannot be
validated or resumed, dispatch fails with a recoverable error and leaves the
requested ID in the run record for inspection.

For a known existing session, the Coordinator SHOULD persist the task's
assignment before attempting CLI delivery. If adapter preflight cannot start
delivery, it SHOULD return a distinct queued-for-inbox outcome under RFC 0005.
If the provider call may already have sent the prompt, the outcome is uncertain:
the Coordinator MUST preserve the attempted run and require reconciliation
before retrying. It MUST NOT create a new session or send the prompt again as
an automatic fallback. A `new` session request cannot fall back to a session
inbox unless the provider has returned an effective session ID.

### Identity and binding

The Coordinator MUST distinguish:

- the provider-neutral task ID;
- the provider-neutral run ID for one execution attempt;
- the provider name and effective provider session ID;
- the absolute, resolved working directory or worktree for that attempt.

The run MUST record both the requested session selection and the effective
session ID returned by the adapter. The adapter MUST capture the effective ID
from provider output even when it becomes available only after the first
turn. If a provider returns a different ID after resume or forks a session,
the run MUST retain the requested ID and update the effective ID. A later run
may resume only the effective ID selected explicitly by its caller.

A reusable session binding SHOULD be scoped to the provider, its configuration
profile, and the resolved working directory. An adapter SHOULD validate that
an existing ID belongs to that scope before sending the prompt. If ownership
cannot be verified, the adapter MUST state that limitation and MUST NOT infer
ownership from the current directory alone.

An agent session registered for discovery under RFC 0004 MAY help a caller
choose a target. Registration alone does not grant a transport or prove that
the provider session is safe to resume. The caller must still select its
session explicitly for CLI dispatch.

### Execution safety and results

The Coordinator MUST prevent two local dispatches from writing to the same
provider session concurrently. If it cannot determine whether an external
process is already using that session, it MUST refuse the CLI resume with a
clear error unless the provider offers a safe attach mechanism. A session lock
does not grant tool permissions; adapter execution follows the configured
harness permission policy.

The adapter MUST preserve the provider event stream or a diagnostic log,
process exit status, and final response as run artifacts. A successful process
exit MUST NOT by itself mark the task complete. The run's verification and the
task's acceptance conditions determine completion under the existing
execution-evidence contract.

`--dry-run` MUST resolve the task, adapter, worktree, and session selection
without starting a provider process or changing a provider session.

## Adapter boundary

The core Coordinator owns selection, run lifecycle, locking, and persistence.
Each provider adapter owns argument construction, process lifecycle, JSON event
parsing, and provider-specific session validation. The first CLI adapters may
cover Codex and Claude Code; Cursor follows the same contract once its local
session and output behavior is verified. The core must not depend on provider
event names such as `thread.started` or `session_id`.

## Scenarios

### Resume an existing Codex session

- **GIVEN** a Codex session ID and a worktree are specified
- **WHEN** the Coordinator dispatches a ready task
- **THEN** the Codex adapter resumes exactly that ID in the specified worktree
- **AND** the run records the requested and effective IDs

### Resume fails

- **GIVEN** a specified Claude Code session ID cannot be resumed
- **WHEN** the adapter reports the failure
- **THEN** the run fails with the resume error
- **AND** no new conversation is created implicitly

### Cross-harness continuation

- **GIVEN** a Codex run ends with a handoff and recorded artifacts
- **WHEN** a Claude Code run starts with `--session new`
- **THEN** the new run receives the project task and handoff context
- **AND** it gets a distinct Claude Code session ID

## Compatibility and migration

Existing run records without session metadata remain valid as historical or
manually executed runs. `hp run create` remains a record-creation command;
the explicit-session requirement applies when `hp` starts a provider CLI.

## Open questions

- Should reusable session bindings be a separate indexed resource or a view
  over run records?
- How should the CLI expose an active desktop/IDE session that cannot safely
  be resumed by a second local process?
- Which provider-specific validation checks are reliable across installations?
