package cli

import (
	"fmt"

	"github.com/chenhg5/harness-projects/internal/model"
	"github.com/chenhg5/harness-projects/internal/store"
)

func (a app) runCreate(args []string) int {
	fs := newFlags("run create")
	projectPath := fs.String("project", "", "project directory (required)")
	id := fs.String("id", "", "run id (required)")
	taskID := fs.String("task-id", "", "task id (required)")
	adapter := fs.String("adapter", "local", "execution adapter name")
	harness := fs.String("harness", "", "coding harness")
	agentID := fs.String("agent-id", "", "agent identity")
	objective := fs.String("objective", "", "run objective")
	worktree := fs.String("worktree", "", "worktree path")
	command := fs.String("command", "", "execution command")
	format, jsonFlag := outputFlags(fs)
	dryRun := fs.Bool("dry-run", false, "preview changes without writing")
	ifNotExists := fs.Bool("if-not-exists", false, "succeed if the run already exists")
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	if *projectPath == "" || *id == "" || *taskID == "" {
		return a.fail(failure{2, "invalid_arguments", "--project, --id, and --task-id are required", "Run hp run create --help", ""})
	}
	s, err := store.Open(*projectPath)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	if _, err := s.Task(*taskID); err != nil {
		return a.operationError(fmt.Errorf("task %q: %w", *taskID, err), *taskID)
	}
	run := model.Run{SchemaVersion: model.ResourceSchemaVersion, ID: *id, TaskID: *taskID, Adapter: *adapter, Harness: *harness, AgentID: *agentID, Objective: *objective, Status: "planned", Worktree: *worktree, Command: *command, CreatedAt: timestamp(), UpdatedAt: timestamp()}
	if err := model.ValidateRun(run); err != nil {
		return a.operationError(err, *id)
	}
	existing, existingErr := s.Run(*id)
	if existingErr == nil {
		if !*ifNotExists {
			return a.fail(failure{5, "conflict", fmt.Sprintf("run already exists: %s", *id), "Use --if-not-exists or a different id", *id})
		}
		run = existing
	} else if !store.IsNotFound(existingErr) {
		return a.operationError(existingErr, *id)
	}
	result := map[string]any{"action": "run.create", "created": existingErr != nil, "run": run}
	if *dryRun {
		a.emit(result, "Would create run "+run.ID, "json")
		return 10
	}
	if existingErr != nil {
		if err := s.SaveRun(run, false); err != nil {
			return a.operationError(err, *id)
		}
	}
	a.emit(result, "Run ready: "+run.ID, selectedFormat(*format, *jsonFlag, a.out))
	return 0
}

func (a app) runList(args []string) int {
	fs := newFlags("run list")
	projectPath := fs.String("project", "", "project directory (required)")
	taskID := fs.String("task-id", "", "filter by task id")
	status := fs.String("status", "", "filter by status")
	format, jsonFlag := outputFlags(fs)
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	if *projectPath == "" {
		return a.fail(failure{2, "invalid_arguments", "--project is required", "Run hp run list --help", "--project"})
	}
	s, err := store.Open(*projectPath)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	runs, err := s.Runs()
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	filtered := runs[:0]
	for _, run := range runs {
		if *taskID != "" && run.TaskID != *taskID {
			continue
		}
		if *status != "" && run.Status != *status {
			continue
		}
		filtered = append(filtered, run)
	}
	a.emit(filtered, fmt.Sprintf("%d run(s)", len(filtered)), selectedFormat(*format, *jsonFlag, a.out))
	return 0
}

func (a app) runGet(args []string) int {
	fs := newFlags("run get")
	projectPath := fs.String("project", "", "project directory (required)")
	id := fs.String("id", "", "run id (required)")
	format, jsonFlag := outputFlags(fs)
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	if err := requiredResourceLookup(*projectPath, *id); err != nil {
		return a.fail(failure{2, "invalid_arguments", err.Error(), "Run hp run get --help", ""})
	}
	s, err := store.Open(*projectPath)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	run, err := s.Run(*id)
	if err != nil {
		return a.operationError(err, *id)
	}
	a.emit(run, run.ID, selectedFormat(*format, *jsonFlag, a.out))
	return 0
}

