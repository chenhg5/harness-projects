package project

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chenhg5/harness-projects/internal/apperror"
	"github.com/chenhg5/harness-projects/internal/hooks"
	"github.com/chenhg5/harness-projects/internal/layout"
	"github.com/chenhg5/harness-projects/internal/skillmount"
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
	".claude/skills", ".cursor/rules", ".cursor/skills", ".agents/skills",
}

var optionalDirectories = map[string]bool{
	".cursor/skills": true,
	".agents/skills": true,
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
			return InitResult{}, nil, apperror.Conflict("project path already exists: %s", root)
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
	layoutContract, err := layout.DefaultBytes()
	if err != nil {
		return InitResult{}, nil, err
	}
	files := map[string][]byte{
		"PROJECT.md":                             []byte("# " + name + "\n\nDescribe the project goal, scope, constraints, and participating repositories here.\n\n## Current work\n\nRead `state/current-plan.md`, the active goal in `goals/`, the latest work log in `worklogs/`, and the latest handoff in `handoffs/`.\n"),
		"AGENTS.md":                              []byte("# Agent instructions\n\nThis directory is a Harness Project: the shared memory for every coding agent working on it. Chat history is not shared between sessions or tools, so these files are the source of truth.\n\n## Start of every session\n\nRun `hp brief` and read it. It shows the goal, plan, in-progress and ready tasks, and what the last session left unfinished. Then read the files it points to before changing code. Run `hp project doctor --path .` when the brief reports drift. Code repositories live in `repos/` and keep their own Git history.\n\n## While working\n\n- Continue the relevant in-progress task shown by `hp brief`, or use `hp task ready --project .` to find unstarted work. Mark a chosen task with `hp task update --project . --id <task> --status in_progress`.\n- Record durable decisions in `spec/` and research in `context/`; `hp context search` searches structured memories and ordinary notes. Every whitespace-separated query term must match, so retry with fewer or more specific terms when needed.\n- Register this session with `hp agent register --project . --focus <what you are doing>`. The session ID is detected from `HP_SESSION_ID` or Codex `CODEX_THREAD_ID`; never invent one. Pass the real `--harness` and `--session-id` when detection fails, and pass `--worktree` when working outside the project directory. Keep the returned registration ID for `hp agent unregister --project . --id <registration-id>`.\n\n## Before you stop\n\n- Use `hp finish --project . --task-id <task> --id <slug> --objective ... --completed ... --remaining ... --next ...` after meaningful work. Use `--status in_progress` for a partial iteration; it still writes the work log and reconciles task-backed state.\n- Create `hp handoff create` only when responsibility moves to another agent or person. A handoff supplements `hp finish`; it does not replace it.\n- `hp finish` reconciles task-backed state only. Update goal status, project phase, health, and project-level next actions separately when they changed.\n- Run `hp project doctor --path .`. Do not create commits, branches, pushes, or pull requests unless the user or repository workflow explicitly requests them.\n\n## Advanced: coordination\n\nUse `hp agent discover --project .` to find other agents and `hp task inbox --project .` for work assigned to this session. A Coordinator can use `hp coordinator dispatch --project . --task-id <id> --harness <name> --session-id <id> --worktree <path>` for explicit CLI delivery; preflight failure leaves the assignment in the inbox. Inspect an uncertain run before any `--retry`. Participation remains registered; `hp agent heartbeat` refreshes the local delivery hint and `hp agent unregister --project . --id <registration-id>` ends it. Use `hp cleanup plan` to keep transient files bounded.\n"),
		"CLAUDE.md":                              []byte("# Claude Code project instructions\n\nFollow `AGENTS.md` and read `PROJECT.md` for the shared project context. The canonical project files are outside `.claude/`.\n"),
		"spec/README.md":                         []byte("# Project specifications\n\nKeep the current requirements, architecture, interfaces, and decisions here. Use separate files for separate concerns.\n"),
		"goals/README.md":                        []byte("# Goals\n\nRecord long-lived project outcomes, constraints, and acceptance criteria here. A goal may span many tasks and runs.\n\nUse Markdown for the explanation: problem, outcome, scope, doneWhen conditions, and evidence required. YAML frontmatter MAY carry stable metadata such as id, status, owner, and method. A matching JSON file is a machine-readable index for filtering and joins; OKR, milestones, experiments, and metrics are optional methods.\n"),
		"tasks/README.md":                        []byte("# Tasks\n\nStore task records and dependency metadata here. A task may have multiple execution runs.\n"),
		"runs/README.md":                         []byte("# Runs\n\nStore execution attempts, status transitions, inputs, outputs, and provider references here.\n"),
		"agents/README.md":                       []byte("# Agents\n\nSession participation records live in `agents/sessions/` and remain discoverable. The current Coordinator and assignment history live in `agents/coordinator.json`; assign a registered session with `hp coordinator assign`. Its optional tracked schedule lives in `agents/coordinator-wakeup.json`, and its editable routine in `agents/coordinator-wakeup.md`. `hp wakeup start` runs a foreground scheduler; local execution state and logs remain under ignored `artifacts/`. Machine-local direct-delivery hints live under `.harness-project/agent-sessions/` and may expire without deleting participation.\n"),
		"agents/coordinator-wakeup.md":           []byte("# Coordinator wakeup routine\n\n1. Read the current goal, plan, latest work log, and assigned task inbox.\n2. Check for ready tasks, blockers, and work that needs a handoff.\n3. Continue the highest priority authorized work and record meaningful progress in a work log.\n4. If there is nothing actionable, report that briefly; do not invent work.\n"),
		"context/README.md":                      []byte("# Context and memory\n\nStore durable research, project knowledge, and indexed context sources here.\n"),
		"handoffs/README.md":                     []byte("# Handoffs\n\nAdd one dated Markdown file per handoff. Include the goal, completed work, next actions, repository branches and commits, verification, and blockers. Do not overwrite previous handoffs.\n"),
		"events/README.md":                       []byte("# Events\n\nRecord incoming events and trigger definitions here. Event processing must be idempotent.\n"),
		"verification/README.md":                 []byte("# Verification\n\nRecord acceptance checks, test runs, review results, and evidence for project outputs here.\n"),
		"worklogs/README.md":                     []byte("# Work logs\n\nAdd one dated Markdown file after each meaningful agent work session. Record the objective, completed work, remaining work, verification, and next action. Work logs are chronological activity records; use `handoffs/` when responsibility moves between agents or harnesses.\n"),
		"archive/README.md":                      []byte("# Archive\n\n`hp cleanup` moves stale transient files here before removal from active project folders. Keep archived material only when it may help audit or recovery; do not archive goals, specifications, task records, or handoffs automatically.\n"),
		"skills/shared/project-context/SKILL.md": []byte("---\nname: project-context\ndescription: Read, maintain, diagnose, and safely close work in a portable Harness Project. Use when working in a directory that contains .harness-project/.\n---\n\n# Project context\n\n1. Run `hp brief` (from anywhere inside the project) and read it: goal, plan, next actions, in-progress and ready tasks, and the remaining work from the latest work log and handoff.\n2. Read the files the brief points to, relevant `spec/` files, and each repository in `repos/` you will edit.\n3. Register the session with `hp agent register --project . --focus <responsibility>`; the session ID is detected from `HP_SESSION_ID` or `CODEX_THREAD_ID`. Do not invent an ID. Pass `--worktree` when working outside the project and keep the returned registration ID for `hp agent unregister --id <registration-id>`.\n4. Continue the relevant in-progress task from `hp brief`, or use `hp task ready --project .` for unstarted work. Mark it with `hp task update --project . --id <task> --status in_progress`.\n5. Use focused `hp context search` queries; every whitespace-separated term must match, so retry with fewer terms if needed.\n6. Before stopping, use `hp finish`; use `--status in_progress` for partial work. A handoff is additional and only applies when responsibility moves.\n7. Run `hp project doctor --path .`. Do not commit, branch, push, or open a pull request unless explicitly requested.\n8. Advanced: find peers with `hp agent discover`, assigned work with `hp task inbox --project .`, and use `hp coordinator dispatch` for explicit CLI delivery. Inspect an uncertain run before any `--retry`.\n"),
		"scripts/shared/README.md":               []byte("# Shared scripts\n\nPut project-wide scripts here. Keep repository-specific scripts in their repositories.\n"),
		"repos/README.md":                        []byte("# Repositories\n\nPlace linked repository worktrees here. Record their purpose and branch in `PROJECT.md`.\n"),
		"state/status.md":                        []byte("# Status\n\nProject created.\n"),
		"state/current-plan.md":                  []byte("# Current plan\n\n1. Define the project goal and acceptance criteria in `PROJECT.md`.\n2. Add repository worktrees under `repos/`.\n3. Write the first task plan here.\n"),
		"state/project-state.json":               []byte(fmt.Sprintf("{\n  \"schemaVersion\": \"1\",\n  \"updatedAt\": \"%s\",\n  \"currentPhase\": \"discovery\",\n  \"health\": \"unknown\",\n  \"progress\": \"0%%\",\n  \"nextActions\": [\"Define the first goal and its doneWhen conditions\"],\n  \"sourceRefs\": [\"PROJECT.md\"]\n}\n", now.UTC().Format(time.RFC3339))),
		"artifacts/README.md":                    []byte("# Artifacts\n\nStore generated project outputs that are not part of a source repository here.\n"),
		"integrations/README.md":                 []byte("# Integrations\n\nRecord provider, repository, event, and runner adapter configuration here.\n"),
		".harness-project/project.json":          append(manifest, '\n'),
		".harness-project/layout.json":           layoutContract,
		".cursor/rules/harness-project.mdc":      []byte("---\ndescription: Shared Harness Project context\nalwaysApply: true\n---\n\nRun `hp brief` at the start of work and read `AGENTS.md` and `PROJECT.md`. Use `state/current-plan.md`, `spec/`, and `context/` for portable project context. Close meaningful work with `hp finish`; add a handoff only when responsibility moves. Do not commit or push unless explicitly requested.\n"),
		".gitignore":                             []byte("# Source checkouts and generated worktrees keep their own Git history.\n/repos/*\n!/repos/README.md\n\n# Machine-local and generated material.\n/archive/*\n!/archive/README.md\n/artifacts/*\n!/artifacts/README.md\n/context/imports/\n*.log\n.DS_Store\n"),
	}
	for relative, body := range map[string]string{
		"AGENTS.md":                              managedWorkflowInstructions(),
		"CLAUDE.md":                              managedWorkflowInstructions(),
		"skills/shared/project-context/SKILL.md": managedSkillInstructions(),
		"context/README.md":                      managedContextInstructions(),
		".cursor/rules/harness-project.mdc":      managedCursorInstructions(),
	} {
		files[relative], err = upsertManagedBlock(files[relative], body)
		if err != nil {
			return InitResult{}, nil, err
		}
	}
	settings, _, err := hooks.MergeClaudeSettings(nil)
	if err != nil {
		return InitResult{}, nil, err
	}
	files[".claude/settings.json"] = settings
	files["AGENTS.md"] = append(files["AGENTS.md"], []byte("\nThe current Coordinator is an explicit project role, not a capability label or an online status. Check `hp coordinator get --project .` before acting as Coordinator. Assign a registered session with `hp coordinator assign --project . --registration-id <id>`; use `--replace` for a handoff. If `hp wakeup status --project .` shows an enabled schedule, the foreground `hp wakeup start --project .` process delivers the routine in `agents/coordinator-wakeup.md` to the current session. Inspect uncertain delivery logs before any manual `hp wakeup once --retry`.\n")...)
	files["skills/shared/project-context/SKILL.md"] = append(files["skills/shared/project-context/SKILL.md"], []byte("\n8. Check `hp coordinator get --project .` for the current Coordinator. A Coordinator may configure `hp wakeup` with an interval or cron schedule and run the foreground scheduler; the role is explicit and transferable. Inspect `hp wakeup status` and delivery logs before retrying an uncertain wakeup.\n")...)
	result.Changes = append(result.Changes, Action{Operation: "create_directory", Path: root})
	for _, dir := range directories {
		result.Changes = append(result.Changes, Action{Operation: "create_directory", Path: filepath.Join(root, dir)})
	}
	for _, relative := range fileOrder {
		result.Changes = append(result.Changes, Action{Operation: "create_file", Path: filepath.Join(root, relative)})
	}
	mountPlan, err := skillmount.BuildPlanForSkills(root, []skillmount.Skill{{
		Name:   "project-context",
		Source: filepath.Join(root, "skills/shared/project-context"),
		Scope:  "shared",
	}}, []string{"codex", "claude", "cursor", "agents"}, false)
	if err != nil {
		return InitResult{}, nil, err
	}
	for _, change := range mountPlan.Changes {
		result.Changes = append(result.Changes, Action{Operation: change.Operation, Path: change.Path, Target: change.Target})
	}
	return result, files, nil
}

