package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/chenhg5/harness-projects/internal/model"
)

func dispatchTestProject(t *testing.T, harness, sessionID, availability string) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "project")
	for _, args := range [][]string{
		{"project", "init", "--path", root},
		{"task", "create", "--project", root, "--id", "sample", "--title", "Sample delivery", "--description", "Make one verified change", "--acceptance", "evidence recorded"},
		{"agent", "register", "--project", root, "--harness", harness, "--session-id", sessionID, "--worktree", root, "--availability", availability},
	} {
		if code, _, stderr := run(t, args...); code != 0 {
			t.Fatalf("%v: code=%d stderr=%s", args, code, stderr)
		}
	}
	return root
}

func fakeProvider(t *testing.T, name, script string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell-script provider fixture requires Unix")
	}
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\n"+script+"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return bin
}

func dispatchCall(t *testing.T, root, harness, sessionID string, extra ...string) (int, model.Run, string) {
	t.Helper()
	args := []string{"coordinator", "dispatch", "--project", root, "--task-id", "sample", "--harness", harness, "--session-id", sessionID, "--worktree", root}
	args = append(args, extra...)
	code, stdout, stderr := run(t, args...)
	var result struct {
		Run model.Run `json:"run"`
	}
	if stdout != "" {
		if err := json.Unmarshal([]byte(stdout), &result); err != nil {
			t.Fatalf("invalid dispatch output: %s: %v", stdout, err)
		}
	}
	return code, result.Run, stderr
}

func TestCoordinatorDispatchCodexQueueAndRepeatGuard(t *testing.T) {
	root := dispatchTestProject(t, "codex", "codex-session", "busy")
	argsPath := filepath.Join(t.TempDir(), "args.txt")
	t.Setenv("HP_FAKE_ARGS", argsPath)
	fakeProvider(t, "codex", "printf '%s\\n' \"$@\" > \"$HP_FAKE_ARGS\"\necho queued")
	code, preview, stderr := dispatchCall(t, root, "codex", "codex-session", "--dry-run")
	if code != 10 || stderr != "" || preview.ID == "" {
		t.Fatalf("dry-run: code=%d run=%+v stderr=%s", code, preview, stderr)
	}
	if _, err := os.Stat(argsPath); !os.IsNotExist(err) {
		t.Fatal("dry-run invoked provider")
	}
	if _, err := os.Stat(filepath.Join(root, "runs", preview.ID+".json")); !os.IsNotExist(err) {
		t.Fatal("dry-run wrote a run")
	}
	code, taskJSON, _ := run(t, "task", "get", "--project", root, "--id", "sample")
	var previewTask model.Task
	if code != 0 || json.Unmarshal([]byte(taskJSON), &previewTask) != nil || previewTask.Assignment != nil {
		t.Fatalf("dry-run changed task assignment: %s", taskJSON)
	}
	code, result, stderr := dispatchCall(t, root, "codex", "codex-session", "--run-id", "delivery-1")
	if code != 0 || stderr != "" || result.DeliveryStatus != "accepted" || result.Status != "running" || result.Transport != "codex-queue" || result.EffectiveSessionID != "codex-session" {
		t.Fatalf("dispatch: code=%d run=%+v stderr=%s", code, result, stderr)
	}
	args, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, part := range []string{"queue\n", "--thread\ncodex-session\n", "--message\n", "Sample delivery"} {
		if !strings.Contains(string(args), part) {
			t.Fatalf("missing %q in provider args: %s", part, args)
		}
	}
	code, _, _ = dispatchCall(t, root, "codex", "codex-session", "--run-id", "delivery-2")
	if code != 5 {
		t.Fatalf("repeat should need --retry: code=%d", code)
	}
	if _, err := os.Stat(filepath.Join(root, "runs", "delivery-2.json")); !os.IsNotExist(err) {
		t.Fatal("repeat wrote a new run")
	}
}

