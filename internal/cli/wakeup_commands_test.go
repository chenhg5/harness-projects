package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chenhg5/harness-projects/internal/presence"
	"github.com/chenhg5/harness-projects/internal/wakeup"
)

func TestCoordinatorRoleHandoffAndDiscovery(t *testing.T) {
	root := filepath.Join(t.TempDir(), "project")
	if code, _, err := run(t, "project", "init", "--path", root); code != 0 {
		t.Fatal(err)
	}
	for _, session := range []string{"first", "second"} {
		if code, _, err := run(t, "agent", "register", "--project", root, "--harness", "codex", "--session-id", session, "--worktree", root); code != 0 {
			t.Fatal(err)
		}
	}
	first, second := presence.RegistrationID("codex", "first"), presence.RegistrationID("codex", "second")
	if code, _, err := run(t, "coordinator", "assign", "--project", root, "--registration-id", first); code != 0 {
		t.Fatal(err)
	}
	if code, _, _ := run(t, "coordinator", "assign", "--project", root, "--registration-id", second); code != 5 {
		t.Fatalf("expected explicit handoff, got %d", code)
	}
	if code, _, err := run(t, "coordinator", "assign", "--project", root, "--registration-id", second, "--replace"); code != 0 {
		t.Fatal(err)
	}
	code, out, err := run(t, "agent", "discover", "--project", root, "--role", "coordinator")
	if code != 0 || err != "" {
		t.Fatalf("discover %d %s", code, err)
	}
	var items []struct{ ID, Role string }
	if err := json.Unmarshal([]byte(out), &items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != second || items[0].Role != "coordinator" {
		t.Fatalf("unexpected Coordinator: %s", out)
	}
	if code, _, err := run(t, "coordinator", "clear", "--project", root, "--registration-id", second); code != 0 {
		t.Fatal(err)
	}
	if code, out, _ := run(t, "agent", "discover", "--project", root); code != 0 || !strings.Contains(out, first) || !strings.Contains(out, second) {
		t.Fatalf("participants were lost: %s", out)
	}
}

func TestWakeupConfigurationAndExactSessionQueue(t *testing.T) {
	root := filepath.Join(t.TempDir(), "project")
	if code, _, err := run(t, "project", "init", "--path", root); code != 0 {
		t.Fatal(err)
	}
	if code, _, err := run(t, "agent", "register", "--project", root, "--harness", "codex", "--session-id", "target-session", "--worktree", root); code != 0 {
		t.Fatal(err)
	}
	id := presence.RegistrationID("codex", "target-session")
	if code, _, err := run(t, "coordinator", "assign", "--project", root, "--registration-id", id); code != 0 {
		t.Fatal(err)
	}
	code, out, stderr := run(t, "wakeup", "configure", "--project", root, "--every", "30m", "--worktree", ".", "--prompt-file", "agents/coordinator-wakeup.md", "--dry-run")
	if code != 10 || stderr != "" || !strings.Contains(out, "firstDueAt") {
		t.Fatalf("dry run: %d %s %s", code, out, stderr)
	}
	if _, err := os.Stat(filepath.Join(root, "agents", "coordinator-wakeup.json")); !os.IsNotExist(err) {
		t.Fatal("dry run wrote config")
	}
	if code, _, err := run(t, "wakeup", "configure", "--project", root, "--every", "30m", "--worktree", ".", "--prompt-file", "agents/coordinator-wakeup.md"); code != 0 {
		t.Fatalf("configure: %s", err)
	}
	argsPath := filepath.Join(t.TempDir(), "args.txt")
	t.Setenv("HP_FAKE_ARGS", argsPath)
	fakeProvider(t, "codex", "printf '%s\\n' \"$@\" > \"$HP_FAKE_ARGS\"\necho queued")
	code, out, stderr = run(t, "wakeup", "once", "--project", root)
	if code != 0 || stderr != "" {
		t.Fatalf("once: %d %s %s", code, out, stderr)
	}
	state, err := (wakeup.Registry{Root: root}).GetState()
	if err != nil || state.LastStatus != "accepted" || state.LastDeliveryID == "" {
		t.Fatalf("state: %+v %v", state, err)
	}
	args, err := os.ReadFile(argsPath)
	if err != nil || !strings.Contains(string(args), "--thread\ntarget-session\n") || !strings.Contains(string(args), "Read PROJECT.md") {
		t.Fatalf("exact target missing: %s %v", args, err)
	}
}

func TestWakeupUncertainDeliveryRequiresExplicitRetry(t *testing.T) {
	root := filepath.Join(t.TempDir(), "project")
	if code, _, err := run(t, "project", "init", "--path", root); code != 0 {
		t.Fatal(err)
	}
	if code, _, err := run(t, "agent", "register", "--project", root, "--harness", "codex", "--session-id", "target", "--worktree", root); code != 0 {
		t.Fatal(err)
	}
	if code, _, err := run(t, "coordinator", "assign", "--project", root, "--registration-id", presence.RegistrationID("codex", "target")); code != 0 {
		t.Fatal(err)
	}
	if code, _, err := run(t, "wakeup", "configure", "--project", root, "--every", "1h", "--worktree", ".", "--prompt-file", "agents/coordinator-wakeup.md"); code != 0 {
		t.Fatal(err)
	}
	called := filepath.Join(t.TempDir(), "called")
	t.Setenv("HP_FAKE_CALLED", called)
	fakeProvider(t, "codex", "echo x >> \"$HP_FAKE_CALLED\"\nexit 1")
	if code, _, _ := run(t, "wakeup", "once", "--project", root); code != 1 {
		t.Fatalf("expected uncertain, got %d", code)
	}
	if code, _, _ := run(t, "wakeup", "once", "--project", root); code == 0 {
		t.Fatal("resent uncertain wakeup")
	}
	if code, _, err := run(t, "wakeup", "configure", "--project", root, "--disable"); code != 0 {
		t.Fatal(err)
	}
	if code, _, _ := run(t, "wakeup", "once", "--project", root); code == 0 {
		t.Fatal("config edit bypassed uncertain guard")
	}
	data, err := os.ReadFile(called)
	if err != nil || strings.Count(string(data), "x") != 1 {
		t.Fatalf("unexpected call count: %q %v", data, err)
	}
	if code, _, _ := run(t, "wakeup", "once", "--project", root, "--retry"); code != 1 {
		t.Fatalf("explicit retry outcome: %d", code)
	}
}
