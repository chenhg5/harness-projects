package model

import (
	"fmt"
	"strings"
)

// ProjectState is a materialized control-plane snapshot. It is deliberately
// derived from goals, tasks, runs, and verification records so agents can
// orient quickly without treating a prose status file as the source of truth.
type ProjectState struct {
	SchemaVersion  string   `json:"schemaVersion"`
	UpdatedAt      string   `json:"updatedAt"`
	CurrentGoalID  string   `json:"currentGoalId,omitempty"`
	CurrentPhase   string   `json:"currentPhase"`
	Health         string   `json:"health"`
	Progress       string   `json:"progress"`
	ActiveTaskIDs  []string `json:"activeTaskIds,omitempty"`
	BlockedTaskIDs []string `json:"blockedTaskIds,omitempty"`
	NextActions    []string `json:"nextActions"`
	Risks          []string `json:"risks,omitempty"`
	LastVerifiedAt string   `json:"lastVerifiedAt,omitempty"`
	SourceRefs     []string `json:"sourceRefs"`
}

func ValidateProjectState(state ProjectState) error {
	if strings.TrimSpace(state.UpdatedAt) == "" {
		return fmt.Errorf("updatedAt is required")
	}
	if strings.TrimSpace(state.CurrentPhase) == "" {
		return fmt.Errorf("currentPhase is required")
	}
	if strings.TrimSpace(state.Health) == "" {
		return fmt.Errorf("health is required")
	}
	if !map[string]bool{"unknown": true, "on_track": true, "at_risk": true, "blocked": true, "complete": true}[state.Health] {
		return fmt.Errorf("invalid health %q", state.Health)
	}
	if strings.TrimSpace(state.Progress) == "" {
		return fmt.Errorf("progress is required")
	}
	if len(state.NextActions) == 0 {
		return fmt.Errorf("nextActions must contain at least one action")
	}
	if len(state.SourceRefs) == 0 {
		return fmt.Errorf("sourceRefs must identify the records used to derive state")
	}
	return nil
}
