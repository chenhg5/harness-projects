package cli

import (
	"fmt"
	"sort"
	"strings"
)

type flagSchemaOutput struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Required    bool     `json:"required"`
	Values      []string `json:"values,omitempty"`
	Default     any      `json:"default,omitempty"`
	Description string   `json:"description,omitempty"`
}

type commandSchemaOutput struct {
	Command     string              `json:"command"`
	Description string              `json:"description"`
	Flags       []flagSchemaOutput  `json:"flags"`
	Examples    []string            `json:"examples"`
	Output      outputSchemaOutput  `json:"output"`
	Errors      []errorSchemaOutput `json:"errors"`
}

type outputSchemaOutput struct {
	Formats   []string `json:"formats"`
	Default   string   `json:"default"`
	JSONShape string   `json:"jsonShape,omitempty"`
	DryRun    bool     `json:"dryRun"`
}

type errorSchemaOutput struct {
	ExitCode  int    `json:"exitCode"`
	Error     string `json:"error"`
	Retryable bool   `json:"retryable"`
	Meaning   string `json:"meaning"`
}

type flagDetail struct {
	Default     any
	Description string
}

var commonFlagDetails = map[string]flagDetail{
	"acceptance":        {Description: "Comma-separated acceptance criteria."},
	"action":            {Description: "Cleanup action to perform."},
	"active-tasks":      {Description: "Comma-separated active task IDs."},
	"adapter":           {Description: "Execution adapter name."},
	"agent":             {Description: "Agent identity recorded in the work log."},
	"agent-id":          {Description: "Agent identity associated with the resource."},
	"all":               {Description: "Include terminal resources normally hidden from results."},
	"approval":          {Description: "Approval record or reference."},
	"artifacts":         {Description: "Comma-separated artifact paths or references."},
	"availability":      {Description: "Local delivery availability state."},
	"base":              {Description: "Base branch, tag, or revision."},
	"blocked-reason":    {Description: "Reason work is blocked."},
	"blocked-tasks":     {Description: "Comma-separated blocked task IDs."},
	"blockers":          {Description: "Blockers and risks to record."},
	"branch":            {Description: "Git branch name."},
	"capabilities":      {Description: "Comma-separated agent skills or capabilities."},
	"capability":        {Description: "Filter participants by capability."},
	"changed-files":     {Description: "Comma-separated changed file paths."},
	"changes":           {Description: "Summary of files or decisions changed."},
	"command":           {Description: "Command name or execution command, as documented by this operation."},
	"commit":            {Description: "Resulting Git commit SHA or reference."},
	"completed":         {Description: "Summary of completed work."},
	"cron":              {Description: "Five-field cron schedule."},
	"current-goal-id":   {Description: "Current project goal ID."},
	"default-branch":    {Description: "Repository default branch or revision."},
	"deliverables":      {Description: "Comma-separated task deliverables."},
	"delivery":          {Description: "Task delivery mode."},
	"depends-on":        {Description: "Comma-separated prerequisite task IDs."},
	"description":       {Description: "Human-readable description."},
	"disable":           {Description: "Disable the configured schedule."},
	"done-when":         {Description: "Comma-separated completion conditions."},
	"dry-run":           {Description: "Preview the operation without writing; successful previews exit 10."},
	"due-date":          {Description: "Task due date."},
	"enable":            {Description: "Enable the configured schedule."},
	"error":             {Description: "Failure or blocker description."},
	"every":             {Description: "Recurring interval such as 30m or 24h."},
	"evidence":          {Description: "Comma-separated evidence references."},
	"evidence-required": {Description: "Comma-separated evidence required before completion."},
	"exit-code":         {Description: "Process exit code recorded for the run."},
	"focus":             {Description: "Current responsibility or work focus."},
	"force":             {Description: "Allow the documented destructive or exceptional action."},
	"format":            {Default: "auto", Description: "Output format; auto-selects text on a TTY and JSON when piped."},
	"from":              {Description: "Source file, source identity, or '-' for stdin as documented."},
	"full":              {Description: "Include complete text sections instead of summaries."},
	"goal-id":           {Description: "Goal ID associated with the task."},
	"harness":           {Description: "Harness name such as codex, claude-code, cursor, or other."},
	"health":            {Description: "Project health state."},
	"id":                {Description: "Stable resource identifier."},
	"if-not-exists":     {Description: "Return success without replacing an existing resource."},
	"include-stale":     {Description: "Deprecated compatibility flag; durable discovery already includes participants."},
	"json":              {Description: "Force JSON output."},
	"key-results":       {Description: "JSON array of measurable key results."},
	"kind":              {Description: "Resource kind."},
	"last-verified-at":  {Description: "Last verification timestamp."},
	"lines":             {Description: "Maximum lines shown for each text section."},
	"logs":              {Description: "Comma-separated log or artifact references."},
	"message":           {Description: "Additional instructions delivered with the task."},
	"method":            {Description: "Goal management method."},
	"module":            {Description: "Filter participants by module or repository area."},
	"modules":           {Description: "Comma-separated modules or repository areas."},
	"name":              {Description: "Human-readable name or directory name, as documented."},
	"next":              {Description: "Next action for the receiving agent."},
	"next-actions":      {Description: "Comma-separated project next actions."},
	"no-git":            {Description: "Skip Git repository creation and the initial commit."},
	"objective":         {Description: "Goal, run, or session objective."},
	"older-than":        {Description: "Age threshold such as 30d or 720h."},
	"open":              {Description: "Open the dashboard in the default browser."},
	"outcome":           {Description: "Observable outcome the goal must produce."},
	"owner":             {Description: "Owner identity."},
	"path":              {Description: "Filesystem path, interpreted as documented by the command."},
	"phase":             {Description: "Current project phase."},
	"port":              {Description: "Local TCP port; 0 selects a free port."},
	"priority":          {Description: "Task priority."},
	"progress":          {Description: "Project progress summary."},
	"project":           {Description: "Project root or a directory from which the project can be resolved."},
	"prompt-file":       {Description: "Project-relative Markdown wakeup prompt."},
	"provider":          {Description: "Repository provider name."},
	"prune":             {Description: "Remove stale managed symlinks only."},
	"query":             {Description: "Whitespace-separated search terms; every term must match."},
	"ready-only":        {Description: "Show only tasks ready to start."},
	"registration-id":   {Description: "Durable participant registration ID."},
	"remaining":         {Description: "Summary of work not completed."},
	"replace":           {Description: "Explicitly replace an existing value or ownership record."},
	"repositories":      {Description: "Repository branches, commits, and worktree state."},
	"repository-id":     {Description: "Registered repository ID."},
	"result":            {Description: "Task result summary."},
	"retry":             {Description: "Retry after inspecting the previous uncertain attempt."},
	"review-cadence":    {Description: "Goal review cadence."},
	"risks":             {Description: "Comma-separated project risks."},
	"role":              {Description: "Current project role filter."},
	"run-id":            {Description: "Execution run ID."},
	"section":           {Description: "Canonical project section."},
	"sections":          {Description: "Comma-separated transient sections to inspect."},
	"session-id":        {Description: "Actual provider session ID; never invent this value."},
	"source":            {Description: "Source reference for the memory."},
	"source-refs":       {Description: "Comma-separated source references supporting project state."},
	"status":            {Description: "Resource status or status filter."},
	"tags":              {Description: "Comma-separated memory tags."},
	"target-date":       {Description: "Goal target date."},
	"targets":           {Description: "Comma-separated skill mount targets or 'all'."},
	"task-id":           {Description: "Task ID associated with the operation."},
	"timeout":           {Description: "Provider command timeout."},
	"timezone":          {Description: "IANA time zone used by the cron schedule."},
	"title":             {Description: "Human-readable resource title."},
	"to":                {Description: "Destination harness or agent."},
	"ttl":               {Description: "Local delivery hint lease duration."},
	"url":               {Description: "Repository URL."},
	"verification":      {Description: "Verification commands, checks, or evidence."},
	"worktree":          {Description: "Execution worktree path."},
}

