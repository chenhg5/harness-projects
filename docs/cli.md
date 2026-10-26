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
hp project init --path ./my-project --name "My Project"   # own Git repo; --no-git to skip
hp brief --project ./my-project
hp project doctor --path ./my-project
hp project upgrade --path ./my-project --dry-run
hp layout list --project ./my-project --format text
hp layout add --project ./my-project --name lib --description "Third-party reference material"
hp hook install --project ./my-project
hp skill sync --project ./my-project --targets claude,agents --dry-run
hp skill sync --project ./my-project --targets claude,agents
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
hp state reconcile --project ./my-project
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
hp worklog create --project ./my-project --id memory-search --agent codex \
  --objective "Implement memory" --completed "Added search" \
  --remaining "Add docs" --verification "go test ./..." --next "Review"
hp finish --project ./my-project --task-id inventory-api --id api-iteration-1 \
  --objective "Implement API" --completed "Added routes" \
  --remaining "None" --verification "go test ./..." --next "Review"
hp worklog latest --project ./my-project --format text
hp cleanup plan --project ./my-project --older-than 30d
hp cleanup apply --project ./my-project --older-than 30d
hp coordinator plan --project ./my-project --format json
```

Commands that write files support `--dry-run`. A successful dry run exits with
code `10`; `hp finish --dry-run` also returns `"dryRun": true`. Non-TTY output
defaults to JSON. JSON data goes to stdout and machine-readable errors go to
stderr.

## Discoverability and machine contract

Command help and machine-readable schemas come from the same command contract:

```bash
hp help task create
hp schema get --command "task create" --json
```

Help puts runnable examples before prose and lists every long flag, accepted
values, whether it is required, its default, and the output/error contract.
`schema get` returns the same description, flags, defaults, examples, output
shape, and semantic failures as JSON. Every command that supports formatted
output accepts both `--format json|text` and the `--json` shortcut.
Do not infer flags from another command or from stale documentation; inspect the
command contract when generating a call.

Failures are JSON on stderr with `error`, `message`, `suggestion`, `retryable`,
`input`, and `exitCode`. Exit codes are `1` for operation or validation failure,
`2` for invalid arguments, `3` for not found, `4` for permission denied, `5`
for conflict, and `10` for a successful dry run. Conflict classification uses
typed errors rather than matching message text.

Work logs are chronological session records. `--id` names a work log; it is
distinct from the provider's session ID. `hp finish` and `hp worklog create`
detect the current session where supported, and also accept an explicit
`--session-id <actual-provider-session-id>`. Never invent that ID. The generated
template uses English headings, while the work log specification permits
historical and translated prose that carries the same core information.
Handoffs are responsibility transfers and may include branch, commit, and
blocker details. Cleanup only
selects transient sections (`worklogs`, `artifacts`, and `context/imports`);
the default action archives them under `archive/` and never selects durable
project history.

Use `hp finish` when the session has a task. Pass `--status in_progress` after a
partial iteration so the task stays open while the work log and state remain
consistent. It validates all three records
before writing, updates the task, creates the work log, and derives active and
blocked task IDs, task progress, next action, verification time, and source
references in `state/project-state.json`. `hp worklog create` remains available
for meaningful work that has no task record. Create a handoff in addition only
when responsibility moves to another agent or harness; a handoff never replaces
`hp finish`.

## Project health and upgrades

`hp project validate` checks the portable file structure. `hp project doctor`
checks operational consistency: state freshness, task/state agreement, source
references, work-log shape, all four skill mount targets, and registered versus
physical repositories and worktrees. It also checks every top-level directory
against the built-in layout plus `.harness-project/layout.json`. It exits `1`
when it finds drift and does not modify the project.

Inspect the directory contract before adding a new top-level folder:

```bash
hp layout list --project . --format text
hp layout add --project . --name lib \
  --description "Third-party reference material" --dry-run
hp layout add --project . --name lib \
  --description "Third-party reference material" --if-not-exists
