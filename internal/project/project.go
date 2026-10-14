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
	"spec", "goals", "tasks", "runs", "agents", "context", "handoffs", "events",
	"verification", "worklogs", "archive", "skills/shared/project-context", "scripts/shared", "repos",
	"state", "artifacts", "integrations", ".harness-project", ".codex/skills",
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
		"PROJECT.md":                             []byte("# " + name + "\n\nDescribe the project goal, scope, constraints, and participating repositories here.\n\n## Current work\n\nRead `state/current-plan.md`, the active goal in `goals/`, the latest work log in `worklogs/`, and the latest handoff in `handoffs/`.\n"),
		"AGENTS.md":                              []byte("# Agent instructions\n\nThis directory is a Harness Project. Read `PROJECT.md`, `.harness-project/project.json`, `goals/`, `state/current-plan.md`, and the latest work log before changing code. Read the latest handoff when responsibility moves between agents. Project specifications live in `spec/`. Tasks and runs live in `tasks/` and `runs/`. Code repositories live in `repos/` and keep their own Git history. Record durable decisions in `spec/`, write a work log after each meaningful session, and write a handoff when transferring work. Use `hp cleanup plan` to keep transient files bounded.\n"),
		"CLAUDE.md":                              []byte("# Claude Code project instructions\n\nFollow `AGENTS.md` and read `PROJECT.md` for the shared project context. The canonical project files are outside `.claude/`.\n"),
		"spec/README.md":                         []byte("# Project specifications\n\nKeep the current requirements, architecture, interfaces, and decisions here. Use separate files for separate concerns.\n"),
		"goals/README.md":                        []byte("# Goals\n\nRecord long-lived project outcomes, constraints, and acceptance criteria here. A goal may span many tasks and runs.\n\nUse Markdown for the explanation: problem, outcome, scope, doneWhen conditions, and evidence required. YAML frontmatter MAY carry stable metadata such as id, status, owner, and method. A matching JSON file is a machine-readable index for filtering and joins; OKR, milestones, experiments, and metrics are optional methods.\n"),
		"tasks/README.md":                        []byte("# Tasks\n\nStore task records and dependency metadata here. A task may have multiple execution runs.\n"),
		"runs/README.md":                         []byte("# Runs\n\nStore execution attempts, status transitions, inputs, outputs, and provider references here.\n"),
		"agents/README.md":                       []byte("# Agents\n\nRecord provider-neutral Agent roles and their execution adapter configuration here.\n"),
		"context/README.md":                      []byte("# Context and memory\n\nStore durable research, project knowledge, and indexed context sources here.\n"),
		"handoffs/README.md":                     []byte("# Handoffs\n\nAdd one dated Markdown file per handoff. Include the goal, completed work, next actions, repository branches and commits, verification, and blockers. Do not overwrite previous handoffs.\n"),
		"events/README.md":                       []byte("# Events\n\nRecord incoming events and trigger definitions here. Event processing must be idempotent.\n"),
		"verification/README.md":                 []byte("# Verification\n\nRecord acceptance checks, test runs, review results, and evidence for project outputs here.\n"),
		"worklogs/README.md":                     []byte("# Work logs\n\nAdd one dated Markdown file after each meaningful agent work session. Record the objective, completed work, remaining work, verification, and next action. Work logs are chronological activity records; use `handoffs/` when responsibility moves between agents or harnesses.\n"),
		"archive/README.md":                      []byte("# Archive\n\n`hp cleanup` moves stale transient files here before removal from active project folders. Keep archived material only when it may help audit or recovery; do not archive goals, specifications, task records, or handoffs automatically.\n"),
		"skills/shared/project-context/SKILL.md": []byte("---\nname: project-context\ndescription: Read the portable context of a Harness Project before work and record a work log after each session.\n---\n\n# Project context\n\n1. Read `PROJECT.md` and `.harness-project/project.json`.\n2. Read `state/current-plan.md`, relevant files in `spec/`, the latest work log, and the latest handoff when present.\n3. Inspect each relevant repository in `repos/` before editing.\n4. Write a work log with objective, completed work, remaining work, verification, and next action. Write a handoff when transferring responsibility.\n5. Use `hp cleanup plan` periodically to archive stale transient files.\n"),
		"scripts/shared/README.md":               []byte("# Shared scripts\n\nPut project-wide scripts here. Keep repository-specific scripts in their repositories.\n"),
		"repos/README.md":                        []byte("# Repositories\n\nPlace linked repository worktrees here. Record their purpose and branch in `PROJECT.md`.\n"),
		"state/status.md":                        []byte("# Status\n\nProject created.\n"),
		"state/current-plan.md":                  []byte("# Current plan\n\n1. Define the project goal and acceptance criteria in `PROJECT.md`.\n2. Add repository worktrees under `repos/`.\n3. Write the first task plan here.\n"),
		"state/project-state.json":               []byte(fmt.Sprintf("{\n  \"schemaVersion\": \"1\",\n  \"updatedAt\": \"%s\",\n  \"currentPhase\": \"discovery\",\n  \"health\": \"unknown\",\n  \"progress\": \"0%%\",\n  \"nextActions\": [\"Define the first goal and its doneWhen conditions\"],\n  \"sourceRefs\": [\"PROJECT.md\"]\n}\n", now.UTC().Format(time.RFC3339))),
		"artifacts/README.md":                    []byte("# Artifacts\n\nStore generated project outputs that are not part of a source repository here.\n"),
		"integrations/README.md":                 []byte("# Integrations\n\nRecord provider, repository, event, and runner adapter configuration here.\n"),
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
	"PROJECT.md", "AGENTS.md", "CLAUDE.md", "spec/README.md", "goals/README.md",
	"tasks/README.md", "runs/README.md", "agents/README.md", "context/README.md",
	"handoffs/README.md", "events/README.md", "verification/README.md", "worklogs/README.md", "archive/README.md",
	"skills/shared/project-context/SKILL.md", "scripts/shared/README.md", "repos/README.md",
	"state/status.md", "state/current-plan.md", "state/project-state.json", "artifacts/README.md", "integrations/README.md",
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