var commandFlagDetails = map[string]map[string]flagDetail{
	"layout list": {
		"project": {Default: ".", Description: "Project root or any directory inside it."},
		"format":  {Default: "auto", Description: "Output format; auto means text on a TTY and JSON when piped."},
		"json":    {Default: false, Description: "Force JSON output."},
	},
	"layout add": {
		"project":       {Default: ".", Description: "Project root or any directory inside it."},
		"name":          {Description: "One top-level directory name without path separators."},
		"description":   {Description: "A concise statement of what belongs in the directory."},
		"if-not-exists": {Default: false, Description: "Return success without changing an existing entry."},
		"replace":       {Default: false, Description: "Replace an existing project-specific description."},
		"dry-run":       {Default: false, Description: "Emit the planned change as JSON without writing; exits 10."},
		"format":        {Default: "auto", Description: "Output format; auto means text on a TTY and JSON when piped."},
		"json":          {Default: false, Description: "Force JSON output."},
	},
	"brief": {
		"project": {Default: "."},
		"lines":   {Default: 30},
	},
	"finish": {
		"project": {Default: "."},
		"status":  {Default: "completed"},
	},
	"project doctor":     {"path": {Default: "."}},
	"project upgrade":    {"path": {Default: "."}},
	"project init":       {"name": {Description: "Project name; defaults to the target directory name."}},
	"state reconcile":    {"project": {Default: "."}},
	"hook install":       {"project": {Default: "."}},
	"hook session-start": {"project": {Description: "Project directory; defaults to the hook working directory."}},
	"hook stop":          {"project": {Description: "Project directory; defaults to the hook working directory."}},
	"skill sync": {
		"project": {Default: "."},
		"targets": {Default: "all"},
	},
	"goal create":          {"status": {Default: "draft"}},
	"task create":          {"status": {Default: "pending"}},
	"run create":           {"adapter": {Default: "local"}},
	"run update":           {"exit-code": {Default: -1}},
	"agent register":       {"availability": {Default: "busy"}, "ttl": {Default: "2h"}},
	"agent heartbeat":      {"ttl": {Default: "2h"}},
	"agent prune":          {"older-than": {Default: "7d"}},
	"cleanup plan":         {"older-than": {Default: "30d"}},
	"cleanup apply":        {"older-than": {Default: "30d"}, "action": {Default: "archive"}},
	"coordinator dispatch": {"delivery": {Default: "auto"}, "timeout": {Default: "30m"}},
	"wakeup once":          {"timeout": {Default: "30m"}},
	"wakeup start":         {"timeout": {Default: "30m"}},
}

