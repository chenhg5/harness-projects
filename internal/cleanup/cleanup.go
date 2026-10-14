package cleanup

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var DefaultSections = []string{"worklogs", "artifacts", "context/imports"}

type Candidate struct {
	Path        string    `json:"path"`
	Section     string    `json:"section"`
	Size        int64     `json:"size"`
	ModifiedAt  time.Time `json:"modifiedAt"`
	Destination string    `json:"destination,omitempty"`
}

type Plan struct {
	Action     string      `json:"action"`
	Project    string      `json:"project"`
	OlderThan  string      `json:"olderThan"`
	Cutoff     time.Time   `json:"cutoff"`
	Sections   []string    `json:"sections"`
	Candidates []Candidate `json:"candidates"`
}

func ParseAge(value string) (time.Duration, error) {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return 30 * 24 * time.Hour, nil
	}
	if strings.HasSuffix(value, "d") {
		var days int
		if _, err := fmt.Sscanf(strings.TrimSuffix(value, "d"), "%d", &days); err != nil || days <= 0 {
			return 0, fmt.Errorf("invalid age %q; use a positive duration such as 30d or 720h", value)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	if strings.HasSuffix(value, "w") {
		var weeks int
		if _, err := fmt.Sscanf(strings.TrimSuffix(value, "w"), "%d", &weeks); err != nil || weeks <= 0 {
			return 0, fmt.Errorf("invalid age %q; use a positive duration such as 4w or 720h", value)
		}
		return time.Duration(weeks) * 7 * 24 * time.Hour, nil
	}
	duration, err := time.ParseDuration(value)
	if err != nil || duration <= 0 {
		return 0, fmt.Errorf("invalid age %q; use a positive duration such as 30d or 720h", value)
	}
	return duration, nil
}

func NormalizeSections(value string) ([]string, error) {
	if strings.TrimSpace(value) == "" {
		return append([]string(nil), DefaultSections...), nil
	}
	seen := map[string]bool{}
	sections := []string{}
	for _, raw := range strings.Split(value, ",") {
		section := filepath.ToSlash(strings.TrimSpace(raw))
		if section == "" || seen[section] {
			continue
		}
		if section != "worklogs" && section != "artifacts" && section != "context/imports" {
			return nil, fmt.Errorf("section %q is not transient; cleanup only supports worklogs, artifacts, and context/imports", section)
		}
		seen[section] = true
		sections = append(sections, section)
	}
	if len(sections) == 0 {
		return nil, fmt.Errorf("at least one cleanup section is required")
	}
	return sections, nil
}

func Build(root, olderThan, sectionsValue, action string, now time.Time) (Plan, error) {
	age, err := ParseAge(olderThan)
	if err != nil {
		return Plan{}, err
	}
	sections, err := NormalizeSections(sectionsValue)
	if err != nil {
		return Plan{}, err
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return Plan{}, err
	}
	if action == "" {
		action = "archive"
	}
	if action != "archive" && action != "delete" {
		return Plan{}, fmt.Errorf("invalid cleanup action %q; use archive or delete", action)
	}
	cutoff := now.Add(-age)
	plan := Plan{Action: action, Project: root, OlderThan: olderThan, Cutoff: cutoff, Sections: sections, Candidates: []Candidate{}}
	for _, section := range sections {
		directory := filepath.Join(root, filepath.FromSlash(section))
		entries, err := os.ReadDir(directory)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return Plan{}, err
		}
		_ = entries
		err = filepath.WalkDir(directory, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			if entry.Name() == "README.md" || strings.HasPrefix(entry.Name(), ".") {
				return nil
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			if !info.Mode().IsRegular() || info.ModTime().After(cutoff) {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			candidate := Candidate{Path: path, Section: section, Size: info.Size(), ModifiedAt: info.ModTime()}
			if action == "archive" {
				date := info.ModTime().UTC().Format("2006-01-02")
				candidate.Destination = filepath.Join(root, "archive", filepath.FromSlash(rel), "..")
				candidate.Destination = filepath.Join(root, "archive", filepath.FromSlash(section), date, filepath.Base(path))
			}
			plan.Candidates = append(plan.Candidates, candidate)
			return nil
		})
		if err != nil {
			return Plan{}, err
		}
	}
	sort.Slice(plan.Candidates, func(i, j int) bool { return plan.Candidates[i].Path < plan.Candidates[j].Path })
	return plan, nil
}

func Apply(plan Plan, force bool) (int, error) {
	if plan.Action == "delete" && !force {
		return 0, fmt.Errorf("delete requires --force; use archive (the default) for reversible cleanup")
	}
	changed := 0
	for _, candidate := range plan.Candidates {
		info, err := os.Lstat(candidate.Path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return changed, err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return changed, fmt.Errorf("cleanup target is not a regular file: %s", candidate.Path)
		}
		if !info.ModTime().Equal(candidate.ModifiedAt) || info.Size() != candidate.Size {
			return changed, fmt.Errorf("cleanup target changed since preview: %s", candidate.Path)
		}
		if plan.Action == "delete" {
			if err := os.Remove(candidate.Path); err != nil {
				return changed, err
			}
		} else {
			destination := candidate.Destination
			if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
				return changed, err
			}
			if _, err := os.Stat(destination); err == nil {
				for i := 1; ; i++ {
					candidatePath := fmt.Sprintf("%s-%d", destination, i)
					if _, statErr := os.Stat(candidatePath); os.IsNotExist(statErr) {
						destination = candidatePath
						break
					}
				}
			}
			if err := os.Rename(candidate.Path, destination); err != nil {
				return changed, err
			}
		}
		changed++
	}
	return changed, nil
}
