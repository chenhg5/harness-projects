package project

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const SchemaVersion = "1"

type Manifest struct {
	SchemaVersion string `json:"schemaVersion"`
	Name          string `json:"name"`
	CreatedAt     string `json:"createdAt"`
}

type Action struct {
	Operation string `json:"operation"`
	Path      string `json:"path"`
	Target    string `json:"target,omitempty"`
}

type InitResult struct {
	Action  string   `json:"action"`
	Path    string   `json:"path"`
	Name    string   `json:"name"`
	Created bool     `json:"created"`
	Changes []Action `json:"changes"`
}

type Status struct {
	Path     string   `json:"path"`
	Name     string   `json:"name,omitempty"`
	Version  string   `json:"schemaVersion,omitempty"`
	Valid    bool     `json:"valid"`
	Problems []string `json:"problems"`
}

var directories = []string{
	"spec", "handoffs", "skills/shared/project-context", "scripts/shared",
	"repos", "state", "artifacts", ".harness-project", ".codex/skills",
	".claude/skills", ".cursor/rules",
}

func ResolvePath(path string) (string, error) {
	if path == "" {
		return "", errors.New("project path is required")
	}
	return filepath.Abs(path)
}

func PlanInit(path, name string, ifNotExists bool, now time.Time) (InitResult, map[string][]byte, error) {
	root, err := ResolvePath(path)
	if err != nil {
		return InitResult{}, nil, err
	}
	if name == "" {
		name = filepath.Base(root)
	}
	if strings.TrimSpace(name) == "" || name == "." || name == string(filepath.Separator) || strings.ContainsAny(name, "\r\n") {
		return InitResult{}, nil, fmt.Errorf("invalid project name %q", name)
	}
	result := InitResult{Action: "project.init", Path: root, Name: name, Created: true, Changes: []Action{}}
	if _, err := os.Lstat(root); err == nil {
		if !ifNotExists {
			return InitResult{}, nil, fmt.Errorf("project path already exists: %s", root)
		}
		status := Inspect(root)
		if !status.Valid {
			return InitResult{}, nil, fmt.Errorf("project path exists but is not a valid Harness Project: %s", strings.Join(status.Problems, "; "))
		}
		result.Name = status.Name
		result.Created = false
		return result, nil, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return InitResult{}, nil, err
	}

	manifest, err := json.MarshalIndent(Manifest{SchemaVersion: SchemaVersion, Name: name, CreatedAt: now.UTC().Format(time.RFC3339)}, "", "  ")
	if err != nil {
		return InitResult{}, nil, err
	}
	files := map[string][]byte{
		"PROJECT.md":                             []byte("# " + name + "\n\nDescribe the project goal, scope, and participating repositories here.\n\n## Current work\n\nRead `state/current-plan.md` and the latest file in `handoffs/`.\n"),
		"AGENTS.md":                              []byte("# Agent instructions\n\nThis directory is a Harness Project. Read `PROJECT.md`, `.harness-project/project.json`, `state/current-plan.md`, and the latest handoff before changing code. Project specifications live in `spec/`. Code repositories live in `repos/`. Record durable decisions in `spec/` and write a new handoff when transferring work.\n"),
		"CLAUDE.md":                              []byte("# Claude Code project instructions\n\nFollow `AGENTS.md` and read `PROJECT.md` for the shared project context. The canonical project files are outside `.claude/`.\n"),
		"spec/README.md":                         []byte("# Project specifications\n\nKeep the current requirements, architecture, interfaces, and decisions here. Use separate files for separate concerns.\n"),
		"handoffs/README.md":                     []byte("# Handoffs\n\nAdd one dated Markdown file per handoff. Include the goal, completed work, next actions, repository branches and commits, verification, and blockers. Do not overwrite previous handoffs.\n"),
		"skills/shared/project-context/SKILL.md": []byte("---\nname: project-context\ndescription: Read the portable context of a Harness Project before work and record a handoff when transferring it.\n---\n\n# Project context\n\n1. Read `PROJECT.md` and `.harness-project/project.json`.\n2. Read `state/current-plan.md`, relevant files in `spec/`, and the latest handoff.\n3. Inspect each relevant repository in `repos/` before editing.\n4. Update the plan and append a handoff when transferring work.\n"),
		"scripts/shared/README.md":               []byte("# Shared scripts\n\nPut project-wide scripts here. Keep repository-specific scripts in their repositories.\n"),
		"repos/README.md":                        []byte("# Repositories\n\nPlace linked repository worktrees here. Record their purpose and branch in `PROJECT.md`.\n"),
		"state/status.md":                        []byte("# Status\n\nProject created.\n"),
		"state/current-plan.md":                  []byte("# Current plan\n\n1. Define the project goal and acceptance criteria in `PROJECT.md`.\n2. Add repository worktrees under `repos/`.\n3. Write the first task plan here.\n"),
		"artifacts/README.md":                    []byte("# Artifacts\n\nStore generated project outputs that are not part of a source repository here.\n"),
		".harness-project/project.json":          append(manifest, '\n'),
		".cursor/rules/harness-project.mdc":      []byte("---\ndescription: Shared Harness Project context\nalwaysApply: true\n---\n\nRead `AGENTS.md` and `PROJECT.md` before work. Use `state/current-plan.md`, `spec/`, and `handoffs/` for portable project context.\n"),
	}
	result.Changes = append(result.Changes, Action{Operation: "create_directory", Path: root})
	for _, dir := range directories {
		result.Changes = append(result.Changes, Action{Operation: "create_directory", Path: filepath.Join(root, dir)})
	}
	for _, relative := range fileOrder {
		result.Changes = append(result.Changes, Action{Operation: "create_file", Path: filepath.Join(root, relative)})
	}
	for _, link := range []string{".codex/skills/project-context", ".claude/skills/project-context"} {
		linkPath := filepath.Join(root, link)
		target, err := filepath.Rel(filepath.Dir(linkPath), filepath.Join(root, "skills/shared/project-context"))
		if err != nil {
			return InitResult{}, nil, err
		}
		result.Changes = append(result.Changes, Action{Operation: "create_symlink", Path: linkPath, Target: target})
	}
	return result, files, nil
}

