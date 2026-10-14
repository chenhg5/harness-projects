package store

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/chenhg5/harness-projects/internal/model"
	"github.com/chenhg5/harness-projects/internal/project"
)

func testStore(t *testing.T) Store {
	t.Helper()
	root := filepath.Join(t.TempDir(), "project")
	plan, files, err := project.PlanInit(root, "Test", false, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := project.ApplyInit(plan, files); err != nil {
		t.Fatal(err)
	}
	s, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestResourcesRoundTrip(t *testing.T) {
	s := testStore(t)
	goal := model.Goal{SchemaVersion: model.ResourceSchemaVersion, ID: "goal", Title: "Goal", Status: "draft", CreatedAt: "now", UpdatedAt: "now"}
	if err := s.SaveGoal(goal, false); err != nil {
		t.Fatal(err)
	}
	task := model.Task{SchemaVersion: model.ResourceSchemaVersion, ID: "task", GoalID: "goal", Title: "Task", Status: "pending", CreatedAt: "now", UpdatedAt: "now"}
	if err := s.SaveTask(task, false); err != nil {
		t.Fatal(err)
	}
	run := model.Run{SchemaVersion: model.ResourceSchemaVersion, ID: "run", TaskID: "task", Adapter: "local", Status: "planned", CreatedAt: "now", UpdatedAt: "now"}
	if err := s.SaveRun(run, false); err != nil {
		t.Fatal(err)
	}
	gotGoal, err := s.Goal("goal")
	if err != nil || gotGoal.Title != goal.Title {
		t.Fatalf("goal round trip: %#v %v", gotGoal, err)
	}
	gotTasks, err := s.Tasks()
	if err != nil || len(gotTasks) != 1 {
		t.Fatalf("task list: %#v %v", gotTasks, err)
	}
	gotRuns, err := s.Runs()
	if err != nil || len(gotRuns) != 1 {
		t.Fatalf("run list: %#v %v", gotRuns, err)
	}
}
