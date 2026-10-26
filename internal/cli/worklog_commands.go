package cli

import (
	"fmt"

	"github.com/chenhg5/harness-projects/internal/fileops"
	"github.com/chenhg5/harness-projects/internal/session"
	"github.com/chenhg5/harness-projects/internal/store"
	"github.com/chenhg5/harness-projects/internal/worklog"
)

func (a app) worklogCreate(args []string) int {
	fs := newFlags("worklog create")
	projectPath := fs.String("project", "", "project directory")
	id := fs.String("id", "", "work log id (not the provider session ID)")
	agent := fs.String("agent", "", "agent identity")
	harness := fs.String("harness", "", "coding harness")
	sessionID := fs.String("session-id", "", "actual provider session ID (detected when available)")
	taskID := fs.String("task-id", "", "related task id")
	runID := fs.String("run-id", "", "related run id")
	objective := fs.String("objective", "", "session objective")
	completed := fs.String("completed", "", "completed work")
	remaining := fs.String("remaining", "", "work not completed")
	changes := fs.String("changes", "", "files or decisions changed")
	verification := fs.String("verification", "", "verification evidence")
	next := fs.String("next", "", "next action")
	blockers := fs.String("blockers", "", "blockers and risks")
	format, jsonFlag := outputFlags(fs)
	dryRun := fs.Bool("dry-run", false, "preview without writing")
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	if *projectPath == "" {
		return a.parseError(fmt.Errorf("--project is required"))
	}
	session.Fill(harness, sessionID)
	if *agent == "" {
		*agent = *harness
	}
	s, err := store.Open(*projectPath)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	record := worklog.Record{ID: *id, Agent: *agent, Harness: *harness, SessionID: *sessionID, TaskID: *taskID, RunID: *runID, CreatedAt: timestamp(), Objective: *objective, Completed: *completed, Remaining: *remaining, Changes: *changes, Verification: *verification, Next: *next, Blockers: *blockers}
	if err := record.Validate(); err != nil {
		return a.parseError(err)
	}
	name, err := record.Filename()
	if err != nil {
		return a.operationError(err, *id)
	}
	content := []byte(record.Markdown())
	plan, err := fileops.PlanWrite(s.Root, "worklogs", name, content, false, false)
	if err != nil {
		return a.operationError(err, name)
	}
	result := map[string]any{"action": "worklog.create", "record": record, "path": plan.Path}
	if *dryRun {
		a.emit(result, "Would create work log "+name, "json")
		return 10
	}
	if err := fileops.ApplyWrite(plan, content); err != nil {
		return a.operationError(err, name)
	}
	a.emit(result, "Work log created: "+name, selectedFormat(*format, *jsonFlag, a.out))
	return 0
}

func (a app) worklogList(args []string) int {
	fs := newFlags("worklog list")
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
	names, err := worklog.List(s.Root)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	a.emit(names, fmt.Sprintf("%d work log(s)", len(names)), selectedFormat(*format, *jsonFlag, a.out))
	return 0
}

func (a app) worklogLatest(args []string) int {
	fs := newFlags("worklog latest")
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
	name, content, err := worklog.Latest(s.Root)
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
