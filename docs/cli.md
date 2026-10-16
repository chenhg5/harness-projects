# `hp` CLI

`hp` is the reference CLI for creating and maintaining Harness Projects. It
uses a noun-verb command tree so agents can discover commands incrementally.
The CLI is the local control plane for the Coordinator and adapters.

## Build

```bash
go build -o ./bin/hp ./cmd/hp
```

## Commands

```bash
hp project init --path ./my-project --name "My Project"
hp project status --path ./my-project --format json
hp project validate --path ./my-project --format json
hp project serve --path ./my-project --open
hp file write --project ./my-project --section spec \
  --path architecture/api.md --from ./api.md --dry-run
hp file read --project ./my-project --section state \
  --path current-plan.md --format text
hp schema list --format json
hp schema get --command "file write" --format json

hp goal create --project ./my-project --id modernize-api \
  --title "Modernize API" --description "Reduce failed releases" \
  --outcome "API releases meet the reliability target" \
  --done-when "reliability target is met,verification is recorded" \
  --key-results '[{"id":"reliability","metric":"successful requests","direction":"increase","target":99,"status":"planned"}]' \
  --status active
hp task create --project ./my-project --id inventory-api \
  --title "Inventory API" --goal-id modernize-api \
  --deliverables "API endpoints" --acceptance "integration tests" --priority high
hp task ready --project ./my-project --format json
hp task assign --project ./my-project --id inventory-api \
  --registration-id <registration-id>
hp task inbox --project ./my-project --harness codex \
  --session-id <actual-provider-session-id>
hp coordinator dispatch --project ./my-project --task-id inventory-api \
  --harness codex --session-id <actual-provider-session-id> \
  --worktree ./repos/api --dry-run
hp run create --project ./my-project --id run-001 --task-id inventory-api
hp run update --project ./my-project --id run-001 --status running
hp agent register --project ./my-project --harness codex \
  --session-id <actual-provider-session-id> --modules api,coordinator \
  --capabilities go,testing --focus "Implement API"
hp agent discover --project ./my-project --module api
hp coordinator assign --project ./my-project --registration-id <registration-id>
hp coordinator get --project ./my-project
hp agent discover --project ./my-project --role coordinator
hp wakeup configure --project ./my-project --every 30m \
  --worktree repos/api --prompt-file agents/coordinator-wakeup.md
hp wakeup status --project ./my-project
hp wakeup once --project ./my-project --dry-run
hp wakeup start --project ./my-project
hp agent heartbeat --project ./my-project --id <registration-id> \
  --availability available
hp agent unregister --project ./my-project --id <registration-id>
hp agent prune --project ./my-project --older-than 7d --dry-run
hp state validate --project ./my-project
hp repository add --project ./my-project --id api --path ../api --default-branch main
hp worktree create --project ./my-project --id api-task --repository-id api
hp worktree list --project ./my-project --status active
hp worktree remove --project ./my-project --id api-task
hp context add --project ./my-project --id api-decision \
  --kind decision --title "API boundary" --from ./decision.md
hp context search --project ./my-project --query "API boundary"
hp handoff create --project ./my-project --id api-work \
  --from codex --to claude --objective "Build API" \
  --completed "Added routes" --next "Add tests"
hp handoff latest --project ./my-project --format text
hp worklog create --project ./my-project --id session-001 --agent codex \
  --objective "Implement memory" --completed "Added search" \
  --remaining "Add docs" --verification "go test ./..." --next "Review"
hp worklog latest --project ./my-project --format text
hp cleanup plan --project ./my-project --older-than 30d
hp cleanup apply --project ./my-project --older-than 30d
hp coordinator plan --project ./my-project --format json
```

Commands that write files support `--dry-run`. A successful dry run exits with
code `10`. Non-TTY output defaults to JSON. JSON data goes to stdout and
machine-readable errors go to stderr.

Work logs are chronological session records. Handoffs are responsibility
transfers and may include branch, commit, and blocker details. Cleanup only
selects transient sections (`worklogs`, `artifacts`, and `context/imports`);
the default action archives them under `archive/` and never selects durable
project history.

## Agent session discovery

At the start of work, an agent should register its actual provider session ID
with `hp agent register`. `cc` is accepted as an alias for `claude-code`.
Agents can describe their current focus, modules, capabilities, worktree, and
task. The registration command returns a registration ID for `heartbeat`,
`get`, and `unregister`. `hp agent discover` returns JSON by default for other
agents to filter by harness, module, or capability.

