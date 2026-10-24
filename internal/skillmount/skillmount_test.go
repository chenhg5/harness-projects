package skillmount

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildPlanAndApply(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "shared/common")
	writeSkill(t, root, "codex/codex-only")
	writeSkill(t, root, "vendor/tool")
	writeSkill(t, root, "vendor/tool/upstream")

	targets, err := ParseTargets("claude,agents")
	if err != nil {
		t.Fatal(err)
	}
	plan, err := BuildPlan(root, targets, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Skills) != 3 {
		t.Fatalf("expected outer skills only, got %#v", plan.Skills)
	}
	if len(plan.Changes) != 4 {
		t.Fatalf("expected two shared skills for two targets, got %#v", plan.Changes)
	}
	if err := Apply(plan); err != nil {
		t.Fatal(err)
	}
	assertLink(t, filepath.Join(root, ".claude/skills/common"), filepath.FromSlash("../../skills/shared/common"))
	assertLink(t, filepath.Join(root, ".agents/skills/tool"), filepath.FromSlash("../../skills/vendor/tool"))
	if _, err := os.Lstat(filepath.Join(root, ".claude/skills/codex-only")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("codex-only skill leaked to claude: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(root, ".claude/skills/upstream")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("nested skill was mounted: %v", err)
	}

	second, err := BuildPlan(root, targets, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Changes) != 0 || second.Unchanged != 4 {
		t.Fatalf("expected idempotent plan, got %#v", second)
	}
}

func TestBuildPlanRejectsNonSymlinkCollision(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "shared/common")
	path := filepath.Join(root, ".claude/skills/common")
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatal(err)
	}
	targets, _ := ParseTargets("claude")
	_, err := BuildPlan(root, targets, false)
	var conflict *ConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestPruneOnlyRemovesProjectSkillLinks(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "shared/current")
	mountRoot := filepath.Join(root, ".claude/skills")
	if err := os.MkdirAll(mountRoot, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../../skills/shared/removed", filepath.Join(mountRoot, "removed")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/tmp/external-skill", filepath.Join(mountRoot, "external")); err != nil {
		t.Fatal(err)
	}
	targets, _ := ParseTargets("claude")
	plan, err := BuildPlan(root, targets, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(plan); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(mountRoot, "removed")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("stale project link remains: %v", err)
	}
	assertLink(t, filepath.Join(mountRoot, "external"), "/tmp/external-skill")
}

func TestDuplicateNamesConflictPerTarget(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "shared/common")
	writeSkill(t, root, "vendor/common")
	targets, _ := ParseTargets("agents")
	_, err := BuildPlan(root, targets, false)
	var conflict *ConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("expected duplicate-name conflict, got %v", err)
	}
}

func TestParseTargets(t *testing.T) {
	targets, err := ParseTargets("claude-code,agents,claude")
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 2 || targets[0] != "claude" || targets[1] != "agents" {
		t.Fatalf("unexpected targets: %#v", targets)
	}
	if _, err := ParseTargets("agent"); err == nil {
		t.Fatal("expected singular agent target to be rejected")
	}
}

func writeSkill(t *testing.T, root, relative string) {
	t.Helper()
	directory := filepath.Join(root, "skills", filepath.FromSlash(relative))
	if err := os.MkdirAll(directory, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "SKILL.md"), []byte("---\nname: test\n---\n"), 0644); err != nil {
		t.Fatal(err)
	}
}

func assertLink(t *testing.T, path, target string) {
	t.Helper()
	actual, err := os.Readlink(path)
	if err != nil {
		t.Fatal(err)
	}
	if actual != target {
		t.Fatalf("%s points to %q, want %q", path, actual, target)
	}
}
