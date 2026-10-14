package model

import (
	"fmt"
	"regexp"
	"strings"
)

const ResourceSchemaVersion = "1"

var idPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)

type Goal struct {
	SchemaVersion    string      `json:"schemaVersion"`
	ID               string      `json:"id"`
	Title            string      `json:"title"`
	Description      string      `json:"description,omitempty"`
	Outcome          string      `json:"outcome,omitempty"`
	Method           string      `json:"method,omitempty"`
	Owner            string      `json:"owner,omitempty"`
	TargetDate       string      `json:"targetDate,omitempty"`
	ReviewCadence    string      `json:"reviewCadence,omitempty"`
	Status           string      `json:"status"`
	Acceptance       []string    `json:"acceptance,omitempty"`
	DoneWhen         []string    `json:"doneWhen,omitempty"`
	EvidenceRequired []string    `json:"evidenceRequired,omitempty"`
	KeyResults       []KeyResult `json:"keyResults,omitempty"`
	Evidence         []string    `json:"evidence,omitempty"`
	CreatedAt        string      `json:"createdAt"`
	UpdatedAt        string      `json:"updatedAt"`
}

type KeyResult struct {
	ID          string   `json:"id"`
	Metric      string   `json:"metric"`
	Description string   `json:"description,omitempty"`
	Unit        string   `json:"unit,omitempty"`
	Direction   string   `json:"direction"`
	Baseline    *float64 `json:"baseline,omitempty"`
	Target      *float64 `json:"target"`
	Current     *float64 `json:"current,omitempty"`
	Status      string   `json:"status"`
	DueDate     string   `json:"dueDate,omitempty"`
	Evidence    []string `json:"evidence,omitempty"`
}

type Task struct {
	SchemaVersion string   `json:"schemaVersion"`
	ID            string   `json:"id"`
	GoalID        string   `json:"goalId,omitempty"`
	Title         string   `json:"title"`
	Description   string   `json:"description,omitempty"`
	Deliverables  []string `json:"deliverables,omitempty"`
	Acceptance    []string `json:"acceptance,omitempty"`
	Priority      string   `json:"priority,omitempty"`
	Owner         string   `json:"owner,omitempty"`
	DueDate       string   `json:"dueDate,omitempty"`
	EstimateHours *float64 `json:"estimateHours,omitempty"`
	Status        string   `json:"status"`
	DependsOn     []string `json:"dependsOn,omitempty"`
	AgentID       string   `json:"agentId,omitempty"`
	BlockedReason string   `json:"blockedReason,omitempty"`
	Result        string   `json:"result,omitempty"`
	Evidence      []string `json:"evidence,omitempty"`
	Verification  []string `json:"verification,omitempty"`
	CreatedAt     string   `json:"createdAt"`
	UpdatedAt     string   `json:"updatedAt"`
}

type Run struct {
	SchemaVersion string   `json:"schemaVersion"`
	ID            string   `json:"id"`
	TaskID        string   `json:"taskId"`
	Adapter       string   `json:"adapter"`
	Harness       string   `json:"harness,omitempty"`
	AgentID       string   `json:"agentId,omitempty"`
	Objective     string   `json:"objective,omitempty"`
	Status        string   `json:"status"`
	Worktree      string   `json:"worktree,omitempty"`
	Command       string   `json:"command,omitempty"`
	Commit        string   `json:"commit,omitempty"`
	ChangedFiles  []string `json:"changedFiles,omitempty"`
	Verification  []string `json:"verification,omitempty"`
	Logs          []string `json:"logs,omitempty"`
	Approval      string   `json:"approval,omitempty"`
	ExitCode      *int     `json:"exitCode,omitempty"`
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
var keyResultStatuses = map[string]bool{"planned": true, "in_progress": true, "achieved": true, "at_risk": true, "missed": true}
var keyResultDirections = map[string]bool{"increase": true, "decrease": true, "maintain": true, "boolean": true}

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
	if goal.Status == "active" || goal.Status == "completed" {
		if strings.TrimSpace(goal.Description) == "" {
			return fmt.Errorf("active or completed goals require a description")
		}
		if strings.TrimSpace(goal.Outcome) == "" {
			return fmt.Errorf("active or completed goals require an outcome")
		}
		if len(goal.DoneWhen) == 0 {
			return fmt.Errorf("active or completed goals require at least one doneWhen condition")
		}
	}
	if goal.Method != "" && !map[string]bool{"okr": true, "milestone": true, "experiment": true, "metric": true, "custom": true, "none": true}[goal.Method] {
		return fmt.Errorf("invalid goal method %q", goal.Method)
	}
	for _, result := range goal.KeyResults {
		if err := ValidateKeyResult(result); err != nil {
			return fmt.Errorf("invalid key result %q: %w", result.ID, err)
		}
	}
	if goal.Status == "completed" {
		if len(goal.Evidence) == 0 {
			return fmt.Errorf("completed goals require evidence")
		}
	}
	return nil
}

func ValidateKeyResult(result KeyResult) error {
	if err := ValidateID(result.ID); err != nil {
		return err
	}
	if strings.TrimSpace(result.Metric) == "" {
		return fmt.Errorf("metric is required")
	}
	if !keyResultDirections[result.Direction] {
		return fmt.Errorf("invalid direction %q", result.Direction)
	}
	if result.Target == nil {
		return fmt.Errorf("target is required")
	}
	if !keyResultStatuses[result.Status] {
		return fmt.Errorf("invalid status %q", result.Status)
	}
	if result.Status == "achieved" && (result.Current == nil || len(result.Evidence) == 0) {
		return fmt.Errorf("achieved key results require current value and evidence")
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
	if task.Priority != "" && !map[string]bool{"critical": true, "high": true, "medium": true, "low": true}[task.Priority] {
		return fmt.Errorf("invalid task priority %q", task.Priority)
	}
	if task.Status == "blocked" && strings.TrimSpace(task.BlockedReason) == "" {
		return fmt.Errorf("blocked tasks require blockedReason")
	}
	if task.Status == "completed" && len(task.Evidence) == 0 && strings.TrimSpace(task.Result) == "" {
		return fmt.Errorf("completed tasks require result or evidence")
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
	if run.Status == "succeeded" && len(run.Verification) == 0 {
		return fmt.Errorf("succeeded runs require verification evidence")
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
