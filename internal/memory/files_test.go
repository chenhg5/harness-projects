package memory

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSearchFilesFindsPlainContextNotes(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "context", "research"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "context", "research", "api-boundary.md"), []byte("Portable API boundary decision"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "context", "README.md"), []byte("Portable API boundary decision"), 0644); err != nil {
		t.Fatal(err)
	}
	matches, err := SearchFiles(root, "portable boundary")
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 || matches[0].Path != "context/research/api-boundary.md" {
		t.Fatalf("unexpected matches: %#v", matches)
	}
}
