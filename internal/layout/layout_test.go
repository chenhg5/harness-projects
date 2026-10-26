package layout

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingUsesBuiltIns(t *testing.T) {
	contract, exists, err := Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatal("missing contract should not report an existing file")
	}
	if !Allowed(contract, "tasks") || Allowed(contract, "lib") {
		t.Fatalf("unexpected built-in layout: %#v", Entries(contract))
	}
}

func TestAddSaveAndLoadProjectDirectory(t *testing.T) {
	root := t.TempDir()
	contract, err := Add(Default(), "lib", "Third-party reference material.", false)
	if err != nil {
		t.Fatal(err)
	}
	if err := Save(root, contract); err != nil {
		t.Fatal(err)
	}
	loaded, exists, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if !exists || !Allowed(loaded, "lib") || len(ProjectEntries(loaded)) != 1 {
		t.Fatalf("unexpected project layout: %#v", loaded)
	}
	info, err := os.Stat(filepath.Join(root, filepath.FromSlash(RelativePath)))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0644 {
		t.Fatalf("unexpected contract permissions: %v", info.Mode().Perm())
	}
}

func TestAddRejectsTraversalAndBuiltInOverrides(t *testing.T) {
	for _, path := range []string{"../lib", "nested/lib", "tasks"} {
		if _, err := Add(Default(), path, "Description", false); err == nil {
			t.Fatalf("expected %q to be rejected", path)
		}
	}
	dirty := Default()
	dirty.Directories[" lib"] = "Description"
	if err := Validate(dirty); err == nil {
		t.Fatal("expected a manually authored path with whitespace to be rejected")
	}
}

func TestAddRequiresReplaceForExistingEntry(t *testing.T) {
	contract, err := Add(Default(), "lib", "First", false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Add(contract, "lib", "Second", false); err == nil {
		t.Fatal("expected duplicate directory to require replace")
	}
	updated, err := Add(contract, "lib", "Second", true)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Directories["lib"] != "Second" {
		t.Fatalf("unexpected replacement: %#v", updated)
	}
}
