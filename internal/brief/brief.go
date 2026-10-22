// Package brief condenses a Harness Project into the short orientation a new
// agent session needs before it continues work.
package brief

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/chenhg5/harness-projects/internal/dashboard"
	"github.com/chenhg5/harness-projects/internal/runtime"
	"github.com/chenhg5/harness-projects/internal/store"
)

// DefaultLines bounds each free-text section so a brief stays readable at the
// start of a session. Callers pass 0 for the full text.
const DefaultLines = 30

// charsPerLine converts the line budget into a character budget, because
// plans are often written as a few very long paragraphs.
const charsPerLine = 120

// maxTasks bounds each task list unless the full brief is requested.
const maxTasks = 8

type Goal struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Outcome   string   `json:"outcome,omitempty"`
	DoneWhen  []string `json:"doneWhen,omitempty"`
	Completed int      `json:"completedTasks"`
	Total     int      `json:"taskCount"`
}

type Task struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Status   string `json:"status"`
	Priority string `json:"priority,omitempty"`
	Assignee string `json:"assignee,omitempty"`
	Reason   string `json:"reason,omitempty"`
}

type Note struct {
	Name      string `json:"name"`
	Excerpt   string `json:"excerpt"`
	Truncated bool   `json:"truncated,omitempty"`
}

type Brief struct {
	Project      string   `json:"project"`
	Path         string   `json:"path"`
	Phase        string   `json:"phase,omitempty"`
	Health       string   `json:"health,omitempty"`
	Progress     string   `json:"progress,omitempty"`
	UpdatedAt    string   `json:"updatedAt,omitempty"`
	Goal         *Goal    `json:"goal,omitempty"`
	NextActions  []string `json:"nextActions,omitempty"`
	Risks        []string `json:"risks,omitempty"`
	Plan         *Note    `json:"plan,omitempty"`
	Warnings     []string `json:"warnings,omitempty"`
	InProgress   []Task   `json:"inProgress"`
	Ready        []Task   `json:"ready"`
	Blocked      []Task   `json:"blocked"`
	MoreTasks    int      `json:"moreTasks,omitempty"`
	Worklog      *Note    `json:"latestWorklog,omitempty"`
	Handoff      *Note    `json:"latestHandoff,omitempty"`
	Repositories []string `json:"repositories,omitempty"`
	WhenDone     []string `json:"whenDone"`
}

// Load reads the project at root. maxLines limits each free-text section; 0 keeps everything.
func Load(root string, maxLines int) (Brief, error) {
	summary, err := dashboard.Load(root)
	if err != nil {
		return Brief{}, err
	}
	b := Brief{
		Project: summary.Name, Path: summary.Path,
		Phase: summary.State.CurrentPhase, Health: summary.State.Health, Progress: summary.State.Progress,
		UpdatedAt: summary.State.UpdatedAt, NextActions: summary.State.NextActions, Risks: summary.State.Risks,
		InProgress: []Task{}, Ready: []Task{}, Blocked: []Task{},
	}
	for _, goal := range summary.Goals {
		if goal.ID == summary.State.CurrentGoalID || (summary.State.CurrentGoalID == "" && goal.Status == "active") {
			b.Goal = &Goal{ID: goal.ID, Title: goal.Title, Outcome: goal.Outcome, DoneWhen: goal.DoneWhen, Completed: goal.CompletedTasks, Total: goal.TaskCount}
			break
		}
	}
	if err := b.loadTasks(summary.Path, maxLines > 0); err != nil {
		return Brief{}, err
	}
	plan, err := os.ReadFile(filepath.Join(summary.Path, "state", "current-plan.md"))
	if err == nil {
		b.Plan = excerpt("state/current-plan.md", string(plan), maxLines)
	} else if !errors.Is(err, os.ErrNotExist) {
		return Brief{}, fmt.Errorf("read current plan: %w", err)
	}
	if summary.LatestWorklog != nil {
		b.Worklog = excerpt("worklogs/"+summary.LatestWorklog.Name, focus(summary.LatestWorklog.Content, "Remaining", "Next action", "Blockers"), maxLines)
	}
	if summary.LatestHandoff != nil {
		b.Handoff = excerpt("handoffs/"+summary.LatestHandoff.Name, focus(summary.LatestHandoff.Content, "Next actions", "Blockers and risks"), maxLines)
	}
	if summary.LatestWorklog != nil && summary.State.UpdatedAt != "" && worklogTime(summary.LatestWorklog.Name) > summary.State.UpdatedAt {
		b.Warnings = append(b.Warnings, "The state snapshot is older than the latest work log; trust the work log and plan, then run hp state update.")
	}
	for _, repository := range summary.Repositories {
		b.Repositories = append(b.Repositories, repository.ID)
	}
	b.WhenDone = []string{
		"hp task update --project " + quote(b.Path) + " --id <task> --status <in_progress|completed|blocked>",
		"hp worklog create --project " + quote(b.Path) + " --id <slug> --agent <you> --objective ... --completed ... --remaining ... --next ...",
	}
	return b, nil
}

