package cli

import (
	"fmt"

	"github.com/chenhg5/harness-projects/internal/model"
	"github.com/chenhg5/harness-projects/internal/runtime"
	"github.com/chenhg5/harness-projects/internal/store"
)

func (a app) taskCreate(args []string) int {
	fs := newFlags("task create")
	projectPath := fs.String("project", "", "project directory (required)")
	id := fs.String("id", "", "task id (required)")
	title := fs.String("title", "", "task title (required)")
	description := fs.String("description", "", "task description")
	deliverables := fs.String("deliverables", "", "comma-separated deliverables")
	acceptance := fs.String("acceptance", "", "comma-separated acceptance criteria")
	priority := fs.String("priority", "", "critical|high|medium|low")
	owner := fs.String("owner", "", "task owner")
	dueDate := fs.String("due-date", "", "due date")
	blockedReason := fs.String("blocked-reason", "", "reason the task is blocked")
	goalID := fs.String("goal-id", "", "parent goal id")
	dependsOn := fs.String("depends-on", "", "comma-separated task ids")
	agentID := fs.String("agent-id", "", "assigned Agent id")
	status := fs.String("status", "pending", "pending|ready|blocked")
	evidence := fs.String("evidence", "", "comma-separated verification evidence")
	resultText := fs.String("result", "", "result summary")
	verification := fs.String("verification", "", "comma-separated verification commands or checks")
	format, jsonFlag := outputFlags(fs)
	dryRun := fs.Bool("dry-run", false, "preview changes without writing")
	ifNotExists := fs.Bool("if-not-exists", false, "succeed if the task already exists")
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	if err := requiredResourceFlags(*projectPath, *id, *title); err != nil {
		return a.fail(failure{2, "invalid_arguments", err.Error(), "Run hp task create --help", ""})
	}
	dependencies, err := commaList(*dependsOn)
	if err != nil {
		return a.fail(failure{2, "invalid_arguments", err.Error(), "Use comma-separated task ids", "--depends-on"})
	}
	deliverableList, err := commaList(*deliverables)
	if err != nil {
		return a.parseError(err)
	}
	acceptanceList, err := commaList(*acceptance)
	if err != nil {
		return a.parseError(err)
	}
	evidenceList, err := commaList(*evidence)
	if err != nil {
		return a.parseError(err)
	}
	verificationList, err := commaList(*verification)
	if err != nil {
		return a.parseError(err)
	}
	task := model.Task{SchemaVersion: model.ResourceSchemaVersion, ID: *id, GoalID: *goalID, Title: *title, Description: *description, Deliverables: deliverableList, Acceptance: acceptanceList, Priority: *priority, Owner: *owner, DueDate: *dueDate, Status: *status, DependsOn: dependencies, AgentID: *agentID, BlockedReason: *blockedReason, Result: *resultText, Evidence: evidenceList, Verification: verificationList, CreatedAt: timestamp(), UpdatedAt: timestamp()}
	if err := model.ValidateTask(task); err != nil {
		return a.operationError(err, *id)
	}
	s, err := store.Open(*projectPath)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	if task.GoalID != "" {
		if _, err := s.Goal(task.GoalID); err != nil {
			return a.operationError(fmt.Errorf("goal %q: %w", task.GoalID, err), task.GoalID)
		}
	}
	for _, dependencyID := range task.DependsOn {
		if _, err := s.Task(dependencyID); err != nil {
			return a.operationError(fmt.Errorf("dependency task %q: %w", dependencyID, err), dependencyID)
		}
	}
	existing, existingErr := s.Task(*id)
	if existingErr == nil {
		if !*ifNotExists {
			return a.fail(failure{5, "conflict", fmt.Sprintf("task already exists: %s", *id), "Use --if-not-exists or hp task update", *id})
		}
		task = existing
	} else if !store.IsNotFound(existingErr) {
		return a.operationError(existingErr, *id)
	}
	result := map[string]any{"action": "task.create", "created": existingErr != nil, "task": task}
	if *dryRun {
		a.emit(result, "Would create task "+task.ID, "json")
		return 10
	}
	if existingErr != nil {
		if err := s.SaveTask(task, false); err != nil {
			return a.operationError(err, *id)
		}
	}
	a.emit(result, "Task ready: "+task.ID, selectedFormat(*format, *jsonFlag, a.out))
	return 0
}

func (a app) taskList(args []string) int {
	fs := newFlags("task list")
	projectPath := fs.String("project", "", "project directory (required)")
	status := fs.String("status", "", "filter by status")
	goalID := fs.String("goal-id", "", "filter by goal id")
	format, jsonFlag := outputFlags(fs)
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	if *projectPath == "" {
		return a.fail(failure{2, "invalid_arguments", "--project is required", "Run hp task list --help", "--project"})
	}
	s, err := store.Open(*projectPath)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	tasks, err := s.Tasks()
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	filtered := tasks[:0]
	for _, task := range tasks {
		if *status != "" && task.Status != *status {
			continue
		}
		if *goalID != "" && task.GoalID != *goalID {
			continue
		}
		filtered = append(filtered, task)
	}
	a.emit(filtered, fmt.Sprintf("%d task(s)", len(filtered)), selectedFormat(*format, *jsonFlag, a.out))
	return 0
}

