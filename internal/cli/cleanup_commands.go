package cli

import (
	"fmt"
	"time"

	"github.com/chenhg5/harness-projects/internal/cleanup"
	"github.com/chenhg5/harness-projects/internal/store"
)

func (a app) cleanup(args []string, apply bool) int {
	name := "cleanup plan"
	if apply {
		name = "cleanup apply"
	}
	fs := newFlags(name)
	projectPath := fs.String("project", "", "project directory")
	olderThan := fs.String("older-than", "30d", "age threshold, for example 30d or 720h")
	sections := fs.String("sections", "", "comma-separated transient sections")
	action := fs.String("action", "archive", "archive or delete")
	force := fs.Bool("force", false, "confirm destructive deletion")
	format, jsonFlag := outputFlags(fs)
	dryRun := fs.Bool("dry-run", false, "preview without changing files")
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	if *projectPath == "" {
		return a.parseError(fmt.Errorf("--project is required"))
	}
	s, err := store.Open(*projectPath)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	selectedAction := "archive"
	if apply {
		selectedAction = *action
	}
	plan, err := cleanup.Build(s.Root, *olderThan, *sections, selectedAction, time.Now())
	if err != nil {
		return a.parseError(err)
	}
	result := map[string]any{"action": "cleanup.plan", "plan": plan}
	if !apply {
		a.emit(result, fmt.Sprintf("%d stale file(s) eligible for %s", len(plan.Candidates), plan.Action), selectedFormat(*format, *jsonFlag, a.out))
		return 0
	}
	result["action"] = "cleanup.apply"
	if *dryRun {
		a.emit(result, fmt.Sprintf("Would %s %d stale file(s)", plan.Action, len(plan.Candidates)), "json")
		return 10
	}
	changed, err := cleanup.Apply(plan, *force)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	result["changed"] = changed
	a.emit(result, fmt.Sprintf("%s %d stale file(s)", plan.Action, changed), selectedFormat(*format, *jsonFlag, a.out))
	return 0
}

func (a app) cleanupPlan(args []string) int  { return a.cleanup(args, false) }
func (a app) cleanupApply(args []string) int { return a.cleanup(args, true) }
