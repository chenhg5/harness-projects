package cli

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"
)

func run(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := Run(args, bytes.NewBuffer(nil), &out, &errOut)
	return code, out.String(), errOut.String()
}

func TestProjectLifecycle(t *testing.T) {
	root := filepath.Join(t.TempDir(), "demo")
	code, _, stderr := run(t, "project", "init", "--path", root, "--name", "Demo")
	if code != 0 {
		t.Fatalf("init code=%d stderr=%s", code, stderr)
	}
	code, stdout, stderr := run(t, "project", "validate", "--path", root, "--json")
	if code != 0 {
		t.Fatalf("validate code=%d stderr=%s", code, stderr)
	}
	var result struct {
		Valid bool `json:"valid"`
	}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatal(err)
	}
	if !result.Valid {
		t.Fatalf("expected valid project: %s", stdout)
	}
}

func TestDryRunExitCode(t *testing.T) {
	root := filepath.Join(t.TempDir(), "demo")
	code, _, stderr := run(t, "project", "init", "--path", root, "--dry-run")
	if code != 10 {
		t.Fatalf("expected exit 10, got %d stderr=%s", code, stderr)
	}
}

func TestGoalTaskRunLifecycle(t *testing.T) {
	root := filepath.Join(t.TempDir(), "demo")
	if code, _, stderr := run(t, "project", "init", "--path", root); code != 0 {
		t.Fatalf("init code=%d stderr=%s", code, stderr)
	}
	if code, _, stderr := run(t, "goal", "create", "--project", root, "--id", "modernize", "--title", "Modernize API", "--status", "active"); code != 0 {
		t.Fatalf("goal create code=%d stderr=%s", code, stderr)
	}
	if code, _, stderr := run(t, "task", "create", "--project", root, "--id", "inventory", "--title", "Inventory API", "--goal-id", "modernize"); code != 0 {
		t.Fatalf("task create code=%d stderr=%s", code, stderr)
	}
	if code, _, stderr := run(t, "task", "create", "--project", root, "--id", "frontend", "--title", "Frontend integration", "--depends-on", "inventory"); code != 0 {
		t.Fatalf("dependent task create code=%d stderr=%s", code, stderr)
	}
	code, stdout, stderr := run(t, "task", "ready", "--project", root, "--json")
	if code != 0 || stderr != "" {
		t.Fatalf("task ready code=%d stderr=%s", code, stderr)
	}
	var ready []map[string]any
	if err := json.Unmarshal([]byte(stdout), &ready); err != nil {
		t.Fatal(err)
	}
	if len(ready) != 1 {
		t.Fatalf("unexpected ready tasks: %s", stdout)
	}
	task, ok := ready[0]["task"].(map[string]any)
	if !ok || task["id"] != "inventory" {
		t.Fatalf("unexpected ready task: %s", stdout)
	}
	if code, _, stderr := run(t, "task", "update", "--project", root, "--id", "inventory", "--status", "completed"); code != 0 {
		t.Fatalf("task update code=%d stderr=%s", code, stderr)
	}
	if code, _, stderr := run(t, "run", "create", "--project", root, "--id", "run-1", "--task-id", "frontend"); code != 0 {
		t.Fatalf("run create code=%d stderr=%s", code, stderr)
	}
	if code, _, stderr := run(t, "run", "update", "--project", root, "--id", "run-1", "--status", "running"); code != 0 {
		t.Fatalf("run update code=%d stderr=%s", code, stderr)
	}
}

func TestCoordinatorPlan(t *testing.T) {
	root := filepath.Join(t.TempDir(), "demo")
	if code, _, stderr := run(t, "project", "init", "--path", root); code != 0 {
		t.Fatalf("init code=%d stderr=%s", code, stderr)
	}
	if code, _, stderr := run(t, "task", "create", "--project", root, "--id", "first", "--title", "First"); code != 0 {
		t.Fatalf("task create code=%d stderr=%s", code, stderr)
	}
	code, stdout, stderr := run(t, "coordinator", "plan", "--project", root, "--json")
	if code != 0 || stderr != "" {
		t.Fatalf("plan code=%d stderr=%s", code, stderr)
	}
	var plan map[string]any
	if err := json.Unmarshal([]byte(stdout), &plan); err != nil {
		t.Fatal(err)
	}
	if plan["readyCount"] != float64(1) || plan["hasWork"] != true {
		t.Fatalf("unexpected plan: %s", stdout)
	}
}
