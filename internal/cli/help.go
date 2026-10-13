package cli

import "github.com/chenhg5/harness-projects/internal/model"

const rootHelp = `Harness Projects CLI (hp)

Examples:
  hp project init --path ./my-project --name "My Project"
  hp project validate --path ./my-project --format json
  hp file write --project ./my-project --section spec --path architecture.md --from ./draft.md --dry-run

Commands:
  project init|status|validate    Create and inspect Harness Projects
  goal create|list|get|update     Manage long-lived project goals
  task create|list|get|update|ready Manage dependent work items
  run create|list|get|update      Manage execution attempts
  repository add|list|get         Register participating code repositories
  worktree create|list|get|remove Manage isolated Git worktrees
  coordinator plan                Plan currently dispatchable work
  file write|read                 Write and read project files
  schema list|get                 Inspect the command interface

Use hp help <noun> <verb> for command details.
Exit codes: 0 success, 1 operation/validation error, 2 usage, 3 not found,
4 permission denied, 5 conflict, 10 successful dry run.
`

var groupHelp = map[string]string{
	"project":     "Project commands: init, status, validate\nExample: hp project init --path ./my-project\n",
	"goal":        "Goal commands: create, list, get, update\nExample: hp goal list --project ./my-project\n",
	"task":        "Task commands: create, list, get, update, ready\nExample: hp task ready --project ./my-project\n",
	"run":         "Run commands: create, list, get, update\nExample: hp run list --project ./my-project\n",
	"repository":  "Repository commands: add, list, get\nExample: hp repository add --project ./my-project --id api --path ../api\n",
	"worktree":    "Worktree commands: create, list, get, remove\nExample: hp worktree create --project ./my-project --id api-task --repository-id api\n",
	"coordinator": "Coordinator commands: plan\nExample: hp coordinator plan --project ./my-project\n",
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
	"goal create": `hp goal create --project <directory> --id <id> --title <title> [--status draft|active|completed|archived] [--acceptance a,b] [--dry-run]
  Create a long-lived project goal.
`,
	"goal list": `hp goal list --project <directory> [--status <status>] [--format json|text]
  List project goals.
`,
	"goal get": `hp goal get --project <directory> --id <id> [--format json|text]
  Read one goal.
`,
	"goal update": `hp goal update --project <directory> --id <id> [--title <title>] [--status <status>] [--dry-run]
  Update selected goal fields.
`,
	"task create": `hp task create --project <directory> --id <id> --title <title> [--goal-id <id>] [--depends-on a,b] [--status pending|ready|blocked] [--dry-run]
  Create a task and validate its dependency references.
`,
	"task list": `hp task list --project <directory> [--status <status>] [--goal-id <id>] [--format json|text]
  List tasks.
`,
	"task get": `hp task get --project <directory> --id <id> [--format json|text]
  Read one task.
`,
	"task update": `hp task update --project <directory> --id <id> [--status <status>] [--depends-on a,b] [--dry-run]
  Update selected task fields.
`,
	"task ready": `hp task ready --project <directory> [--format json|text]
  Compute tasks whose dependencies are complete and which can be dispatched.
`,
	"run create": `hp run create --project <directory> --id <id> --task-id <id> [--adapter local] [--worktree <path>] [--dry-run]
  Record a planned execution attempt. It does not start a provider process.
`,
	"run list": `hp run list --project <directory> [--task-id <id>] [--status <status>] [--format json|text]
  List execution attempts.
`,
	"run get": `hp run get --project <directory> --id <id> [--format json|text]
  Read one execution attempt.
`,
	"run update": `hp run update --project <directory> --id <id> [--status <status>] [--error <message>] [--dry-run]
  Update run state and record timestamps for terminal states.
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
	"project init":     {"project init", "Create a Harness Project", []flagSchema{{"path", "string", true, nil}, {"name", "string", false, nil}, {"if-not-exists", "boolean", false, nil}, {"dry-run", "boolean", false, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp project init --path ./my-project --dry-run"}},
	"project status":   {"project status", "Inspect a Harness Project", []flagSchema{{"path", "string", true, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp project status --path ./my-project --json"}},
	"project validate": {"project validate", "Validate a Harness Project", []flagSchema{{"path", "string", true, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp project validate --path ./my-project --json"}},
	"file write":       {"file write", "Write a project file", []flagSchema{{"project", "string", true, nil}, {"section", "enum", true, []string{"spec", "goals", "tasks", "runs", "agents", "context", "handoffs", "events", "verification", "skills", "scripts", "state", "artifacts", "integrations"}}, {"path", "string", true, nil}, {"from", "string", true, nil}, {"replace", "boolean", false, nil}, {"if-not-exists", "boolean", false, nil}, {"dry-run", "boolean", false, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp file write --project ./my-project --section spec --path requirements.md --from ./draft.md"}},
	"file read":        {"file read", "Read a project file", []flagSchema{{"project", "string", true, nil}, {"section", "enum", true, []string{"spec", "goals", "tasks", "runs", "agents", "context", "handoffs", "events", "verification", "skills", "scripts", "state", "artifacts", "integrations"}}, {"path", "string", true, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp file read --project ./my-project --section spec --path requirements.md --json"}},
	"goal create":      {"goal create", "Create a project goal", []flagSchema{{"project", "string", true, nil}, {"id", "string", true, nil}, {"title", "string", true, nil}, {"description", "string", false, nil}, {"status", "enum", false, model.GoalStatuses()}, {"acceptance", "string", false, nil}, {"dry-run", "boolean", false, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp goal create --project ./my-project --id modernize-api --title 'Modernize API'"}},
	"goal list":        {"goal list", "List project goals", []flagSchema{{"project", "string", true, nil}, {"status", "enum", false, model.GoalStatuses()}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp goal list --project ./my-project --json"}},
	"goal get":         {"goal get", "Read a project goal", []flagSchema{{"project", "string", true, nil}, {"id", "string", true, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp goal get --project ./my-project --id modernize-api --json"}},
	"goal update":      {"goal update", "Update a project goal", []flagSchema{{"project", "string", true, nil}, {"id", "string", true, nil}, {"title", "string", false, nil}, {"description", "string", false, nil}, {"status", "enum", false, model.GoalStatuses()}, {"acceptance", "string", false, nil}, {"dry-run", "boolean", false, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp goal update --project ./my-project --id modernize-api --status active"}},
	"task create":      {"task create", "Create a project task", []flagSchema{{"project", "string", true, nil}, {"id", "string", true, nil}, {"title", "string", true, nil}, {"goal-id", "string", false, nil}, {"depends-on", "string", false, nil}, {"agent-id", "string", false, nil}, {"status", "enum", false, model.TaskStatuses()}, {"dry-run", "boolean", false, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp task create --project ./my-project --id inventory-api --title 'Inventory API'"}},
	"task get":         {"task get", "Read a project task", []flagSchema{{"project", "string", true, nil}, {"id", "string", true, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp task get --project ./my-project --id inventory-api --json"}},
	"task update":      {"task update", "Update a project task", []flagSchema{{"project", "string", true, nil}, {"id", "string", true, nil}, {"status", "enum", false, model.TaskStatuses()}, {"depends-on", "string", false, nil}, {"dry-run", "boolean", false, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp task update --project ./my-project --id inventory-api --status completed"}},
	"task ready":       {"task ready", "List dispatchable tasks", []flagSchema{{"project", "string", true, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp task ready --project ./my-project --json"}},
	"run create":       {"run create", "Create a planned execution run", []flagSchema{{"project", "string", true, nil}, {"id", "string", true, nil}, {"task-id", "string", true, nil}, {"adapter", "string", false, nil}, {"worktree", "string", false, nil}, {"dry-run", "boolean", false, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp run create --project ./my-project --id run-001 --task-id inventory-api"}},
	"run list":         {"run list", "List execution runs", []flagSchema{{"project", "string", true, nil}, {"task-id", "string", false, nil}, {"status", "enum", false, model.RunStatuses()}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp run list --project ./my-project --json"}},
	"run get":          {"run get", "Read an execution run", []flagSchema{{"project", "string", true, nil}, {"id", "string", true, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp run get --project ./my-project --id run-001 --json"}},
	"run update":       {"run update", "Update an execution run", []flagSchema{{"project", "string", true, nil}, {"id", "string", true, nil}, {"status", "enum", false, model.RunStatuses()}, {"worktree", "string", false, nil}, {"error", "string", false, nil}, {"artifacts", "string", false, nil}, {"dry-run", "boolean", false, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp run update --project ./my-project --id run-001 --status running"}},
	"repository add":   {"repository add", "Register a project repository", []flagSchema{{"project", "string", true, nil}, {"id", "string", true, nil}, {"url", "string", false, nil}, {"path", "string", false, nil}, {"default-branch", "string", false, nil}, {"provider", "string", false, nil}, {"if-not-exists", "boolean", false, nil}, {"dry-run", "boolean", false, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp repository add --project ./my-project --id api --path ../api"}},
	"repository list":  {"repository list", "List project repositories", []flagSchema{{"project", "string", true, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp repository list --project ./my-project --json"}},
	"repository get":   {"repository get", "Read a project repository", []flagSchema{{"project", "string", true, nil}, {"id", "string", true, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp repository get --project ./my-project --id api --json"}},
	"worktree create":  {"worktree create", "Create a Git worktree", []flagSchema{{"project", "string", true, nil}, {"id", "string", true, nil}, {"repository-id", "string", true, nil}, {"path", "string", false, nil}, {"branch", "string", false, nil}, {"base", "string", false, nil}, {"dry-run", "boolean", false, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp worktree create --project ./my-project --id api-task --repository-id api"}},
	"worktree list":    {"worktree list", "List project worktrees", []flagSchema{{"project", "string", true, nil}, {"repository-id", "string", false, nil}, {"status", "enum", false, model.WorktreeStatuses()}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp worktree list --project ./my-project --json"}},
	"worktree get":     {"worktree get", "Read a project worktree", []flagSchema{{"project", "string", true, nil}, {"id", "string", true, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp worktree get --project ./my-project --id api-task --json"}},
	"worktree remove":  {"worktree remove", "Remove a Git worktree", []flagSchema{{"project", "string", true, nil}, {"id", "string", true, nil}, {"force", "boolean", false, nil}, {"dry-run", "boolean", false, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp worktree remove --project ./my-project --id api-task --force"}},
	"coordinator plan": {"coordinator plan", "Compute dispatchable tasks", []flagSchema{{"project", "string", true, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp coordinator plan --project ./my-project --json"}},
	"schema list":      {"schema list", "List CLI command schemas", []flagSchema{{"format", "enum", false, []string{"json", "text"}}}, []string{"hp schema list --json"}},
	"schema get":       {"schema get", "Inspect a CLI command schema", []flagSchema{{"command", "string", true, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp schema get --command 'project init' --json"}},
}
