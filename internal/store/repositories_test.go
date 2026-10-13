package store

import (
	"path/filepath"
	"testing"

	"github.com/chenhg5/harness-projects/internal/model"
)

func TestRepositoryAndWorktreeRoundTrip(t *testing.T) {
	s := testStore(t)
	root := s.Root
	repository := model.Repository{
		SchemaVersion: model.ResourceSchemaVersion,
		ID:            "api",
		Path:          filepath.Join(t.TempDir(), "api"),
		CreatedAt:     "2026-01-01T00:00:00Z",
		UpdatedAt:     "2026-01-01T00:00:00Z",
	}
	if err := s.SaveRepository(repository, false); err != nil {
		t.Fatal(err)
	}
	worktree := model.Worktree{
		SchemaVersion: model.ResourceSchemaVersion,
		ID:            "api-task",
		RepositoryID:  "api",
		Path:          filepath.Join(root, "repos", "api", "api-task"),
		Branch:        "hp/api/api-task",
		Status:        "active",
		CreatedAt:     repository.CreatedAt,
		UpdatedAt:     repository.UpdatedAt,
	}
	if err := s.SaveWorktree(worktree, false); err != nil {
		t.Fatal(err)
	}
	gotRepository, err := s.Repository("api")
	if err != nil {
		t.Fatal(err)
	}
	if gotRepository.Path != repository.Path {
		t.Fatalf("repository path = %q, want %q", gotRepository.Path, repository.Path)
	}
	gotWorktree, err := s.Worktree("api-task")
	if err != nil {
		t.Fatal(err)
	}
	if gotWorktree.Branch != worktree.Branch {
		t.Fatalf("worktree branch = %q, want %q", gotWorktree.Branch, worktree.Branch)
	}
}
