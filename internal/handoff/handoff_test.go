package handoff

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/chenhg5/harness-projects/internal/project"
)

func TestRecordMarkdownAndLatest(t *testing.T) {
	root := filepath.Join(t.TempDir(), "project")
	plan, files, err := project.PlanInit(root, "Test", false, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := project.ApplyInit(plan, files); err != nil {
		t.Fatal(err)
	}
	record := Record{
		ID: "handoff-one", From: "codex", To: "claude",
		CreatedAt: "2026-10-14T12:00:00Z",
		Objective: "Build portable memory", Completed: "Added search",
		Next: "Add import", Verification: "go test ./...",
	}
	if err := record.Validate(); err != nil {
		t.Fatal(err)
	}
	name, err := record.Filename()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "handoffs", name), []byte(record.Markdown()), 0644); err != nil {
		t.Fatal(err)
	}
	gotName, content, err := Latest(root)
	if err != nil {
		t.Fatal(err)
	}
	if gotName != name || content == "" {
		t.Fatalf("latest = %q, %q", gotName, content)
	}
}
