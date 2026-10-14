package worklog

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/chenhg5/harness-projects/internal/fileops"
	"github.com/chenhg5/harness-projects/internal/model"
)

// Record is the short, chronological summary written after an agent session.
// It intentionally complements handoff.Record: work logs describe activity,
// while handoffs describe a transfer of responsibility.
type Record struct {
	ID           string `json:"id"`
	Agent        string `json:"agent"`
	Harness      string `json:"harness,omitempty"`
	TaskID       string `json:"taskId,omitempty"`
	RunID        string `json:"runId,omitempty"`
	CreatedAt    string `json:"createdAt"`
	Objective    string `json:"objective"`
	Completed    string `json:"completed"`
	Remaining    string `json:"remaining"`
	Verification string `json:"verification,omitempty"`
	Next         string `json:"next"`
	Blockers     string `json:"blockers,omitempty"`
	Changes      string `json:"changes,omitempty"`
}

func (r Record) Validate() error {
	if err := model.ValidateID(r.ID); err != nil {
		return err
	}
	if strings.TrimSpace(r.Agent) == "" {
		return fmt.Errorf("worklog agent is required")
	}
	if strings.TrimSpace(r.Objective) == "" || strings.TrimSpace(r.Completed) == "" || strings.TrimSpace(r.Remaining) == "" || strings.TrimSpace(r.Next) == "" {
		return fmt.Errorf("worklog objective, completed, remaining, and next are required")
	}
	if _, err := time.Parse(time.RFC3339, r.CreatedAt); err != nil {
		return fmt.Errorf("invalid createdAt: %w", err)
	}
	return nil
}

func (r Record) Filename() (string, error) {
	created, err := time.Parse(time.RFC3339, r.CreatedAt)
	if err != nil {
		return "", err
	}
	return created.UTC().Format("2006-01-02T15-04-05Z") + "-" + r.ID + ".md", nil
}

func (r Record) Markdown() string {
	return fmt.Sprintf("# Work log: %s\n\n- Agent: %s\n- Harness: %s\n- Created: %s\n- Task: %s\n- Run: %s\n\n## Objective\n\n%s\n\n## Completed\n\n%s\n\n## Remaining\n\n%s\n\n## Changes\n\n%s\n\n## Verification\n\n%s\n\n## Next action\n\n%s\n\n## Blockers\n\n%s\n",
		r.ID, r.Agent, optional(r.Harness), r.CreatedAt, optional(r.TaskID), optional(r.RunID),
		r.Objective, r.Completed, r.Remaining, optional(r.Changes), optional(r.Verification), r.Next, optional(r.Blockers))
}

func optional(value string) string {
	if strings.TrimSpace(value) == "" {
		return "None recorded."
	}
	return value
}

func List(root string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(root, "worklogs"))
	if err != nil {
		return nil, err
	}
	names := []string{}
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == "README.md" || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	return names, nil
}

func Latest(root string) (string, string, error) {
	names, err := List(root)
	if err != nil {
		return "", "", err
	}
	if len(names) == 0 {
		return "", "", os.ErrNotExist
	}
	name := names[len(names)-1]
	_, data, err := fileops.Read(root, "worklogs", name)
	if err != nil {
		return "", "", err
	}
	return name, string(data), nil
}
