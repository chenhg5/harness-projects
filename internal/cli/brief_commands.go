package cli

import (
	"fmt"

	"github.com/chenhg5/harness-projects/internal/brief"
	"github.com/chenhg5/harness-projects/internal/project"
)

func (a app) brief(args []string) int {
	fs := newFlags("brief")
	projectPath := fs.String("project", ".", "project directory or any directory inside it")
	full := fs.Bool("full", false, "show complete plan, work log, and handoff text")
	lines := fs.Int("lines", brief.DefaultLines, "maximum lines per text section")
	format, jsonFlag := outputFlags(fs)
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	if *lines < 1 {
		return a.parseError(fmt.Errorf("--lines must be at least 1"))
	}
	root, err := project.Find(*projectPath)
	if err != nil {
		return a.fail(failure{3, "not_found", err.Error(), "Run inside a Harness Project, pass --project, or create one with hp project init", *projectPath})
	}
	maxLines := *lines
	if *full {
		maxLines = 0
	}
	result, err := brief.Load(root, maxLines)
	if err != nil {
		return a.operationError(err, root)
	}
	if selectedFormat(*format, *jsonFlag, a.out) == "text" {
		fmt.Fprint(a.out, result.Markdown())
		return 0
	}
	a.emit(result, "", "json")
	return 0
}
