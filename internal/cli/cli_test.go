package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func run(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := Run(args, bytes.NewBuffer(nil), &out, &errOut)
	return code, out.String(), errOut.String()
}

func runInput(t *testing.T, input string, args ...string) (int, string, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := Run(args, bytes.NewBufferString(input), &out, &errOut)
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
	if code, _, stderr := run(t, "goal", "create", "--project", root, "--id", "modernize", "--title", "Modernize API", "--description", "Reduce API failure rate with measurable delivery evidence.", "--outcome", "API releases meet the agreed reliability target.", "--done-when", "reliability target is met", "--key-results", `[{"id":"reliability","metric":"successful API requests","direction":"increase","target":99,"status":"planned"}]`, "--status", "active"); code != 0 {
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
	if code, _, stderr := run(t, "task", "update", "--project", root, "--id", "inventory", "--status", "completed", "--result", "API implementation verified", "--evidence", "verification/api.md"); code != 0 {
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

func TestContextAndHandoffLifecycle(t *testing.T) {
	root := filepath.Join(t.TempDir(), "demo")
	if code, _, stderr := run(t, "project", "init", "--path", root); code != 0 {
		t.Fatalf("init code=%d stderr=%s", code, stderr)
	}
	if code, _, stderr := runInput(t, "Use project files as the portable source of truth.\n", "context", "add", "--project", root, "--id", "portable-context", "--kind", "decision", "--title", "Portable context", "--from", "-", "--tags", "memory,agents"); code != 0 {
		t.Fatalf("context add code=%d stderr=%s", code, stderr)
	}
	code, stdout, stderr := run(t, "context", "search", "--project", root, "--query", "portable agents", "--json")
	if code != 0 || stderr != "" {
		t.Fatalf("context search code=%d stderr=%s", code, stderr)
	}
	var matches []map[string]any
	if err := json.Unmarshal([]byte(stdout), &matches); err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("unexpected context matches: %s", stdout)
	}
	if code, _, stderr := run(t, "handoff", "create", "--project", root, "--id", "handoff-one", "--from", "codex", "--to", "claude", "--objective", "Build memory", "--completed", "Added search", "--next", "Add import"); code != 0 {
		t.Fatalf("handoff create code=%d stderr=%s", code, stderr)
	}
	code, stdout, stderr = run(t, "handoff", "latest", "--project", root, "--format", "text")
	if code != 0 || stderr != "" || !bytes.Contains([]byte(stdout), []byte("Build memory")) {
		t.Fatalf("handoff latest code=%d stderr=%s stdout=%s", code, stderr, stdout)
	}
}

func TestContextSearchIncludesPlainNotes(t *testing.T) {
	root := filepath.Join(t.TempDir(), "demo")
	if code, _, stderr := run(t, "project", "init", "--path", root, "--no-git"); code != 0 {
		t.Fatalf("init code=%d stderr=%s", code, stderr)
	}
	if err := os.WriteFile(filepath.Join(root, "context", "api-boundary.md"), []byte("Portable API boundary decision"), 0644); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr := run(t, "context", "search", "--project", root, "--query", "portable boundary", "--json")
	if code != 0 || stderr != "" {
		t.Fatalf("search code=%d stderr=%s", code, stderr)
	}
	var matches []struct {
		Type string `json:"type"`
		Path string `json:"path"`
	}
	if err := json.Unmarshal([]byte(stdout), &matches); err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 || matches[0].Type != "file" || matches[0].Path != "context/api-boundary.md" {
		t.Fatalf("unexpected matches: %s", stdout)
	}
}

func TestFinishClosesTaskWorklogAndStateTogether(t *testing.T) {
	t.Setenv("HP_SESSION_ID", "")
	t.Setenv("CODEX_THREAD_ID", "real-codex-session")
	root := filepath.Join(t.TempDir(), "demo")
	if code, _, stderr := run(t, "project", "init", "--path", root, "--no-git"); code != 0 {
		t.Fatalf("init code=%d stderr=%s", code, stderr)
	}
	if code, _, stderr := run(t, "task", "create", "--project", root, "--id", "api", "--title", "Build API", "--status", "ready"); code != 0 {
		t.Fatalf("task create code=%d stderr=%s", code, stderr)
	}
	if code, _, stderr := run(t, "task", "update", "--project", root, "--id", "api", "--status", "in_progress"); code != 0 {
		t.Fatalf("task update code=%d stderr=%s", code, stderr)
	}
	code, _, stderr := run(t, "finish", "--project", root, "--task-id", "api", "--id", "api-session", "--agent", "codex", "--objective", "Build API", "--completed", "Added routes", "--remaining", "None", "--verification", "go test ./...", "--next", "Review output")
	if code != 0 || stderr != "" {
		t.Fatalf("finish code=%d stderr=%s", code, stderr)
	}
	logs, err := filepath.Glob(filepath.Join(root, "worklogs", "*-api-session.md"))
	if err != nil || len(logs) != 1 {
		t.Fatalf("expected one named work log: %v, %v", logs, err)
	}
	logData, err := os.ReadFile(logs[0])
	if err != nil || !bytes.Contains(logData, []byte("- Session ID: real-codex-session")) {
		t.Fatalf("provider session was not linked to work log: %s, %v", logData, err)
	}
	code, stdout, stderr := run(t, "task", "get", "--project", root, "--id", "api", "--json")
	if code != 0 || stderr != "" {
		t.Fatalf("task get code=%d stderr=%s", code, stderr)
	}
	var task struct {
		Status string `json:"status"`
		Result string `json:"result"`
	}
	if err := json.Unmarshal([]byte(stdout), &task); err != nil {
		t.Fatal(err)
	}
	if task.Status != "completed" || task.Result != "Added routes" {
		t.Fatalf("unexpected task: %s", stdout)
	}
	code, stdout, stderr = run(t, "state", "get", "--project", root, "--json")
	if code != 0 || stderr != "" {
		t.Fatalf("state get code=%d stderr=%s", code, stderr)
	}
	var state struct {
		Progress    string   `json:"progress"`
		NextActions []string `json:"nextActions"`
	}
	if err := json.Unmarshal([]byte(stdout), &state); err != nil {
		t.Fatal(err)
	}
	if state.Progress != "1 of 1 tasks completed" || len(state.NextActions) < 1 || state.NextActions[0] != "Review output" {
		t.Fatalf("unexpected state: %s", stdout)
	}
	if code, stdout, stderr := run(t, "project", "doctor", "--path", root, "--json"); code != 0 || stderr != "" {
		t.Fatalf("doctor after finish code=%d stderr=%s stdout=%s", code, stderr, stdout)
	}
}

func TestWorklogCreateAcceptsExplicitProviderSession(t *testing.T) {
	t.Setenv("HP_SESSION_ID", "")
	t.Setenv("CODEX_THREAD_ID", "environment-session")
	root := filepath.Join(t.TempDir(), "demo")
	if code, _, stderr := run(t, "project", "init", "--path", root, "--no-git"); code != 0 {
		t.Fatalf("init code=%d stderr=%s", code, stderr)
	}
	code, stdout, stderr := run(t, "worklog", "create", "--project", root, "--id", "api-review", "--agent", "claude", "--harness", "claude-code", "--session-id", "claude-session", "--objective", "Review API", "--completed", "Read implementation", "--remaining", "Tests", "--next", "Run tests")
	if code != 0 || stderr != "" {
		t.Fatalf("worklog code=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
	var result struct {
		Record struct {
			ID        string `json:"id"`
			SessionID string `json:"sessionId"`
		} `json:"record"`
	}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil || result.Record.ID != "api-review" || result.Record.SessionID != "claude-session" {
		t.Fatalf("worklog IDs differ from request: %s, %v", stdout, err)
	}
	code, stdout, stderr = run(t, "worklog", "create", "--project", root, "--id", "api-follow-up", "--objective", "Continue API", "--completed", "Ran tests", "--remaining", "Review", "--next", "Request review")
	if code != 0 || stderr != "" {
		t.Fatalf("detected worklog code=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil || result.Record.ID != "api-follow-up" || result.Record.SessionID != "environment-session" {
		t.Fatalf("provider session was not detected: %s, %v", stdout, err)
	}
}

func TestDoctorAcceptsLegacyWorklogAndFlagsIncompleteOne(t *testing.T) {
	root := filepath.Join(t.TempDir(), "demo")
	if code, _, stderr := run(t, "project", "init", "--path", root, "--no-git"); code != 0 {
		t.Fatalf("init code=%d stderr=%s", code, stderr)
	}
	legacy := "# Work log\n- **Objective:** Fix search\n## Changes\nDebounced input.\n## Verification\nTests passed.\n## Follow-up\nPreview still needed; review the PR.\n"
	if err := os.WriteFile(filepath.Join(root, "worklogs", "legacy.md"), []byte(legacy), 0644); err != nil {
		t.Fatal(err)
	}
	if code, stdout, stderr := run(t, "project", "doctor", "--path", root, "--json"); code != 0 || stderr != "" {
		t.Fatalf("legacy log should be accepted: code=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
	incomplete := "# Work log: incomplete\n- Agent: codex\n- Harness: codex\n- Created: 2026-10-26T00:00:00Z\n- Task: api\n- Run: None recorded.\n## Objective\nFix search\n## Completed\n...\n## Remaining\nTests\n## Next action\nRun tests\n## Changes\nNone recorded.\n## Verification\nNone recorded.\n## Blockers\nNone recorded.\n"
	if err := os.WriteFile(filepath.Join(root, "worklogs", "incomplete.md"), []byte(incomplete), 0644); err != nil {
		t.Fatal(err)
	}
	if code, stdout, stderr := run(t, "project", "doctor", "--path", root, "--json"); code != 1 || stderr != "" || !bytes.Contains([]byte(stdout), []byte("incomplete_worklogs")) || !bytes.Contains([]byte(stdout), []byte("incomplete.md")) {
		t.Fatalf("incomplete log should be reported: code=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
}

func TestFinishDryRunIsExplicitAndDoesNotWrite(t *testing.T) {
	root := filepath.Join(t.TempDir(), "demo")
	if code, _, stderr := run(t, "project", "init", "--path", root, "--no-git"); code != 0 {
		t.Fatalf("init code=%d stderr=%s", code, stderr)
	}
	if code, _, stderr := run(t, "task", "create", "--project", root, "--id", "api", "--title", "Build API", "--status", "in_progress"); code != 0 {
		t.Fatalf("task create code=%d stderr=%s", code, stderr)
	}
	code, stdout, stderr := run(t, "finish", "--project", root, "--task-id", "api", "--id", "api-session", "--agent", "codex", "--objective", "Build API", "--completed", "Drafted routes", "--remaining", "Tests", "--next", "Run tests", "--dry-run")
	if code != 10 || stderr != "" {
		t.Fatalf("finish dry-run code=%d stderr=%s", code, stderr)
	}
	var result struct {
		DryRun bool `json:"dryRun"`
	}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil || !result.DryRun {
		t.Fatalf("dry-run result is not explicit: %s err=%v", stdout, err)
	}
	if matches, err := filepath.Glob(filepath.Join(root, "worklogs", "*-api-session.md")); err != nil || len(matches) != 0 {
		t.Fatalf("dry-run wrote worklog: matches=%v err=%v", matches, err)
	}
}

func TestProjectDoctorAndUpgradeRepairSkillDrift(t *testing.T) {
	root := filepath.Join(t.TempDir(), "demo")
	if code, _, stderr := run(t, "project", "init", "--path", root, "--no-git"); code != 0 {
		t.Fatalf("init code=%d stderr=%s", code, stderr)
	}
	link := filepath.Join(root, ".agents", "skills", "project-context")
	if err := os.Remove(link); err != nil {
		t.Fatal(err)
	}
	if code, stdout, _ := run(t, "project", "doctor", "--path", root, "--json"); code != 1 || !bytes.Contains([]byte(stdout), []byte("skill_mount_drift")) {
		t.Fatalf("doctor did not report drift: code=%d stdout=%s", code, stdout)
	}
	if code, _, stderr := run(t, "project", "upgrade", "--path", root); code != 0 || stderr != "" {
		t.Fatalf("upgrade code=%d stderr=%s", code, stderr)
	}
	if code, stdout, stderr := run(t, "project", "doctor", "--path", root, "--json"); code != 0 || stderr != "" {
		t.Fatalf("doctor after upgrade code=%d stderr=%s stdout=%s", code, stderr, stdout)
	}
}

func TestStateReconcileRepairsTaskStateDrift(t *testing.T) {
	root := filepath.Join(t.TempDir(), "demo")
	if code, _, stderr := run(t, "project", "init", "--path", root, "--no-git"); code != 0 {
		t.Fatalf("init code=%d stderr=%s", code, stderr)
	}
	if code, _, stderr := run(t, "task", "create", "--project", root, "--id", "api", "--title", "Build API", "--status", "ready"); code != 0 {
		t.Fatalf("task create code=%d stderr=%s", code, stderr)
	}
	if code, _, stderr := run(t, "task", "update", "--project", root, "--id", "api", "--status", "in_progress"); code != 0 {
		t.Fatalf("task update code=%d stderr=%s", code, stderr)
	}
	if code, stdout, _ := run(t, "project", "doctor", "--path", root, "--json"); code != 1 || !bytes.Contains([]byte(stdout), []byte("task_state_drift")) {
		t.Fatalf("doctor did not report state drift: code=%d stdout=%s", code, stdout)
	}
	if code, _, stderr := run(t, "state", "reconcile", "--project", root); code != 0 || stderr != "" {
		t.Fatalf("reconcile code=%d stderr=%s", code, stderr)
	}
	if code, stdout, stderr := run(t, "project", "doctor", "--path", root, "--json"); code != 0 || stderr != "" {
		t.Fatalf("doctor after reconcile code=%d stderr=%s stdout=%s", code, stderr, stdout)
	}
}
