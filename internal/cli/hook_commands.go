package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chenhg5/harness-projects/internal/brief"
	"github.com/chenhg5/harness-projects/internal/hooks"
	"github.com/chenhg5/harness-projects/internal/model"
	"github.com/chenhg5/harness-projects/internal/presence"
	"github.com/chenhg5/harness-projects/internal/project"
	"github.com/chenhg5/harness-projects/internal/store"
)

// Hook commands run inside a harness lifecycle. They must never break the
// session, so every failure is reported on stderr and exits 0.

func (a app) readHookInput() (hooks.Input, error) {
	var input hooks.Input
	data, err := io.ReadAll(io.LimitReader(a.in, 1<<20))
	if err != nil {
		return input, err
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return input, nil
	}
	if err := json.Unmarshal(data, &input); err != nil {
		return input, fmt.Errorf("invalid hook input: %w", err)
	}
	return input, nil
}

func (a app) hookProject(flagValue string, input hooks.Input) (string, bool) {
	start := flagValue
	if start == "" {
		start = input.Cwd
	}
	if start == "" {
		start = "."
	}
	root, err := project.Find(start)
	return root, err == nil
}

func (a app) hookWarn(format string, args ...any) int {
	fmt.Fprintf(a.err, "hp hook: "+format+"\n", args...)
	return 0
}

func (a app) hookSessionStart(args []string) int {
	fs := newFlags("hook session-start")
	projectPath := fs.String("project", "", "project directory (default: the hook cwd)")
	harnessFlag := fs.String("harness", "", "claude-code|codex|cursor|other (required)")
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	harness, err := presence.CanonicalHarness(*harnessFlag)
	if err != nil || *harnessFlag == "" {
		return a.parseError(fmt.Errorf("--harness is required"))
	}
	input, err := a.readHookInput()
	if err != nil {
		return a.hookWarn("%v", err)
	}
	root, ok := a.hookProject(*projectPath, input)
	if !ok {
		return 0
	}
	registered := ""
	if model.ValidateProviderSessionID(input.SessionID) == nil {
		quiet := app{in: bytes.NewReader(nil), out: io.Discard, err: io.Discard}
		worktree := input.Cwd
		if worktree == "" {
			worktree = root
		}
		if code := quiet.agentRegister([]string{"--project", root, "--harness", harness, "--session-id", input.SessionID, "--worktree", worktree}); code == 0 {
			registered = harness + ":" + input.SessionID
		} else {
			a.hookWarn("could not register session (exit %d)", code)
		}
		if envFile := os.Getenv("CLAUDE_ENV_FILE"); envFile != "" {
			if err := appendEnv(envFile, harness, input.SessionID); err != nil {
				a.hookWarn("could not export session variables: %v", err)
			}
		}
		s, err := store.Open(root)
		if err == nil {
			marker := hooks.Marker{Harness: harness, SessionID: input.SessionID, StartedAt: time.Now().UTC().Format(time.RFC3339Nano)}
			if existing, err := hooks.LoadMarker(root, harness, input.SessionID); err == nil && input.Source != "startup" {
				marker = existing // resume, clear, or compact keep the original baseline
			} else {
				marker.Fingerprints = hooks.Fingerprints(context.Background(), root, repositoryPaths(s))
			}
			if err := hooks.SaveMarker(root, marker); err != nil {
				a.hookWarn("could not save session marker: %v", err)
			}
		}
	}
	result, err := brief.Load(root, brief.DefaultLines)
	if err != nil {
		return a.hookWarn("could not build brief: %v", err)
	}
	fmt.Fprint(a.out, result.Markdown())
	if registered != "" {
		fmt.Fprintf(a.out, "\nThis session is registered as `%s`; `hp` commands detect it automatically.\n", registered)
	}
	fmt.Fprintln(a.out, "Before you stop after meaningful work, record it with `hp worklog create` (or `hp handoff create` when someone else continues).")
	return 0
}

