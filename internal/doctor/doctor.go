package doctor

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	hpgit "github.com/chenhg5/harness-projects/internal/git"
	"github.com/chenhg5/harness-projects/internal/layout"
	"github.com/chenhg5/harness-projects/internal/model"
	"github.com/chenhg5/harness-projects/internal/project"
	"github.com/chenhg5/harness-projects/internal/skillmount"
	"github.com/chenhg5/harness-projects/internal/store"
	"github.com/chenhg5/harness-projects/internal/worklog"
)

type Finding struct {
	Code       string `json:"code"`
	Severity   string `json:"severity"`
	Message    string `json:"message"`
	Path       string `json:"path,omitempty"`
	Suggestion string `json:"suggestion,omitempty"`
}

type Report struct {
	Action   string    `json:"action"`
	Project  string    `json:"project"`
	Healthy  bool      `json:"healthy"`
	Findings []Finding `json:"findings"`
}

func Inspect(root string) (Report, error) {
	root, err := project.ResolvePath(root)
	if err != nil {
		return Report{}, err
	}
	report := Report{Action: "project.doctor", Project: root, Findings: []Finding{}}
	status := project.Inspect(root)
	for _, problem := range status.Problems {
		report.add("invalid_structure", "error", problem, "", "Repair the project structure before continuing")
	}
	if !status.Valid {
		return report, nil
	}
	contract, exists, layoutErr := layout.Load(root)
	if layoutErr != nil {
		report.add("invalid_layout_contract", "error", layoutErr.Error(), layout.RelativePath, "Repair the contract or regenerate it with hp project upgrade")
		contract = layout.Default()
	} else if !exists {
		report.add("missing_layout_contract", "warning", "project directory contract is missing", layout.RelativePath, "Run hp project upgrade --path .")
	}
	checkProjectDirectories(&report, root, contract)
	s, err := store.Open(root)
	if err != nil {
		return Report{}, err
	}
	state, err := s.State()
	if err != nil {
		report.add("invalid_state", "error", err.Error(), "state/project-state.json", "Run hp state validate and repair the state file")
	} else {
		if err := model.ValidateProjectState(state); err != nil {
			report.add("invalid_state", "error", err.Error(), "state/project-state.json", "Run hp state update with the required fields")
		}
		checkStateFreshness(&report, root, state)
		checkStateTasks(&report, s, state)
		checkSourceRefs(&report, root, state.SourceRefs)
	}
	checkSkillMounts(&report, root)
	checkRepositories(&report, s)
	checkWorklogs(&report, root)
	report.Healthy = len(report.Findings) == 0
	sort.SliceStable(report.Findings, func(i, j int) bool {
		if report.Findings[i].Severity != report.Findings[j].Severity {
			return report.Findings[i].Severity == "error"
		}
		if report.Findings[i].Code != report.Findings[j].Code {
			return report.Findings[i].Code < report.Findings[j].Code
		}
		return report.Findings[i].Path < report.Findings[j].Path
	})
	return report, nil
}

func checkProjectDirectories(report *Report, root string, contract layout.Contract) {
	entries, err := os.ReadDir(root)
	if err != nil {
		report.add("layout_scan_error", "error", err.Error(), ".", "Repair project directory permissions")
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			if entry.Type()&os.ModeSymlink == 0 {
				continue
			}
			info, err := os.Stat(filepath.Join(root, entry.Name()))
			if err != nil || !info.IsDir() {
				continue
			}
		}
		if layout.Allowed(contract, entry.Name()) {
			continue
		}
		report.add(
			"unexpected_project_directory",
			"warning",
			"top-level directory is not registered in the project layout",
			entry.Name(),
			fmt.Sprintf("Move it into a registered location or run hp layout add --project . --name %q --description %q", entry.Name(), "Describe what belongs here"),
		)
	}
}

func (r *Report) add(code, severity, message, path, suggestion string) {
	r.Findings = append(r.Findings, Finding{Code: code, Severity: severity, Message: message, Path: path, Suggestion: suggestion})
}

func checkStateFreshness(report *Report, root string, state model.ProjectState) {
	names, err := worklog.List(root)
	if err != nil || len(names) == 0 {
		return
	}
	latest := names[len(names)-1]
	if len(latest) < 20 {
		return
	}
	worklogAt, err := time.Parse("2006-01-02T15-04-05Z", latest[:20])
	if err != nil {
		return
	}
	stateAt, err := time.Parse(time.RFC3339, state.UpdatedAt)
	if err == nil && worklogAt.After(stateAt) {
		report.add("stale_state", "warning", "state snapshot is older than the latest work log", "state/project-state.json", "Use hp finish for session closeout or run hp state reconcile")
	}
}

