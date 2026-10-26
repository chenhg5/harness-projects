package worklog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRecordMarkdownAndList(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "worklogs"), 0755); err != nil {
		t.Fatal(err)
	}
	record := Record{ID: "log-1", Agent: "codex", SessionID: "session-1", CreatedAt: "2026-01-02T03:04:05Z", Objective: "build", Completed: "code", Remaining: "docs", Next: "test"}
	if err := record.Validate(); err != nil {
		t.Fatal(err)
	}
	name, err := record.Filename()
	if err != nil {
		t.Fatal(err)
	}
	if name != "2026-01-02T03-04-05Z-log-1.md" {
		t.Fatalf("unexpected filename %s", name)
	}
	if got := record.Markdown(); !strings.Contains(got, "## Remaining") {
		t.Fatal("markdown omitted remaining section")
	}
	if got := record.Markdown(); !strings.Contains(got, "- Session ID: session-1") {
		t.Fatal("markdown omitted provider session ID")
	}
	if _, err := time.Parse(time.RFC3339, record.CreatedAt); err != nil {
		t.Fatal(err)
	}
}
