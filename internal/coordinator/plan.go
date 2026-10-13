package coordinator

import (
	"sort"

	"github.com/chenhg5/harness-projects/internal/runtime"
	"github.com/chenhg5/harness-projects/internal/store"
)

type Plan struct {
	Project      string                  `json:"project"`
	Ready        []runtime.TaskReadiness `json:"ready"`
	Waiting      []runtime.TaskReadiness `json:"waiting"`
	TotalTasks   int                     `json:"totalTasks"`
	ReadyCount   int                     `json:"readyCount"`
	WaitingCount int                     `json:"waitingCount"`
	HasWork      bool                    `json:"hasWork"`
}

func Build(s store.Store) (Plan, error) {
	tasks, err := s.Tasks()
	if err != nil {
		return Plan{}, err
	}
	readiness, err := runtime.ReadyTasks(tasks)
	if err != nil {
		return Plan{}, err
	}
	plan := Plan{Project: s.Root, TotalTasks: len(readiness), Ready: make([]runtime.TaskReadiness, 0), Waiting: make([]runtime.TaskReadiness, 0)}
	for _, item := range readiness {
		if item.Ready {
			plan.Ready = append(plan.Ready, item)
			continue
		}
		if item.Task.Status == "pending" || item.Task.Status == "ready" || item.Task.Status == "blocked" {
			plan.Waiting = append(plan.Waiting, item)
		}
	}
	sort.Slice(plan.Ready, func(i, j int) bool { return plan.Ready[i].Task.ID < plan.Ready[j].Task.ID })
	sort.Slice(plan.Waiting, func(i, j int) bool { return plan.Waiting[i].Task.ID < plan.Waiting[j].Task.ID })
	plan.ReadyCount = len(plan.Ready)
	plan.WaitingCount = len(plan.Waiting)
	plan.HasWork = plan.ReadyCount > 0
	return plan, nil
}
