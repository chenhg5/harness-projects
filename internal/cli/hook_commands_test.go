package cli

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func newHookProject(t *testing.T) string {
	t.Helper()
	t.Setenv("GIT_AUTHOR_NAME", "hp test")
	t.Setenv("GIT_AUTHOR_EMAIL", "hp@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "hp test")
	t.Setenv("GIT_COMMITTER_EMAIL", "hp@example.com")
	t.Setenv("HP_SESSION_ID", "")
	t.Setenv("CODEX_THREAD_ID", "")
	root := filepath.Join(t.TempDir(), "demo")
	if code, _, stderr := run(t, "project", "init", "--path", root, "--name", "Demo", "--json"); code != 0 {
		t.Fatalf("init code=%d stderr=%s", code, stderr)
	}
	return root
}

func TestProjectInitCreatesGitRepositoryAndHooks(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	root := newHookProject(t)
	out, err := exec.Command("git", "-C", root, "log", "--oneline").Output()
	if err != nil || !strings.Contains(string(out), "Create Harness Project") {
		t.Fatalf("expected an initial commit, got %q err=%v", out, err)
	}
	if status, _ := exec.Command("git", "-C", root, "status", "--porcelain").Output(); len(status) != 0 {
		t.Fatalf("new project should be clean, got %s", status)
	}
	settings, err := os.ReadFile(filepath.Join(root, ".claude", "settings.json"))
	if err != nil || !strings.Contains(string(settings), "hp hook session-start --harness claude-code") || !strings.Contains(string(settings), "hp hook stop --harness claude-code") {
		t.Fatalf("expected Claude Code hooks, got %s err=%v", settings, err)
	}
	agents, _ := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if !strings.Contains(string(agents), "Run `hp brief`") {
		t.Fatalf("AGENTS.md should start sessions with hp brief:\n%s", agents)
	}

	other := filepath.Join(t.TempDir(), "plain")
	if code, stdout, _ := run(t, "project", "init", "--path", other, "--no-git", "--json"); code != 0 || !strings.Contains(stdout, `"git": "skipped"`) {
		t.Fatalf("--no-git should skip Git: code=%d %s", code, stdout)
	}
	if _, err := os.Stat(filepath.Join(other, ".git")); !os.IsNotExist(err) {
		t.Fatalf("--no-git created a repository")
	}
}

func TestSessionStartHookRegistersAndBriefs(t *testing.T) {
	root := newHookProject(t)
	envFile := filepath.Join(t.TempDir(), "claude.env")
	t.Setenv("CLAUDE_ENV_FILE", envFile)
	input := `{"session_id":"abc-123","cwd":"` + root + `","hook_event_name":"SessionStart","source":"startup"}`
	code, stdout, stderr := runInput(t, input, "hook", "session-start", "--harness", "claude-code")
	if code != 0 || stderr != "" {
		t.Fatalf("hook code=%d stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, "# Project brief: Demo") || !strings.Contains(stdout, "claude-code:abc-123") {
		t.Fatalf("expected brief and registration note:\n%s", stdout)
	}
	env, _ := os.ReadFile(envFile)
	if !strings.Contains(string(env), "export HP_SESSION_ID='abc-123'") || !strings.Contains(string(env), "export HP_HARNESS='claude-code'") {
		t.Fatalf("expected exported session variables, got %q", env)
	}
	if code, stdout, _ := run(t, "agent", "discover", "--project", root, "--json"); code != 0 || !strings.Contains(stdout, "abc-123") {
		t.Fatalf("session should be discoverable: code=%d %s", code, stdout)
	}

	outside := t.TempDir()
	code, stdout, stderr = runInput(t, `{"session_id":"x","cwd":"`+outside+`"}`, "hook", "session-start", "--harness", "claude-code")
	if code != 0 || stdout != "" || stderr != "" {
		t.Fatalf("hook outside a project should be silent: code=%d out=%q err=%q", code, stdout, stderr)
	}
	if code, _, _ := runInput(t, "not json", "hook", "session-start", "--harness", "claude-code", "--project", root); code != 0 {
		t.Fatalf("invalid input must not fail the session, code=%d", code)
	}
}

func TestStopHookAsksOnceAfterChanges(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	root := newHookProject(t)
	t.Setenv("CLAUDE_ENV_FILE", "")
	start := `{"session_id":"s1","cwd":"` + root + `","source":"startup"}`
	stop := `{"session_id":"s1","cwd":"` + root + `","stop_hook_active":false}`
	if code, _, stderr := runInput(t, start, "hook", "session-start", "--harness", "claude-code"); code != 0 {
		t.Fatalf("start code=%d stderr=%s", code, stderr)
	}
	// Registration itself wrote agents/sessions; commit it so only real work counts.
	if out, err := exec.Command("git", "-C", root, "add", "-A").CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	if out, err := exec.Command("git", "-C", root, "commit", "-qm", "register").CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	if code, _, stderr := runInput(t, start, "hook", "session-start", "--harness", "claude-code"); code != 0 {
		t.Fatalf("restart code=%d stderr=%s", code, stderr)
	}
	if code, stdout, _ := runInput(t, stop, "hook", "stop", "--harness", "claude-code"); code != 0 || stdout != "" {
		t.Fatalf("no changes should not block: code=%d %s", code, stdout)
	}
	if err := os.WriteFile(filepath.Join(root, "spec", "api.md"), []byte("# API\n"), 0644); err != nil {
		t.Fatal(err)
	}
	code, stdout, _ := runInput(t, stop, "hook", "stop", "--harness", "claude-code")
	var decision map[string]string
	if code != 0 || json.Unmarshal([]byte(stdout), &decision) != nil || decision["decision"] != "block" || !strings.Contains(decision["reason"], "hp finish") {
		t.Fatalf("changes without a work log should block once: code=%d %s", code, stdout)
	}
	if code, stdout, _ := runInput(t, stop, "hook", "stop", "--harness", "claude-code"); code != 0 || stdout != "" {
		t.Fatalf("second stop must not block again: %s", stdout)
	}
}

func TestStopHookAcceptsRecordedWork(t *testing.T) {
	root := newHookProject(t)
	t.Setenv("CLAUDE_ENV_FILE", "")
	start := `{"session_id":"s2","cwd":"` + root + `","source":"startup"}`
	if code, _, _ := runInput(t, start, "hook", "session-start", "--harness", "claude-code"); code != 0 {
		t.Fatal("start failed")
	}
	if err := os.WriteFile(filepath.Join(root, "spec", "api.md"), []byte("# API\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if code, _, stderr := run(t, "worklog", "create", "--project", root, "--id", "api", "--agent", "claude", "--objective", "o", "--completed", "c", "--remaining", "r", "--next", "n"); code != 0 {
		t.Fatalf("worklog: %s", stderr)
	}
	if code, stdout, _ := runInput(t, `{"session_id":"s2","cwd":"`+root+`"}`, "hook", "stop", "--harness", "claude-code"); code != 0 || stdout != "" {
		t.Fatalf("recorded work should not block: %s", stdout)
	}
}

func TestHookInstallMergesSettings(t *testing.T) {
	root := filepath.Join(t.TempDir(), "demo")
	if code, _, _ := run(t, "project", "init", "--path", root, "--no-git"); code != 0 {
		t.Fatal("init failed")
	}
	path := filepath.Join(root, ".claude", "settings.json")
	existing := `{"permissions":{"allow":["Bash(go test:*)"]},"hooks":{"Stop":[{"hooks":[{"type":"command","command":"echo done"}]}]}}`
	if err := os.WriteFile(path, []byte(existing), 0644); err != nil {
		t.Fatal(err)
	}
	if code, stdout, stderr := run(t, "hook", "install", "--project", root, "--json"); code != 0 || !strings.Contains(stdout, `"changed": true`) {
		t.Fatalf("install code=%d %s %s", code, stdout, stderr)
	}
	data, _ := os.ReadFile(path)
	for _, want := range []string{"Bash(go test:*)", "echo done", "hp hook stop", "hp hook session-start"} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("merged settings lost %q:\n%s", want, data)
		}
	}
	if code, stdout, _ := run(t, "hook", "install", "--project", root, "--json"); code != 0 || !strings.Contains(stdout, `"changed": false`) {
		t.Fatalf("second install should be a no-op: %s", stdout)
	}
}

func TestRegisterDetectsSession(t *testing.T) {
	root := newHookProject(t)
	t.Setenv("CODEX_THREAD_ID", "thread-42")
	code, stdout, stderr := run(t, "agent", "register", "--project", root, "--focus", "api", "--json")
	if code != 0 || !strings.Contains(stdout, `"harness": "codex"`) || !strings.Contains(stdout, "thread-42") {
		t.Fatalf("expected Codex detection: code=%d %s %s", code, stdout, stderr)
	}
	t.Setenv("HP_HARNESS", "claude-code")
	t.Setenv("HP_SESSION_ID", "claude-7")
	if code, stdout, _ := run(t, "task", "inbox", "--project", root, "--json"); code != 0 {
		t.Fatalf("inbox should use detected session: code=%d %s", code, stdout)
	}
	if code, _, _ := run(t, "agent", "register", "--project", root, "--harness", "cursor", "--json"); code != 2 {
		t.Fatalf("a different explicit harness must not borrow the detected session, code=%d", code)
	}
	t.Setenv("HP_SESSION_ID", "")
	t.Setenv("CODEX_THREAD_ID", "")
	if code, _, stderr := run(t, "agent", "register", "--project", root); code != 2 || !strings.Contains(stderr, "could not detect") {
		t.Fatalf("missing session should explain detection: code=%d %s", code, stderr)
	}
}
