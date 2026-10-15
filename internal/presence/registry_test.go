package presence

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRegisterDiscoverExpireAndRevive(t *testing.T) {
	root := t.TempDir()
	r := Registry{Root: root}
	now := time.Date(2026, 10, 15, 8, 0, 0, 0, time.UTC)
	registered, err := r.Register(Registration{
		Harness: "cc", SessionID: "session-1", Modules: []string{"API"},
		Capabilities: []string{"Go", "tests"}, Focus: "Implement API",
	}, time.Hour, now)
	if err != nil {
		t.Fatal(err)
	}
	if registered.Harness != "claude-code" || registered.Availability != "busy" || !registered.Live {
		t.Fatalf("unexpected registration: %#v", registered)
	}
	reregistered, err := r.Register(Registration{Harness: "cc", SessionID: "session-1"}, time.Hour, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if reregistered.RegisteredAt != registered.RegisteredAt || reregistered.Focus != registered.Focus ||
		len(reregistered.Modules) != 1 || reregistered.Worktree != registered.Worktree {
		t.Fatalf("re-register lost session metadata: %#v", reregistered)
	}
	if _, err := os.Stat(filepath.Join(root, ".harness-project", "agent-sessions", ".gitignore")); err != nil {
		t.Fatalf("missing machine-local ignore file: %v", err)
	}
	items, err := r.Discover(Filter{Harness: "claude-code", Module: "api", Capability: "go"}, now.Add(30*time.Minute))
	if err != nil || len(items) != 1 || items[0].ID != registered.ID {
		t.Fatalf("unexpected discovery: items=%#v err=%v", items, err)
	}
	items, err = r.Discover(Filter{}, now.Add(time.Hour+time.Minute))
	if err != nil || len(items) != 0 {
		t.Fatalf("expired registration should be hidden: items=%#v err=%v", items, err)
	}
	items, err = r.Discover(Filter{IncludeStale: true}, now.Add(time.Hour+time.Minute))
	if err != nil || len(items) != 1 || items[0].Live {
		t.Fatalf("expired registration should be inspectable: items=%#v err=%v", items, err)
	}
	renewed, err := r.Heartbeat(registered.ID, "available", "Review API", time.Hour, now.Add(2*time.Hour))
	if err != nil || !renewed.Live || renewed.Availability != "available" || renewed.Focus != "Review API" {
		t.Fatalf("unexpected renewal: item=%#v err=%v", renewed, err)
	}
	if renewed.RegisteredAt != registered.RegisteredAt {
		t.Fatalf("renewal changed registeredAt: %s -> %s", registered.RegisteredAt, renewed.RegisteredAt)
	}
	if err := r.Unregister(registered.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Get(registered.ID, now.Add(2*time.Hour)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected removed registration, got %v", err)
	}
}

func TestRegisterRejectsInvalidSessionAndTTL(t *testing.T) {
	r := Registry{Root: t.TempDir()}
	now := time.Now()
	for _, sessionID := range []string{"", " leading", "line\nbreak"} {
		if _, err := r.Register(Registration{Harness: "codex", SessionID: sessionID}, time.Hour, now); err == nil {
			t.Fatalf("accepted invalid session id %q", sessionID)
		}
	}
	if _, err := r.Register(Registration{Harness: "codex", SessionID: "real-id"}, 0, now); err == nil {
		t.Fatal("accepted zero TTL")
	}
	if _, err := os.Stat(r.directory()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("invalid registrations wrote state: %v", err)
	}
}

func TestPruneOnlyLongExpiredRegistrations(t *testing.T) {
	r := Registry{Root: t.TempDir()}
	now := time.Date(2026, 10, 15, 8, 0, 0, 0, time.UTC)
	old, err := r.Register(Registration{Harness: "codex", SessionID: "old-session"}, time.Hour, now)
	if err != nil {
		t.Fatal(err)
	}
	current, err := r.Register(Registration{Harness: "cursor", SessionID: "current-session"}, time.Hour, now.Add(25*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	candidates, err := r.PruneCandidates(24*time.Hour, now.Add(26*time.Hour))
	if err != nil || len(candidates) != 1 || candidates[0].ID != old.ID {
		t.Fatalf("unexpected prune candidates: %#v, %v", candidates, err)
	}
	removed, err := r.Prune(24*time.Hour, now.Add(26*time.Hour))
	if err != nil || len(removed) != 1 || removed[0] != old.ID {
		t.Fatalf("unexpected prune result: %#v, %v", removed, err)
	}
	if _, err := r.Get(current.ID, now.Add(26*time.Hour)); err != nil {
		t.Fatalf("prune removed current registration: %v", err)
	}
}
