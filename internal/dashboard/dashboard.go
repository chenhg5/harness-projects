package dashboard

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/chenhg5/harness-projects/internal/handoff"
	"github.com/chenhg5/harness-projects/internal/model"
	"github.com/chenhg5/harness-projects/internal/project"
	"github.com/chenhg5/harness-projects/internal/store"
	"github.com/chenhg5/harness-projects/internal/worklog"
)

//go:embed assets/index.html assets/style.css
var assets embed.FS

var page = template.Must(template.ParseFS(assets, "assets/index.html"))

type GoalView struct {
	model.Goal
	Document       string `json:"document,omitempty"`
	TaskCount      int    `json:"taskCount"`
	CompletedTasks int    `json:"completedTasks"`
	TaskPercent    int    `json:"taskPercent"`
}

type Note struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

type Summary struct {
	Name          string             `json:"name"`
	Path          string             `json:"path"`
	State         model.ProjectState `json:"state"`
	Goals         []GoalView         `json:"goals"`
	Tasks         []model.Task       `json:"tasks"`
	Repositories  []model.Repository `json:"repositories"`
	LatestWorklog *Note              `json:"latestWorklog,omitempty"`
	LatestHandoff *Note              `json:"latestHandoff,omitempty"`
	TaskCount     int                `json:"taskCount"`
	Completed     int                `json:"completed"`
	Active        int                `json:"active"`
	Blocked       int                `json:"blocked"`
	TaskPercent   int                `json:"taskPercent"`
}

func Load(root string) (Summary, error) {
	s, err := store.Open(root)
	if err != nil {
		return Summary{}, err
	}
	state, err := s.State()
	if err != nil {
		return Summary{}, fmt.Errorf("read project state: %w", err)
	}
	goals, err := s.Goals()
	if err != nil {
		return Summary{}, fmt.Errorf("read goals: %w", err)
	}
	tasks, err := s.Tasks()
	if err != nil {
		return Summary{}, fmt.Errorf("read tasks: %w", err)
	}
	repositories, err := s.Repositories()
	if err != nil {
		return Summary{}, fmt.Errorf("read repositories: %w", err)
	}
	status := project.Inspect(s.Root)
	result := Summary{Path: s.Root, Name: status.Name, State: state, Goals: make([]GoalView, 0, len(goals)), Tasks: tasks, Repositories: repositories, TaskCount: len(tasks)}
	for _, task := range tasks {
		switch task.Status {
		case "completed":
			result.Completed++
		case "in_progress":
			result.Active++
		case "blocked":
			result.Blocked++
		}
	}
	result.TaskPercent = percent(result.Completed, result.TaskCount)
	for _, goal := range goals {
		view := GoalView{Goal: goal}
		path := filepath.Join(s.Root, "goals", goal.ID+".md")
		if info, err := os.Lstat(path); err == nil && info.Mode().IsRegular() {
			data, err := os.ReadFile(path)
			if err != nil {
				return Summary{}, fmt.Errorf("read goal document %s: %w", goal.ID, err)
			}
			view.Document = string(data)
		} else if err != nil && !errors.Is(err, os.ErrNotExist) {
			return Summary{}, fmt.Errorf("read goal document %s: %w", goal.ID, err)
		}
		for _, task := range tasks {
			if task.GoalID == goal.ID {
				view.TaskCount++
				if task.Status == "completed" {
					view.CompletedTasks++
				}
			}
		}
		view.TaskPercent = percent(view.CompletedTasks, view.TaskCount)
		result.Goals = append(result.Goals, view)
	}
	sort.SliceStable(result.Goals, func(i, j int) bool {
		if result.Goals[i].ID == state.CurrentGoalID {
			return true
		}
		if result.Goals[j].ID == state.CurrentGoalID {
			return false
		}
		return result.Goals[i].ID < result.Goals[j].ID
	})
	if name, content, err := worklog.Latest(s.Root); err == nil {
		result.LatestWorklog = &Note{Name: name, Content: content}
	} else if !errors.Is(err, os.ErrNotExist) {
		return Summary{}, fmt.Errorf("read latest work log: %w", err)
	}
	if name, content, err := handoff.Latest(s.Root); err == nil {
		result.LatestHandoff = &Note{Name: name, Content: content}
	} else if !errors.Is(err, os.ErrNotExist) {
		return Summary{}, fmt.Errorf("read latest handoff: %w", err)
	}
	return result, nil
}

func percent(done, total int) int {
	if total == 0 {
		return 0
	}
	return done * 100 / total
}

// Handler serves only a read-only summary. It never exposes arbitrary project files.
func Handler(root string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		summary, err := Load(root)
		if err != nil {
			http.Error(w, "Could not read project data", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := page.Execute(w, summary); err != nil {
			return
		}
	})
	mux.HandleFunc("GET /style.css", func(w http.ResponseWriter, r *http.Request) {
		data, err := assets.ReadFile("assets/style.css")
		if err != nil {
			http.Error(w, "Styles unavailable", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		_, _ = w.Write(data)
	})
	mux.HandleFunc("GET /api/project", func(w http.ResponseWriter, r *http.Request) {
		summary, err := Load(root)
		if err != nil {
			http.Error(w, "Could not read project data", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(summary)
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if index := strings.LastIndex(host, ":"); index >= 0 {
			host = host[:index]
		}
		if host != "127.0.0.1" && host != "localhost" {
			http.Error(w, "Invalid host", http.StatusForbidden)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'self' 'unsafe-inline'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'")
		mux.ServeHTTP(w, r)
	})
}