var commandOutputDetails = map[string]string{
	"project status":   "A project status object with path, identity, validity, and structural problems.",
	"project validate": "A project status object with path, identity, validity, and structural problems.",
	"project doctor":   "A doctor report object with project, healthy, and findings.",
	"project serve":    "A server result object containing the local dashboard address.",
	"layout list":      "An object with action, project, contractPath, schemaVersion, contractExists, and directories.",
	"layout add":       "An object describing the affected directory and whether the contract changed.",
	"brief":            "A bounded project brief object with goal, state, tasks, plan, latest work log, and handoff.",
	"schema list":      "An object whose commands field is the sorted list of introspectable command names.",
	"schema get":       "A command contract object with description, flags, examples, output, and errors.",
	"cleanup plan":     "An object containing action and the cleanup plan.",
	"cleanup apply":    "An object containing action, cleanup plan, and changed count after apply.",
	"coordinator plan": "A dispatch plan object describing ready and blocked work.",
	"task inbox":       "An array of assigned tasks with readiness and dependency details.",
	"context search":   "An array of ranked structured-memory and text-file matches.",
}

var standardErrorSchemas = []errorSchemaOutput{
	{ExitCode: exitOperationFailed, Error: "operation_failed", Meaning: "The command was valid but the operation or resource validation failed."},
	{ExitCode: exitOperationFailed, Error: "transient_failure", Retryable: true, Meaning: "A time-bounded external operation failed and may be retried after inspection."},
	{ExitCode: exitInvalidArguments, Error: "invalid_arguments", Meaning: "Flags, values, or positional arguments are invalid."},
	{ExitCode: exitNotFound, Error: "not_found", Meaning: "The requested path or resource does not exist."},
	{ExitCode: exitPermissionDenied, Error: "permission_denied", Meaning: "The operating system denied access."},
	{ExitCode: exitConflict, Error: "conflict", Meaning: "Current state conflicts with the requested mutation."},
}

