package cli

const rootHelp = `Harness Projects CLI (hp)

Examples:
  hp project init --path ./my-project --name "My Project"
  hp project validate --path ./my-project --format json
  hp file write --project ./my-project --section spec --path architecture.md --from ./draft.md --dry-run

Commands:
  project init|status|validate    Create and inspect Harness Projects
  file write|read                 Write and read project files
  schema list|get                 Inspect the command interface

Use hp help <noun> <verb> for command details.
Exit codes: 0 success, 1 operation/validation error, 2 usage, 3 not found,
4 permission denied, 5 conflict, 10 successful dry run.
`

var groupHelp = map[string]string{
	"project": "Project commands: init, status, validate\nExample: hp project init --path ./my-project\n",
	"file":    "File commands: write, read\nExample: hp file read --project ./my-project --section spec --path architecture.md\n",
	"schema":  "Schema commands: list, get\nExample: hp schema get --command 'project init'\n",
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
	"schema list":      {"schema list", "List CLI command schemas", []flagSchema{{"format", "enum", false, []string{"json", "text"}}}, []string{"hp schema list --json"}},
	"schema get":       {"schema get", "Inspect a CLI command schema", []flagSchema{{"command", "string", true, nil}, {"format", "enum", false, []string{"json", "text"}}}, []string{"hp schema get --command 'project init' --json"}},
}
