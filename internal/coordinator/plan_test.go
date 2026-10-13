package coordinator

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/chenhg5/harness-projects/internal/model"
	"github.com/chenhg5/harness-projects/internal/project"
	"github.com/chenhg5/harness-projects/internal/store"
)

func TestBuildSeparatesReadyAndWaiting(t *testing.T) {
	root := filepath.Join(t.TempDir(), "project")
	initPlan, files, err := project.PlanInit(root, "Test", false, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := project.ApplyInit(initPlan, files); err != nil {
		t.Fatal(err)
	}
	s, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SaveTask(model.Task{SchemaVersion: model.ResourceSchemaVersion, ID: "first", Title: "First", Status: "pending", CreatedAt: "now", UpdatedAt: "now"}, false); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveTask(model.Task{SchemaVersion: model.ResourceSchemaVersion, ID: "second", Title: "Second", Status: "pending", DependsOn: []string{"first"}, CreatedAt: "now", UpdatedAt: "now"}, false); err != nil {
		t.Fatal(err)
	}
	plan, err := Build(s)
	if err != nil {
		t.Fatal(err)
	}
	if plan.ReadyCount != 1 || plan.WaitingCount != 1 || !plan.HasWork {
		t.Fatalf("unexpected plan: %#v", plan)
	}
}
