package skillmount

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var targetOrder = []string{"codex", "claude", "cursor", "agents"}

var targetDirectories = map[string]string{
	"codex":  ".codex/skills",
	"claude": ".claude/skills",
	"cursor": ".cursor/skills",
	"agents": ".agents/skills",
}

type Skill struct {
	Name   string `json:"name"`
	Source string `json:"source"`
	Scope  string `json:"scope"`
}

type Change struct {
	Operation string `json:"operation"`
	Path      string `json:"path"`
	Target    string `json:"target,omitempty"`
	Previous  string `json:"-"`
}

type Plan struct {
	Action    string   `json:"action"`
	Project   string   `json:"project"`
	Targets   []string `json:"targets"`
	Skills    []Skill  `json:"skills"`
	Changes   []Change `json:"changes"`
	Unchanged int      `json:"unchanged"`
}

type ConflictError struct {
	Path   string
	Reason string
}

func (err *ConflictError) Error() string {
	return fmt.Sprintf("skill mount conflict at %s: %s", err.Path, err.Reason)
}

func (err *ConflictError) Is(target error) bool { return errors.Is(target, os.ErrExist) }

func ParseTargets(value string) ([]string, error) {
	selected := map[string]bool{}
	if strings.TrimSpace(value) == "" || strings.TrimSpace(value) == "all" {
		for _, target := range targetOrder {
			selected[target] = true
		}
	} else {
		for _, raw := range strings.Split(value, ",") {
			target := strings.TrimSpace(raw)
			if target == "claude-code" {
				target = "claude"
			}
			if _, ok := targetDirectories[target]; !ok {
				return nil, fmt.Errorf("invalid target %q; use codex, claude, cursor, agents, or all", raw)
			}
			selected[target] = true
		}
	}
	targets := make([]string, 0, len(selected))
	for _, target := range targetOrder {
		if selected[target] {
			targets = append(targets, target)
		}
	}
	return targets, nil
}

func Discover(root string) ([]Skill, error) {
	skillsRoot := filepath.Join(root, "skills")
	info, err := os.Stat(skillsRoot)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("skills path is not a directory: %s", skillsRoot)
	}
	skills := []Skill{}
	err = filepath.WalkDir(skillsRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == skillsRoot || !entry.IsDir() {
			return nil
		}
		skillFile := filepath.Join(path, "SKILL.md")
		fileInfo, statErr := os.Stat(skillFile)
		if errors.Is(statErr, os.ErrNotExist) {
			return nil
		}
		if statErr != nil {
			return statErr
		}
		if !fileInfo.Mode().IsRegular() {
			return fmt.Errorf("skill manifest is not a regular file: %s", skillFile)
		}
		relative, relErr := filepath.Rel(skillsRoot, path)
		if relErr != nil {
			return relErr
		}
		skills = append(skills, Skill{
			Name:   filepath.Base(path),
			Source: path,
			Scope:  scopeFor(relative),
		})
		return filepath.SkipDir
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(skills, func(i, j int) bool {
		if skills[i].Name == skills[j].Name {
			return skills[i].Source < skills[j].Source
		}
		return skills[i].Name < skills[j].Name
	})
	return skills, nil
}

func BuildPlan(root string, targets []string, prune bool) (Plan, error) {
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return Plan{}, err
	}
	skills, err := Discover(absoluteRoot)
	if err != nil {
		return Plan{}, err
	}
	return BuildPlanForSkills(absoluteRoot, skills, targets, prune)
}

