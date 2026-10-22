// Package hooks keeps the bookkeeping that harness lifecycle hooks need: what a
// session looked like when it started, and whether it already left a record.
package hooks

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/chenhg5/harness-projects/internal/git"
)

// Input is the subset of the hook payload that Claude Code and Codex send on stdin.
type Input struct {
	SessionID      string `json:"session_id"`
	Cwd            string `json:"cwd"`
	HookEventName  string `json:"hook_event_name"`
	Source         string `json:"source"`
	StopHookActive bool   `json:"stop_hook_active"`
}

// Marker records the start of a session on this machine. It is local state,
// never project history.
type Marker struct {
	Harness      string            `json:"harness"`
	SessionID    string            `json:"sessionId"`
	StartedAt    string            `json:"startedAt"`
	Fingerprints map[string]string `json:"fingerprints"`
	Reminded     bool              `json:"reminded,omitempty"`
}

func markerDir(root string) string {
	return filepath.Join(root, ".harness-project", "local", "hooks")
}

func markerPath(root, harness, sessionID string) string {
	sum := sha256.Sum256([]byte(harness + "\x00" + sessionID))
	return filepath.Join(markerDir(root), hex.EncodeToString(sum[:8])+".json")
}

// SaveMarker writes the marker and keeps the local directory out of Git.
func SaveMarker(root string, marker Marker) error {
	dir := markerDir(root)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	ignore := filepath.Join(root, ".harness-project", "local", ".gitignore")
	if _, err := os.Stat(ignore); errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(ignore, []byte("*\n"), 0600); err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(marker, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(markerPath(root, marker.Harness, marker.SessionID), append(data, '\n'), 0600)
}

func LoadMarker(root, harness, sessionID string) (Marker, error) {
	var marker Marker
	data, err := os.ReadFile(markerPath(root, harness, sessionID))
	if err != nil {
		return marker, err
	}
	if err := json.Unmarshal(data, &marker); err != nil {
		return marker, fmt.Errorf("invalid hook marker: %w", err)
	}
	return marker, nil
}

// Fingerprints summarizes the project repository, Git repositories under
// repos/, and registered repository paths.
func Fingerprints(ctx context.Context, root string, registered []string) map[string]string {
	dirs := map[string]bool{root: true}
	for _, path := range registered {
		if path == "" {
			continue
		}
		if !filepath.IsAbs(path) {
			path = filepath.Join(root, path)
		}
		dirs[filepath.Clean(path)] = true
	}
	reposRoot := filepath.Join(root, "repos")
	_ = filepath.WalkDir(reposRoot, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || !entry.IsDir() {
			return nil
		}
		if depth := strings.Count(strings.TrimPrefix(path, reposRoot), string(filepath.Separator)); depth > 3 {
			return filepath.SkipDir
		}
		if path != reposRoot && git.IsRepositoryRoot(path) {
			dirs[path] = true
			return filepath.SkipDir
		}
		return nil
	})
	result := map[string]string{}
	paths := make([]string, 0, len(dirs))
	for dir := range dirs {
		paths = append(paths, dir)
	}
	sort.Strings(paths)
	for _, dir := range paths {
		if fingerprint := git.Fingerprint(ctx, dir); fingerprint != "" {
			result[dir] = fingerprint
		}
	}
	return result
}

// Changed reports the directories whose fingerprint differs from the start.
func Changed(before, after map[string]string) []string {
	changed := []string{}
	for dir, now := range after {
		if before[dir] != now {
			changed = append(changed, dir)
		}
	}
	sort.Strings(changed)
	return changed
}

// RecordedSince reports whether a work log or handoff was written after start.
func RecordedSince(root string, start time.Time) bool {
	for _, section := range []string{"worklogs", "handoffs"} {
		entries, err := os.ReadDir(filepath.Join(root, section))
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || entry.Name() == "README.md" || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}
			if info, err := entry.Info(); err == nil && info.ModTime().After(start) {
				return true
			}
		}
	}
	return false
}

// HookCommand is the shell command a harness runs for one lifecycle event. It
// stays silent and successful when hp is not installed.
func HookCommand(event, harness string) string {
	return "command -v hp >/dev/null 2>&1 && hp hook " + event + " --harness " + harness + " || true"
}

// MergeClaudeSettings adds the hp SessionStart and Stop hooks to a Claude Code
// settings document, keeping every other setting and hook unchanged.
func MergeClaudeSettings(existing []byte) ([]byte, bool, error) {
	settings := map[string]any{}
	if len(strings.TrimSpace(string(existing))) > 0 {
		if err := json.Unmarshal(existing, &settings); err != nil {
			return nil, false, fmt.Errorf("invalid Claude Code settings JSON: %w", err)
		}
	}
	hooksValue, _ := settings["hooks"].(map[string]any)
	if hooksValue == nil {
		if settings["hooks"] != nil {
			return nil, false, fmt.Errorf("Claude Code settings field hooks is not an object")
		}
		hooksValue = map[string]any{}
	}
	changed := false
	for _, item := range []struct{ event, name string }{{"SessionStart", "session-start"}, {"Stop", "stop"}} {
		command := HookCommand(item.name, "claude-code")
		groups, _ := hooksValue[item.event].([]any)
		if containsCommand(groups, "hp hook "+item.name) {
			continue
		}
		groups = append(groups, map[string]any{"hooks": []any{map[string]any{"type": "command", "command": command, "timeout": 30}}})
		hooksValue[item.event] = groups
		changed = true
	}
	settings["hooks"] = hooksValue
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return nil, false, err
	}
	return append(data, '\n'), changed, nil
}

func containsCommand(groups []any, fragment string) bool {
	for _, group := range groups {
		object, _ := group.(map[string]any)
		entries, _ := object["hooks"].([]any)
		for _, entry := range entries {
			hook, _ := entry.(map[string]any)
			if command, _ := hook["command"].(string); strings.Contains(command, fragment) {
				return true
			}
		}
	}
	return false
}
