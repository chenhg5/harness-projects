package dashboard

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chenhg5/harness-projects/internal/model"
	"github.com/chenhg5/harness-projects/internal/project"
	"github.com/chenhg5/harness-projects/internal/store"
)

func TestDashboardReadsCurrentProjectDataAndEscapesContent(t *testing.T) {
	root := filepath.Join(t.TempDir(), "demo")
	plan, files, err := project.PlanInit(root, "Demo", false, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := project.ApplyInit(plan, files); err != nil {
		t.Fatal(err)
	}
	s, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	goal := model.Goal{SchemaVersion: "1", ID: "launch", Title: "Launch <script>alert(1)</script>", Description: "Launch the project", Outcome: "Users can use it", DoneWhen: []string{"Release is verified"}, Status: "active", CreatedAt: "2026-10-14T00:00:00Z", UpdatedAt: "2026-10-14T00:00:00Z"}
	if err := s.SaveGoal(goal, false); err != nil {
		t.Fatal(err)
	}
	task := model.Task{SchemaVersion: "1", ID: "release", GoalID: "launch", Title: "Release", Status: "ready", CreatedAt: "2026-10-14T00:00:00Z", UpdatedAt: "2026-10-14T00:00:00Z"}
	if err := s.SaveTask(task, false); err != nil {
		t.Fatal(err)
	}
	handler := Handler(root)

	request := httptest.NewRequest(http.MethodGet, "http://127.0.0.1/", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "Release is verified") {
		t.Fatalf("unexpected dashboard response: code=%d body=%s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), "<script>alert(1)</script>") {
		t.Fatal("goal title was not escaped")
	}

	if err := s.SaveTask(model.Task{SchemaVersion: "1", ID: "release", GoalID: "launch", Title: "Release", Status: "completed", Result: "verified", CreatedAt: task.CreatedAt, UpdatedAt: task.UpdatedAt}, true); err != nil {
		t.Fatal(err)
	}
	api := httptest.NewRecorder()
	handler.ServeHTTP(api, httptest.NewRequest(http.MethodGet, "http://127.0.0.1/api/project", nil))
	var summary Summary
	if err := json.Unmarshal(api.Body.Bytes(), &summary); err != nil {
		t.Fatal(err)
	}
	if summary.TaskPercent != 100 || summary.Goals[0].TaskPercent != 100 {
		t.Fatalf("dashboard did not pick up updated task: %+v", summary)
	}
	if summary.Goals[0].Status != "active" {
		t.Fatal("task completion must not imply goal completion")
	}
}

func TestDashboardRejectsForeignHostsAndWrites(t *testing.T) {
	handler := Handler(t.TempDir())
	foreign := httptest.NewRecorder()
	handler.ServeHTTP(foreign, httptest.NewRequest(http.MethodGet, "http://example.com/", nil))
	if foreign.Code != http.StatusForbidden {
		t.Fatalf("foreign host code=%d", foreign.Code)
	}
	write := httptest.NewRecorder()
	handler.ServeHTTP(write, httptest.NewRequest(http.MethodPost, "http://127.0.0.1/api/project", nil))
	if write.Code != http.StatusMethodNotAllowed {
		t.Fatalf("write code=%d", write.Code)
	}
}