// worklogTime turns a work log file name (2006-01-02T15-04-05Z-id.md) into a
// comparable RFC 3339 timestamp; it returns "" for other names.
func worklogTime(name string) string {
	if len(name) < 20 || name[10] != 'T' || name[19] != 'Z' {
		return ""
	}
	return name[:13] + ":" + name[14:16] + ":" + name[17:20]
}

func (b *Brief) loadTasks(root string, limit bool) error {
	s, err := store.Open(root)
	if err != nil {
		return err
	}
	tasks, err := s.Tasks()
	if err != nil {
		return err
	}
	readiness, err := runtime.ReadyTasks(tasks)
	if err != nil {
		return err
	}
	for _, item := range readiness {
		task := item.Task
		line := Task{ID: task.ID, Title: task.Title, Status: task.Status, Priority: task.Priority, Reason: task.BlockedReason}
		if task.Assignment != nil && task.Assignment.EndedAt == "" {
			line.Assignee = task.Assignment.Harness + ":" + task.Assignment.SessionID
		}
		switch {
		case task.Status == "in_progress":
			b.InProgress = append(b.InProgress, line)
		case task.Status == "blocked":
			b.Blocked = append(b.Blocked, line)
		case item.Ready:
			b.Ready = append(b.Ready, line)
		}
	}
	for _, list := range []*[]Task{&b.InProgress, &b.Ready, &b.Blocked} {
		items := *list
		sort.SliceStable(items, func(i, j int) bool { return rank(items[i].Priority) < rank(items[j].Priority) })
		if limit && len(items) > maxTasks {
			b.MoreTasks += len(items) - maxTasks
			*list = items[:maxTasks]
		}
	}
	return nil
}

func rank(priority string) int {
	switch priority {
	case "critical":
		return 0
	case "high":
		return 1
	case "medium", "":
		return 2
	default:
		return 3
	}
}

// focus keeps only the named "## " sections when they exist, so a brief shows
// what is left rather than the whole history of a session.
func focus(content string, sections ...string) string {
	wanted := map[string]bool{}
	for _, name := range sections {
		wanted[strings.ToLower(name)] = true
	}
	var kept []string
	keep := false
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "## ") {
			keep = wanted[strings.ToLower(strings.TrimSpace(strings.TrimPrefix(line, "## ")))]
		}
		if keep {
			kept = append(kept, line)
		}
	}
	if len(kept) == 0 {
		return content
	}
	return strings.Join(kept, "\n")
}

func excerpt(name, content string, maxLines int) *Note {
	lines := strings.Split(strings.TrimSpace(content), "\n")
	note := &Note{Name: name}
	if maxLines > 0 && len(lines) > maxLines {
		lines = lines[:maxLines]
		note.Truncated = true
	}
	note.Excerpt = strings.Join(lines, "\n")
	if budget := maxLines * charsPerLine; maxLines > 0 && len(note.Excerpt) > budget {
		cut := strings.LastIndex(note.Excerpt[:budget], "\n")
		if cut <= 0 {
			cut = strings.LastIndexAny(note.Excerpt[:budget], " 。.")
		}
		if cut <= 0 {
			cut = budget
		}
		for cut > 0 && !utf8.RuneStart(note.Excerpt[cut]) {
			cut--
		}
		note.Excerpt = strings.TrimSpace(note.Excerpt[:cut])
		note.Truncated = true
	}
	return note
}

