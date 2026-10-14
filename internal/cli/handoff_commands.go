package cli

import (
	"fmt"

	"github.com/chenhg5/harness-projects/internal/fileops"
	"github.com/chenhg5/harness-projects/internal/handoff"
	"github.com/chenhg5/harness-projects/internal/store"
)

func (a app) handoffCreate(args []string) int {
	fs := newFlags("handoff create")
	projectPath := fs.String("project", "", "project directory")
	id := fs.String("id", "", "handoff id")
	from := fs.String("from", "", "source harness or agent")
	to := fs.String("to", "", "destination harness or agent")
	objective := fs.String("objective", "", "current objective")
	completed := fs.String("completed", "", "completed work")
	next := fs.String("next", "", "next actions")
	repositories := fs.String("repositories", "", "repository branches and commits")
	verification := fs.String("verification", "", "verification evidence")
	blockers := fs.String("blockers", "", "blockers and risks")
	format, jsonFlag := outputFlags(fs)
	dryRun := fs.Bool("dry-run", false, "preview without writing")
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
	record := handoff.Record{ID: *id, From: *from, To: *to, CreatedAt: timestamp(), Objective: *objective, Completed: *completed, Next: *next, Repositories: *repositories, Verification: *verification, Blockers: *blockers}
	if err := record.Validate(); err != nil {
		return a.parseError(err)
	}
	name, err := record.Filename()
	if err != nil {
		return a.operationError(err, *id)
	}
	content := []byte(record.Markdown())
	plan, err := fileops.PlanWrite(s.Root, "handoffs", name, content, false, false)
	if err != nil {
		return a.operationError(err, name)
	}
	result := map[string]any{"action": "handoff.create", "record": record, "path": plan.Path}
	if *dryRun {
		a.emit(result, "Would create handoff "+name, "json")
		return 10
	}
	if err := fileops.ApplyWrite(plan, content); err != nil {
		return a.operationError(err, name)
	}
	a.emit(result, "Handoff created: "+name, selectedFormat(*format, *jsonFlag, a.out))
	return 0
}

func (a app) handoffList(args []string) int {
	fs := newFlags("handoff list")
	projectPath := fs.String("project", "", "project directory")
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
	names, err := handoff.List(s.Root)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	a.emit(names, fmt.Sprintf("%d handoff(s)", len(names)), selectedFormat(*format, *jsonFlag, a.out))
	return 0
}

func (a app) handoffLatest(args []string) int {
	fs := newFlags("handoff latest")
	projectPath := fs.String("project", "", "project directory")
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
	name, content, err := handoff.Latest(s.Root)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	if selectedFormat(*format, *jsonFlag, a.out) == "text" {
		_, _ = fmt.Fprint(a.out, content)
	} else {
		a.emit(map[string]any{"name": name, "content": content}, "", "json")
	}
	return 0
}
