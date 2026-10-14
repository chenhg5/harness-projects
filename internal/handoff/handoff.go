package handoff

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

type Record struct {
	ID           string `json:"id"`
	From         string `json:"from"`
	To           string `json:"to"`
	CreatedAt    string `json:"createdAt"`
	Objective    string `json:"objective"`
	Completed    string `json:"completed"`
	Next         string `json:"next"`
	Repositories string `json:"repositories,omitempty"`
	Verification string `json:"verification,omitempty"`
	Blockers     string `json:"blockers,omitempty"`
}

func (r Record) Validate() error {
	if err := model.ValidateID(r.ID); err != nil {
		return err
	}
	if strings.TrimSpace(r.From) == "" || strings.TrimSpace(r.To) == "" {
		return fmt.Errorf("handoff from and to are required")
	}
	if strings.TrimSpace(r.Objective) == "" || strings.TrimSpace(r.Completed) == "" || strings.TrimSpace(r.Next) == "" {
		return fmt.Errorf("handoff objective, completed, and next actions are required")
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
	return fmt.Sprintf("# Handoff: %s\n\n- From: %s\n- To: %s\n- Created: %s\n\n## Objective\n\n%s\n\n## Completed\n\n%s\n\n## Next actions\n\n%s\n\n## Repositories and commits\n\n%s\n\n## Verification\n\n%s\n\n## Blockers and risks\n\n%s\n",
		r.ID, r.From, r.To, r.CreatedAt, r.Objective, r.Completed, r.Next,
		optional(r.Repositories), optional(r.Verification), optional(r.Blockers))
}

func optional(value string) string {
	if strings.TrimSpace(value) == "" {
		return "None recorded."
	}
	return value
}

func List(root string) ([]string, error) {
	directory := filepath.Join(root, "handoffs")
	entries, err := os.ReadDir(directory)
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
	_, data, err := fileops.Read(root, "handoffs", name)
	if err != nil {
		return "", "", err
	}
	return name, string(data), nil
}
