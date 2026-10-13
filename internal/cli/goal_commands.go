package cli

import (
	"fmt"

	"github.com/chenhg5/harness-projects/internal/model"
	"github.com/chenhg5/harness-projects/internal/store"
)

func (a app) goalCreate(args []string) int {
	fs := newFlags("goal create")
	projectPath := fs.String("project", "", "project directory (required)")
	id := fs.String("id", "", "goal id (required)")
	title := fs.String("title", "", "goal title (required)")
	description := fs.String("description", "", "goal description")
	status := fs.String("status", "draft", "draft|active|completed|archived")
	acceptance := fs.String("acceptance", "", "comma-separated acceptance criteria")
	format, jsonFlag := outputFlags(fs)
	dryRun := fs.Bool("dry-run", false, "preview changes without writing")
	ifNotExists := fs.Bool("if-not-exists", false, "succeed if the goal already exists")
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	if err := requiredResourceFlags(*projectPath, *id, *title); err != nil {
		return a.fail(failure{2, "invalid_arguments", err.Error(), "Run hp goal create --help", ""})
	}
	criteria, err := commaList(*acceptance)
	if err != nil {
		return a.fail(failure{2, "invalid_arguments", err.Error(), "Use comma-separated non-empty values", "--acceptance"})
	}
	goal := model.Goal{SchemaVersion: model.ResourceSchemaVersion, ID: *id, Title: *title, Description: *description, Status: *status, Acceptance: criteria, CreatedAt: timestamp(), UpdatedAt: timestamp()}
	if err := model.ValidateGoal(goal); err != nil {
		return a.operationError(err, *id)
	}
	s, err := store.Open(*projectPath)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	existing, existingErr := s.Goal(*id)
	if existingErr == nil {
		if !*ifNotExists {
			return a.fail(failure{5, "conflict", fmt.Sprintf("goal already exists: %s", *id), "Use --if-not-exists or hp goal update", *id})
		}
		goal = existing
	} else if !store.IsNotFound(existingErr) {
		return a.operationError(existingErr, *id)
	}
	result := map[string]any{"action": "goal.create", "created": existingErr != nil, "goal": goal}
	if *dryRun {
		a.emit(result, "Would create goal "+goal.ID, "json")
		return 10
	}
	if existingErr != nil {
		if err := s.SaveGoal(goal, false); err != nil {
			return a.operationError(err, *id)
		}
	}
	a.emit(result, "Goal ready: "+goal.ID, selectedFormat(*format, *jsonFlag, a.out))
	return 0
}

func (a app) goalList(args []string) int {
	fs := newFlags("goal list")
	projectPath := fs.String("project", "", "project directory (required)")
	status := fs.String("status", "", "filter by status")
	format, jsonFlag := outputFlags(fs)
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	if *projectPath == "" {
		return a.fail(failure{2, "invalid_arguments", "--project is required", "Run hp goal list --help", "--project"})
	}
	s, err := store.Open(*projectPath)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	goals, err := s.Goals()
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	if *status != "" {
		filtered := goals[:0]
		for _, goal := range goals {
			if goal.Status == *status {
				filtered = append(filtered, goal)
			}
		}
		goals = filtered
	}
	a.emit(goals, fmt.Sprintf("%d goal(s)", len(goals)), selectedFormat(*format, *jsonFlag, a.out))
	return 0
}

func (a app) goalGet(args []string) int {
	fs := newFlags("goal get")
	projectPath := fs.String("project", "", "project directory (required)")
	id := fs.String("id", "", "goal id (required)")
	format, jsonFlag := outputFlags(fs)
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	if err := requiredResourceLookup(*projectPath, *id); err != nil {
		return a.fail(failure{2, "invalid_arguments", err.Error(), "Run hp goal get --help", ""})
	}
	s, err := store.Open(*projectPath)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	goal, err := s.Goal(*id)
	if err != nil {
		return a.operationError(err, *id)
	}
	a.emit(goal, goal.ID, selectedFormat(*format, *jsonFlag, a.out))
	return 0
}

func (a app) goalUpdate(args []string) int {
	fs := newFlags("goal update")
	projectPath := fs.String("project", "", "project directory (required)")
	id := fs.String("id", "", "goal id (required)")
	title := fs.String("title", "", "new title")
	description := fs.String("description", "", "new description")
	status := fs.String("status", "", "draft|active|completed|archived")
	acceptance := fs.String("acceptance", "", "comma-separated acceptance criteria")
	format, jsonFlag := outputFlags(fs)
	dryRun := fs.Bool("dry-run", false, "preview changes without writing")
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	if err := requiredResourceLookup(*projectPath, *id); err != nil {
		return a.fail(failure{2, "invalid_arguments", err.Error(), "Run hp goal update --help", ""})
	}
	s, err := store.Open(*projectPath)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	goal, err := s.Goal(*id)
	if err != nil {
		return a.operationError(err, *id)
	}
	if flagSet(fs, "title") {
		goal.Title = *title
	}
	if flagSet(fs, "description") {
		goal.Description = *description
	}
	if flagSet(fs, "status") {
		goal.Status = *status
	}
	if flagSet(fs, "acceptance") {
		goal.Acceptance, err = commaList(*acceptance)
		if err != nil {
			return a.fail(failure{2, "invalid_arguments", err.Error(), "Use comma-separated non-empty values", "--acceptance"})
		}
	}
	goal.UpdatedAt = timestamp()
	if err := model.ValidateGoal(goal); err != nil {
		return a.operationError(err, *id)
	}
	result := map[string]any{"action": "goal.update", "goal": goal}
	if *dryRun {
		a.emit(result, "Would update goal "+goal.ID, "json")
		return 10
	}
	if err := s.SaveGoal(goal, true); err != nil {
		return a.operationError(err, *id)
	}
	a.emit(result, "Goal updated: "+goal.ID, selectedFormat(*format, *jsonFlag, a.out))
	return 0
}
