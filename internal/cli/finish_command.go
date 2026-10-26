package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chenhg5/harness-projects/internal/fileops"
	"github.com/chenhg5/harness-projects/internal/model"
	"github.com/chenhg5/harness-projects/internal/project"
	"github.com/chenhg5/harness-projects/internal/projectstate"
	"github.com/chenhg5/harness-projects/internal/session"
	"github.com/chenhg5/harness-projects/internal/store"
	"github.com/chenhg5/harness-projects/internal/worklog"
)

func (a app) finish(args []string) int {
	fs := newFlags("finish")
	projectPath := fs.String("project", ".", "project directory")
	id := fs.String("id", "", "work log id (not the provider session ID)")
	taskID := fs.String("task-id", "", "task to update")
	status := fs.String("status", "completed", "in_progress|completed|blocked|failed|cancelled")
	agent := fs.String("agent", "", "agent identity (defaults to detected harness)")
	harness := fs.String("harness", "", "coding harness (detected when omitted)")
	sessionID := fs.String("session-id", "", "actual provider session ID (detected when available)")
	objective := fs.String("objective", "", "session objective")
	completed := fs.String("completed", "", "completed work")
	remaining := fs.String("remaining", "", "work not completed")
	changes := fs.String("changes", "", "files or decisions changed")
	verification := fs.String("verification", "", "verification evidence")
	next := fs.String("next", "", "next action")
	blockers := fs.String("blockers", "", "blockers and risks")
	resultText := fs.String("result", "", "task result summary")
	evidence := fs.String("evidence", "", "comma-separated task evidence")
	format, jsonFlag := outputFlags(fs)
	dryRun := fs.Bool("dry-run", false, "preview without writing")
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	if *id == "" || *taskID == "" {
		return a.parseError(fmt.Errorf("--id and --task-id are required"))
	}
	session.Fill(harness, sessionID)
	if *agent == "" {
		*agent = *harness
	}
	now := timestamp()
	record := worklog.Record{ID: *id, Agent: *agent, Harness: *harness, SessionID: *sessionID, TaskID: *taskID, CreatedAt: now, Objective: *objective, Completed: *completed, Remaining: *remaining, Changes: *changes, Verification: *verification, Next: *next, Blockers: *blockers}
	if err := record.Validate(); err != nil {
		return a.parseError(err)
	}
	root, err := project.Find(*projectPath)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	s, err := store.Open(root)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	task, err := s.Task(*taskID)
	if err != nil {
		return a.operationError(err, *taskID)
	}
	task.Status = *status
	task.UpdatedAt = now
	if task.Status == "blocked" {
		task.BlockedReason = strings.TrimSpace(*blockers)
	} else {
		task.BlockedReason = ""
	}
	if task.Assignment != nil && (task.Status == "completed" || task.Status == "failed" || task.Status == "cancelled") {
		previous := *task.Assignment
		previous.EndedAt = now
		task.AssignmentHistory = append(task.AssignmentHistory, previous)
		task.Assignment = nil
	}
	evidenceList, err := commaList(*evidence)
	if err != nil {
		return a.parseError(err)
	}
	task.Evidence = appendUniqueStrings(task.Evidence, evidenceList...)
	if strings.TrimSpace(*verification) != "" {
		task.Verification = appendUniqueStrings(task.Verification, strings.TrimSpace(*verification))
	}
	if strings.TrimSpace(*resultText) != "" {
		task.Result = strings.TrimSpace(*resultText)
	} else if task.Status == "completed" && len(task.Evidence) == 0 && strings.TrimSpace(task.Result) == "" {
		task.Result = strings.TrimSpace(*completed)
	}
	if err := model.ValidateTask(task); err != nil {
		return a.parseError(err)
	}
	tasks, err := s.Tasks()
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	for index := range tasks {
		if tasks[index].ID == task.ID {
			tasks[index] = task
		}
	}
	state, err := s.State()
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	if state.CurrentGoalID == "" && task.GoalID != "" {
		state.CurrentGoalID = task.GoalID
	}
	name, err := record.Filename()
	if err != nil {
		return a.operationError(err, *id)
	}
	content := []byte(record.Markdown())
	writePlan, err := fileops.PlanWrite(s.Root, "worklogs", name, content, false, false)
	if err != nil {
		return a.operationError(err, name)
	}
	state = projectstate.Reconcile(state, tasks, now, record.Next, filepath.ToSlash(filepath.Join("worklogs", name)), filepath.ToSlash(filepath.Join("tasks", task.ID+".json")), strings.TrimSpace(record.Verification) != "")
	if err := model.ValidateProjectState(state); err != nil {
		return a.operationError(err, *projectPath)
	}
	result := map[string]any{"action": "finish", "task": task, "worklog": record, "worklogPath": writePlan.Path, "state": state}
	if *dryRun {
		result["dryRun"] = true
		a.emit(result, "Would finish task "+task.ID+", write "+name+", and reconcile project state", "json")
		return 10
	}
	taskPath := filepath.Join(s.Root, "tasks", task.ID+".json")
	statePath := filepath.Join(s.Root, "state", "project-state.json")
	originalTask, taskReadErr := os.ReadFile(taskPath)
	originalState, stateReadErr := os.ReadFile(statePath)
	if taskReadErr != nil || stateReadErr != nil {
		return a.operationError(fmt.Errorf("read transaction baseline: task=%v state=%v", taskReadErr, stateReadErr), *projectPath)
	}
	if err := s.SaveTask(task, true); err != nil {
		return a.operationError(err, task.ID)
	}
	if err := fileops.ApplyWrite(writePlan, content); err != nil {
		_ = os.WriteFile(taskPath, originalTask, 0644)
		return a.operationError(err, name)
	}
	if err := s.SaveState(state); err != nil {
		_ = os.Remove(writePlan.Path)
		_ = os.WriteFile(taskPath, originalTask, 0644)
		_ = os.WriteFile(statePath, originalState, 0644)
		return a.operationError(err, *projectPath)
	}
	a.emit(result, "Finished task "+task.ID+"; wrote "+name+" and reconciled project state", selectedFormat(*format, *jsonFlag, a.out))
	return 0
}

func appendUniqueStrings(values []string, additions ...string) []string {
	seen := map[string]bool{}
	result := []string{}
	for _, value := range append(append([]string(nil), values...), additions...) {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}
