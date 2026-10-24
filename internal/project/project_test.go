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