var fileOrder = []string{
	"PROJECT.md", "AGENTS.md", "CLAUDE.md", "spec/README.md", "handoffs/README.md",
	"skills/shared/project-context/SKILL.md", "scripts/shared/README.md", "repos/README.md",
	"state/status.md", "state/current-plan.md", "artifacts/README.md",
	".harness-project/project.json", ".cursor/rules/harness-project.mdc",
}

func ApplyInit(plan InitResult, files map[string][]byte) error {
	if !plan.Created {
		return nil
	}
	if err := os.Mkdir(plan.Path, 0755); err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = os.RemoveAll(plan.Path)
		}
	}()
	for _, dir := range directories {
		if err := os.MkdirAll(filepath.Join(plan.Path, dir), 0755); err != nil {
			return err
		}
	}
	for _, relative := range fileOrder {
		if err := os.WriteFile(filepath.Join(plan.Path, relative), files[relative], 0644); err != nil {
			return err
		}
	}
	for _, action := range plan.Changes {
		if action.Operation == "create_symlink" {
			if err := os.Symlink(action.Target, action.Path); err != nil {
				return err
			}
		}
	}
	committed = true
	return nil
}

func Inspect(path string) Status {
	root, err := ResolvePath(path)
	if err != nil {
		return Status{Path: path, Problems: []string{err.Error()}}
	}
	status := Status{Path: root, Problems: []string{}}
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err == nil {
		root = canonicalRoot
		status.Path = root
	}
	info, err := os.Stat(root)
	if err != nil {
		status.Problems = append(status.Problems, "project directory does not exist")
		return status
	}
	if !info.IsDir() {
		status.Problems = append(status.Problems, "project path is not a directory")
		return status
	}
	data, err := os.ReadFile(filepath.Join(root, ".harness-project/project.json"))
	if err != nil {
		status.Problems = append(status.Problems, "missing or unreadable .harness-project/project.json")
	} else {
		var manifest Manifest
		if err := json.Unmarshal(data, &manifest); err != nil {
			status.Problems = append(status.Problems, "invalid project manifest JSON")
		} else {
			status.Name, status.Version = manifest.Name, manifest.SchemaVersion
			if manifest.SchemaVersion != SchemaVersion {
				status.Problems = append(status.Problems, "unsupported schemaVersion: "+manifest.SchemaVersion)
			}
			if strings.TrimSpace(manifest.Name) == "" {
				status.Problems = append(status.Problems, "project name is empty")
			}
		}
	}
	for _, dir := range directories {
		info, err := os.Stat(filepath.Join(root, dir))
		if err != nil || !info.IsDir() {
			status.Problems = append(status.Problems, "missing directory: "+dir)
		}
	}
	for _, relative := range fileOrder {
		info, err := os.Stat(filepath.Join(root, relative))
		if err != nil || info.IsDir() {
			status.Problems = append(status.Problems, "missing file: "+relative)
		}
	}
	for _, link := range []string{".codex/skills/project-context", ".claude/skills/project-context"} {
		linkPath := filepath.Join(root, link)
		info, err := os.Lstat(linkPath)
		if err != nil || info.Mode()&os.ModeSymlink == 0 {
			status.Problems = append(status.Problems, "missing skill symlink: "+link)
			continue
		}
		target, err := filepath.EvalSymlinks(linkPath)
		canonicalTarget, targetErr := filepath.EvalSymlinks(filepath.Join(root, "skills/shared/project-context"))
		if targetErr != nil || err != nil || target != canonicalTarget {
			status.Problems = append(status.Problems, "invalid skill symlink: "+link)
		}
	}
	status.Valid = len(status.Problems) == 0
	return status
}
