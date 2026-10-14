package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

var Version = "v0.0.1-dev"

type app struct {
	in  io.Reader
	out io.Writer
	err io.Writer
}

type failure struct {
	code       int
	kind       string
	message    string
	suggestion string
	input      string
}

func Run(args []string, in io.Reader, out, errOut io.Writer) int {
	a := app{in: in, out: out, err: errOut}
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" || args[0] == "help" && len(args) == 1 {
		fmt.Fprint(out, rootHelp)
		return 0
	}
	if args[0] == "--version" {
		fmt.Fprintln(out, Version)
		return 0
	}
	if args[0] == "help" {
		args = args[1:]
		if len(args) == 0 {
			fmt.Fprint(out, rootHelp)
			return 0
		}
		args = append(args, "--help")
	}
	if len(args) < 2 {
		if help, ok := groupHelp[args[0]]; ok {
			fmt.Fprint(out, help)
			return 0
		}
		return a.fail(failure{2, "invalid_arguments", "expected a noun and verb command", "Run hp --help", strings.Join(args, " ")})
	}
	if args[1] == "--help" || args[1] == "-h" {
		if help, ok := groupHelp[args[0]]; ok {
			fmt.Fprint(out, help)
			return 0
		}
	}
	command := args[0] + " " + args[1]
	if hasHelp(args[2:]) {
		if help, ok := commandHelp[command]; ok {
			fmt.Fprint(out, help)
			return 0
		}
	}
	switch command {
	case "project init":
		return a.projectInit(args[2:])
	case "project status":
		return a.projectStatus(args[2:], false)
	case "project validate":
		return a.projectStatus(args[2:], true)
	case "file write":
		return a.fileWrite(args[2:])
	case "file read":
		return a.fileRead(args[2:])
	case "goal create":
		return a.goalCreate(args[2:])
	case "goal list":
		return a.goalList(args[2:])
	case "goal get":
		return a.goalGet(args[2:])
	case "goal update":
		return a.goalUpdate(args[2:])
	case "task create":
		return a.taskCreate(args[2:])
	case "task list":
		return a.taskList(args[2:])
	case "task get":
		return a.taskGet(args[2:])
	case "task update":
		return a.taskUpdate(args[2:])
	case "task ready":
		return a.taskReady(args[2:])
	case "run create":
		return a.runCreate(args[2:])
	case "run list":
		return a.runList(args[2:])
	case "run get":
		return a.runGet(args[2:])
	case "run update":
		return a.runUpdate(args[2:])
	case "repository add":
		return a.repositoryAdd(args[2:])
	case "repository list":
		return a.repositoryList(args[2:])
	case "repository get":
		return a.repositoryGet(args[2:])
	case "worktree create":
		return a.worktreeCreate(args[2:])
	case "worktree list":
		return a.worktreeList(args[2:])
	case "worktree get":
		return a.worktreeGet(args[2:])
	case "worktree remove":
		return a.worktreeRemove(args[2:])
	case "context add":
		return a.contextAdd(args[2:])
	case "context list":
		return a.contextList(args[2:])
	case "context get":
		return a.contextGet(args[2:])
	case "context search":
		return a.contextSearch(args[2:])
	case "handoff create":
		return a.handoffCreate(args[2:])
	case "handoff list":
		return a.handoffList(args[2:])
	case "handoff latest":
		return a.handoffLatest(args[2:])
	case "worklog create":
		return a.worklogCreate(args[2:])
	case "worklog list":
		return a.worklogList(args[2:])
	case "worklog latest":
		return a.worklogLatest(args[2:])
	case "cleanup plan":
		return a.cleanupPlan(args[2:])
	case "cleanup apply":
		return a.cleanupApply(args[2:])
	case "coordinator plan":
		return a.coordinatorPlan(args[2:])
	case "schema list":
		return a.schemaList(args[2:])
	case "schema get":
		return a.schemaGet(args[2:])
	default:
		return a.fail(failure{2, "invalid_arguments", "unknown command: " + command, "Run hp --help", command})
	}
}

func newFlags(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

func outputFlags(fs *flag.FlagSet) (*string, *bool) {
	return fs.String("format", "", "json|text (default: json when piped)"), fs.Bool("json", false, "output JSON")
}

func parse(fs *flag.FlagSet, args []string) error {
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("unexpected positional arguments: %s", strings.Join(fs.Args(), " "))
	}
	format := fs.Lookup("format")
	if format != nil && format.Value.String() != "" && format.Value.String() != "json" && format.Value.String() != "text" {
		return fmt.Errorf("invalid --format %q; use json or text", format.Value.String())
	}
	return nil
}

func hasHelp(args []string) bool {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return true
		}
	}
	return false
}

func selectedFormat(requested string, jsonFlag bool, out io.Writer) string {
	if jsonFlag || requested == "json" {
		return "json"
	}
	if requested == "text" {
		return "text"
	}
	if f, ok := out.(*os.File); ok {
		if info, err := f.Stat(); err == nil && info.Mode()&os.ModeCharDevice != 0 && os.Getenv("TERM") != "dumb" {
			return "text"
		}
	}
	return "json"
}

func (a app) emit(data any, human, format string) {
	if format == "text" {
		fmt.Fprintln(a.out, human)
		return
	}
	encoder := json.NewEncoder(a.out)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(data)
}

func (a app) fail(err failure) int {
	_ = json.NewEncoder(a.err).Encode(map[string]any{
		"error": err.kind, "message": err.message, "suggestion": err.suggestion,
		"retryable": false, "input": err.input,
	})
	return err.code
}

func (a app) parseError(err error) int {
	return a.fail(failure{2, "invalid_arguments", err.Error(), "Run hp <noun> <verb> --help", ""})
}

func (a app) operationError(err error, input string) int {
	var pathErr *os.PathError
	if errors.As(err, &pathErr) && errors.Is(err, os.ErrPermission) {
		return a.fail(failure{4, "permission_denied", err.Error(), "Check directory permissions", input})
	}
	if errors.Is(err, os.ErrNotExist) {
		return a.fail(failure{3, "not_found", err.Error(), "Check the path and create the resource if needed", input})
	}
	message := err.Error()
	if strings.Contains(message, "already exists") || strings.Contains(message, "appeared since preview") || strings.Contains(message, "changed since preview") {
		return a.fail(failure{5, "conflict", message, "Use --if-not-exists or --replace where supported, or choose another path", input})
	}
	return a.fail(failure{1, "operation_failed", message, "Inspect the project path and retry", input})
}
