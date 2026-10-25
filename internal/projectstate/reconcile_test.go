package projectstate

import (
	"reflect"
	"testing"

	"github.com/chenhg5/harness-projects/internal/model"
)

func TestReconcileDerivesTaskBackedState(t *testing.T) {
	state := model.ProjectState{
		CurrentGoalID: "goal", CurrentPhase: "build", Health: "on_track", Progress: "stale",
		NextActions: []string{"stale"}, SourceRefs: []string{"PROJECT.md"},
	}
	tasks := []model.Task{
		{ID: "done", GoalID: "goal", Status: "completed"},
		{ID: "active", GoalID: "goal", Status: "in_progress"},
		{ID: "blocked", GoalID: "other", Status: "blocked"},
	}
	got := Reconcile(state, tasks, "2026-10-25T00:00:00Z", "Review output", "worklogs/latest.md", "tasks/active.json", true)
	if got.Progress != "1 of 2 tasks completed" {
		t.Fatalf("progress=%q", got.Progress)
	}
	if !reflect.DeepEqual(got.ActiveTaskIDs, []string{"active"}) || !reflect.DeepEqual(got.BlockedTaskIDs, []string{"blocked"}) {
		t.Fatalf("unexpected task sets: active=%v blocked=%v", got.ActiveTaskIDs, got.BlockedTaskIDs)
	}
	if !reflect.DeepEqual(got.NextActions, []string{"Review output", "stale"}) || got.LastVerifiedAt != got.UpdatedAt {
		t.Fatalf("unexpected derived state: %#v", got)
	}
}