func BuildPlanForSkills(root string, skills []Skill, targets []string, prune bool) (Plan, error) {
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return Plan{}, err
	}
	plan := Plan{Action: "skill.sync", Project: absoluteRoot, Targets: append([]string(nil), targets...), Skills: skills, Changes: []Change{}}
	desired := map[string]string{}
	for _, target := range targets {
		targetDirectory, ok := targetDirectories[target]
		if !ok {
			return Plan{}, fmt.Errorf("invalid target %q", target)
		}
		mountRoot := filepath.Join(absoluteRoot, targetDirectory)
		if err := checkMountPath(absoluteRoot, mountRoot); err != nil {
			return Plan{}, err
		}
		for _, skill := range skills {
			if skill.Scope != "shared" && skill.Scope != target {
				continue
			}
			linkPath := filepath.Join(mountRoot, skill.Name)
			if previousSource, exists := desired[linkPath]; exists && previousSource != skill.Source {
				return Plan{}, &ConflictError{Path: linkPath, Reason: fmt.Sprintf("multiple skills named %q: %s and %s", skill.Name, previousSource, skill.Source)}
			}
			desired[linkPath] = skill.Source
		}
	}

	paths := make([]string, 0, len(desired))
	for path := range desired {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, linkPath := range paths {
		target, err := filepath.Rel(filepath.Dir(linkPath), desired[linkPath])
		if err != nil {
			return Plan{}, err
		}
		info, err := os.Lstat(linkPath)
		switch {
		case errors.Is(err, os.ErrNotExist):
			plan.Changes = append(plan.Changes, Change{Operation: "create_symlink", Path: linkPath, Target: target})
		case err != nil:
			return Plan{}, err
		case info.Mode()&os.ModeSymlink == 0:
			return Plan{}, &ConflictError{Path: linkPath, Reason: "a non-symlink entry already exists"}
		default:
			previous, readErr := os.Readlink(linkPath)
			if readErr != nil {
				return Plan{}, readErr
			}
			if previous == target {
				plan.Unchanged++
				continue
			}
			plan.Changes = append(plan.Changes, Change{Operation: "replace_symlink", Path: linkPath, Target: target, Previous: previous})
		}
	}

	if prune {
		for _, target := range targets {
			mountRoot := filepath.Join(absoluteRoot, targetDirectories[target])
			entries, err := os.ReadDir(mountRoot)
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			if err != nil {
				return Plan{}, err
			}
			for _, entry := range entries {
				linkPath := filepath.Join(mountRoot, entry.Name())
				if _, exists := desired[linkPath]; exists || entry.Type()&os.ModeSymlink == 0 {
					continue
				}
				previous, err := os.Readlink(linkPath)
				if err != nil {
					return Plan{}, err
				}
				if pointsInside(filepath.Join(absoluteRoot, "skills"), filepath.Dir(linkPath), previous) {
					plan.Changes = append(plan.Changes, Change{Operation: "remove_symlink", Path: linkPath, Previous: previous})
				}
			}
		}
	}
	sort.Slice(plan.Changes, func(i, j int) bool {
		if plan.Changes[i].Path == plan.Changes[j].Path {
			return plan.Changes[i].Operation < plan.Changes[j].Operation
		}
		return plan.Changes[i].Path < plan.Changes[j].Path
	})
	return plan, nil
}

func Apply(plan Plan) error {
	for _, change := range plan.Changes {
		if err := os.MkdirAll(filepath.Dir(change.Path), 0755); err != nil {
			return err
		}
		switch change.Operation {
		case "create_symlink":
			if _, err := os.Lstat(change.Path); err == nil {
				return &ConflictError{Path: change.Path, Reason: "target appeared since preview"}
			} else if !errors.Is(err, os.ErrNotExist) {
				return err
			}
			if err := os.Symlink(change.Target, change.Path); err != nil {
				return err
			}
		case "replace_symlink":
			if err := verifySymlink(change.Path, change.Previous); err != nil {
				return err
			}
			if err := os.Remove(change.Path); err != nil {
				return err
			}
			if err := os.Symlink(change.Target, change.Path); err != nil {
				_ = os.Symlink(change.Previous, change.Path)
				return err
			}
		case "remove_symlink":
			if err := verifySymlink(change.Path, change.Previous); err != nil {
				return err
			}
			if err := os.Remove(change.Path); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown skill mount operation %q", change.Operation)
		}
	}
	return nil
}

func scopeFor(relative string) string {
	parts := strings.Split(filepath.ToSlash(relative), "/")
	if len(parts) < 2 {
		return "shared"
	}
	switch parts[0] {
	case "codex":
		return "codex"
	case "claude", "claude-code":
		return "claude"
	case "cursor":
		return "cursor"
	case "agents":
		return "agents"
	default:
		return "shared"
	}
}

func checkMountPath(root, mountRoot string) error {
	for current := filepath.Dir(mountRoot); current != root; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return &ConflictError{Path: current, Reason: "mount parent is a symlink"}
		}
		if !info.IsDir() {
			return &ConflictError{Path: current, Reason: "mount parent is not a directory"}
		}
	}
	info, err := os.Lstat(mountRoot)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return &ConflictError{Path: mountRoot, Reason: "mount root must be a real directory"}
	}
	return nil
}

func pointsInside(root, linkDirectory, target string) bool {
	resolved := target
	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(linkDirectory, resolved)
	}
	resolved = filepath.Clean(resolved)
	relative, err := filepath.Rel(filepath.Clean(root), resolved)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func verifySymlink(path, expected string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return &ConflictError{Path: path, Reason: "target changed since preview"}
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return &ConflictError{Path: path, Reason: "target changed since preview"}
	}
	actual, err := os.Readlink(path)
	if err != nil {
		return err
	}
	if actual != expected {
		return &ConflictError{Path: path, Reason: "target changed since preview"}
	}
	return nil
}
