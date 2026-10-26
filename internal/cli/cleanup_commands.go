package cli

import (
	"fmt"
	"time"

	"github.com/chenhg5/harness-projects/internal/cleanup"
	"github.com/chenhg5/harness-projects/internal/store"
)

func (a app) cleanupPlan(args []string) int {
	fs := newFlags("cleanup plan")
	projectPath := fs.String("project", "", "project directory")
	olderThan := fs.String("older-than", "30d", "age threshold, for example 30d or 720h")
	sections := fs.String("sections", "", "comma-separated transient sections")
	format, jsonFlag := outputFlags(fs)
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
	plan, err := cleanup.Build(s.Root, *olderThan, *sections, "archive", time.Now())
	if err != nil {
		return a.parseError(err)
	}
	result := map[string]any{"action": "cleanup.plan", "plan": plan}
	a.emit(result, fmt.Sprintf("%d stale file(s) eligible for %s", len(plan.Candidates), plan.Action), selectedFormat(*format, *jsonFlag, a.out))
	return 0
}

func (a app) cleanupApply(args []string) int {
	fs := newFlags("cleanup apply")
	projectPath := fs.String("project", "", "project directory")
	olderThan := fs.String("older-than", "30d", "age threshold, for example 30d or 720h")
	sections := fs.String("sections", "", "comma-separated transient sections")
	action := fs.String("action", "archive", "archive or delete")
	force := fs.Bool("force", false, "confirm destructive deletion")
	dryRun := fs.Bool("dry-run", false, "preview without changing files")
	format, jsonFlag := outputFlags(fs)
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
	plan, err := cleanup.Build(s.Root, *olderThan, *sections, *action, time.Now())
	if err != nil {
		return a.parseError(err)
	}
	result := map[string]any{"action": "cleanup.apply", "plan": plan}
	if *dryRun {
		a.emit(result, fmt.Sprintf("Would %s %d stale file(s)", plan.Action, len(plan.Candidates)), "json")
		return exitDryRun
	}
	changed, err := cleanup.Apply(plan, *force)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	result["changed"] = changed
	a.emit(result, fmt.Sprintf("%s %d stale file(s)", plan.Action, changed), selectedFormat(*format, *jsonFlag, a.out))
	return 0
}
