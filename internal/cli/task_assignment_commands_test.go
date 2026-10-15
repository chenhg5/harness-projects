package cli

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chenhg5/harness-projects/internal/model"
)

func TestTaskAssignmentAndSessionInbox(t *testing.T) {
	root := filepath.Join(t.TempDir(), "project")
	mustRun := func(args ...string) string {
		t.Helper()
		code, out, stderr := run(t, args...)
		if code != 0 {
			t.Fatalf("%v: code=%d stderr=%s", args, code, stderr)
		}
		return out
	}
	mustRun("project", "init", "--path", root)
	mustRun("task", "create", "--project", root, "--id", "foundation", "--title", "Foundation")
	mustRun("task", "create", "--project", root, "--id", "dependent", "--title", "Dependent", "--depends-on", "foundation")
	registered := mustRun("agent", "register", "--project", root, "--harness", "cc", "--session-id", "session-a")
	var registration struct {
		Agent struct {
			ID string `json:"id"`
		} `json:"agent"`
	}
	if err := json.Unmarshal([]byte(registered), &registration); err != nil {
		t.Fatal(err)
	}
	mustRun("task", "assign", "--project", root, "--id", "dependent", "--registration-id", registration.Agent.ID)
	code, out, stderr := run(t, "task", "inbox", "--project", root, "--harness", "claude-code", "--session-id", "session-a")
	if code != 0 || stderr != "" {
		t.Fatalf("inbox: code=%d stderr=%s", code, stderr)
	}
	var inbox []struct {
		Task       model.Task `json:"task"`
		Ready      bool       `json:"ready"`
		Incomplete []string   `json:"incomplete"`
	}
	if err := json.Unmarshal([]byte(out), &inbox); err != nil {
		t.Fatal(err)
	}
	if len(inbox) != 1 || inbox[0].Task.Assignment == nil || inbox[0].Task.Assignment.Harness != "claude-code" || inbox[0].Task.Assignment.SessionID != "session-a" || inbox[0].Ready || len(inbox[0].Incomplete) != 1 {
		t.Fatalf("unexpected inbox: %s", out)
	}
	if got := mustRun("task", "inbox", "--project", root, "--harness", "codex", "--session-id", "session-a"); got != "[]\n" {
		t.Fatalf("wrong harness saw task: %s", got)
	}
	if got := mustRun("task", "inbox", "--project", root, "--harness", "cc", "--session-id", "session-a", "--ready-only"); got != "[]\n" {
		t.Fatalf("blocked dependency appeared ready: %s", got)
	}
	mustRun("task", "update", "--project", root, "--id", "foundation", "--status", "completed", "--result", "Verified")
	ready := mustRun("task", "inbox", "--project", root, "--harness", "cc", "--session-id", "session-a", "--ready-only")
	if err := json.Unmarshal([]byte(ready), &inbox); err != nil {
		t.Fatal(err)
	}
	if len(inbox) != 1 || !inbox[0].Ready {
		t.Fatalf("expected ready task: %s", ready)
	}
	mustRun("agent", "unregister", "--project", root, "--id", registration.Agent.ID)
	if got := mustRun("agent", "discover", "--project", root); !strings.Contains(got, registration.Agent.ID) {
		t.Fatalf("participant disappeared after local lease ended: %s", got)
	}
	if got := mustRun("task", "inbox", "--project", root, "--harness", "cc", "--session-id", "session-a", "--ready-only"); got == "[]\n" {
		t.Fatal("assignment disappeared when local presence was removed")
	}
	if code, _, _ := run(t, "task", "assign", "--project", root, "--id", "dependent", "--harness", "codex", "--session-id", "session-b"); code != 5 {
		t.Fatalf("reassignment without --replace: code=%d", code)
	}
	mustRun("task", "assign", "--project", root, "--id", "dependent", "--harness", "codex", "--session-id", "session-b", "--replace")
	if got := mustRun("task", "inbox", "--project", root, "--harness", "cc", "--session-id", "session-a"); got != "[]\n" {
		t.Fatalf("old session still owns task: %s", got)
	}
	var task model.Task
	if err := json.Unmarshal([]byte(mustRun("task", "get", "--project", root, "--id", "dependent")), &task); err != nil {
		t.Fatal(err)
	}
	if task.Assignment.SessionID != "session-b" || len(task.AssignmentHistory) != 1 || task.AssignmentHistory[0].SessionID != "session-a" || task.AssignmentHistory[0].EndedAt == "" {
		t.Fatalf("assignment history missing: %+v", task)
	}
	mustRun("task", "unassign", "--project", root, "--id", "dependent")
	task = model.Task{}
	if err := json.Unmarshal([]byte(mustRun("task", "get", "--project", root, "--id", "dependent")), &task); err != nil {
		t.Fatal(err)
	}
	if task.Assignment != nil || len(task.AssignmentHistory) != 2 {
		t.Fatalf("unassign history missing: %+v", task)
	}
	mustRun("task", "assign", "--project", root, "--id", "dependent", "--registration-id", registration.Agent.ID)
	if err := json.Unmarshal([]byte(mustRun("task", "get", "--project", root, "--id", "dependent")), &task); err != nil {
		t.Fatal(err)
	}
	if task.Assignment == nil || task.Assignment.RegistrationID != registration.Agent.ID {
		t.Fatalf("durable participant could not be used for assignment: %+v", task.Assignment)
	}
}
