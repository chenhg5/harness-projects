package presence

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/chenhg5/harness-projects/internal/model"
)

const SchemaVersion = "1"

const DefaultTTL = 2 * time.Hour

type Registration struct {
	SchemaVersion string   `json:"schemaVersion"`
	ID            string   `json:"id"`
	Harness       string   `json:"harness"`
	SessionID     string   `json:"sessionId"`
	Availability  string   `json:"availability"`
	Focus         string   `json:"focus,omitempty"`
	Modules       []string `json:"modules,omitempty"`
	Capabilities  []string `json:"capabilities,omitempty"`
	Worktree      string   `json:"worktree"`
	TaskID        string   `json:"taskId,omitempty"`
	RegisteredAt  string   `json:"registeredAt"`
	LastSeenAt    string   `json:"lastSeenAt"`
	ExpiresAt     string   `json:"expiresAt"`
}

type View struct {
	Registration
	Live bool `json:"live"`
}

type Registry struct{ Root string }

type Filter struct {
	Harness      string
	Module       string
	Capability   string
	IncludeStale bool
}

func CanonicalHarness(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "cc", "claude", "claudecode":
		value = "claude-code"
	}
	if err := model.ValidateID(value); err != nil {
		return "", fmt.Errorf("invalid harness: %w", err)
	}
	return value, nil
}

func RegistrationID(harness, sessionID string) string {
	sum := sha256.Sum256([]byte(harness + "\x00" + sessionID))
	return hex.EncodeToString(sum[:])
}

func Validate(reg Registration, now time.Time) error {
	if reg.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported registration schemaVersion %q", reg.SchemaVersion)
	}
	harness, err := CanonicalHarness(reg.Harness)
	if err != nil || harness != reg.Harness {
		return fmt.Errorf("registration harness must be canonical")
	}
	if err := validateSessionID(reg.SessionID); err != nil {
		return err
	}
	if reg.ID != RegistrationID(reg.Harness, reg.SessionID) {
		return fmt.Errorf("registration id does not match harness and session id")
	}
	if !validAvailability(reg.Availability) {
		return fmt.Errorf("invalid availability %q", reg.Availability)
	}
	if !filepath.IsAbs(reg.Worktree) {
		return fmt.Errorf("worktree must be an absolute path")
	}
	if reg.TaskID != "" {
		if err := model.ValidateID(reg.TaskID); err != nil {
			return fmt.Errorf("invalid task id: %w", err)
		}
	}
	for _, list := range [][]string{reg.Modules, reg.Capabilities} {
		for _, item := range list {
			if strings.TrimSpace(item) == "" || len(item) > 120 {
				return fmt.Errorf("modules and capabilities must contain nonempty items of at most 120 bytes")
			}
		}
	}
	registered, err := time.Parse(time.RFC3339Nano, reg.RegisteredAt)
	if err != nil {
		return fmt.Errorf("invalid registeredAt: %w", err)
	}
	seen, err := time.Parse(time.RFC3339Nano, reg.LastSeenAt)
	if err != nil {
		return fmt.Errorf("invalid lastSeenAt: %w", err)
	}
	expires, err := time.Parse(time.RFC3339Nano, reg.ExpiresAt)
	if err != nil {
		return fmt.Errorf("invalid expiresAt: %w", err)
	}
	if seen.Before(registered) || !expires.After(seen) || seen.After(now.Add(time.Minute)) {
		return fmt.Errorf("invalid registration timestamps")
	}
	return nil
}

func validateSessionID(id string) error {
	if id == "" || strings.TrimSpace(id) != id || len(id) > 512 {
		return fmt.Errorf("session id must be nonempty, trimmed, and at most 512 bytes")
	}
	for _, r := range id {
		if unicode.IsControl(r) {
			return fmt.Errorf("session id contains a control character")
		}
	}
	return nil
}

func validAvailability(value string) bool {
	return value == "available" || value == "busy" || value == "paused"
}

