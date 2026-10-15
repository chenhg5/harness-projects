package wakeup

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

const SchemaVersion = "1"

// Config is project Git state. It schedules the current Coordinator role,
// rather than a particular provider session, so a handoff keeps the routine.
type Config struct {
	SchemaVersion string `json:"schemaVersion"`
	Enabled       bool   `json:"enabled"`
	Every         string `json:"every,omitempty"`
	Cron          string `json:"cron,omitempty"`
	TimeZone      string `json:"timeZone,omitempty"`
	Worktree      string `json:"worktree"`
	PromptFile    string `json:"promptFile"`
	CreatedAt     string `json:"createdAt"`
	UpdatedAt     string `json:"updatedAt"`
}

// State is a machine-local execution checkpoint. It is intentionally outside
// project Git because every wakeup can update it.
type State struct {
	SchemaVersion   string   `json:"schemaVersion"`
	RegistrationID  string   `json:"registrationId"`
	ConfigUpdatedAt string   `json:"configUpdatedAt"`
	NextDueAt       string   `json:"nextDueAt,omitempty"`
	LastAttemptAt   string   `json:"lastAttemptAt,omitempty"`
	LastStatus      string   `json:"lastStatus,omitempty"`
	LastReason      string   `json:"lastReason,omitempty"`
	LastDeliveryID  string   `json:"lastDeliveryId,omitempty"`
	Logs            []string `json:"logs,omitempty"`
}

type Registry struct{ Root string }

func (r Registry) configPath() string {
	return filepath.Join(r.Root, "agents", "coordinator-wakeup.json")
}
func (r Registry) statePath() string {
	return filepath.Join(r.Root, "artifacts", "wakeup", "state.json")
}

func (r Registry) GetConfig() (Config, error) {
	data, err := os.ReadFile(r.configPath())
	if err != nil {
		return Config{}, err
	}
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return Config{}, fmt.Errorf("invalid wakeup configuration: %w", err)
	}
	if err := Validate(config); err != nil {
		return Config{}, err
	}
	return config, nil
}

func (r Registry) SaveConfig(config Config) error {
	if err := Validate(config); err != nil {
		return err
	}
	return atomicWrite(r.configPath(), config)
}

func (r Registry) GetState() (State, error) {
	data, err := os.ReadFile(r.statePath())
	if err != nil {
		return State{}, err
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}, fmt.Errorf("invalid wakeup state: %w", err)
	}
	if state.SchemaVersion != SchemaVersion {
		return State{}, fmt.Errorf("unsupported wakeup state schemaVersion %q", state.SchemaVersion)
	}
	return state, nil
}

func (r Registry) SaveState(state State) error {
	if state.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported wakeup state schemaVersion %q", state.SchemaVersion)
	}
	return atomicWrite(r.statePath(), state)
}

func Validate(config Config) error {
	if config.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported wakeup schemaVersion %q", config.SchemaVersion)
	}
	if (config.Every == "") == (config.Cron == "") {
		return fmt.Errorf("configure exactly one of --every or --cron")
	}
	if config.Every != "" {
		if config.TimeZone != "" {
			return fmt.Errorf("--timezone is only valid with --cron")
		}
		interval, err := time.ParseDuration(config.Every)
		if err != nil || interval < time.Minute || interval > 7*24*time.Hour {
			return fmt.Errorf("--every must be between 1m and 168h")
		}
	}
	if config.Cron != "" {
		if _, err := cronParser.Parse(config.Cron); err != nil {
			return fmt.Errorf("invalid cron schedule: %w", err)
		}
		if config.TimeZone == "" {
			return fmt.Errorf("--timezone is required with --cron")
		}
		if _, err := time.LoadLocation(config.TimeZone); err != nil {
			return fmt.Errorf("invalid time zone: %w", err)
		}
	}
	for name, path := range map[string]string{"worktree": config.Worktree, "promptFile": config.PromptFile} {
		if path == "" || filepath.IsAbs(path) || filepath.Clean(path) != path || path == ".." || strings.HasPrefix(path, ".."+string(filepath.Separator)) {
			return fmt.Errorf("%s must be a project-relative path", name)
		}
	}
	if _, err := time.Parse(time.RFC3339Nano, config.CreatedAt); err != nil {
		return fmt.Errorf("invalid createdAt: %w", err)
	}
	if _, err := time.Parse(time.RFC3339Nano, config.UpdatedAt); err != nil {
		return fmt.Errorf("invalid updatedAt: %w", err)
	}
	return nil
}

var cronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

func Next(config Config, after time.Time) (time.Time, error) {
	if err := Validate(config); err != nil {
		return time.Time{}, err
	}
	if config.Every != "" {
		interval, _ := time.ParseDuration(config.Every)
		return after.Add(interval).UTC(), nil
	}
	location, _ := time.LoadLocation(config.TimeZone)
	schedule, _ := cronParser.Parse(config.Cron)
	next := schedule.Next(after.In(location))
	if next.IsZero() {
		return time.Time{}, fmt.Errorf("cron schedule has no future occurrence")
	}
	return next.UTC(), nil
}

// ResolveFile keeps configured paths inside the project even through symlinks.
func ResolveFile(root, relative string, directory bool) (string, error) {
	if relative == "" || filepath.IsAbs(relative) || filepath.Clean(relative) != relative || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path must be project-relative: %q", relative)
	}
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	path, err := filepath.EvalSymlinks(filepath.Join(canonicalRoot, relative))
	if err != nil {
		return "", err
	}
	within, err := filepath.Rel(canonicalRoot, path)
	if err != nil || within == ".." || strings.HasPrefix(within, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("configured path escapes project: %s", relative)
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if info.IsDir() != directory {
		return "", fmt.Errorf("configured path has the wrong type: %s", relative)
	}
	if !directory && info.Size() > 64<<10 {
		return "", fmt.Errorf("wakeup prompt file exceeds 64 KiB")
	}
	return path, nil
}

func atomicWrite(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".hp-wakeup-*")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	if _, err := temp.Write(append(data, '\n')); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(temp.Name(), path)
}

func (r Registry) LoadStateOrEmpty() (State, error) {
	state, err := r.GetState()
	if errors.Is(err, os.ErrNotExist) {
		return State{SchemaVersion: SchemaVersion}, nil
	}
	return state, err
}