func checkStateTasks(report *Report, s store.Store, state model.ProjectState) {
	tasks, err := s.Tasks()
	if err != nil {
		report.add("invalid_tasks", "error", err.Error(), "tasks", "Repair invalid task records")
		return
	}
	active := []string{}
	blocked := []string{}
	known := map[string]bool{}
	for _, task := range tasks {
		known[task.ID] = true
		if task.Status == "in_progress" {
			active = append(active, task.ID)
		}
		if task.Status == "blocked" {
			blocked = append(blocked, task.ID)
		}
	}
	sort.Strings(active)
	sort.Strings(blocked)
	if !sameStrings(active, state.ActiveTaskIDs) {
		report.add("task_state_drift", "warning", fmt.Sprintf("activeTaskIds=%v but task records say %v", state.ActiveTaskIDs, active), "state/project-state.json", "Run hp state reconcile")
	}
	if !sameStrings(blocked, state.BlockedTaskIDs) {
		report.add("task_state_drift", "warning", fmt.Sprintf("blockedTaskIds=%v but task records say %v", state.BlockedTaskIDs, blocked), "state/project-state.json", "Run hp state reconcile")
	}
	if state.CurrentGoalID != "" {
		if _, err := s.Goal(state.CurrentGoalID); err != nil {
			report.add("missing_current_goal", "error", "currentGoalId does not resolve to a goal record", "state/project-state.json", "Set --current-goal-id to an existing goal")
		}
	}
	for _, id := range append(append([]string{}, state.ActiveTaskIDs...), state.BlockedTaskIDs...) {
		if !known[id] {
			report.add("missing_state_task", "error", "state references missing task "+id, "state/project-state.json", "Remove the stale task id or recreate the task")
		}
	}
}

func checkSourceRefs(report *Report, root string, refs []string) {
	for _, ref := range refs {
		if strings.Contains(ref, "://") || strings.ContainsAny(ref, "*?[") {
			continue
		}
		path := ref
		if !filepath.IsAbs(path) {
			path = filepath.Join(root, filepath.FromSlash(ref))
		}
		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			report.add("missing_source_ref", "warning", "state source reference does not exist", ref, "Update sourceRefs to point at current evidence")
		}
	}
}

func checkSkillMounts(report *Report, root string) {
	targets, _ := skillmount.ParseTargets("all")
	plan, err := skillmount.BuildPlan(root, targets, false)
	if err != nil {
		report.add("skill_mount_error", "error", err.Error(), "skills", "Resolve the conflict, then run hp project upgrade")
		return
	}
	if len(plan.Changes) > 0 {
		report.add("skill_mount_drift", "warning", fmt.Sprintf("%d skill mount change(s) are pending", len(plan.Changes)), "skills", "Run hp project upgrade or hp skill sync --targets all")
	}
}

func checkRepositories(report *Report, s store.Store) {
	registered := map[string]bool{}
	repositories, err := s.Repositories()
	if err != nil {
		report.add("invalid_repositories", "error", err.Error(), ".harness-project/repositories", "Repair invalid repository records")
		return
	}
	for _, repository := range repositories {
		if repository.Path == "" {
			continue
		}
		registered[filepath.Clean(repository.Path)] = true
		if _, err := os.Stat(repository.Path); errors.Is(err, os.ErrNotExist) {
			report.add("missing_repository_path", "warning", "registered repository path does not exist", repository.Path, "Update or remove the repository registration")
		}
	}
	worktrees, err := s.Worktrees()
	if err != nil {
		report.add("invalid_worktrees", "error", err.Error(), ".harness-project/worktrees", "Repair invalid worktree records")
		return
	}
	for _, worktree := range worktrees {
		if worktree.Status == "removed" {
			continue
		}
		registered[filepath.Clean(worktree.Path)] = true
		if _, err := os.Stat(worktree.Path); errors.Is(err, os.ErrNotExist) {
			report.add("missing_worktree_path", "warning", "registered worktree path does not exist", worktree.Path, "Remove the stale record or recreate the worktree")
		}
	}
	for _, path := range discoverCheckouts(filepath.Join(s.Root, "repos")) {
		if !registered[filepath.Clean(path)] {
			relative, _ := filepath.Rel(s.Root, path)
			report.add("unregistered_checkout", "warning", "Git checkout under repos/ is not registered", filepath.ToSlash(relative), "Register the repository/worktree so hp can track its lifecycle")
		}
	}
}

func discoverCheckouts(root string) []string {
	paths := []string{}
	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || !entry.IsDir() {
			return nil
		}
		if path != root && hpgit.IsRepositoryRoot(path) {
			paths = append(paths, path)
			return filepath.SkipDir
		}
		if relative, relErr := filepath.Rel(root, path); relErr == nil && strings.Count(filepath.ToSlash(relative), "/") >= 4 {
			return filepath.SkipDir
		}
		return nil
	})
	sort.Strings(paths)
	return paths
}

func checkWorklogs(report *Report, root string) {
	names, err := worklog.List(root)
	if err != nil {
		return
	}
	invalid := 0
	examples := []string{}
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(root, "worklogs", name))
		if err != nil {
			continue
		}
		if missing := worklog.MissingCoreFields(string(data)); len(missing) > 0 {
			invalid++
			if len(examples) < 3 {
				examples = append(examples, name+" ("+strings.Join(missing, ", ")+")")
			}
		}
	}
	if invalid > 0 {
		report.add("incomplete_worklogs", "warning", fmt.Sprintf("%d work log(s) have empty or missing core fields in the CLI template; examples: %s", invalid, strings.Join(examples, "; ")), "worklogs", "Review the examples and record objective, completed work, and unfinished work or next action; use hp finish for new logs")
	}
}

func sameStrings(left, right []string) bool {
	left = append([]string(nil), left...)
	right = append([]string(nil), right...)
	sort.Strings(left)
	sort.Strings(right)
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