Participation records are stored under `agents/sessions/` and tracked by project
Git. They do not expire or carry an online/offline state. `hp agent discover`
returns all registered participants; `--include-stale` is accepted for
compatibility but is now a no-op. The actual session must register; old local
leases cannot establish historical participation by themselves.

## Coordinator and scheduled wakeups

`hp coordinator assign` designates exactly one registered session as the
project's current Coordinator. Use `--replace` for an explicit handoff; the
assignment history stays in `agents/coordinator.json`. `hp coordinator clear`
ends the current term without removing that session's participation record.
Discovery marks the current Coordinator and accepts `--role coordinator`.

The optional wakeup schedule follows the **role**, so a handoff changes the
target session without rewriting the schedule. Configure either an interval
(`--every 30m`) or a five-field cron expression with an IANA time zone, for
example `--cron '0 9 * * 1-5' --timezone Asia/Shanghai`. The project-relative
worktree must match the Coordinator's registered worktree. Edit the Markdown
routine at `agents/coordinator-wakeup.md` to state what the Coordinator should
check and do at each wakeup. `hp wakeup once` tests one wakeup; `hp wakeup
start` runs a foreground scheduler until stopped. Keep that process running
with a process supervisor if persistent scheduling is wanted. Configuration is
tracked in Git; local checkpoint and bounded delivery logs live under ignored
`artifacts/`.

Wakeups always address an existing provider session ID through its CLI.
Codex uses `codex queue`; Claude Code is probed for an active session before a
separate resume; Cursor additionally requires a current local delivery hint.
If preflight fails, the wakeup is locally `queued` for another attempt. If a
provider process starts but delivery cannot be confirmed, the status is
`uncertain` and automatic retries stop. Inspect the logs shown by `hp wakeup
status` before `hp wakeup once --retry`. No scheduler is started by project
initialization, and a session registration alone does not create a wakeup.
The wakeup routine is a generated convenience file, not a required project
file: older projects remain valid and agents can register without it.

Machine-local direct-delivery hints live under `.harness-project/agent-sessions/`
and are ignored by Git. They expire after two hours by default. `hp agent
heartbeat` renews a hint, `hp agent unregister` ends it, and `hp agent prune`
removes old hints. None of those commands removes a participant. A delivery
hint is self-reported and does not prove a session is running or reachable.
`hp task assign` can use a durable registration or an
explicit `--harness` and `--session-id` pair to write a durable assignment
into the task. `hp task inbox` lets the assigned session find active work and
dependency readiness on its next visit. It does not require a live lease or
start a provider process. Reassignment requires `--replace`; `hp task
unassign` clears the current address while retaining assignment history.
Cross-harness handoffs still use project files. `hp coordinator dispatch`
records an explicit session assignment and run before trying a provider CLI.
Codex uses `codex queue --thread <id>`, so a successful CLI exit confirms that
the message was accepted for the existing session, not that the task is done.
Claude Code and Cursor run non-interactively with an explicit resume ID and
their structured result streams. The task is never marked complete solely
because the provider command exited successfully.

The default `--delivery auto` keeps the task in the target session inbox when
preflight cannot safely begin delivery, for example when its local delivery
hint is missing, expired, busy for a separate resume, tied to another worktree,
or its CLI is unavailable. `--delivery direct` reports that case as an error
while preserving the assignment; `--delivery inbox` deliberately skips the
provider call. `--dry-run` previews local checks without writing or invoking
a provider CLI; its `providerProbePending` field signals that Claude Code's
active-session check still needs to run. The command requires a real existing session ID and a directory
inside the project. New-session creation is not implemented by this command.

Once a provider process starts, an error or missing terminal result is
recorded as uncertain delivery. The CLI never automatically resends it.
Inspect the run and local `artifacts/dispatch/` logs before an intentional
`--retry`. Logs are local and ignored by the project Git repository. An
exclusive local lock prevents overlapping `hp` calls to the same session;
provider processes started outside `hp` are not covered by that lock.

## Local dashboard

`hp project serve --path ./my-project --open` starts a local, read-only Web
dashboard. The server listens on `127.0.0.1` and chooses a free port unless
`--port` is provided. It prints its URL on startup; stop it with Ctrl-C. The
section navigation lets you jump between the overview, goals, tasks, next
actions, repositories, and activity. Use Refresh view to load updated files;
`GET /api/project` returns the same project summary as JSON. The dashboard reads
the project files again for each request. The page supports English and
Simplified Chinese through its header switch or `?lang=en` / `?lang=zh-CN`.
Task percentages count completed tasks; they do not change goal status or claim
that a goal's acceptance conditions have been met.
