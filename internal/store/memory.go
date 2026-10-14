package store

import (
	"fmt"
	"path/filepath"

	"github.com/chenhg5/harness-projects/internal/model"
)

func (s Store) memoryPath(kind, id string) (string, error) {
	if err := model.ValidateID(id); err != nil {
		return "", err
	}
	if !validMemoryKind(kind) {
		return "", fmt.Errorf("invalid memory kind %q", kind)
	}
	return filepath.Join(s.Root, "context", "memory", kind, id+".json"), nil
}

func validMemoryKind(kind string) bool {
	for _, candidate := range model.MemoryKinds() {
		if kind == candidate {
			return true
		}
	}
	return false
}

func (s Store) SaveMemory(memory model.Memory, replace bool) error {
	if err := model.ValidateMemory(memory); err != nil {
		return err
	}
	path, err := s.memoryPath(memory.Kind, memory.ID)
	if err != nil {
		return err
	}
	return writeJSON(path, memory, replace)
}

func (s Store) Memory(kind, id string) (model.Memory, error) {
	path, err := s.memoryPath(kind, id)
	if err != nil {
		return model.Memory{}, err
	}
	return readJSON[model.Memory](path)
}

func (s Store) Memories(kind string) ([]model.Memory, error) {
	if kind != "" && !validMemoryKind(kind) {
		return nil, fmt.Errorf("invalid memory kind %q", kind)
	}
	kinds := model.MemoryKinds()
	if kind != "" {
		kinds = []string{kind}
	}
	all := []model.Memory{}
	for _, itemKind := range kinds {
		items, err := listJSON[model.Memory](filepath.Join(s.Root, "context", "memory", itemKind))
		if err != nil {
			return nil, err
		}
		all = append(all, items...)
	}
	return all, nil
}
