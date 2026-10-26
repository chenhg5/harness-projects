// Package layout defines the allowed top-level directories in a Harness Project.
package layout

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

const (
	SchemaVersion = "1"
	RelativePath  = ".harness-project/layout.json"
)

var (
	ErrBuiltIn = errors.New("layout directory is built in")
	ErrExists  = errors.New("layout directory already exists")
)

type Contract struct {
	SchemaVersion string            `json:"schemaVersion"`
	Directories   map[string]string `json:"directories"`
}

type Entry struct {
	Path        string `json:"path"`
	Description string `json:"description"`
	Source      string `json:"source"`
}

var builtIns = map[string]string{
	"spec":             "Current requirements, architecture, interfaces, and durable decisions.",
	"goals":            "Long-lived outcomes, success criteria, and evidence requirements.",
	"tasks":            "Execution units, dependencies, ownership, and acceptance criteria.",
	"runs":             "Execution attempts, provider references, outputs, and verification.",
	"agents":           "Durable participant records and Coordinator configuration.",
	"context":          "Durable research, project knowledge, and searchable memory.",
	"handoffs":         "Transfers of responsibility between agents or people.",
	"events":           "Incoming events and idempotent trigger definitions.",
	"verification":     "Acceptance checks, test results, reviews, and evidence.",
	"worklogs":         "Chronological records of meaningful work sessions.",
	"archive":          "Inactive material retained for audit or recovery.",
	"skills":           "Canonical project skills shared across agent harnesses.",
	"scripts":          "Project-wide scripts; repository-specific scripts stay in repositories.",
	"repos":            "Participating repository checkouts and worktrees.",
	"state":            "The current project control snapshot and active plan.",
	"artifacts":        "Generated outputs that are not source-repository content.",
	"integrations":     "Provider, repository, event, and runner adapter configuration.",
	".harness-project": "hp metadata and machine-local operational state.",
	".codex":           "Codex adapter files and mounted project skills.",
	".claude":          "Claude Code settings and mounted project skills.",
	".cursor":          "Cursor rules and mounted project skills.",
	".agents":          "Agent Skills-compatible mounted project skills.",
	".git":             "Git repository metadata.",
}

func Default() Contract {
	return Contract{SchemaVersion: SchemaVersion, Directories: map[string]string{}}
}

func DefaultBytes() ([]byte, error) {
	return Marshal(Default())
}

func Marshal(contract Contract) ([]byte, error) {
	if err := Validate(contract); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(contract, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

// Load returns the project contract and whether a contract file exists.
// Projects created before the contract was introduced use the built-ins until upgraded.
func Load(root string) (Contract, bool, error) {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(RelativePath)))
	if errors.Is(err, os.ErrNotExist) {
		return Default(), false, nil
	}
	if err != nil {
		return Contract{}, false, err
	}
	var contract Contract
	if err := json.Unmarshal(data, &contract); err != nil {
		return Contract{}, true, fmt.Errorf("invalid layout contract JSON: %w", err)
	}
	if err := Validate(contract); err != nil {
		return Contract{}, true, err
	}
	return contract, true, nil
}

func Validate(contract Contract) error {
	if contract.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported layout schemaVersion: %q", contract.SchemaVersion)
	}
	if contract.Directories == nil {
		return errors.New("layout directories must be an object")
	}
	for path, description := range contract.Directories {
		if err := ValidateName(path); err != nil {
			return err
		}
		if _, reserved := builtIns[path]; reserved {
			return fmt.Errorf("layout directory %q is built in and cannot be overridden", path)
		}
		if strings.TrimSpace(description) == "" {
			return fmt.Errorf("layout directory %q requires a description", path)
		}
	}
	return nil
}

func Add(contract Contract, path, description string, replace bool) (Contract, error) {
	path = strings.TrimSpace(path)
	description = strings.TrimSpace(description)
	if err := ValidateName(path); err != nil {
		return Contract{}, err
	}
	if _, reserved := builtIns[path]; reserved {
		return Contract{}, fmt.Errorf("%w: %q cannot be overridden", ErrBuiltIn, path)
	}
	if description == "" {
		return Contract{}, errors.New("directory description is required")
	}
	if contract.Directories == nil {
		contract.Directories = map[string]string{}
	}
	if _, exists := contract.Directories[path]; exists && !replace {
		return Contract{}, fmt.Errorf("%w: %q; use --replace to update it", ErrExists, path)
	}
	contract.SchemaVersion = SchemaVersion
	contract.Directories[path] = description
	return contract, Validate(contract)
}

func Save(root string, contract Contract) error {
	data, err := Marshal(contract)
	if err != nil {
		return err
	}
	path := filepath.Join(root, filepath.FromSlash(RelativePath))
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".hp-layout-*")
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
	if err := os.Chmod(temp.Name(), 0644); err != nil {
		return err
	}
	return os.Rename(temp.Name(), path)
}

func Entries(contract Contract) []Entry {
	entries := make([]Entry, 0, len(builtIns)+len(contract.Directories))
	for path, description := range builtIns {
		entries = append(entries, Entry{Path: path, Description: description, Source: "built-in"})
	}
	for path, description := range contract.Directories {
		entries = append(entries, Entry{Path: path, Description: description, Source: "project"})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	return entries
}

func ProjectEntries(contract Contract) []Entry {
	entries := make([]Entry, 0, len(contract.Directories))
	for path, description := range contract.Directories {
		entries = append(entries, Entry{Path: path, Description: description, Source: "project"})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	return entries
}

func Allowed(contract Contract, path string) bool {
	if _, ok := builtIns[path]; ok {
		return true
	}
	_, ok := contract.Directories[path]
	return ok
}

func ValidateName(path string) error {
	if path == "" || strings.TrimSpace(path) != path || path == "." || path == ".." || filepath.IsAbs(path) || filepath.Base(path) != path || strings.ContainsAny(path, `/\\`) {
		return fmt.Errorf("layout directory name %q must be one top-level name", path)
	}
	for _, character := range path {
		if unicode.IsControl(character) {
			return fmt.Errorf("layout directory name %q contains control characters", path)
		}
	}
	return nil
}
