package project

import (
	"path/filepath"
	"testing"
	"time"
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
