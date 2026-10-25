package projectstate

import (
	"fmt"
	"sort"
	"strings"

	"github.com/chenhg5/harness-projects/internal/model"
)

// Reconcile derives the task-backed fields of a project state snapshot while
// preserving human judgments such as phase, health, and risks.
func Reconcile(state model.ProjectState, tasks []model.Task, now, nextAction, worklogRef, taskRef string, verified bool) model.ProjectState {
	active := []string{}
	blocked := []string{}
	completed := 0
	total := 0
	for _, task := range tasks {
		if task.Status == "in_progress" {
			active = append(active, task.ID)
		}
		if task.Status == "blocked" {
			blocked = append(blocked, task.ID)
		}
		if state.CurrentGoalID == "" || task.GoalID == state.CurrentGoalID {
			total++
			if task.Status == "completed" {
				completed++
			}
		}
	}
	sort.Strings(active)
	sort.Strings(blocked)
	state.ActiveTaskIDs = active
	state.BlockedTaskIDs = blocked
	if total > 0 {
		state.Progress = fmt.Sprintf("%d of %d tasks completed", completed, total)
	}
	if strings.TrimSpace(nextAction) != "" {
		state.NextActions = appendUnique([]string{strings.TrimSpace(nextAction)}, state.NextActions...)
		if len(state.NextActions) > 5 {
			state.NextActions = state.NextActions[:5]
		}
	}
	state.SourceRefs = appendUnique(state.SourceRefs, taskRef, worklogRef)
	state.UpdatedAt = now
	if verified {
		state.LastVerifiedAt = now
	}
	return state
}

func appendUnique(values []string, additions ...string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values)+len(additions))
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
