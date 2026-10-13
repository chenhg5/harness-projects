package model

import (
	"fmt"
	"regexp"
	"strings"
)

const ResourceSchemaVersion = "1"

var idPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)

type Goal struct {
	SchemaVersion string   `json:"schemaVersion"`
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	Description   string   `json:"description,omitempty"`
	Status        string   `json:"status"`
	Acceptance    []string `json:"acceptance,omitempty"`
	CreatedAt     string   `json:"createdAt"`
	UpdatedAt     string   `json:"updatedAt"`
}

type Task struct {
	SchemaVersion string   `json:"schemaVersion"`
	ID            string   `json:"id"`
	GoalID        string   `json:"goalId,omitempty"`
	Title         string   `json:"title"`
	Description   string   `json:"description,omitempty"`
	Status        string   `json:"status"`
	DependsOn     []string `json:"dependsOn,omitempty"`
	AgentID       string   `json:"agentId,omitempty"`
	CreatedAt     string   `json:"createdAt"`
	UpdatedAt     string   `json:"updatedAt"`
}

type Run struct {
	SchemaVersion string   `json:"schemaVersion"`
	ID            string   `json:"id"`
	TaskID        string   `json:"taskId"`
	Adapter       string   `json:"adapter"`
	Status        string   `json:"status"`
	Worktree      string   `json:"worktree,omitempty"`
	Error         string   `json:"error,omitempty"`
	Artifacts     []string `json:"artifacts,omitempty"`
	CreatedAt     string   `json:"createdAt"`
	UpdatedAt     string   `json:"updatedAt"`
	StartedAt     string   `json:"startedAt,omitempty"`
	FinishedAt    string   `json:"finishedAt,omitempty"`
}

var goalStatuses = map[string]bool{"draft": true, "active": true, "completed": true, "archived": true}
var taskStatuses = map[string]bool{"pending": true, "ready": true, "in_progress": true, "blocked": true, "completed": true, "failed": true, "cancelled": true}
var runStatuses = map[string]bool{"planned": true, "running": true, "succeeded": true, "failed": true, "paused": true, "cancelled": true}

func ValidateID(id string) error {
	if !idPattern.MatchString(id) {
		return fmt.Errorf("invalid id %q; use lowercase letters, digits, dot, underscore, or hyphen", id)
	}
	return nil
}

func ValidateGoal(goal Goal) error {
	if err := ValidateID(goal.ID); err != nil {
		return err
	}
	if strings.TrimSpace(goal.Title) == "" {
		return fmt.Errorf("goal title is required")
	}
	if !goalStatuses[goal.Status] {
		return fmt.Errorf("invalid goal status %q", goal.Status)
	}
	return nil
}

func ValidateTask(task Task) error {
	if err := ValidateID(task.ID); err != nil {
		return err
	}
	if strings.TrimSpace(task.Title) == "" {
		return fmt.Errorf("task title is required")
	}
	if !taskStatuses[task.Status] {
		return fmt.Errorf("invalid task status %q", task.Status)
	}
	seen := map[string]bool{}
	for _, dependency := range task.DependsOn {
		if err := ValidateID(dependency); err != nil {
			return fmt.Errorf("invalid dependency: %w", err)
		}
		if dependency == task.ID {
			return fmt.Errorf("task cannot depend on itself")
		}
		if seen[dependency] {
			return fmt.Errorf("duplicate dependency %q", dependency)
		}
		seen[dependency] = true
	}
	return nil
}

func ValidateRun(run Run) error {
	if err := ValidateID(run.ID); err != nil {
		return err
	}
	if err := ValidateID(run.TaskID); err != nil {
		return fmt.Errorf("invalid task id: %w", err)
	}
	if strings.TrimSpace(run.Adapter) == "" {
		return fmt.Errorf("run adapter is required")
	}
	if !runStatuses[run.Status] {
		return fmt.Errorf("invalid run status %q", run.Status)
	}
	return nil
}

func GoalStatuses() []string { return []string{"draft", "active", "completed", "archived"} }
func TaskStatuses() []string {
	return []string{"pending", "ready", "in_progress", "blocked", "completed", "failed", "cancelled"}
}
func RunStatuses() []string {
	return []string{"planned", "running", "succeeded", "failed", "paused", "cancelled"}
}