func TestCoordinatorDispatchPreflightDowngradesWithoutProviderCall(t *testing.T) {
	root := dispatchTestProject(t, "claude-code", "claude-session", "busy")
	called := filepath.Join(t.TempDir(), "called")
	t.Setenv("HP_FAKE_CALLED", called)
	fakeProvider(t, "claude", "touch \"$HP_FAKE_CALLED\"")
	code, result, stderr := dispatchCall(t, root, "claude-code", "claude-session", "--run-id", "queued-1")
	if code != 0 || stderr != "" || result.DeliveryStatus != "queued" || !strings.Contains(result.DeliveryReason, "busy") {
		t.Fatalf("queued: code=%d run=%+v stderr=%s", code, result, stderr)
	}
	if _, err := os.Stat(called); !os.IsNotExist(err) {
		t.Fatal("busy session was resumed")
	}
	code, result, _ = dispatchCall(t, root, "claude-code", "claude-session", "--run-id", "queued-2", "--delivery", "direct")
	if code != 5 || result.DeliveryStatus != "queued" {
		t.Fatalf("strict direct: code=%d run=%+v", code, result)
	}
	code, taskJSON, _ := run(t, "task", "get", "--project", root, "--id", "sample")
	if code != 0 {
		t.Fatalf("task get: code=%d", code)
	}
	var task model.Task
	if err := json.Unmarshal([]byte(taskJSON), &task); err != nil {
		t.Fatal(err)
	}
	if task.Assignment == nil || task.Assignment.SessionID != "claude-session" {
		t.Fatalf("queued task lost assignment: %+v", task)
	}
}

func TestCoordinatorDispatchClaudeTerminalAndUncertain(t *testing.T) {
	root := dispatchTestProject(t, "claude-code", "claude-session", "available")
	argsPath := filepath.Join(t.TempDir(), "args.txt")
	t.Setenv("HP_FAKE_ARGS", argsPath)
	fakeProvider(t, "claude", "if [ \"$1\" = agents ]; then echo '[]'; exit 0; fi\nprintf '%s\\n' \"$@\" > \"$HP_FAKE_ARGS\"\necho '{\"type\":\"system\",\"session_id\":\"claude-session\"}'\necho '{\"type\":\"result\",\"session_id\":\"claude-session\",\"result\":\"done\",\"is_error\":false}'")
	code, result, stderr := dispatchCall(t, root, "claude-code", "claude-session", "--run-id", "claude-1")
	if code != 0 || stderr != "" || result.DeliveryStatus != "completed" || result.Status != "succeeded" || result.EffectiveSessionID != "claude-session" || result.Summary != "done" {
		t.Fatalf("Claude result: code=%d run=%+v stderr=%s", code, result, stderr)
	}
	args, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(args), "--resume\nclaude-session\n") || !strings.Contains(string(args), "--print\n") {
		t.Fatalf("wrong Claude args: %s", args)
	}
	code, taskJSON, _ := run(t, "task", "get", "--project", root, "--id", "sample")
	var task model.Task
	if code != 0 || json.Unmarshal([]byte(taskJSON), &task) != nil || task.Status != "pending" {
		t.Fatalf("provider exit marked task complete: %s", taskJSON)
	}
	// A separate task verifies an exit after process start is never silently queued.
	if code, _, stderr := run(t, "task", "create", "--project", root, "--id", "uncertain", "--title", "Uncertain delivery"); code != 0 {
		t.Fatalf("create: %s", stderr)
	}
	provider := filepath.Join(strings.Split(os.Getenv("PATH"), string(os.PathListSeparator))[0], "claude")
	if err := os.WriteFile(provider, []byte("#!/bin/sh\nif [ \"$1\" = agents ]; then echo '[]'; exit 0; fi\necho '{\"type\":\"system\",\"session_id\":\"claude-session\"}'\nexit 1\n"), 0700); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr := run(t, "coordinator", "dispatch", "--project", root, "--task-id", "uncertain", "--harness", "claude-code", "--session-id", "claude-session", "--worktree", root, "--run-id", "uncertain-1")
	var uncertain struct {
		Run model.Run `json:"run"`
	}
	if json.Unmarshal([]byte(stdout), &uncertain) != nil || code != 1 || uncertain.Run.DeliveryStatus != "uncertain" || stderr != "" {
		t.Fatalf("uncertain: code=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
	code, _, _ = run(t, "coordinator", "dispatch", "--project", root, "--task-id", "uncertain", "--harness", "claude-code", "--session-id", "claude-session", "--worktree", root, "--run-id", "uncertain-2")
	if code != 5 {
		t.Fatalf("uncertain delivery repeated without --retry: code=%d", code)
	}
}

func TestCoordinatorDispatchCursorExactSession(t *testing.T) {
	root := dispatchTestProject(t, "cursor", "cursor-session", "available")
	argsPath := filepath.Join(t.TempDir(), "args.txt")
	t.Setenv("HP_FAKE_ARGS", argsPath)
	fakeProvider(t, "cursor-agent", "printf '%s\\n' \"$@\" > \"$HP_FAKE_ARGS\"\necho '{\"type\":\"result\",\"session_id\":\"cursor-session\",\"result\":\"done\",\"is_error\":false}'")
	code, result, stderr := dispatchCall(t, root, "cursor", "cursor-session", "--run-id", "cursor-1")
	if code != 0 || stderr != "" || result.DeliveryStatus != "completed" || result.EffectiveSessionID != "cursor-session" {
		t.Fatalf("Cursor result: code=%d run=%+v stderr=%s", code, result, stderr)
	}
	args, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(args), "--resume=cursor-session\n") || !strings.Contains(string(args), "--print\n") {
		t.Fatalf("wrong Cursor args: %s", args)
	}
}

