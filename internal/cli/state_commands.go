package cli

import (
	"fmt"

	"github.com/chenhg5/harness-projects/internal/model"
	"github.com/chenhg5/harness-projects/internal/store"
)

func (a app) stateGet(args []string, validate bool) int {
	command := "state get"
	if validate {
		command = "state validate"
	}
	fs := newFlags(command)
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
	state, err := s.State()
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	if err := model.ValidateProjectState(state); err != nil {
		a.emit(map[string]any{"valid": false, "error": err.Error(), "state": state}, "Project state is invalid: "+err.Error(), selectedFormat(*format, *jsonFlag, a.out))
		if validate {
			return 1
		}
		return 0
	}
	a.emit(state, "Project state is valid", selectedFormat(*format, *jsonFlag, a.out))
	return 0
}

func (a app) stateUpdate(args []string) int {
	fs := newFlags("state update")
	projectPath := fs.String("project", "", "project directory")
	phase := fs.String("phase", "", "current phase")
	health := fs.String("health", "", "unknown|on_track|at_risk|blocked|complete")
	progress := fs.String("progress", "", "progress summary")
	goalID := fs.String("current-goal-id", "", "current goal id")
	active := fs.String("active-tasks", "", "comma-separated active task ids")
	blocked := fs.String("blocked-tasks", "", "comma-separated blocked task ids")
	next := fs.String("next-actions", "", "comma-separated next actions")
	risks := fs.String("risks", "", "comma-separated risks")
	verified := fs.String("last-verified-at", "", "last verification timestamp")
	refs := fs.String("source-refs", "", "comma-separated source references")
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
	state, err := s.State()
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	if flagSet(fs, "phase") {
		state.CurrentPhase = *phase
	}
	if flagSet(fs, "health") {
		state.Health = *health
	}
	if flagSet(fs, "progress") {
		state.Progress = *progress
	}
	if flagSet(fs, "current-goal-id") {
		state.CurrentGoalID = *goalID
	}
	if flagSet(fs, "active-tasks") {
		state.ActiveTaskIDs, err = commaList(*active)
		if err != nil {
			return a.parseError(err)
		}
	}
	if flagSet(fs, "blocked-tasks") {
		state.BlockedTaskIDs, err = commaList(*blocked)
		if err != nil {
			return a.parseError(err)
		}
	}
	if flagSet(fs, "next-actions") {
		state.NextActions, err = commaList(*next)
		if err != nil {
			return a.parseError(err)
		}
	}
	if flagSet(fs, "risks") {
		state.Risks, err = commaList(*risks)
		if err != nil {
			return a.parseError(err)
		}
	}
	if flagSet(fs, "last-verified-at") {
		state.LastVerifiedAt = *verified
	}
	if flagSet(fs, "source-refs") {
		state.SourceRefs, err = commaList(*refs)
		if err != nil {
			return a.parseError(err)
		}
	}
	state.UpdatedAt = timestamp()
	if err := model.ValidateProjectState(state); err != nil {
		return a.operationError(err, *projectPath)
	}
	result := map[string]any{"action": "state.update", "state": state}
	if *dryRun {
		a.emit(result, "Would update project state", "json")
		return 10
	}
	if err := s.SaveState(state); err != nil {
		return a.operationError(err, *projectPath)
	}
	a.emit(result, "Project state updated", selectedFormat(*format, *jsonFlag, a.out))
	return 0
}
