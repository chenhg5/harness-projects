package store

import (
	"os"
	"path/filepath"

	"github.com/chenhg5/harness-projects/internal/model"
)

func (s Store) metadataPath(kind, id string) (string, error) {
	if err := model.ValidateID(id); err != nil {
		return "", err
	}
	return filepath.Join(s.Root, ".harness-project", kind, id+".json"), nil
}

func (s Store) Repository(id string) (model.Repository, error) {
	path, err := s.metadataPath("repositories", id)
	if err != nil {
		return model.Repository{}, err
	}
	return readJSON[model.Repository](path)
}

func (s Store) Repositories() ([]model.Repository, error) {
	return listJSON[model.Repository](filepath.Join(s.Root, ".harness-project", "repositories"))
}

func (s Store) SaveRepository(repository model.Repository, replace bool) error {
	if err := model.ValidateRepository(repository); err != nil {
		return err
	}
	path, err := s.metadataPath("repositories", repository.ID)
	if err != nil {
		return err
	}
	return writeJSON(path, repository, replace)
}

func (s Store) Worktree(id string) (model.Worktree, error) {
	path, err := s.metadataPath("worktrees", id)
	if err != nil {
		return model.Worktree{}, err
	}
	return readJSON[model.Worktree](path)
}

func (s Store) Worktrees() ([]model.Worktree, error) {
	return listJSON[model.Worktree](filepath.Join(s.Root, ".harness-project", "worktrees"))
}

func (s Store) SaveWorktree(worktree model.Worktree, replace bool) error {
	if err := model.ValidateWorktree(worktree); err != nil {
		return err
	}
	path, err := s.metadataPath("worktrees", worktree.ID)
	if err != nil {
		return err
	}
	return writeJSON(path, worktree, replace)
}

func (s Store) DeleteWorktree(id string) error {
	path, err := s.metadataPath("worktrees", id)
	if err != nil {
		return err
	}
	return os.Remove(path)
}