func (a app) hookStop(args []string) int {
	fs := newFlags("hook stop")
	projectPath := fs.String("project", "", "project directory (default: the hook cwd)")
	harnessFlag := fs.String("harness", "", "claude-code|codex|cursor|other (required)")
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	harness, err := presence.CanonicalHarness(*harnessFlag)
	if err != nil || *harnessFlag == "" {
		return a.parseError(fmt.Errorf("--harness is required"))
	}
	input, err := a.readHookInput()
	if err != nil {
		return a.hookWarn("%v", err)
	}
	if input.StopHookActive || input.SessionID == "" {
		return 0
	}
	root, ok := a.hookProject(*projectPath, input)
	if !ok {
		return 0
	}
	marker, err := hooks.LoadMarker(root, harness, input.SessionID)
	if err != nil || marker.Reminded {
		return 0
	}
	started, err := time.Parse(time.RFC3339Nano, marker.StartedAt)
	if err != nil || hooks.RecordedSince(root, started) {
		return 0
	}
	s, err := store.Open(root)
	if err != nil {
		return 0
	}
	changed := hooks.Changed(marker.Fingerprints, hooks.Fingerprints(context.Background(), root, repositoryPaths(s)))
	if len(changed) == 0 {
		return 0
	}
	marker.Reminded = true
	if err := hooks.SaveMarker(root, marker); err != nil {
		return a.hookWarn("could not save session marker: %v", err)
	}
	for i, dir := range changed {
		if relative, err := filepath.Rel(root, dir); err == nil && !strings.HasPrefix(relative, "..") {
			changed[i] = relative
		}
	}
	reason := fmt.Sprintf("This session changed files (%s) but has not recorded progress in the Harness Project at %s. "+
		"Run `hp worklog create --project %s --id <slug> --agent %s --objective ... --completed ... --remaining ... --next ...` "+
		"(and `hp task update` for task status), then stop. If the work is trivial, reply briefly and stop; you will not be asked again this session.",
		strings.Join(changed, ", "), root, root, harness)
	_ = json.NewEncoder(a.out).Encode(map[string]string{"decision": "block", "reason": reason})
	return 0
}

func (a app) hookInstall(args []string) int {
	fs := newFlags("hook install")
	projectPath := fs.String("project", ".", "project directory")
	format, jsonFlag := outputFlags(fs)
	dryRun := fs.Bool("dry-run", false, "preview without writing")
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	root, err := project.Find(*projectPath)
	if err != nil {
		return a.fail(failure{3, "not_found", err.Error(), "Run inside a Harness Project or pass --project", *projectPath})
	}
	path := filepath.Join(root, ".claude", "settings.json")
	existing, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return a.operationError(err, path)
	}
	data, changed, err := hooks.MergeClaudeSettings(existing)
	if err != nil {
		return a.operationError(err, path)
	}
	result := map[string]any{"action": "hook.install", "path": path, "changed": changed}
	if *dryRun {
		a.emit(result, "Would update "+path, "json")
		return 10
	}
	if changed {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return a.operationError(err, path)
		}
		if err := os.WriteFile(path, data, 0644); err != nil {
			return a.operationError(err, path)
		}
	}
	message := "Claude Code hooks already installed in " + path
	if changed {
		message = "Installed Claude Code hooks in " + path
	}
	a.emit(result, message, selectedFormat(*format, *jsonFlag, a.out))
	return 0
}

func appendEnv(path, harness, sessionID string) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(file, "export HP_HARNESS=%s\nexport HP_SESSION_ID=%s\n", shellQuote(harness), shellQuote(sessionID))
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	return err
}

func repositoryPaths(s store.Store) []string {
	repositories, err := s.Repositories()
	if err != nil {
		return nil
	}
	paths := make([]string, 0, len(repositories))
	for _, repository := range repositories {
		paths = append(paths, repository.Path)
	}
	return paths
}

const sessionHint = "Pass --harness and --session-id, set HP_HARNESS and HP_SESSION_ID, or install the session hooks with hp hook install (Claude Code). Codex exposes CODEX_THREAD_ID automatically."

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}