func (a app) taskGet(args []string) int {
	fs := newFlags("task get")
	projectPath := fs.String("project", "", "project directory (required)")
	id := fs.String("id", "", "task id (required)")
	format, jsonFlag := outputFlags(fs)
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	if err := requiredResourceLookup(*projectPath, *id); err != nil {
		return a.fail(failure{2, "invalid_arguments", err.Error(), "Run hp task get --help", ""})
	}
	s, err := store.Open(*projectPath)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	task, err := s.Task(*id)
	if err != nil {
		return a.operationError(err, *id)
	}
	a.emit(task, task.ID, selectedFormat(*format, *jsonFlag, a.out))
	return 0
}

func (a app) taskUpdate(args []string) int {
	fs := newFlags("task update")
	projectPath := fs.String("project", "", "project directory (required)")
	id := fs.String("id", "", "task id (required)")
	title := fs.String("title", "", "new title")
	description := fs.String("description", "", "new description")
	deliverables := fs.String("deliverables", "", "comma-separated deliverables")
	acceptance := fs.String("acceptance", "", "comma-separated acceptance criteria")
	priority := fs.String("priority", "", "critical|high|medium|low")
	owner := fs.String("owner", "", "task owner")
	dueDate := fs.String("due-date", "", "due date")
	blockedReason := fs.String("blocked-reason", "", "reason the task is blocked")
	status := fs.String("status", "", "pending|ready|in_progress|blocked|completed|failed|cancelled")
	dependsOn := fs.String("depends-on", "", "comma-separated task ids")
	agentID := fs.String("agent-id", "", "assigned Agent id")
	evidence := fs.String("evidence", "", "comma-separated verification evidence")
	resultText := fs.String("result", "", "result summary")
	verification := fs.String("verification", "", "comma-separated verification commands or checks")
	format, jsonFlag := outputFlags(fs)
	dryRun := fs.Bool("dry-run", false, "preview changes without writing")
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	if err := requiredResourceLookup(*projectPath, *id); err != nil {
		return a.fail(failure{2, "invalid_arguments", err.Error(), "Run hp task update --help", ""})
	}
	s, err := store.Open(*projectPath)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	task, err := s.Task(*id)
	if err != nil {
		return a.operationError(err, *id)
	}
	if flagSet(fs, "title") {
		task.Title = *title
	}
	if flagSet(fs, "description") {
		task.Description = *description
	}
	if flagSet(fs, "deliverables") {
		task.Deliverables, err = commaList(*deliverables)
		if err != nil {
			return a.parseError(err)
		}
	}
	if flagSet(fs, "acceptance") {
		task.Acceptance, err = commaList(*acceptance)
		if err != nil {
			return a.parseError(err)
		}
	}
	if flagSet(fs, "priority") {
		task.Priority = *priority
	}
	if flagSet(fs, "owner") {
		task.Owner = *owner
	}
	if flagSet(fs, "due-date") {
		task.DueDate = *dueDate
	}
	if flagSet(fs, "blocked-reason") {
		task.BlockedReason = *blockedReason
	}
	if flagSet(fs, "status") {
		task.Status = *status
	}
	if flagSet(fs, "depends-on") {
		task.DependsOn, err = commaList(*dependsOn)
		if err != nil {
			return a.fail(failure{2, "invalid_arguments", err.Error(), "Use comma-separated task ids", "--depends-on"})
		}
	}
	if flagSet(fs, "agent-id") {
		task.AgentID = *agentID
	}
	if flagSet(fs, "evidence") {
		task.Evidence, err = commaList(*evidence)
		if err != nil {
			return a.parseError(err)
		}
	}
	if flagSet(fs, "result") {
		task.Result = *resultText
	}
	if flagSet(fs, "verification") {
		task.Verification, err = commaList(*verification)
		if err != nil {
			return a.parseError(err)
		}
	}
	for _, dependencyID := range task.DependsOn {
		if dependencyID == task.ID {
			return a.fail(failure{2, "invalid_arguments", "task cannot depend on itself", "Remove the task id from --depends-on", dependencyID})
		}
		if _, err := s.Task(dependencyID); err != nil {
			return a.operationError(fmt.Errorf("dependency task %q: %w", dependencyID, err), dependencyID)
		}
	}
	task.UpdatedAt = timestamp()
	if err := model.ValidateTask(task); err != nil {
		return a.operationError(err, *id)
	}
	result := map[string]any{"action": "task.update", "task": task}
	if *dryRun {
		a.emit(result, "Would update task "+task.ID, "json")
		return 10
	}
	if err := s.SaveTask(task, true); err != nil {
		return a.operationError(err, *id)
	}
	a.emit(result, "Task updated: "+task.ID, selectedFormat(*format, *jsonFlag, a.out))
	return 0
}

func (a app) taskReady(args []string) int {
	fs := newFlags("task ready")
	projectPath := fs.String("project", "", "project directory (required)")
	format, jsonFlag := outputFlags(fs)
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	if *projectPath == "" {
		return a.fail(failure{2, "invalid_arguments", "--project is required", "Run hp task ready --help", "--project"})
	}
	s, err := store.Open(*projectPath)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	tasks, err := s.Tasks()
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	readiness, err := runtime.ReadyTasks(tasks)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	ready := make([]runtime.TaskReadiness, 0, len(readiness))
	for _, item := range readiness {
		if item.Ready {
			ready = append(ready, item)
		}
	}
	a.emit(ready, fmt.Sprintf("%d ready task(s)", len(ready)), selectedFormat(*format, *jsonFlag, a.out))
	return 0
}
