package model

import "testing"

func TestProjectStateRequiresNavigationAndSources(t *testing.T) {
	state := ProjectState{UpdatedAt: "2026-10-14T00:00:00Z", CurrentPhase: "build", Health: "on_track", Progress: "1/2", NextActions: []string{"verify"}, SourceRefs: []string{"goals/goal.json"}}
	if err := ValidateProjectState(state); err != nil {
		t.Fatal(err)
	}
	state.NextActions = nil
	if err := ValidateProjectState(state); err == nil {
		t.Fatal("expected next actions to be required")
	}
}