func (a app) runUpdate(args []string) int {
	fs := newFlags("run update")
	projectPath := fs.String("project", "", "project directory (required)")
	id := fs.String("id", "", "run id (required)")
	status := fs.String("status", "", "planned|running|succeeded|failed|paused|cancelled")
	worktree := fs.String("worktree", "", "worktree path")
	runError := fs.String("error", "", "failure or blocker description")
	artifacts := fs.String("artifacts", "", "comma-separated artifact paths")
	harness := fs.String("harness", "", "coding harness")
	agentID := fs.String("agent-id", "", "agent identity")
	objective := fs.String("objective", "", "run objective")
	command := fs.String("command", "", "execution command")
	commit := fs.String("commit", "", "resulting commit")
	changedFiles := fs.String("changed-files", "", "comma-separated changed files")
	verification := fs.String("verification", "", "comma-separated verification evidence")
	logs := fs.String("logs", "", "comma-separated log or artifact references")
	approval := fs.String("approval", "", "approval reference")
	exitCode := fs.Int("exit-code", -1, "process exit code")
	format, jsonFlag := outputFlags(fs)
	dryRun := fs.Bool("dry-run", false, "preview changes without writing")
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	if err := requiredResourceLookup(*projectPath, *id); err != nil {
		return a.fail(failure{2, "invalid_arguments", err.Error(), "Run hp run update --help", ""})
	}
	s, err := store.Open(*projectPath)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	run, err := s.Run(*id)
	if err != nil {
		return a.operationError(err, *id)
	}
	if flagSet(fs, "status") {
		run.Status = *status
		now := timestamp()
		if *status == "running" && run.StartedAt == "" {
			run.StartedAt = now
		}
		if *status == "succeeded" || *status == "failed" || *status == "cancelled" {
			run.FinishedAt = now
		}
	}
	if flagSet(fs, "worktree") {
		run.Worktree = *worktree
	}
	if flagSet(fs, "error") {
		run.Error = *runError
	}
	if flagSet(fs, "artifacts") {
		run.Artifacts, err = commaList(*artifacts)
		if err != nil {
			return a.fail(failure{2, "invalid_arguments", err.Error(), "Use comma-separated artifact paths", "--artifacts"})
		}
	}
	if flagSet(fs, "harness") {
		run.Harness = *harness
	}
	if flagSet(fs, "agent-id") {
		run.AgentID = *agentID
	}
	if flagSet(fs, "objective") {
		run.Objective = *objective
	}
	if flagSet(fs, "command") {
		run.Command = *command
	}
	if flagSet(fs, "commit") {
		run.Commit = *commit
	}
	if flagSet(fs, "changed-files") {
		run.ChangedFiles, err = commaList(*changedFiles)
		if err != nil {
			return a.parseError(err)
		}
	}
	if flagSet(fs, "verification") {
		run.Verification, err = commaList(*verification)
		if err != nil {
			return a.parseError(err)
		}
	}
	if flagSet(fs, "logs") {
		run.Logs, err = commaList(*logs)
		if err != nil {
			return a.parseError(err)
		}
	}
	if flagSet(fs, "approval") {
		run.Approval = *approval
	}
	if flagSet(fs, "exit-code") {
		run.ExitCode = exitCode
	}
	run.UpdatedAt = timestamp()
	if err := model.ValidateRun(run); err != nil {
		return a.operationError(err, *id)
	}
	result := map[string]any{"action": "run.update", "run": run}
	if *dryRun {
		a.emit(result, "Would update run "+run.ID, "json")
		return 10
	}
	if err := s.SaveRun(run, true); err != nil {
		return a.operationError(err, *id)
	}
	a.emit(result, "Run updated: "+run.ID, selectedFormat(*format, *jsonFlag, a.out))
	return 0
}