func TestCoordinatorDispatchClaudeActiveSessionStaysInInbox(t *testing.T) {
	root := dispatchTestProject(t, "claude-code", "claude-session", "available")
	called := filepath.Join(t.TempDir(), "called")
	t.Setenv("HP_FAKE_CALLED", called)
	fakeProvider(t, "claude", "if [ \"$1\" = agents ]; then echo '[{\"sessionId\":\"claude-session\",\"status\":\"idle\"}]'; exit 0; fi\ntouch \"$HP_FAKE_CALLED\"")
	code, result, stderr := dispatchCall(t, root, "claude-code", "claude-session", "--run-id", "active-1")
	if code != 0 || stderr != "" || result.DeliveryStatus != "queued" || !strings.Contains(result.DeliveryReason, "still active") {
		t.Fatalf("active session: code=%d run=%+v stderr=%s", code, result, stderr)
	}
	if _, err := os.Stat(called); !os.IsNotExist(err) {
		t.Fatal("active Claude session was resumed")
	}
}

func TestCoordinatorDispatchClaudeDryRunDoesNotProbeProvider(t *testing.T) {
	root := dispatchTestProject(t, "claude-code", "claude-session", "available")
	called := filepath.Join(t.TempDir(), "called")
	t.Setenv("HP_FAKE_CALLED", called)
	fakeProvider(t, "claude", "touch \"$HP_FAKE_CALLED\"")
	code, _, stderr := dispatchCall(t, root, "claude-code", "claude-session", "--dry-run")
	if code != 10 || stderr != "" {
		t.Fatalf("dry-run: code=%d stderr=%s", code, stderr)
	}
	if _, err := os.Stat(called); !os.IsNotExist(err) {
		t.Fatal("dry-run invoked Claude CLI")
	}
}

func TestCoordinatorDispatchMissingAdapterFallsBack(t *testing.T) {
	root := dispatchTestProject(t, "cursor", "cursor-session", "available")
	t.Setenv("PATH", t.TempDir())
	code, result, stderr := dispatchCall(t, root, "cursor", "cursor-session", "--run-id", "missing-cli")
	if code != 0 || stderr != "" || result.DeliveryStatus != "queued" || !strings.Contains(result.DeliveryReason, "unavailable") {
		t.Fatalf("missing adapter: code=%d run=%+v stderr=%s", code, result, stderr)
	}
}

func TestCoordinatorDispatchNoTerminalResultIsUncertain(t *testing.T) {
	root := dispatchTestProject(t, "cursor", "cursor-session", "available")
	fakeProvider(t, "cursor-agent", "echo '{\"type\":\"system\",\"session_id\":\"cursor-session\"}'")
	code, result, stderr := dispatchCall(t, root, "cursor", "cursor-session", "--run-id", "no-result")
	if code != 1 || stderr != "" || result.DeliveryStatus != "uncertain" || result.Status != "paused" {
		t.Fatalf("missing terminal result: code=%d run=%+v stderr=%s", code, result, stderr)
	}
	code, _, _ = dispatchCall(t, root, "cursor", "cursor-session", "--run-id", "no-result-2")
	if code != 5 {
		t.Fatalf("uncertain delivery repeated without --retry: code=%d", code)
	}
}
