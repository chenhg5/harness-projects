package fileops

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chenhg5/harness-projects/internal/project"
)

func makeProject(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "demo")
	plan, files, err := project.PlanInit(root, "Demo", false, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := project.ApplyInit(plan, files); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestWritePlanAndApply(t *testing.T) {
	root := makeProject(t)
	content := []byte("# Architecture\n")
	plan, err := PlanWrite(root, "spec", "architecture/api.md", content, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := ApplyWrite(plan, content); err != nil {
		t.Fatal(err)
	}
	_, got, err := Read(root, "spec", "architecture/api.md")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(content) {
		t.Fatalf("got %q, want %q", got, content)
	}
}

func TestRejectsUnsafePath(t *testing.T) {
	root := makeProject(t)
	for _, relative := range []string{"../outside.md", ".ssh/key", "a\\b.md"} {
		if _, err := Target(root, "spec", relative); err == nil || !strings.Contains(err.Error(), "unsafe") && !strings.Contains(err.Error(), "invalid") {
			t.Fatalf("expected unsafe path rejection for %q, got %v", relative, err)
		}
	}
}
