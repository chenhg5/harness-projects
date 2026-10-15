package participants

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRegistrationPersistsAndUpdatesWithoutPresence(t *testing.T) {
	root := t.TempDir()
	registry := Registry{Root: root}
	firstTime := time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)
	first, err := registry.Register(Session{Harness: "cc", SessionID: "session-one", Focus: "API", Modules: []string{"api"}, Worktree: "repos/api"}, firstTime)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "agents", "sessions", first.ID+".json")); err != nil {
		t.Fatalf("durable registration missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "agents", "sessions", ".gitignore")); !os.IsNotExist(err) {
		t.Fatalf("participant records should be trackable by project Git: %v", err)
	}
	// Discovery has no lease clock. Participating once remains discoverable.
	items, err := registry.Discover(Filter{Harness: "claude-code", Module: "api"})
	if err != nil || len(items) != 1 || items[0].ID != first.ID {
		t.Fatalf("participant missing from discovery: items=%+v err=%v", items, err)
	}
	later := firstTime.Add(30 * 24 * time.Hour)
	updated, err := registry.Register(Session{Harness: "claude-code", SessionID: "session-one", Focus: "Review"}, later)
	if err != nil {
		t.Fatal(err)
	}
	if updated.RegisteredAt != first.RegisteredAt || updated.UpdatedAt == first.UpdatedAt || updated.Modules[0] != "api" || updated.Focus != "Review" {
		t.Fatalf("upsert lost original participation or metadata: first=%+v updated=%+v", first, updated)
	}
}

func TestProjectRelativeWorktree(t *testing.T) {
	root := t.TempDir()
	if got := ProjectRelativeWorktree(root, filepath.Join(root, "repos", "a")); got != filepath.Join("repos", "a") {
		t.Fatalf("inside project: %q", got)
	}
	if got := ProjectRelativeWorktree(root, filepath.Dir(root)); got != "" {
		t.Fatalf("outside project path must not be persisted: %q", got)
	}
}