func ValidateTTL(ttl time.Duration) error {
	if ttl < time.Minute || ttl > 7*24*time.Hour {
		return fmt.Errorf("ttl must be between 1m and 168h")
	}
	return nil
}

func (r Registry) directory() string {
	return filepath.Join(r.Root, ".harness-project", "agent-sessions")
}

func (r Registry) path(id string) (string, error) {
	if len(id) != 64 {
		return "", fmt.Errorf("invalid registration id %q", id)
	}
	if _, err := hex.DecodeString(id); err != nil {
		return "", fmt.Errorf("invalid registration id %q", id)
	}
	return filepath.Join(r.directory(), id+".json"), nil
}

func (r Registry) Get(id string, now time.Time) (View, error) {
	path, err := r.path(id)
	if err != nil {
		return View{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return View{}, err
	}
	var reg Registration
	if err := json.Unmarshal(data, &reg); err != nil {
		return View{}, fmt.Errorf("invalid registration %s: %w", path, err)
	}
	if reg.ID != id {
		return View{}, fmt.Errorf("registration id mismatch in %s", path)
	}
	if err := Validate(reg, now); err != nil {
		return View{}, fmt.Errorf("invalid registration %s: %w", path, err)
	}
	return view(reg, now), nil
}

func view(reg Registration, now time.Time) View {
	expires, _ := time.Parse(time.RFC3339Nano, reg.ExpiresAt)
	return View{Registration: reg, Live: now.Before(expires)}
}

func (r Registry) Register(reg Registration, ttl time.Duration, now time.Time) (View, error) {
	prepared, err := r.PrepareRegister(reg, ttl, now)
	if err != nil {
		return View{}, err
	}
	if err := r.save(prepared.Registration); err != nil {
		return View{}, err
	}
	return prepared, nil
}

func (r Registry) PrepareRegister(reg Registration, ttl time.Duration, now time.Time) (View, error) {
	if err := ValidateTTL(ttl); err != nil {
		return View{}, err
	}
	harness, err := CanonicalHarness(reg.Harness)
	if err != nil {
		return View{}, err
	}
	if err := validateSessionID(reg.SessionID); err != nil {
		return View{}, err
	}
	reg.Harness = harness
	reg.ID = RegistrationID(harness, reg.SessionID)
	reg.SchemaVersion = SchemaVersion
	reg.RegisteredAt = now.UTC().Format(time.RFC3339Nano)
	if existing, err := r.Get(reg.ID, now); err == nil {
		reg.RegisteredAt = existing.RegisteredAt
		if reg.Availability == "" {
			reg.Availability = existing.Availability
		}
		if reg.Focus == "" {
			reg.Focus = existing.Focus
		}
		if reg.Modules == nil {
			reg.Modules = existing.Modules
		}
		if reg.Capabilities == nil {
			reg.Capabilities = existing.Capabilities
		}
		if reg.Worktree == "" {
			reg.Worktree = existing.Worktree
		}
		if reg.TaskID == "" {
			reg.TaskID = existing.TaskID
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return View{}, err
	}
	if reg.Availability == "" {
		reg.Availability = "busy"
	}
	if reg.Worktree == "" {
		reg.Worktree = r.Root
	}
	if !filepath.IsAbs(reg.Worktree) {
		reg.Worktree, err = filepath.Abs(reg.Worktree)
		if err != nil {
			return View{}, err
		}
	}
	if canonical, err := filepath.EvalSymlinks(reg.Worktree); err == nil {
		reg.Worktree = canonical
	}
	if info, err := os.Stat(reg.Worktree); err != nil || !info.IsDir() {
		return View{}, fmt.Errorf("worktree is not a directory: %s", reg.Worktree)
	}
	reg.LastSeenAt = now.UTC().Format(time.RFC3339Nano)
	reg.ExpiresAt = now.Add(ttl).UTC().Format(time.RFC3339Nano)
	if err := Validate(reg, now); err != nil {
		return View{}, err
	}
	return view(reg, now), nil
}

func (r Registry) Heartbeat(id, availability, focus string, ttl time.Duration, now time.Time) (View, error) {
	prepared, err := r.PrepareHeartbeat(id, availability, focus, ttl, now)
	if err != nil {
		return View{}, err
	}
	if err := r.save(prepared.Registration); err != nil {
		return View{}, err
	}
	return prepared, nil
}

func (r Registry) PrepareHeartbeat(id, availability, focus string, ttl time.Duration, now time.Time) (View, error) {
	if err := ValidateTTL(ttl); err != nil {
		return View{}, err
	}
	existing, err := r.Get(id, now)
	if err != nil {
		return View{}, err
	}
	reg := existing.Registration
	if availability != "" {
		reg.Availability = availability
	}
	if focus != "" {
		reg.Focus = focus
	}
	reg.LastSeenAt = now.UTC().Format(time.RFC3339Nano)
	reg.ExpiresAt = now.Add(ttl).UTC().Format(time.RFC3339Nano)
	if err := Validate(reg, now); err != nil {
		return View{}, err
	}
	return view(reg, now), nil
}

func (r Registry) Unregister(id string) error {
	path, err := r.path(id)
	if err != nil {
		return err
	}
	return os.Remove(path)
}

func (r Registry) Discover(filter Filter, now time.Time) ([]View, error) {
	entries, err := os.ReadDir(r.directory())
	if errors.Is(err, os.ErrNotExist) {
		return []View{}, nil
	}
	if err != nil {
		return nil, err
	}
	result := make([]View, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("agent registration is a symlink: %s", entry.Name())
		}
		id := strings.TrimSuffix(entry.Name(), ".json")
		item, err := r.Get(id, now)
		if err != nil {
			return nil, err
		}
		if !filter.IncludeStale && !item.Live {
			continue
		}
		if filter.Harness != "" && item.Harness != filter.Harness {
			continue
		}
		if filter.Module != "" && !containsFold(item.Modules, filter.Module) {
			continue
		}
		if filter.Capability != "" && !containsFold(item.Capabilities, filter.Capability) {
			continue
		}
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

func (r Registry) PruneCandidates(olderThan time.Duration, now time.Time) ([]View, error) {
	if olderThan < 0 {
		return nil, fmt.Errorf("older-than must not be negative")
	}
	items, err := r.Discover(Filter{IncludeStale: true}, now)
	if err != nil {
		return nil, err
	}
	stale := make([]View, 0)
	for _, item := range items {
		expires, _ := time.Parse(time.RFC3339Nano, item.ExpiresAt)
		if !now.Before(expires.Add(olderThan)) {
			stale = append(stale, item)
		}
	}
	return stale, nil
}

func (r Registry) Prune(olderThan time.Duration, now time.Time) ([]string, error) {
	candidates, err := r.PruneCandidates(olderThan, now)
	if err != nil {
		return nil, err
	}
	removed := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		current, err := r.Get(candidate.ID, now)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return removed, err
		}
		if current.ExpiresAt != candidate.ExpiresAt {
			continue // renewed since the plan was built
		}
		if err := r.Unregister(candidate.ID); err != nil {
			return removed, err
		}
		removed = append(removed, candidate.ID)
	}
	return removed, nil
}

func containsFold(items []string, sought string) bool {
	for _, item := range items {
		if strings.EqualFold(item, sought) {
			return true
		}
	}
	return false
}

func (r Registry) save(reg Registration) error {
	dir := r.directory()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	if err := os.Chmod(dir, 0700); err != nil {
		return err
	}
	// Registration records are machine-local leases, not project history.
	ignore := filepath.Join(dir, ".gitignore")
	if file, err := os.OpenFile(ignore, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600); err == nil {
		if _, err := file.WriteString("*\n"); err != nil {
			_ = file.Close()
			return err
		}
		if err := file.Close(); err != nil {
			return err
		}
	} else if !errors.Is(err, os.ErrExist) {
		return err
	}
	data, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	temp, err := os.CreateTemp(dir, ".hp-agent-*")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	path, _ := r.path(reg.ID)
	return os.Rename(temp.Name(), path)
}