func (a app) schemaList(args []string) int {
	fs := newFlags("schema list")
	format, jsonFlag := outputFlags(fs)
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	commands := make([]string, 0, len(commandSchemas))
	for command := range commandSchemas {
		commands = append(commands, command)
	}
	sort.Strings(commands)
	a.emit(map[string]any{"commands": commands}, strings.Join(commands, "\n"), selectedFormat(*format, *jsonFlag, a.out))
	return 0
}

func (a app) schemaGet(args []string) int {
	fs := newFlags("schema get")
	command := fs.String("command", "", "command name, e.g. project init (required)")
	format, jsonFlag := outputFlags(fs)
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	schema, ok := commandSchemas[*command]
	if !ok {
		return a.fail(failure{3, "not_found", "unknown command schema: " + *command, "Run hp schema list", *command})
	}
	a.emit(enrichCommandSchema(schema), renderCommandHelp(schema), selectedFormat(*format, *jsonFlag, a.out))
	return 0
}

func enrichCommandSchema(schema commandSchema) commandSchemaOutput {
	result := commandSchemaOutput{
		Command: schema.Command, Description: schema.Description, Examples: schema.Examples,
		Errors: append([]errorSchemaOutput(nil), standardErrorSchemas...),
	}
	flags := append([]flagSchema(nil), schema.Flags...)
	if hasSchemaFlag(flags, "format") && !hasSchemaFlag(flags, "json") {
		flags = append(flags, flagSchema{Name: "json", Type: "boolean"})
	}
	for _, flag := range flags {
		detail := commonFlagDetails[flag.Name]
		if override, ok := commandFlagDetails[schema.Command][flag.Name]; ok {
			if override.Description != "" {
				detail.Description = override.Description
			}
			if override.Default != nil {
				detail.Default = override.Default
			}
		}
		if detail.Description == "" {
			detail.Description = fmt.Sprintf("Value for --%s.", flag.Name)
		}
		if !flag.Required && detail.Default == nil {
			detail.Default = zeroDefault(flag.Type)
		}
		result.Flags = append(result.Flags, flagSchemaOutput{
			Name: flag.Name, Type: flag.Type, Required: flag.Required, Values: flag.Values,
			Default: detail.Default, Description: detail.Description,
		})
	}
	result.Output = outputContract(schema, result.Flags)
	return result
}

func outputContract(schema commandSchema, flags []flagSchemaOutput) outputSchemaOutput {
	output := outputSchemaOutput{Formats: []string{"text"}, Default: "text"}
	if hasOutputFlag(flags, "format") {
		output.Formats = []string{"json", "text"}
		output.Default = "auto"
		output.JSONShape = inferredJSONShape(schema)
	}
	output.DryRun = hasOutputFlag(flags, "dry-run")
	return output
}

func inferredJSONShape(schema commandSchema) string {
	if description := commandOutputDetails[schema.Command]; description != "" {
		return description
	}
	parts := strings.Fields(schema.Command)
	verb := parts[len(parts)-1]
	switch verb {
	case "list", "discover", "ready":
		return "A JSON array of resources described by this command."
	case "get", "latest", "status", "validate":
		return "A JSON object representing the requested resource or status."
	default:
		return "A JSON object describing the operation and its resulting or planned resource."
	}
}

func hasOutputFlag(flags []flagSchemaOutput, name string) bool {
	for _, flag := range flags {
		if flag.Name == name {
			return true
		}
	}
	return false
}

func hasSchemaFlag(flags []flagSchema, name string) bool {
	for _, flag := range flags {
		if flag.Name == name {
			return true
		}
	}
	return false
}

func zeroDefault(flagType string) any {
	switch flagType {
	case "boolean":
		return false
	case "integer":
		return 0
	default:
		return ""
	}
}
