package participants

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/chenhg5/harness-projects/internal/model"
	"github.com/chenhg5/harness-projects/internal/presence"
)

const SchemaVersion = "1"

// Session records that a provider conversation has participated in a project.
// It is durable project history, not evidence that the provider is running now.
type Session struct {
	SchemaVersion string   `json:"schemaVersion"`
	ID            string   `json:"id"`
	Harness       string   `json:"harness"`
	SessionID     string   `json:"sessionId"`
	Focus         string   `json:"focus,omitempty"`
	Modules       []string `json:"modules,omitempty"`
	Capabilities  []string `json:"capabilities,omitempty"`
	Worktree      string   `json:"worktree,omitempty"` // project-relative; empty when outside the project
	TaskID        string   `json:"taskId,omitempty"`
	RegisteredAt  string   `json:"registeredAt"`
	UpdatedAt     string   `json:"updatedAt"`
}

type Registry struct{ Root string }

type Filter struct {
	Harness    string
	Module     string
	Capability string
}

func (r Registry) directory() string { return filepath.Join(r.Root, "agents", "sessions") }

func (r Registry) path(id string) (string, error) {
	if len(id) != 64 {
		return "", fmt.Errorf("invalid agent registration id %q", id)
	}
	if _, err := hex.DecodeString(id); err != nil {
		return "", fmt.Errorf("invalid agent registration id %q", id)
	}
	return filepath.Join(r.directory(), id+".json"), nil
}

func Validate(session Session) error {
	if session.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported agent registration schemaVersion %q", session.SchemaVersion)
	}
	harness, err := presence.CanonicalHarness(session.Harness)
	if err != nil || harness != session.Harness {
		return fmt.Errorf("registration harness must be canonical")
	}
	if err := model.ValidateProviderSessionID(session.SessionID); err != nil {
		return err
	}
	if session.ID != presence.RegistrationID(session.Harness, session.SessionID) {
		return fmt.Errorf("registration id does not match harness and session id")
	}
	if session.Worktree != "" && (filepath.IsAbs(session.Worktree) || filepath.Clean(session.Worktree) != session.Worktree || session.Worktree == ".." || strings.HasPrefix(session.Worktree, ".."+string(filepath.Separator))) {
		return fmt.Errorf("worktree must be a project-relative path")
	}
	if session.TaskID != "" {
		if err := model.ValidateID(session.TaskID); err != nil {
			return fmt.Errorf("invalid task id: %w", err)
		}
	}
	for _, list := range [][]string{session.Modules, session.Capabilities} {
		for _, item := range list {
			if strings.TrimSpace(item) == "" || len(item) > 120 {
				return fmt.Errorf("modules and capabilities must contain nonempty items of at most 120 bytes")
			}
		}
	}
	registered, err := time.Parse(time.RFC3339Nano, session.RegisteredAt)
	if err != nil {
		return fmt.Errorf("invalid registeredAt: %w", err)
	}
	updated, err := time.Parse(time.RFC3339Nano, session.UpdatedAt)
	if err != nil || updated.Before(registered) {
		return fmt.Errorf("invalid updatedAt")
	}
	return nil
}

func (r Registry) Get(id string) (Session, error) {
	path, err := r.path(id)
	if err != nil {
		return Session{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Session{}, err
	}
	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return Session{}, fmt.Errorf("invalid agent registration %s: %w", path, err)
	}
	if session.ID != id {
		return Session{}, fmt.Errorf("registration id mismatch in %s", path)
	}
	if err := Validate(session); err != nil {
		return Session{}, fmt.Errorf("invalid agent registration %s: %w", path, err)
	}
	return session, nil
}

func (r Registry) PrepareRegister(session Session, now time.Time) (Session, error) {
	harness, err := presence.CanonicalHarness(session.Harness)
	if err != nil {
		return Session{}, err
	}
	if err := model.ValidateProviderSessionID(session.SessionID); err != nil {
		return Session{}, err
	}
	session.Harness = harness
	session.ID = presence.RegistrationID(harness, session.SessionID)
	session.SchemaVersion = SchemaVersion
	session.RegisteredAt = now.UTC().Format(time.RFC3339Nano)
	if existing, err := r.Get(session.ID); err == nil {
		session.RegisteredAt = existing.RegisteredAt
		if session.Focus == "" {
			session.Focus = existing.Focus
		}
		if session.Modules == nil {
			session.Modules = existing.Modules
		}
		if session.Capabilities == nil {
			session.Capabilities = existing.Capabilities
		}
		if session.Worktree == "" {
			session.Worktree = existing.Worktree
		}
		if session.TaskID == "" {
			session.TaskID = existing.TaskID
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return Session{}, err
	}
	session.UpdatedAt = now.UTC().Format(time.RFC3339Nano)
	if err := Validate(session); err != nil {
		return Session{}, err
	}
	return session, nil
}

func (r Registry) Register(session Session, now time.Time) (Session, error) {
	prepared, err := r.PrepareRegister(session, now)
	if err != nil {
		return Session{}, err
	}
	if err := os.MkdirAll(r.directory(), 0755); err != nil {
		return Session{}, err
	}
	data, err := json.MarshalIndent(prepared, "", "  ")
	if err != nil {
		return Session{}, err
	}
	data = append(data, '\n')
	temp, err := os.CreateTemp(r.directory(), ".hp-participant-*")
	if err != nil {
		return Session{}, err
	}
	defer os.Remove(temp.Name())
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return Session{}, err
	}
	if err := temp.Close(); err != nil {
		return Session{}, err
	}
	path, _ := r.path(prepared.ID)
	if err := os.Rename(temp.Name(), path); err != nil {
		return Session{}, err
	}
	return prepared, nil
}

func (r Registry) Discover(filter Filter) ([]Session, error) {
	entries, err := os.ReadDir(r.directory())
	if errors.Is(err, os.ErrNotExist) {
		return []Session{}, nil
	}
	if err != nil {
		return nil, err
	}
	result := make([]Session, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("agent registration is a symlink: %s", entry.Name())
		}
		item, err := r.Get(strings.TrimSuffix(entry.Name(), ".json"))
		if err != nil {
			return nil, err
		}
		if filter.Harness != "" && item.Harness != filter.Harness || filter.Module != "" && !containsFold(item.Modules, filter.Module) || filter.Capability != "" && !containsFold(item.Capabilities, filter.Capability) {
			continue
		}
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

func ProjectRelativeWorktree(root, absolute string) string {
	relative, err := filepath.Rel(root, absolute)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return ""
	}
	return relative
}

func containsFold(items []string, sought string) bool {
	for _, item := range items {
		if strings.EqualFold(item, sought) {
			return true
		}
	}
	return false
}
