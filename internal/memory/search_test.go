package memory

import (
	"testing"

	"github.com/chenhg5/harness-projects/internal/model"
)

func TestSearchRequiresAllTermsAndRanksTitle(t *testing.T) {
	items := []model.Memory{
		{ID: "body", Kind: "lesson", Title: "Worktree note", Content: "The API project uses a shared context file."},
		{ID: "title", Kind: "decision", Title: "Shared API context", Content: "Keep the project memory portable.", Tags: []string{"api"}},
		{ID: "other", Kind: "decision", Title: "Different choice", Content: "Unrelated content"},
	}
	matches := Search(items, "shared api")
	if len(matches) != 2 {
		t.Fatalf("matches = %d, want 2", len(matches))
	}
	if matches[0].Memory.ID != "title" {
		t.Fatalf("first match = %q, want title", matches[0].Memory.ID)
	}
}
