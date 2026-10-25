package cli

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSkillSync(t *testing.T) {
	root := filepath.Join(t.TempDir(), "demo")
	if code, _, stderr := run(t, "project", "init", "--path", root, "--no-git"); code != 0 {
		t.Fatalf("init code=%d stderr=%s", code, stderr)
	}
	code, stdout, stderr := run(t, "skill", "sync", "--project", root, "--targets", "agents", "--json")
	if code != 0 || stderr != "" {
		t.Fatalf("sync code=%d stderr=%s", code, stderr)
	}
	var result struct {
		Action  string `json:"action"`
		Changes []any  `json:"changes"`
	}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatal(err)
	}
	if result.Action != "skill.sync" || len(result.Changes) != 0 {
		t.Fatalf("unexpected result: %s", stdout)
	}
	target, err := os.Readlink(filepath.Join(root, ".agents/skills/project-context"))
	if err != nil {
		t.Fatal(err)
	}
	if target != filepath.FromSlash("../../skills/shared/project-context") {
		t.Fatalf("unexpected target %q", target)
	}
}

func TestSkillSyncDryRunDoesNotWrite(t *testing.T) {
	root := filepath.Join(t.TempDir(), "demo")
	if code, _, stderr := run(t, "project", "init", "--path", root, "--no-git"); code != 0 {
		t.Fatalf("init code=%d stderr=%s", code, stderr)
	}
	if err := os.RemoveAll(filepath.Join(root, ".agents")); err != nil {
		t.Fatal(err)
	}
	code, _, stderr := run(t, "skill", "sync", "--project", root, "--targets", "agents", "--dry-run")
	if code != 10 || stderr != "" {
		t.Fatalf("dry run code=%d stderr=%s", code, stderr)
	}
	if _, err := os.Lstat(filepath.Join(root, ".agents")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("dry run created mount directory: %v", err)
	}
}

func TestSkillSyncReportsConflict(t *testing.T) {
	root := filepath.Join(t.TempDir(), "demo")
	if code, _, stderr := run(t, "project", "init", "--path", root, "--no-git"); code != 0 {
		t.Fatalf("init code=%d stderr=%s", code, stderr)
	}
	path := filepath.Join(root, ".agents/skills/project-context")
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatal(err)
	}
	code, _, stderr := run(t, "skill", "sync", "--project", root, "--targets", "agents")
	if code != 5 || stderr == "" {
		t.Fatalf("conflict code=%d stderr=%s", code, stderr)
	}
}
