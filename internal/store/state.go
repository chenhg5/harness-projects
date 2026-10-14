package store

import (
	"path/filepath"

	"github.com/chenhg5/harness-projects/internal/model"
)

func (s Store) State() (model.ProjectState, error) {
	return readJSON[model.ProjectState](filepath.Join(s.Root, "state", "project-state.json"))
}

func (s Store) SaveState(state model.ProjectState) error {
	if err := model.ValidateProjectState(state); err != nil {
		return err
	}
	return writeJSON(filepath.Join(s.Root, "state", "project-state.json"), state, true)
}
