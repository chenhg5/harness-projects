package cli

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestAgentRegistrationDiscoveryLifecycle(t *testing.T) {
	root := filepath.Join(t.TempDir(), "project")
	if code, _, stderr := run(t, "project", "init", "--path", root); code != 0 {
		t.Fatalf("init: code=%d stderr=%s", code, stderr)
	}
	if code, _, stderr := run(t, "agent", "register", "--project", root, "--harness", "codex"); code != 2 {
		t.Fatalf("missing session ID should be a usage error: code=%d stderr=%s", code, stderr)
	}
	args := []string{"agent", "register", "--project", root, "--harness", "cc", "--session-id", "claude-session-1", "--modules", "api,coordinator", "--capabilities", "go,review", "--focus", "Review API", "--worktree", "repos", "--dry-run"}
	if code, _, stderr := run(t, args...); code != 10 {
		t.Fatalf("register dry-run: code=%d stderr=%s", code, stderr)
	}
	if code, stdout, stderr := run(t, "agent", "discover", "--project", root); code != 0 || stdout != "[]\n" || stderr != "" {
		t.Fatalf("dry-run wrote registration: code=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
	args = []string{"agent", "register", "--project", root, "--harness", "cc", "--session-id", "claude-session-1", "--modules", "api,coordinator", "--capabilities", "go,review", "--focus", "Review API"}
	code, stdout, stderr := run(t, args...)
	if code != 0 || stderr != "" {
		t.Fatalf("register: code=%d stderr=%s", code, stderr)
	}
	var created struct {
		Agent struct {
			ID        string `json:"id"`
			Harness   string `json:"harness"`
			SessionID string `json:"sessionId"`
		} `json:"agent"`
	}
	if err := json.Unmarshal([]byte(stdout), &created); err != nil {
		t.Fatal(err)
	}
	if created.Agent.ID == "" || created.Agent.Harness != "claude-code" || created.Agent.SessionID != "claude-session-1" {
		t.Fatalf("unexpected registration: %s", stdout)
	}
	code, stdout, stderr = run(t, "agent", "discover", "--project", root, "--harness", "claude-code", "--module", "api", "--capability", "go")
	if code != 0 || stderr != "" {
		t.Fatalf("discover: code=%d stderr=%s", code, stderr)
	}
	var found []map[string]any
	if err := json.Unmarshal([]byte(stdout), &found); err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || found[0]["id"] != created.Agent.ID || found[0]["live"] != true {
		t.Fatalf("unexpected discovery: %s", stdout)
	}
	if code, _, stderr := run(t, "agent", "heartbeat", "--project", root, "--id", created.Agent.ID, "--availability", "available"); code != 0 {
		t.Fatalf("heartbeat: code=%d stderr=%s", code, stderr)
	}
	if code, _, stderr := run(t, "agent", "prune", "--project", root, "--dry-run"); code != 10 {
		t.Fatalf("prune dry-run: code=%d stderr=%s", code, stderr)
	}
	if code, _, stderr := run(t, "agent", "unregister", "--project", root, "--id", created.Agent.ID); code != 0 {
		t.Fatalf("unregister: code=%d stderr=%s", code, stderr)
	}
	if code, stdout, stderr := run(t, "agent", "discover", "--project", root); code != 0 || stdout != "[]\n" || stderr != "" {
		t.Fatalf("unregister left registration: code=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
}
