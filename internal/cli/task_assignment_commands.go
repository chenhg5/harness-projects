package cli

import (
	"fmt"
	"github.com/chenhg5/harness-projects/internal/session"
	"strings"

	"github.com/chenhg5/harness-projects/internal/model"
	"github.com/chenhg5/harness-projects/internal/participants"
	"github.com/chenhg5/harness-projects/internal/presence"
	"github.com/chenhg5/harness-projects/internal/runtime"
	"github.com/chenhg5/harness-projects/internal/store"
)

// Assignment is a durable task address. It does not start or resume a provider.
func (a app) taskAssign(args []string) int {
	fs := newFlags("task assign")
	projectPath := fs.String("project", "", "project directory (required)")
	id := fs.String("id", "", "task id (required)")
	registrationID := fs.String("registration-id", "", "registered agent session ID")
	harness := fs.String("harness", "", "target harness")
	sessionID := fs.String("session-id", "", "target provider session ID")
	replace := fs.Bool("replace", false, "replace a different assignment")
	dryRun := fs.Bool("dry-run", false, "preview changes without writing")
	format, jsonFlag := outputFlags(fs)
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	if err := requiredResourceLookup(*projectPath, *id); err != nil {
		return a.parseError(err)
	}
	if (*registrationID == "") == (*harness == "" || *sessionID == "") || (*registrationID != "" && (*harness != "" || *sessionID != "")) {
		return a.parseError(fmt.Errorf("provide either --registration-id or both --harness and --session-id"))
	}
	s, err := store.Open(*projectPath)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	task, err := s.Task(*id)
	if err != nil {
		return a.operationError(err, *id)
	}
	if task.Status == "completed" || task.Status == "cancelled" {
		return a.fail(failure{5, "conflict", "cannot assign a completed or cancelled task", "Choose an active task", *id})
	}
	assignment := model.TaskAssignment{AssignedAt: timestamp()}
	if *registrationID != "" {
		view, err := (participants.Registry{Root: s.Root}).Get(*registrationID)
		if err != nil {
			return a.operationError(err, *registrationID)
		}
		assignment.Harness, assignment.SessionID, assignment.RegistrationID = view.Harness, view.SessionID, view.ID
	} else {
		assignment.Harness, err = presence.CanonicalHarness(*harness)
		if err != nil {
			return a.parseError(err)
		}
		assignment.SessionID = *sessionID
	}
	if err := model.ValidateProviderSessionID(assignment.SessionID); err != nil {
		return a.parseError(err)
	}
	if task.Assignment != nil {
		if task.Assignment.Harness == assignment.Harness && task.Assignment.SessionID == assignment.SessionID {
			a.emit(map[string]any{"action": "task.assign", "changed": false, "task": task}, "Task already assigned: "+task.ID, selectedFormat(*format, *jsonFlag, a.out))
			return 0
		}
		if !*replace {
			return a.fail(failure{5, "conflict", "task is already assigned to another session", "Use --replace to reassign", *id})
		}
		previous := *task.Assignment
		previous.EndedAt = timestamp()
		task.AssignmentHistory = append(task.AssignmentHistory, previous)
	}
	task.Assignment = &assignment
	task.UpdatedAt = timestamp()
	if err := model.ValidateTask(task); err != nil {
		return a.operationError(err, *id)
	}
	result := map[string]any{"action": "task.assign", "changed": true, "task": task}
	if *dryRun {
		a.emit(result, "Would assign task "+task.ID, "json")
		return 10
	}
	if err := s.SaveTask(task, true); err != nil {
		return a.operationError(err, *id)
	}
	a.emit(result, "Assigned task "+task.ID+" to "+assignment.Harness+" / "+assignment.SessionID, selectedFormat(*format, *jsonFlag, a.out))
	return 0
}

func (a app) taskUnassign(args []string) int {
	fs := newFlags("task unassign")
	projectPath := fs.String("project", "", "project directory (required)")
	id := fs.String("id", "", "task id (required)")
	dryRun := fs.Bool("dry-run", false, "preview changes without writing")
	format, jsonFlag := outputFlags(fs)
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	if err := requiredResourceLookup(*projectPath, *id); err != nil {
		return a.parseError(err)
	}
	s, err := store.Open(*projectPath)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	task, err := s.Task(*id)
	if err != nil {
		return a.operationError(err, *id)
	}
	if task.Assignment == nil {
		return a.fail(failure{5, "conflict", "task is not assigned", "Choose an assigned task", *id})
	}
	previous := *task.Assignment
	previous.EndedAt = timestamp()
	task.AssignmentHistory = append(task.AssignmentHistory, previous)
	task.Assignment = nil
	task.UpdatedAt = timestamp()
	result := map[string]any{"action": "task.unassign", "task": task}
	if *dryRun {
		a.emit(result, "Would unassign task "+task.ID, "json")
		return 10
	}
	if err := s.SaveTask(task, true); err != nil {
		return a.operationError(err, *id)
	}
	a.emit(result, "Unassigned task "+task.ID, selectedFormat(*format, *jsonFlag, a.out))
	return 0
}

func (a app) taskInbox(args []string) int {
	fs := newFlags("task inbox")
	projectPath := fs.String("project", "", "project directory (required)")
	harness := fs.String("harness", "", "current harness (detected when omitted)")
	sessionID := fs.String("session-id", "", "current provider session ID (detected when omitted)")
	readyOnly := fs.Bool("ready-only", false, "show only tasks ready to start")
	all := fs.Bool("all", false, "include completed and cancelled tasks")
	format, jsonFlag := outputFlags(fs)
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	session.Fill(harness, sessionID)
	if *projectPath == "" {
		return a.parseError(fmt.Errorf("--project is required"))
	}
	if *harness == "" || *sessionID == "" {
		return a.fail(failure{2, "invalid_arguments", "could not detect the provider session", sessionHint, "--session-id"})
	}
	canonical, err := presence.CanonicalHarness(*harness)
	if err != nil {
		return a.parseError(err)
	}
	if err := model.ValidateProviderSessionID(*sessionID); err != nil {
		return a.parseError(err)
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
	inbox := make([]runtime.TaskReadiness, 0)
	for _, item := range readiness {
		assignment := item.Task.Assignment
		if assignment == nil || assignment.Harness != canonical || assignment.SessionID != *sessionID {
			continue
		}
		if !*all && (item.Task.Status == "completed" || item.Task.Status == "cancelled") {
			continue
		}
		if *readyOnly && !item.Ready {
			continue
		}
		inbox = append(inbox, item)
	}
	human := fmt.Sprintf("%d assigned task(s)", len(inbox))
	if len(inbox) > 0 {
		lines := make([]string, 0, len(inbox))
		for _, item := range inbox {
			lines = append(lines, fmt.Sprintf("%s %s ready=%t — %s", item.Task.ID, item.Task.Status, item.Ready, item.Task.Title))
		}
		human += "\n" + strings.Join(lines, "\n")
	}
	a.emit(inbox, human, selectedFormat(*format, *jsonFlag, a.out))
	return 0
}
