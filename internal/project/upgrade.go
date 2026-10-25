package project

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/chenhg5/harness-projects/internal/hooks"
	"github.com/chenhg5/harness-projects/internal/skillmount"
)

const managedStart = "<!-- hp:managed-workflow v2 -->"
const managedEnd = "<!-- /hp:managed-workflow -->"

type UpgradePlan struct {
	Action    string          `json:"action"`
	Path      string          `json:"path"`
	Changes   []Action        `json:"changes"`
	SkillPlan skillmount.Plan `json:"skillPlan"`
	files     map[string][]byte
}

func PlanUpgrade(path string) (UpgradePlan, error) {
	root, err := ResolvePath(path)
	if err != nil {
		return UpgradePlan{}, err
	}
	manifestData, err := os.ReadFile(filepath.Join(root, ".harness-project", "project.json"))
	if err != nil {
		return UpgradePlan{}, err
	}
	var manifest Manifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return UpgradePlan{}, fmt.Errorf("invalid project manifest: %w", err)
	}
	if manifest.SchemaVersion != SchemaVersion {
		return UpgradePlan{}, fmt.Errorf("unsupported schemaVersion: %s", manifest.SchemaVersion)
	}

	plan := UpgradePlan{Action: "project.upgrade", Path: root, Changes: []Action{}, files: map[string][]byte{}}
	managed := map[string]string{
		"AGENTS.md":                              managedWorkflowInstructions(),
		"skills/shared/project-context/SKILL.md": managedSkillInstructions(),
		"context/README.md":                      managedContextInstructions(),
		".cursor/rules/harness-project.mdc":      managedCursorInstructions(),
	}
	for relative, block := range managed {
		current, readErr := os.ReadFile(filepath.Join(root, relative))
		if readErr != nil && !os.IsNotExist(readErr) {
			return UpgradePlan{}, readErr
		}
		updated, err := upsertManagedBlock(current, block)
		if err != nil {
			return UpgradePlan{}, fmt.Errorf("upgrade %s: %w", relative, err)
		}
		if !bytes.Equal(current, updated) {
			operation := "update_file"
			if os.IsNotExist(readErr) {
				operation = "create_file"
			}
			absolute := filepath.Join(root, relative)
			plan.Changes = append(plan.Changes, Action{Operation: operation, Path: absolute})
			plan.files[absolute] = updated
		}
	}

	settingsPath := filepath.Join(root, ".claude", "settings.json")
	settings, readErr := os.ReadFile(settingsPath)
	if readErr != nil && !os.IsNotExist(readErr) {
		return UpgradePlan{}, readErr
	}
	merged, changed, err := hooks.MergeClaudeSettings(settings)
	if err != nil {
		return UpgradePlan{}, err
	}
	if changed || os.IsNotExist(readErr) {
		operation := "update_file"
		if os.IsNotExist(readErr) {
			operation = "create_file"
		}
		plan.Changes = append(plan.Changes, Action{Operation: operation, Path: settingsPath})
		plan.files[settingsPath] = merged
	}

	ignorePath := filepath.Join(root, ".gitignore")
	if ignore, readErr := os.ReadFile(ignorePath); readErr == nil {
		updated := strings.ReplaceAll(string(ignore), "/.harness-project/worktrees/\n", "")
		if updated != string(ignore) {
			plan.Changes = append(plan.Changes, Action{Operation: "update_file", Path: ignorePath})
			plan.files[ignorePath] = []byte(updated)
		}
	} else if !os.IsNotExist(readErr) {
		return UpgradePlan{}, readErr
	}

	targets, _ := skillmount.ParseTargets("all")
	plan.SkillPlan, err = skillmount.BuildPlan(root, targets, false)
	if err != nil {
		return UpgradePlan{}, err
	}
	for _, change := range plan.SkillPlan.Changes {
		plan.Changes = append(plan.Changes, Action{Operation: change.Operation, Path: change.Path, Target: change.Target})
	}
	sort.Slice(plan.Changes, func(i, j int) bool {
		if plan.Changes[i].Path != plan.Changes[j].Path {
			return plan.Changes[i].Path < plan.Changes[j].Path
		}
		return plan.Changes[i].Operation < plan.Changes[j].Operation
	})
	return plan, nil
}

