package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBriefSummarizesProjectFromNestedDirectory(t *testing.T) {
	root := filepath.Join(t.TempDir(), "demo")
	steps := [][]string{
		{"project", "init", "--path", root, "--name", "Demo"},
		{"goal", "create", "--project", root, "--id", "ship", "--title", "Ship the API", "--description", "Ship a stable API.", "--outcome", "Clients use the API in production.", "--done-when", "first client is live", "--status", "active"},
		{"task", "create", "--project", root, "--id", "schema", "--goal-id", "ship", "--title", "Design schema", "--priority", "high", "--status", "ready"},
		{"task", "create", "--project", root, "--id", "client", "--goal-id", "ship", "--title", "Onboard client", "--depends-on", "schema"},
		{"state", "update", "--project", root, "--current-goal-id", "ship", "--next-actions", "Design the schema"},
		{"worklog", "create", "--project", root, "--id", "kickoff", "--agent", "codex", "--objective", "Start", "--completed", "Set up goal", "--remaining", "Schema not designed", "--next", "Design the schema"},
	}
	for _, step := range steps {
		if code, _, stderr := run(t, step...); code != 0 {
			t.Fatalf("%v code=%d stderr=%s", step, code, stderr)
		}
	}
	nested := filepath.Join(root, "repos", "api")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatal(err)
	}

	code, stdout, stderr := run(t, "brief", "--project", nested, "--json")
	if code != 0 {
		t.Fatalf("brief code=%d stderr=%s", code, stderr)
	}
	var result struct {
		Path        string                   `json:"path"`
		Goal        struct{ ID string }      `json:"goal"`
		NextActions []string                 `json:"nextActions"`
		Ready       []struct{ ID string }    `json:"ready"`
		Worklog     struct{ Excerpt string } `json:"latestWorklog"`
	}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("decode: %v\n%s", err, stdout)
	}
	resolved, _ := filepath.Abs(root)
	if result.Path != resolved || result.Goal.ID != "ship" || len(result.NextActions) != 1 {
		t.Fatalf("unexpected brief: %s", stdout)
	}
	if len(result.Ready) != 1 || result.Ready[0].ID != "schema" {
		t.Fatalf("only the unblocked task should be ready: %s", stdout)
	}
	if !strings.Contains(result.Worklog.Excerpt, "Schema not designed") || strings.Contains(result.Worklog.Excerpt, "Set up goal") {
		t.Fatalf("work log excerpt should focus on what remains: %q", result.Worklog.Excerpt)
	}

	code, stdout, _ = run(t, "brief", "--project", root, "--format", "text")
	if code != 0 || !strings.Contains(stdout, "## Goal: Ship the API") || !strings.Contains(stdout, "`schema` Design schema") {
		t.Fatalf("unexpected text brief code=%d:\n%s", code, stdout)
	}
}

func TestBriefOutsideProject(t *testing.T) {
	code, _, stderr := run(t, "brief", "--project", t.TempDir())
	if code != 3 || !strings.Contains(stderr, "not_found") {
		t.Fatalf("expected not_found, code=%d stderr=%s", code, stderr)
	}
}

func TestBriefTruncatesLongPlans(t *testing.T) {
	root := filepath.Join(t.TempDir(), "demo")
	if code, _, stderr := run(t, "project", "init", "--path", root); code != 0 {
		t.Fatalf("init code=%d stderr=%s", code, stderr)
	}
	plan := "# Plan\n\n" + strings.Repeat("长段落 long paragraph ", 2000)
	if err := os.WriteFile(filepath.Join(root, "state", "current-plan.md"), []byte(plan), 0644); err != nil {
		t.Fatal(err)
	}
	code, stdout, _ := run(t, "brief", "--project", root, "--lines", "5", "--format", "text")
	if code != 0 || len(stdout) > 3000 || !strings.Contains(stdout, "truncated") {
		t.Fatalf("expected a truncated brief, code=%d len=%d", code, len(stdout))
	}
	code, stdout, _ = run(t, "brief", "--project", root, "--full", "--format", "text")
	if code != 0 || len(stdout) < len(plan) {
		t.Fatalf("--full should keep the whole plan, len=%d", len(stdout))
	}
}
