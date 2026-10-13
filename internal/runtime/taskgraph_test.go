package runtime

import (
	"testing"

	"github.com/chenhg5/harness-projects/internal/model"
)

func TestReadyTasksRespectDependencies(t *testing.T) {
	tasks := []model.Task{
		{ID: "one", Title: "One", Status: "pending"},
		{ID: "two", Title: "Two", Status: "pending", DependsOn: []string{"one"}},
		{ID: "three", Title: "Three", Status: "completed", DependsOn: []string{"one"}},
	}
	ready, err := ReadyTasks(tasks)
	if err != nil {
		t.Fatal(err)
	}
	if !ready[0].Ready || ready[1].Ready || ready[2].Ready {
		t.Fatalf("unexpected readiness: %#v", ready)
	}
}

func TestReadyTasksReportsMissingDependency(t *testing.T) {
	tasks := []model.Task{{ID: "one", Title: "One", Status: "pending", DependsOn: []string{"missing"}}}
	ready, err := ReadyTasks(tasks)
	if err != nil {
		t.Fatal(err)
	}
	if ready[0].Ready || len(ready[0].Missing) != 1 {
		t.Fatalf("expected missing dependency: %#v", ready)
	}
}