func ApplyUpgrade(plan UpgradePlan) error {
	for _, change := range plan.Changes {
		content, ok := plan.files[change.Path]
		if !ok {
			continue
		}
		if err := writeGeneratedFile(change.Path, content); err != nil {
			return err
		}
	}
	return skillmount.Apply(plan.SkillPlan)
}

func writeGeneratedFile(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".hp-upgrade-*")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	if _, err := temp.Write(content); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(temp.Name(), path)
}

func upsertManagedBlock(content []byte, body string) ([]byte, error) {
	text := strings.TrimRight(string(content), "\n")
	start := strings.Index(text, managedStart)
	end := strings.Index(text, managedEnd)
	block := managedStart + "\n" + strings.TrimSpace(body) + "\n" + managedEnd
	switch {
	case start < 0 && end < 0:
		if text == "" {
			return []byte(block + "\n"), nil
		}
		return []byte(text + "\n\n" + block + "\n"), nil
	case start < 0 || end < start:
		return nil, fmt.Errorf("malformed managed workflow markers")
	default:
		end += len(managedEnd)
		return []byte(strings.TrimRight(text[:start]+block+text[end:], "\n") + "\n"), nil
	}
}

func managedWorkflowInstructions() string {
	return `## Managed hp workflow

- Start with ` + "`hp brief`" + `; use ` + "`hp project doctor --path .`" + ` when the brief reports drift.
- Continue relevant in-progress work from the brief, or use ` + "`hp task ready`" + ` for unstarted tasks; start one with ` + "`hp task update --status in_progress`" + `.
- Keep durable research in ` + "`context/`" + `; ` + "`hp context search`" + ` searches structured and ordinary notes and requires every whitespace-separated query term to match.
- Use ` + "`hp finish`" + ` at the end of meaningful work, including ` + "`--status in_progress`" + ` for a partial iteration. It updates the task, writes the work log, and reconciles task-backed state.
- Use ` + "`hp handoff create`" + ` only when responsibility moves. A handoff supplements ` + "`hp finish`" + `; it does not replace it.
- Run ` + "`hp project doctor --path .`" + ` before closeout. Do not commit, branch, push, or open a pull request unless the user or repository workflow explicitly requests it.
- Run ` + "`hp project upgrade --path .`" + ` after updating hp; it refreshes this managed block, hooks, and skill mounts without replacing project-specific instructions.`
}

func managedSkillInstructions() string {
	return `## Reliable lifecycle

1. Run ` + "`hp brief`" + ` and inspect the task inbox before editing.
2. Continue relevant in-progress work from the brief, or use ` + "`hp task ready`" + ` for unstarted tasks. Mark the chosen task with ` + "`hp task update --status in_progress`" + `.
3. Use focused ` + "`hp context search`" + ` queries for structured and ordinary notes; every whitespace-separated term must match.
4. End meaningful work with ` + "`hp finish`" + `; use ` + "`--status in_progress`" + ` for partial work. Create a handoff in addition only when responsibility moves.
5. Run ` + "`hp project doctor --path <project>`" + ` and repair reported drift. Do not commit, branch, push, or open a pull request unless explicitly requested.
6. Use ` + "`hp project upgrade --path <project>`" + ` after hp upgrades to refresh managed instructions, hooks, and all supported skill mounts.`
}

func managedContextInstructions() string {
	return `## Search contract

Ordinary Markdown and text notes may live anywhere below this directory except ` + "`context/imports/`" + `. Structured memories created by ` + "`hp context add`" + ` live below ` + "`context/memory/`" + `. ` + "`hp context search`" + ` searches both forms and requires every whitespace-separated query term to match; retry with fewer or more specific terms when a broad query returns nothing.`
}

func managedCursorInstructions() string {
	return `Run ` + "`hp brief`" + ` before work. Use ` + "`hp finish`" + ` for meaningful closeout and add a handoff only when responsibility moves. Run ` + "`hp project doctor --path .`" + ` before stopping. Do not commit or push unless explicitly requested.`
}
