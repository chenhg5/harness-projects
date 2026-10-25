package project

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/chenhg5/harness-projects/internal/skillmount"
)

func TestInitAndInspect(t *testing.T) {
	root := filepath.Join(t.TempDir(), "demo")
	plan, files, err := PlanInit(root, "Demo", false, time.Unix(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	if err := ApplyInit(plan, files); err != nil {
		t.Fatal(err)
	}
	status := Inspect(root)
	if !status.Valid {
		t.Fatalf("expected valid project, got: %#v", status.Problems)
	}
	if status.Name != "Demo" || status.Version != SchemaVersion {
		t.Fatalf("unexpected identity: %#v", status)
	}
}

func TestInitRequiresIfNotExists(t *testing.T) {
	root := filepath.Join(t.TempDir(), "demo")
	plan, files, err := PlanInit(root, "Demo", false, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := ApplyInit(plan, files); err != nil {
		t.Fatal(err)
	}
	if _, _, err := PlanInit(root, "Demo", false, time.Now()); err == nil {
		t.Fatal("expected existing project to require --if-not-exists")
	}
	if plan, _, err := PlanInit(root, "Demo", true, time.Now()); err != nil || plan.Created {
		t.Fatalf("expected idempotent init, plan=%#v err=%v", plan, err)
	}
}

func TestExistingProjectWithoutOptionalWakeupPrompt(t *testing.T) {
	root := filepath.Join(t.TempDir(), "demo")
	plan, files, err := PlanInit(root, "Demo", false, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := ApplyInit(plan, files); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, "agents", "coordinator-wakeup.md")); err != nil {
		t.Fatal(err)
	}
	if status := Inspect(root); !status.Valid {
		t.Fatalf("optional prompt invalidated project: %v", status.Problems)
	}
	if plan, _, err := PlanInit(root, "Demo", true, time.Now()); err != nil || plan.Created {
		t.Fatalf("existing project rejected: plan=%+v err=%v", plan, err)
	}
}

func TestInitPromptsAgentSessionRegistration(t *testing.T) {
	root := filepath.Join(t.TempDir(), "demo")
	_, files, err := PlanInit(root, "Demo", false, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"AGENTS.md", "skills/shared/project-context/SKILL.md"} {
		for _, command := range []string{"hp agent register", "hp agent discover"} {
			if !bytes.Contains(files[path], []byte(command)) {
				t.Fatalf("%s does not instruct agents to run %q", path, command)
			}
		}
	}
}

func TestInspectDetectsAndRecoversSkillMountDrift(t *testing.T) {
	root := filepath.Join(t.TempDir(), "demo")
	plan, files, err := PlanInit(root, "Demo", false, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := ApplyInit(plan, files); err != nil {
		t.Fatal(err)
	}
	skillDirectory := filepath.Join(root, "skills", "shared", "extra")
	if err := os.MkdirAll(skillDirectory, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDirectory, "SKILL.md"), []byte("---\nname: extra\n---\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if status := Inspect(root); status.Valid {
		t.Fatal("expected missing skill mounts to invalidate the project")
	}
	targets, _ := skillmount.ParseTargets("codex,claude")
	mountPlan, err := skillmount.BuildPlan(root, targets, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := skillmount.Apply(mountPlan); err != nil {
		t.Fatal(err)
	}
	if status := Inspect(root); !status.Valid {
		t.Fatalf("expected synchronized project to be valid: %v", status.Problems)
	}
}

func TestInitMountsSharedSkillsForEveryHarness(t *testing.T) {
	root := filepath.Join(t.TempDir(), "demo")
	plan, files, err := PlanInit(root, "Demo", false, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := ApplyInit(plan, files); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		".codex/skills/project-context",
		".claude/skills/project-context",
		".cursor/skills/project-context",
		".agents/skills/project-context",
	} {
		if info, err := os.Lstat(filepath.Join(root, path)); err != nil || info.Mode()&os.ModeSymlink == 0 {
			t.Fatalf("missing skill mount %s: info=%v err=%v", path, info, err)
		}
	}
}

func TestUpgradeRepairsManagedWorkflowAndSkillMounts(t *testing.T) {
	root := filepath.Join(t.TempDir(), "demo")
	initPlan, files, err := PlanInit(root, "Demo", false, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := ApplyInit(initPlan, files); err != nil {
		t.Fatal(err)
	}
	agents := filepath.Join(root, ".agents", "skills", "project-context")
	if err := os.Remove(agents); err != nil {
		t.Fatal(err)
	}
	agentsFile := filepath.Join(root, "AGENTS.md")
	content, err := os.ReadFile(agentsFile)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(agentsFile, append(content, []byte("\nProject-specific instruction.\n")...), 0644); err != nil {
		t.Fatal(err)
	}
	plan, err := PlanUpgrade(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := ApplyUpgrade(plan); err != nil {
		t.Fatal(err)
	}
	updated, err := os.ReadFile(agentsFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(updated, []byte("Project-specific instruction.")) || bytes.Count(updated, []byte(managedStart)) != 1 {
		t.Fatalf("upgrade did not preserve custom instructions: %s", updated)
	}
	if info, err := os.Lstat(agents); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("agents mount not repaired: info=%v err=%v", info, err)
	}
	idempotent, err := PlanUpgrade(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(idempotent.Changes) != 0 {
		t.Fatalf("upgrade should be idempotent: %#v", idempotent.Changes)
	}
}
