package model

import "testing"

func TestCompletedGoalRequiresAchievedKeyResultsAndEvidence(t *testing.T) {
	target, current := 1.0, 0.5
	goal := Goal{ID: "goal", Title: "Goal", Description: "Description", Outcome: "Outcome", Status: "completed", KeyResults: []KeyResult{{ID: "result", Metric: "metric", Direction: "increase", Target: &target, Current: &current, Status: "in_progress"}}}
	if err := ValidateGoal(goal); err == nil {
		t.Fatal("expected incomplete key result to block goal completion")
	}
	goal.KeyResults[0].Status = "achieved"
	if err := ValidateGoal(goal); err == nil {
		t.Fatal("expected missing evidence to block goal completion")
	}
	goal.Evidence = []string{"verification/result.md"}
	goal.KeyResults[0].Evidence = []string{"verification/result.md"}
	if err := ValidateGoal(goal); err != nil {
		t.Fatalf("expected complete goal to validate: %v", err)
	}
}

func TestBlockedAndCompletedTasksNeedEvidence(t *testing.T) {
	task := Task{ID: "task", Title: "Task", Status: "blocked"}
	if err := ValidateTask(task); err == nil {
		t.Fatal("expected blocked task reason")
	}
	task.BlockedReason = "waiting for API"
	if err := ValidateTask(task); err != nil {
		t.Fatal(err)
	}
	task.Status = "completed"
	if err := ValidateTask(task); err == nil {
		t.Fatal("expected completed task evidence")
	}
	task.Result = "verified"
	if err := ValidateTask(task); err != nil {
		t.Fatal(err)
	}
}