```

`hp layout add` records intent but does not create the directory. Built-in
locations cannot be overridden. Use `--if-not-exists` for safe retries and
`--replace` to update an existing description. Project-specific entries appear
in `hp brief`, so a fresh agent knows where that project's extra material belongs.

For agents, `layout list` returns `action`, `project`, `contractPath`,
`schemaVersion`, `contractExists`, and `directories`. `layout add` returns only
the affected `directory` plus `dryRun`, `changed`, `contractCreated`, `replaced`,
and `previousDescription` when relevant; it does not repeat the full catalog.
Successful dry runs exit `10`. Invalid names and mutually exclusive flags exit
`2`; an existing or built-in directory conflict exits `5`. Errors are JSON on
stderr with `error`, `message`, `suggestion`, `retryable`, `input`, and
`exitCode`.

Use `hp state reconcile` to repair task-backed state drift without rewriting
human judgments such as phase, health, risks, or an intentionally curated next
action. It derives active and blocked task IDs and progress from task records.

After installing a newer `hp`, preview `hp project upgrade --path . --dry-run`,
then apply it without `--dry-run`. Upgrade maintains a marked workflow section
inside existing instructions, merges Claude Code hooks, removes the legacy rule
that ignored durable worktree records, and synchronizes Codex, Claude Code,
Cursor, and `.agents/skills` mounts. It also creates the directory contract for
projects made by older versions. Text outside the managed markers is kept.

## Session start: `hp brief`

`hp brief` is the first command of every session. It finds the project from the
current directory or any directory inside it (including nested repository
worktrees under `repos/`) and prints the current goal, state, next actions and
risks, in-progress, ready, and blocked tasks, the current plan, and the
remaining work from the latest work log and handoff.

```bash
hp brief                       # text in a terminal, JSON when piped
hp brief --format text --full  # complete plan, work log, and handoff
hp brief --project ./my-project --lines 15 --json
```

Long sections are bounded (`--lines`, default 30) so the brief stays readable.
It warns when `state/project-state.json` is older than the latest work log.

## Session hooks

`hp project init` writes `.claude/settings.json` with two Claude Code hooks;
`hp hook install` adds them to an existing project without touching other
settings. Both hooks do nothing when `hp` is not installed or the session is
outside a Harness Project, and never fail the session.

| Hook | Command | Effect |
| --- | --- | --- |
| `SessionStart` | `hp hook session-start --harness claude-code` | Registers the session from the hook input, exports `HP_HARNESS` and `HP_SESSION_ID` to later shell commands through `CLAUDE_ENV_FILE`, and prints `hp brief` into the session context. |
| `Stop` | `hp hook stop --harness claude-code` | If the project or any repository under `repos/` (or a registered repository path) changed since the session started and no work log or handoff was written, asks the agent once to record progress. |

Codex reads `AGENTS.md` and runs `hp brief` itself. Its hooks file uses the same
event names, so `hp hook session-start --harness codex` and `hp hook stop
--harness codex` can be added to a Codex hooks configuration; that path has not
been verified by the project yet.

## Skill mounts

`hp skill sync` discovers canonical skills below `skills/` and creates relative
links in the selected harness directories. The default target is `all`; use
`--targets codex,claude,cursor,agents` to be explicit. It never overwrites a
real file or directory. `--prune` removes only stale symlinks that still point
into the project's canonical `skills/` tree.

`hp context search` searches both structured records created by `hp context
add` and ordinary Markdown or text notes below `context/` (excluding imports
and the structured-memory storage tree). Every whitespace-separated query term
must match. Retry with fewer or more specific terms when a broad query returns
no results.

## Agent session discovery

`hp agent register` and `hp task inbox` detect the current session when
`--harness` and `--session-id` are omitted: first `HP_HARNESS` and
`HP_SESSION_ID` (exported by the session hook or set by hand), then Codex's
`CODEX_THREAD_ID`. An explicit flag always wins, and a detected ID is never
used for a different explicit harness. Never invent an ID; when nothing is
detected the command fails and says how to provide one.

At the start of work, an agent should register its actual provider session ID
with `hp agent register`. `cc` is accepted as an alias for `claude-code`.
Agents can describe their current focus, modules, capabilities, worktree, and
task. The registration command returns a registration ID for `heartbeat`,
`get`, and `unregister`. `hp agent discover` returns JSON by default for other
agents to filter by harness, module, or capability. When `--worktree` is omitted,
registration uses the current directory if it is inside the project; otherwise
it uses the project root. Pass `--worktree` explicitly when working from an
external repository checkout.

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
