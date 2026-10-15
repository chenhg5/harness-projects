package cli

import "github.com/chenhg5/harness-projects/internal/model"

const rootHelp = `Harness Projects CLI (hp)

Examples:
  hp project init --path ./my-project --name "My Project"
  hp project validate --path ./my-project --format json
  hp file write --project ./my-project --section spec --path architecture.md --from ./draft.md --dry-run

Commands:
  project init|status|validate|serve Create, inspect, and view Harness Projects
  goal create|list|get|update     Manage long-lived project goals
  task create|list|get|update|ready|assign|unassign|inbox Manage dependent work items
  run create|list|get|update      Manage execution attempts
  agent register|discover|get|heartbeat|unregister|prune Discover agent sessions
  repository add|list|get         Register participating code repositories
  worktree create|list|get|remove Manage isolated Git worktrees
  context add|list|get|search     Manage portable project memory
  handoff create|list|latest      Transfer work between agents
  worklog create|list|latest      Record and read agent work sessions
  cleanup plan|apply              Archive stale transient files safely
  state get|update|validate       Inspect and update the project control snapshot
  coordinator plan|dispatch       Plan work and deliver it to an explicit session
  file write|read                 Write and read project files
  schema list|get                 Inspect the command interface

Use hp help <noun> <verb> for command details.
Exit codes: 0 success, 1 operation/validation error, 2 usage, 3 not found,
4 permission denied, 5 conflict, 10 successful dry run.
`

var groupHelp = map[string]string{
	"project":     "Project commands: init, status, validate, serve\nExample: hp project serve --path ./my-project --open\n",
	"goal":        "Goal commands: create, list, get, update\nExample: hp goal list --project ./my-project\n",
	"task":        "Task commands: create, list, get, update, ready, assign, unassign, inbox\nExample: hp task inbox --project ./my-project --harness codex --session-id <id>\n",
	"run":         "Run commands: create, list, get, update\nExample: hp run list --project ./my-project\n",
	"agent":       "Agent commands: register, discover, get, heartbeat, unregister, prune\nExample: hp agent discover --project ./my-project\n",
	"repository":  "Repository commands: add, list, get\nExample: hp repository add --project ./my-project --id api --path ../api\n",
	"worktree":    "Worktree commands: create, list, get, remove\nExample: hp worktree create --project ./my-project --id api-task --repository-id api\n",
	"context":     "Context commands: add, list, get, search\nExample: hp context search --project ./my-project --query architecture\n",
	"handoff":     "Handoff commands: create, list, latest\nExample: hp handoff latest --project ./my-project\n",
	"worklog":     "Work log commands: create, list, latest\nExample: hp worklog latest --project ./my-project\n",
	"cleanup":     "Cleanup commands: plan, apply\nExample: hp cleanup plan --project ./my-project --older-than 30d\n",
	"state":       "State commands: get, update, validate\nExample: hp state validate --project ./my-project\n",
	"coordinator": "Coordinator commands: plan, dispatch\nExample: hp coordinator dispatch --project ./my-project --task-id api --harness codex --session-id <id> --worktree ./repos/api\n",
	"file":        "File commands: write, read\nExample: hp file read --project ./my-project --section spec --path architecture.md\n",
	"schema":      "Schema commands: list, get\nExample: hp schema get --command 'project init'\n",
}