var fileOrder = []string{
	"PROJECT.md", "AGENTS.md", "CLAUDE.md", "spec/README.md", "goals/README.md",
	"tasks/README.md", "runs/README.md", "agents/README.md", "agents/coordinator-wakeup.md", "context/README.md",
	"handoffs/README.md", "events/README.md", "verification/README.md", "worklogs/README.md", "archive/README.md",
	"skills/shared/project-context/SKILL.md", "scripts/shared/README.md", "repos/README.md",
	"state/status.md", "state/current-plan.md", "state/project-state.json", "artifacts/README.md", "integrations/README.md",
	".harness-project/project.json", ".harness-project/layout.json", ".cursor/rules/harness-project.mdc", ".gitignore", ".claude/settings.json",
}

// Templates can be generated for new projects without becoming requirements
// for projects created before the feature existed.
var optionalFiles = map[string]bool{
	"agents/coordinator-wakeup.md": true,
	".harness-project/layout.json": true,
	".gitignore":                   true,
	".claude/settings.json":        true,
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
		if errors.Is(err, os.ErrNotExist) && optionalDirectories[dir] {
			continue
		}
		if err != nil || !info.IsDir() {
			status.Problems = append(status.Problems, "missing directory: "+dir)
		}
	}
	for _, relative := range fileOrder {
		info, err := os.Stat(filepath.Join(root, relative))
		if errors.Is(err, os.ErrNotExist) && optionalFiles[relative] {
			continue
		}
		if err != nil || info.IsDir() {
			status.Problems = append(status.Problems, "missing file: "+relative)
		}
	}
	targets, _ := skillmount.ParseTargets("codex,claude")
	mountPlan, mountErr := skillmount.BuildPlan(root, targets, false)
	if mountErr != nil {
		status.Problems = append(status.Problems, "invalid skill mounts: "+mountErr.Error())
	} else {
		for _, change := range mountPlan.Changes {
			relative, err := filepath.Rel(root, change.Path)
			if err != nil {
				relative = change.Path
			}
			switch change.Operation {
			case "create_symlink":
				status.Problems = append(status.Problems, "missing skill symlink: "+relative)
			case "replace_symlink":
				status.Problems = append(status.Problems, "invalid skill symlink: "+relative)
			}
		}
	}
	status.Valid = len(status.Problems) == 0
	return status
}

// Find returns the nearest directory at or above start that holds a project
// manifest, so commands work from inside nested repository worktrees.
func Find(start string) (string, error) {
	dir, err := ResolvePath(start)
	if err != nil {
		return "", err
	}
	for {
		if info, err := os.Stat(filepath.Join(dir, ".harness-project", "project.json")); err == nil && info.Mode().IsRegular() {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no Harness Project found at or above %s: %w", start, os.ErrNotExist)
		}
		dir = parent
	}
}
