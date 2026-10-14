package cleanup

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBuildAndApplyArchive(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "worklogs", "old.md")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
	plan, err := Build(root, "24h", "worklogs", "archive", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Candidates) != 1 {
		t.Fatalf("expected one candidate, got %#v", plan.Candidates)
	}
	if _, err := Apply(plan, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("source still exists: %v", err)
	}
	if _, err := os.Stat(plan.Candidates[0].Destination); err != nil {
		t.Fatalf("archive missing: %v", err)
	}
}

func TestDeleteRequiresForce(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "artifacts", "old.txt")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-48 * time.Hour)
	_ = os.Chtimes(path, old, old)
	plan, err := Build(root, "24h", "artifacts", "delete", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(plan, false); err == nil {
		t.Fatal("expected force requirement")
	}
}