var commandHelp = map[string]string{
	"project init": `hp project init --path <directory> [--name <name>] [--if-not-exists] [--dry-run] [--format json|text]
  Create a portable Harness Project with spec, handoff, skill, state, and harness entry points.
  Example: hp project init --path ./checkout --name "Checkout" --dry-run
  Example: hp project init --path ./checkout --if-not-exists --json
`,
	"project status": `hp project status --path <directory> [--format json|text]
  Report project identity and structural problems. Exits 0 even if invalid.
  Example: hp project status --path ./checkout --json
`,
	"project validate": `hp project validate --path <directory> [--format json|text]
  Validate project structure. Exits 1 if invalid.
  Example: hp project validate --path ./checkout --json
`,
	"project serve": `hp project serve --path <directory> [--port 0..65535] [--open] [--format json|text]
  Serve a read-only local dashboard with goals, task progress, state, repositories, and recent agent activity. Binds to 127.0.0.1 only. Press Ctrl-C to stop.
  Example: hp project serve --path ./checkout --open
`,
	"file write": `hp file write --project <directory> --section <section> --path <relative-path> --from <file|-> [--replace|--if-not-exists] [--dry-run] [--format json|text]
  Write a text file under a canonical project section: spec, goals, tasks, runs, agents, context, handoffs, events, verification, skills, scripts, state, artifacts, or integrations.
  Existing files require --replace or --if-not-exists. Dry runs exit 10.
  Example: hp file write --project ./checkout --section spec --path architecture/api.md --from ./api.md --dry-run
  Example: hp file write --project ./checkout --section state --path current-plan.md --from - --replace
`,
	"file read": `hp file read --project <directory> --section <section> --path <relative-path> [--format json|text]
  Read one project file.
  Example: hp file read --project ./checkout --section state --path current-plan.md --json
`,
	"goal create": `hp goal create --project <directory> --id <id> --title <title> [--description <text>] [--outcome <text>] [--method <method>] [--done-when a,b] [--evidence-required a,b] [--owner <id>] [--target-date <date>] [--key-results <json>] [--status draft|active|completed|archived] [--dry-run]
	Create an outcome-oriented goal. Active goals require a description, outcome, and at least one doneWhen condition; completed goals require evidence. Methods such as OKR are optional.
`,
	"goal list": `hp goal list --project <directory> [--status <status>] [--format json|text]
  List project goals.
`,
	"goal get": `hp goal get --project <directory> --id <id> [--format json|text]
  Read one goal.
`,
	"goal update": `hp goal update --project <directory> --id <id> [--title <title>] [--description <text>] [--outcome <text>] [--method <method>] [--done-when a,b] [--evidence-required a,b] [--owner <id>] [--target-date <date>] [--key-results <json>] [--evidence a,b] [--status <status>] [--dry-run]
  Update selected goal fields.
`,
	"task create": `hp task create --project <directory> --id <id> --title <title> [--description <text>] [--deliverables a,b] [--acceptance a,b] [--priority critical|high|medium|low] [--owner <id>] [--due-date <date>] [--goal-id <id>] [--depends-on a,b] [--status pending|ready|blocked] [--dry-run]
	Create an execution unit with deliverables, acceptance, ownership, and dependencies.
`,
	"task list": `hp task list --project <directory> [--status <status>] [--goal-id <id>] [--format json|text]
  List tasks.
`,
	"task get": `hp task get --project <directory> --id <id> [--format json|text]
  Read one task.
`,
	"task update": `hp task update --project <directory> --id <id> [--status <status>] [--deliverables a,b] [--acceptance a,b] [--priority <priority>] [--owner <id>] [--blocked-reason <text>] [--result <text>] [--evidence a,b] [--verification a,b] [--dry-run]
  Update selected task fields.
`,
	"task ready": `hp task ready --project <directory> [--format json|text]
  Compute tasks whose dependencies are complete and which can be dispatched.
`,
	"task assign": `hp task assign --project <directory> --id <task-id> (--registration-id <id> | --harness <name> --session-id <provider-id>) [--replace] [--dry-run]
  Persist a task assignment to a provider session. This does not start the provider. A registration must be live; explicit harness and session ID allow assignment while offline.
`,
	"task unassign": `hp task unassign --project <directory> --id <task-id> [--dry-run]
  Remove the current assignment and retain its history.
`,
	"task inbox": `hp task inbox --project <directory> --harness <name> --session-id <provider-id> [--ready-only] [--all] [--format json|text]
  Discover tasks assigned to this exact provider session. Includes readiness and dependency details; completed and cancelled tasks are hidden unless --all is set.
`,
	"run create": `hp run create --project <directory> --id <id> --task-id <id> [--adapter local] [--harness <name>] [--agent-id <id>] [--objective <text>] [--command <command>] [--worktree <path>] [--dry-run]
  Record a planned execution attempt. It does not start a provider process.
`,
	"run list": `hp run list --project <directory> [--task-id <id>] [--status <status>] [--format json|text]
  List execution attempts.
`,
	"run get": `hp run get --project <directory> --id <id> [--format json|text]
  Read one execution attempt.
`,
	"run update": `hp run update --project <directory> --id <id> [--status <status>] [--commit <sha>] [--changed-files a,b] [--verification a,b] [--logs a,b] [--error <message>] [--dry-run]
  Update run state and record timestamps for terminal states.
`,
	"agent register": `hp agent register --project <directory> --harness <name> --session-id <provider-id> [--worktree <path>] [--task-id <id>] [--availability available|busy|paused] [--focus <text>] [--modules a,b] [--capabilities a,b] [--ttl 2h] [--dry-run]
  Register this live provider session for discovery. cc is an alias for claude-code. The CLI cannot infer your session ID.
`,
	"agent discover": `hp agent discover --project <directory> [--harness <name>] [--module <name>] [--capability <name>] [--include-stale] [--format json|text]
  Find registered agent sessions. Expired registrations are hidden unless --include-stale is set.
`,
	"agent get": `hp agent get --project <directory> --id <registration-id> [--format json|text]
  Read one agent session registration.
`,
	"agent heartbeat": `hp agent heartbeat --project <directory> --id <registration-id> [--availability available|busy|paused] [--focus <text>] [--ttl 2h] [--dry-run]
  Renew an agent session registration after meaningful work or while waiting for more tasks.
`,
	"agent unregister": `hp agent unregister --project <directory> --id <registration-id> [--dry-run]
  Remove an agent session registration when the session leaves this project.
`,
	"agent prune": `hp agent prune --project <directory> [--older-than 7d] [--dry-run] [--format json|text]
  Remove registrations whose lease expired at least this long ago.
`,
	"repository add": `hp repository add --project <directory> --id <id> [--url <url>] [--path <local-git-repository>] [--default-branch <branch>] [--provider <name>] [--if-not-exists] [--dry-run]
  Register a repository used by this project. Registration never clones or changes the repository.
`,
	"repository list": `hp repository list --project <directory> [--format json|text]
  List registered repositories.
`,
	"repository get": `hp repository get --project <directory> --id <id> [--format json|text]
  Read one repository registration.
`,
	"worktree create": `hp worktree create --project <directory> --id <id> --repository-id <id> [--path <directory>] [--branch <branch>] [--base <revision>] [--dry-run]
  Create and register an isolated Git worktree under the project repos directory.
`,
	"worktree list": `hp worktree list --project <directory> [--repository-id <id>] [--status <status>] [--format json|text]
  List registered worktrees.
`,
	"worktree get": `hp worktree get --project <directory> --id <id> [--format json|text]
  Read one worktree registration.
`,
	"worktree remove": `hp worktree remove --project <directory> --id <id> [--force] [--dry-run] [--format json|text]
  Remove the Git worktree and mark its registration as removed.
`,
	"coordinator plan": `hp coordinator plan --project <directory> [--format json|text]
  Compute dispatchable and waiting tasks without starting an Agent.
`,
	"coordinator dispatch": `hp coordinator dispatch --project <directory> --task-id <id> --harness codex|claude-code|cursor --session-id <existing-provider-id> --worktree <directory> [--delivery auto|direct|inbox] [--message <text>] [--run-id <id>] [--timeout 30m] [--replace] [--retry] [--dry-run]
  Persist the task assignment, then try an exact-session provider CLI delivery. Auto mode queues the task in the session inbox when delivery cannot begin safely. Once a provider process starts, an unclear outcome requires reconciliation; it is never silently resent. Codex uses its queue command; Claude Code and Cursor resume by ID in non-interactive mode.
`,
	"context add": `hp context add --project <directory> --id <id> --kind decision|lesson|preference|research --title <title> --from <file|-> [--tags a,b] [--source <reference>] [--dry-run]
  Store one portable project memory record under context/memory/.
`,
	"context list": `hp context list --project <directory> [--kind <kind>] [--format json|text]
  List project memory records.
`,
	"context get": `hp context get --project <directory> --kind <kind> --id <id> [--format json|text]
  Read one project memory record.
`,
	"context search": `hp context search --project <directory> --query <terms> [--kind <kind>] [--format json|text]
  Search title, tags, content, and source; all query terms must match.
`,
	"handoff create": `hp handoff create --project <directory> --id <id> --from <agent> --to <agent> --objective <text> --completed <text> --next <text> [--repositories <text>] [--verification <text>] [--blockers <text>] [--dry-run]
  Append one durable Markdown handoff.
`,
	"handoff list": `hp handoff list --project <directory> [--format json|text]
  List handoff files in chronological filename order.
`,
	"handoff latest": `hp handoff latest --project <directory> [--format json|text]
  Read the latest handoff file.
`,
	"worklog create": `hp worklog create --project <directory> --id <id> --agent <agent> --objective <text> --completed <text> --remaining <text> --next <text> [--harness <name>] [--task-id <id>] [--run-id <id>] [--changes <text>] [--verification <text>] [--blockers <text>] [--dry-run]
  Record one agent session. Create this after each meaningful session; use handoff when responsibility moves between agents.
`,
	"worklog list": `hp worklog list --project <directory> [--format json|text]
  List work logs in chronological filename order.
`,
	"worklog latest": `hp worklog latest --project <directory> [--format json|text]
  Read the latest work log.
`,
	"cleanup plan": `hp cleanup plan --project <directory> [--older-than 30d] [--sections worklogs,artifacts,context/imports] [--format json|text]
  Preview stale transient files eligible for archival. Durable goals, specs, tasks, runs, and handoffs are never selected.
`,
	"cleanup apply": `hp cleanup apply --project <directory> [--older-than 30d] [--sections worklogs,artifacts,context/imports] [--action archive|delete] [--force] [--dry-run]
  Archive stale files by default. Deletion requires --action delete and --force; dry runs exit 10.
`,
	"state get": `hp state get --project <directory> [--format json|text]
  Read the structured project control snapshot.
`,
	"state validate": `hp state validate --project <directory> [--format json|text]
  Validate that the state snapshot has an owner goal, health, progress, next actions, and source references.
`,
	"state update": `hp state update --project <directory> [--phase <phase>] [--health unknown|on_track|at_risk|blocked|complete] [--progress <summary>] [--current-goal-id <id>] [--active-tasks a,b] [--blocked-tasks a,b] [--next-actions a,b] [--risks a,b] [--last-verified-at <timestamp>] [--source-refs a,b] [--dry-run]
  Update the materialized state snapshot after changing goals, tasks, runs, or verification evidence.
`,
	"schema list": `hp schema list [--format json|text]
  List introspectable CLI commands.
  Example: hp schema list --json
`,
	"schema get": `hp schema get --command <noun-verb> [--format json|text]
  Return one command's flags and examples.
  Example: hp schema get --command 'file write' --json
`,
}