func quote(path string) string {
	if strings.ContainsAny(path, " \t'\"") {
		return "'" + strings.ReplaceAll(path, "'", `'\''`) + "'"
	}
	return path
}

// Markdown renders the brief for people and for agent context injection.
func (b Brief) Markdown() string {
	var w strings.Builder
	fmt.Fprintf(&w, "# Project brief: %s\n\n", b.Project)
	fmt.Fprintf(&w, "Path: %s\n", b.Path)
	if status := joinNonEmpty(" · ", label("phase", b.Phase), label("health", b.Health), label("updated", b.UpdatedAt)); status != "" {
		fmt.Fprintf(&w, "Status: %s\n", status)
	}
	if b.Progress != "" {
		fmt.Fprintf(&w, "Progress: %s\n", b.Progress)
	}
	for _, warning := range b.Warnings {
		fmt.Fprintf(&w, "\n> Warning: %s\n", warning)
	}
	if b.Goal != nil {
		fmt.Fprintf(&w, "\n## Goal: %s\n\n", b.Goal.Title)
		if b.Goal.Outcome != "" {
			fmt.Fprintf(&w, "%s\n\n", b.Goal.Outcome)
		}
		fmt.Fprintf(&w, "Tasks completed: %d/%d (`%s`)\n", b.Goal.Completed, b.Goal.Total, b.Goal.ID)
		if len(b.Goal.DoneWhen) > 0 {
			w.WriteString("\nDone when:\n")
			for _, item := range b.Goal.DoneWhen {
				fmt.Fprintf(&w, "- %s\n", item)
			}
		}
	} else {
		w.WriteString("\n## Goal\n\nNo active goal. Create one with `hp goal create`.\n")
	}
	if len(b.NextActions) > 0 || len(b.Risks) > 0 {
		w.WriteString("\n## Next actions\n\n")
		for _, item := range b.NextActions {
			fmt.Fprintf(&w, "- %s\n", item)
		}
		for _, item := range b.Risks {
			fmt.Fprintf(&w, "- Risk: %s\n", item)
		}
	}
	writeTasks(&w, "In progress", b.InProgress)
	writeTasks(&w, "Ready to start", b.Ready)
	writeTasks(&w, "Blocked", b.Blocked)
	if b.MoreTasks > 0 {
		fmt.Fprintf(&w, "\n%d more task(s) not shown; run hp task list or hp brief --full\n", b.MoreTasks)
	}
	writeNote(&w, "Current plan", b.Plan)
	writeNote(&w, "Latest work log", b.Worklog)
	writeNote(&w, "Latest handoff", b.Handoff)
	if len(b.Repositories) > 0 {
		fmt.Fprintf(&w, "\n## Repositories\n\n%s (worktrees under repos/)\n", strings.Join(b.Repositories, ", "))
	}
	w.WriteString("\n## When you finish or stop\n\n")
	for _, command := range b.WhenDone {
		fmt.Fprintf(&w, "- `%s`\n", command)
	}
	return w.String()
}

func writeTasks(w *strings.Builder, title string, tasks []Task) {
	if len(tasks) == 0 {
		return
	}
	fmt.Fprintf(w, "\n## %s\n\n", title)
	for _, task := range tasks {
		extra := joinNonEmpty(", ", task.Priority, task.Assignee, task.Reason)
		if extra != "" {
			extra = " (" + extra + ")"
		}
		fmt.Fprintf(w, "- `%s` %s%s\n", task.ID, task.Title, extra)
	}
}

func writeNote(w *strings.Builder, title string, note *Note) {
	if note == nil {
		return
	}
	fmt.Fprintf(w, "\n## %s — %s\n\n", title, note.Name)
	w.WriteString(demote(note.Excerpt))
	w.WriteString("\n")
	if note.Truncated {
		fmt.Fprintf(w, "\n… truncated; read %s or run hp brief --full\n", note.Name)
	}
}

// demote nests a document's headings below the brief's own section headings.
func demote(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "#") {
			lines[i] = "##" + line
		}
	}
	return strings.Join(lines, "\n")
}

func label(name, value string) string {
	if value == "" {
		return ""
	}
	return name + " " + value
}

func joinNonEmpty(sep string, values ...string) string {
	kept := values[:0:0]
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			kept = append(kept, value)
		}
	}
	return strings.Join(kept, sep)
}
