package runtime

import (
	"fmt"

	"github.com/chenhg5/harness-projects/internal/model"
)

type TaskReadiness struct {
	Task         model.Task `json:"task"`
	Ready        bool       `json:"ready"`
	Missing      []string   `json:"missing,omitempty"`
	Incomplete   []string   `json:"incomplete,omitempty"`
	BlockedByRun bool       `json:"blockedByRun,omitempty"`
}

func ReadyTasks(tasks []model.Task) ([]TaskReadiness, error) {
	byID := make(map[string]model.Task, len(tasks))
	for _, task := range tasks {
		if _, exists := byID[task.ID]; exists {
			return nil, fmt.Errorf("duplicate task id %q", task.ID)
		}
		byID[task.ID] = task
	}
	result := make([]TaskReadiness, 0, len(tasks))
	for _, task := range tasks {
		item := TaskReadiness{Task: task, Ready: task.Status == "pending" || task.Status == "ready"}
		for _, dependencyID := range task.DependsOn {
			dependency, ok := byID[dependencyID]
			if !ok {
				item.Ready = false
				item.Missing = append(item.Missing, dependencyID)
				continue
			}
			if dependency.Status != "completed" {
				item.Ready = false
				item.Incomplete = append(item.Incomplete, dependencyID)
			}
		}
		if task.Status == "blocked" || task.Status == "in_progress" || task.Status == "completed" || task.Status == "failed" || task.Status == "cancelled" {
			item.Ready = false
		}
		result = append(result, item)
	}
	return result, nil
}