type flagSchema struct {
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Required bool     `json:"required"`
	Values   []string `json:"values,omitempty"`
}

type commandSchema struct {
	Command     string       `json:"command"`
	Description string       `json:"description"`
	Flags       []flagSchema `json:"flags"`
	Examples    []string     `json:"examples"`
}

var commandSchemas = map[string]commandSchema{
	"project init":         {"project init", "Create a Harness Project", []flagSchema{{"path", "string", true, nil}, {"name", "string", false, nil}, {"if-not-exists", "boolean", false, nil}, {"dry-run", "boolean", false, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp project init --path ./my-project --dry-run"}},
	"project status":       {"project status", "Inspect a Harness Project", []flagSchema{{"path", "string", true, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp project status --path ./my-project --json"}},
	"project validate":     {"project validate", "Validate a Harness Project", []flagSchema{{"path", "string", true, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp project validate --path ./my-project --json"}},
	"project serve":        {"project serve", "Serve a read-only local project dashboard", []flagSchema{{"path", "string", true, nil}, {"port", "integer", false, nil}, {"open", "boolean", false, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp project serve --path ./my-project --open"}},
	"file write":           {"file write", "Write a project file", []flagSchema{{"project", "string", true, nil}, {"section", "enum", true, []string{"spec", "goals", "tasks", "runs", "agents", "context", "handoffs", "events", "verification", "skills", "scripts", "state", "artifacts", "integrations"}}, {"path", "string", true, nil}, {"from", "string", true, nil}, {"replace", "boolean", false, nil}, {"if-not-exists", "boolean", false, nil}, {"dry-run", "boolean", false, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp file write --project ./my-project --section spec --path requirements.md --from ./draft.md"}},
	"file read":            {"file read", "Read a project file", []flagSchema{{"project", "string", true, nil}, {"section", "enum", true, []string{"spec", "goals", "tasks", "runs", "agents", "context", "handoffs", "events", "verification", "skills", "scripts", "state", "artifacts", "integrations"}}, {"path", "string", true, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp file read --project ./my-project --section spec --path requirements.md --json"}},
	"goal create":          {"goal create", "Create an outcome-oriented project goal", []flagSchema{{"project", "string", true, nil}, {"id", "string", true, nil}, {"title", "string", true, nil}, {"description", "string", false, nil}, {"outcome", "string", false, nil}, {"method", "enum", false, []string{"okr", "milestone", "experiment", "metric", "custom", "none"}}, {"done-when", "string", false, nil}, {"evidence-required", "string", false, nil}, {"owner", "string", false, nil}, {"target-date", "string", false, nil}, {"review-cadence", "string", false, nil}, {"key-results", "json", false, nil}, {"evidence", "string", false, nil}, {"status", "enum", false, model.GoalStatuses()}, {"acceptance", "string", false, nil}, {"dry-run", "boolean", false, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp goal create --project ./my-project --id modernize-api --title 'Modernize API' --description '...' --outcome '...' --key-results '[{\"id\":\"coverage\",\"metric\":\"test coverage\",\"direction\":\"increase\",\"target\":90,\"status\":\"planned\"}]'"}},
	"goal list":            {"goal list", "List project goals", []flagSchema{{"project", "string", true, nil}, {"status", "enum", false, model.GoalStatuses()}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp goal list --project ./my-project --json"}},
	"goal get":             {"goal get", "Read a project goal", []flagSchema{{"project", "string", true, nil}, {"id", "string", true, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp goal get --project ./my-project --id modernize-api --json"}},
	"goal update":          {"goal update", "Update a project goal", []flagSchema{{"project", "string", true, nil}, {"id", "string", true, nil}, {"title", "string", false, nil}, {"description", "string", false, nil}, {"outcome", "string", false, nil}, {"method", "enum", false, []string{"okr", "milestone", "experiment", "metric", "custom", "none"}}, {"done-when", "string", false, nil}, {"evidence-required", "string", false, nil}, {"owner", "string", false, nil}, {"target-date", "string", false, nil}, {"review-cadence", "string", false, nil}, {"key-results", "json", false, nil}, {"evidence", "string", false, nil}, {"status", "enum", false, model.GoalStatuses()}, {"acceptance", "string", false, nil}, {"dry-run", "boolean", false, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp goal update --project ./my-project --id modernize-api --status active"}},
	"task create":          {"task create", "Create an execution task", []flagSchema{{"project", "string", true, nil}, {"id", "string", true, nil}, {"title", "string", true, nil}, {"description", "string", false, nil}, {"deliverables", "string", false, nil}, {"acceptance", "string", false, nil}, {"priority", "enum", false, []string{"critical", "high", "medium", "low"}}, {"owner", "string", false, nil}, {"due-date", "string", false, nil}, {"goal-id", "string", false, nil}, {"depends-on", "string", false, nil}, {"agent-id", "string", false, nil}, {"status", "enum", false, model.TaskStatuses()}, {"dry-run", "boolean", false, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp task create --project ./my-project --id inventory-api --title 'Inventory API' --deliverables 'API endpoints' --acceptance 'integration tests'"}},
	"task get":             {"task get", "Read a project task", []flagSchema{{"project", "string", true, nil}, {"id", "string", true, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp task get --project ./my-project --id inventory-api --json"}},
	"task update":          {"task update", "Update a project task", []flagSchema{{"project", "string", true, nil}, {"id", "string", true, nil}, {"status", "enum", false, model.TaskStatuses()}, {"deliverables", "string", false, nil}, {"acceptance", "string", false, nil}, {"priority", "enum", false, []string{"critical", "high", "medium", "low"}}, {"owner", "string", false, nil}, {"blocked-reason", "string", false, nil}, {"result", "string", false, nil}, {"evidence", "string", false, nil}, {"verification", "string", false, nil}, {"depends-on", "string", false, nil}, {"dry-run", "boolean", false, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp task update --project ./my-project --id inventory-api --status completed --evidence verification/2026-10-14.md"}},
	"task ready":           {"task ready", "List dispatchable tasks", []flagSchema{{"project", "string", true, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp task ready --project ./my-project --json"}},
	"task assign":          {"task assign", "Assign a task to a provider session without starting it", []flagSchema{{"project", "string", true, nil}, {"id", "string", true, nil}, {"registration-id", "string", false, nil}, {"harness", "string", false, nil}, {"session-id", "string", false, nil}, {"replace", "boolean", false, nil}, {"dry-run", "boolean", false, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp task assign --project ./my-project --id inventory-api --registration-id <id>"}},
	"task unassign":        {"task unassign", "Unassign a task while preserving its assignment history", []flagSchema{{"project", "string", true, nil}, {"id", "string", true, nil}, {"dry-run", "boolean", false, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp task unassign --project ./my-project --id inventory-api"}},
	"task inbox":           {"task inbox", "List tasks assigned to a provider session", []flagSchema{{"project", "string", true, nil}, {"harness", "string", true, nil}, {"session-id", "string", true, nil}, {"ready-only", "boolean", false, nil}, {"all", "boolean", false, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp task inbox --project ./my-project --harness codex --session-id <provider-id> --json"}},
	"run create":           {"run create", "Create a planned execution run", []flagSchema{{"project", "string", true, nil}, {"id", "string", true, nil}, {"task-id", "string", true, nil}, {"adapter", "string", false, nil}, {"harness", "string", false, nil}, {"agent-id", "string", false, nil}, {"objective", "string", false, nil}, {"command", "string", false, nil}, {"worktree", "string", false, nil}, {"dry-run", "boolean", false, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp run create --project ./my-project --id run-001 --task-id inventory-api --objective 'Verify API'"}},
	"run list":             {"run list", "List execution runs", []flagSchema{{"project", "string", true, nil}, {"task-id", "string", false, nil}, {"status", "enum", false, model.RunStatuses()}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp run list --project ./my-project --json"}},
	"run get":              {"run get", "Read an execution run", []flagSchema{{"project", "string", true, nil}, {"id", "string", true, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp run get --project ./my-project --id run-001 --json"}},
	"run update":           {"run update", "Update an execution run", []flagSchema{{"project", "string", true, nil}, {"id", "string", true, nil}, {"status", "enum", false, model.RunStatuses()}, {"worktree", "string", false, nil}, {"commit", "string", false, nil}, {"changed-files", "string", false, nil}, {"verification", "string", false, nil}, {"logs", "string", false, nil}, {"approval", "string", false, nil}, {"exit-code", "integer", false, nil}, {"error", "string", false, nil}, {"artifacts", "string", false, nil}, {"dry-run", "boolean", false, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp run update --project ./my-project --id run-001 --status running"}},
	"agent register":       {"agent register", "Register a live agent session", []flagSchema{{"project", "string", true, nil}, {"harness", "string", true, nil}, {"session-id", "string", true, nil}, {"worktree", "string", false, nil}, {"task-id", "string", false, nil}, {"availability", "enum", false, []string{"available", "busy", "paused"}}, {"focus", "string", false, nil}, {"modules", "string", false, nil}, {"capabilities", "string", false, nil}, {"ttl", "duration", false, nil}, {"dry-run", "boolean", false, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp agent register --project ./my-project --harness codex --session-id <provider-id> --modules api --capabilities go,testing"}},
	"agent discover":       {"agent discover", "Find active agent sessions", []flagSchema{{"project", "string", true, nil}, {"harness", "string", false, nil}, {"module", "string", false, nil}, {"capability", "string", false, nil}, {"include-stale", "boolean", false, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp agent discover --project ./my-project --module api --json"}},
	"agent get":            {"agent get", "Read an agent session registration", []flagSchema{{"project", "string", true, nil}, {"id", "string", true, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp agent get --project ./my-project --id <registration-id>"}},
	"agent heartbeat":      {"agent heartbeat", "Renew an agent session registration", []flagSchema{{"project", "string", true, nil}, {"id", "string", true, nil}, {"availability", "enum", false, []string{"available", "busy", "paused"}}, {"focus", "string", false, nil}, {"ttl", "duration", false, nil}, {"dry-run", "boolean", false, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp agent heartbeat --project ./my-project --id <registration-id> --availability available"}},
	"agent unregister":     {"agent unregister", "Remove an agent session registration", []flagSchema{{"project", "string", true, nil}, {"id", "string", true, nil}, {"dry-run", "boolean", false, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp agent unregister --project ./my-project --id <registration-id>"}},
	"agent prune":          {"agent prune", "Remove old expired agent session registrations", []flagSchema{{"project", "string", true, nil}, {"older-than", "duration", false, nil}, {"dry-run", "boolean", false, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp agent prune --project ./my-project --older-than 7d --dry-run"}},
	"repository add":       {"repository add", "Register a project repository", []flagSchema{{"project", "string", true, nil}, {"id", "string", true, nil}, {"url", "string", false, nil}, {"path", "string", false, nil}, {"default-branch", "string", false, nil}, {"provider", "string", false, nil}, {"if-not-exists", "boolean", false, nil}, {"dry-run", "boolean", false, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp repository add --project ./my-project --id api --path ../api"}},
	"repository list":      {"repository list", "List project repositories", []flagSchema{{"project", "string", true, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp repository list --project ./my-project --json"}},
	"repository get":       {"repository get", "Read a project repository", []flagSchema{{"project", "string", true, nil}, {"id", "string", true, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp repository get --project ./my-project --id api --json"}},
	"worktree create":      {"worktree create", "Create a Git worktree", []flagSchema{{"project", "string", true, nil}, {"id", "string", true, nil}, {"repository-id", "string", true, nil}, {"path", "string", false, nil}, {"branch", "string", false, nil}, {"base", "string", false, nil}, {"dry-run", "boolean", false, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp worktree create --project ./my-project --id api-task --repository-id api"}},
	"worktree list":        {"worktree list", "List project worktrees", []flagSchema{{"project", "string", true, nil}, {"repository-id", "string", false, nil}, {"status", "enum", false, model.WorktreeStatuses()}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp worktree list --project ./my-project --json"}},
	"worktree get":         {"worktree get", "Read a project worktree", []flagSchema{{"project", "string", true, nil}, {"id", "string", true, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp worktree get --project ./my-project --id api-task --json"}},
	"worktree remove":      {"worktree remove", "Remove a Git worktree", []flagSchema{{"project", "string", true, nil}, {"id", "string", true, nil}, {"force", "boolean", false, nil}, {"dry-run", "boolean", false, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp worktree remove --project ./my-project --id api-task --force"}},
	"coordinator plan":     {"coordinator plan", "Compute dispatchable tasks", []flagSchema{{"project", "string", true, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp coordinator plan --project ./my-project --json"}},
	"coordinator dispatch": {"coordinator dispatch", "Assign and deliver a ready task to an exact provider session", []flagSchema{{"project", "string", true, nil}, {"task-id", "string", true, nil}, {"harness", "enum", true, []string{"codex", "claude-code", "cursor"}}, {"session-id", "string", true, nil}, {"worktree", "string", true, nil}, {"delivery", "enum", false, []string{"auto", "direct", "inbox"}}, {"message", "string", false, nil}, {"run-id", "string", false, nil}, {"timeout", "duration", false, nil}, {"replace", "boolean", false, nil}, {"retry", "boolean", false, nil}, {"dry-run", "boolean", false, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp coordinator dispatch --project ./my-project --task-id inventory-api --harness codex --session-id <provider-id> --worktree ./repos/api --dry-run"}},
	"context add":          {"context add", "Store a portable project memory", []flagSchema{{"project", "string", true, nil}, {"id", "string", true, nil}, {"kind", "enum", true, model.MemoryKinds()}, {"title", "string", true, nil}, {"from", "string", true, nil}, {"tags", "string", false, nil}, {"source", "string", false, nil}, {"dry-run", "boolean", false, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp context add --project ./my-project --id api-decision --kind decision --title 'API design' --from ./decision.md"}},
	"context list":         {"context list", "List project memories", []flagSchema{{"project", "string", true, nil}, {"kind", "enum", false, model.MemoryKinds()}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp context list --project ./my-project --json"}},
	"context get":          {"context get", "Read one project memory", []flagSchema{{"project", "string", true, nil}, {"kind", "enum", true, model.MemoryKinds()}, {"id", "string", true, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp context get --project ./my-project --kind decision --id api-decision"}},
	"context search":       {"context search", "Search project memories", []flagSchema{{"project", "string", true, nil}, {"query", "string", true, nil}, {"kind", "enum", false, model.MemoryKinds()}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp context search --project ./my-project --query 'API design'"}},
	"handoff create":       {"handoff create", "Append a durable handoff", []flagSchema{{"project", "string", true, nil}, {"id", "string", true, nil}, {"from", "string", true, nil}, {"to", "string", true, nil}, {"objective", "string", true, nil}, {"completed", "string", true, nil}, {"next", "string", true, nil}, {"repositories", "string", false, nil}, {"verification", "string", false, nil}, {"blockers", "string", false, nil}, {"dry-run", "boolean", false, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp handoff create --project ./my-project --id api-work --from codex --to claude --objective 'Build API' --completed 'Added routes' --next 'Add tests'"}},
	"handoff list":         {"handoff list", "List project handoffs", []flagSchema{{"project", "string", true, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp handoff list --project ./my-project"}},
	"handoff latest":       {"handoff latest", "Read the latest project handoff", []flagSchema{{"project", "string", true, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp handoff latest --project ./my-project --format text"}},
	"worklog create":       {"worklog create", "Record an agent work session", []flagSchema{{"project", "string", true, nil}, {"id", "string", true, nil}, {"agent", "string", true, nil}, {"harness", "string", false, nil}, {"task-id", "string", false, nil}, {"run-id", "string", false, nil}, {"objective", "string", true, nil}, {"completed", "string", true, nil}, {"remaining", "string", true, nil}, {"changes", "string", false, nil}, {"verification", "string", false, nil}, {"next", "string", true, nil}, {"blockers", "string", false, nil}, {"dry-run", "boolean", false, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp worklog create --project ./my-project --id session-001 --agent codex --objective 'Implement memory' --completed 'Added search' --remaining 'Add docs' --next 'Run tests'"}},
	"worklog list":         {"worklog list", "List agent work sessions", []flagSchema{{"project", "string", true, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp worklog list --project ./my-project"}},
	"worklog latest":       {"worklog latest", "Read the latest agent work session", []flagSchema{{"project", "string", true, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp worklog latest --project ./my-project --format text"}},
	"cleanup plan":         {"cleanup plan", "Preview stale transient files", []flagSchema{{"project", "string", true, nil}, {"older-than", "duration", false, nil}, {"sections", "string", false, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp cleanup plan --project ./my-project --older-than 30d"}},
	"cleanup apply":        {"cleanup apply", "Archive or delete stale transient files", []flagSchema{{"project", "string", true, nil}, {"older-than", "duration", false, nil}, {"sections", "string", false, nil}, {"action", "enum", false, []string{"archive", "delete"}}, {"force", "boolean", false, nil}, {"dry-run", "boolean", false, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp cleanup apply --project ./my-project --older-than 30d"}},
	"state get":            {"state get", "Read project state", []flagSchema{{"project", "string", true, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp state get --project ./my-project"}},
	"state validate":       {"state validate", "Validate project state", []flagSchema{{"project", "string", true, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp state validate --project ./my-project"}},
	"state update":         {"state update", "Update project state", []flagSchema{{"project", "string", true, nil}, {"phase", "string", false, nil}, {"health", "enum", false, []string{"unknown", "on_track", "at_risk", "blocked", "complete"}}, {"progress", "string", false, nil}, {"current-goal-id", "string", false, nil}, {"active-tasks", "string", false, nil}, {"blocked-tasks", "string", false, nil}, {"next-actions", "string", false, nil}, {"risks", "string", false, nil}, {"last-verified-at", "string", false, nil}, {"source-refs", "string", false, nil}, {"dry-run", "boolean", false, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp state update --project ./my-project --health on_track --progress '1/3 KRs achieved' --next-actions 'Verify projections' --source-refs state/project-state.json"}},
	"schema list":          {"schema list", "List CLI command schemas", []flagSchema{{"format", "enum", false, []string{"json", "text"}}}, []string{"hp schema list --json"}},
	"schema get":           {"schema get", "Inspect a CLI command schema", []flagSchema{{"command", "string", true, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp schema get --command 'project init' --json"}},
}
