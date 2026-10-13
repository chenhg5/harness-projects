package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/chenhg5/harness-projects/internal/model"
	"github.com/chenhg5/harness-projects/internal/project"
)

type Store struct {
	Root string
}

func Open(root string) (Store, error) {
	resolved, err := project.ResolvePath(root)
	if err != nil {
		return Store{}, err
	}
	if status := project.Inspect(resolved); !status.Valid {
		return Store{}, fmt.Errorf("not a valid Harness Project: %s", strings.Join(status.Problems, "; "))
	}
	return Store{Root: resolved}, nil
}

func (s Store) resourcePath(kind, id string) (string, error) {
	if err := model.ValidateID(id); err != nil {
		return "", err
	}
	var directory string
	switch kind {
	case "goal", "task", "run":
		directory = kind + "s"
	default:
		return "", fmt.Errorf("unsupported resource kind %q", kind)
	}
	return filepath.Join(s.Root, directory, id+".json"), nil
}

func writeJSON(path string, value any, replace bool) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if !replace {
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err != nil {
			return err
		}
		if _, err := file.Write(data); err != nil {
			_ = file.Close()
			_ = os.Remove(path)
			return err
		}
		return file.Close()
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".hp-resource-*")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(temp.Name(), path)
}

func readJSON[T any](path string) (T, error) {
	var value T
	data, err := os.ReadFile(path)
	if err != nil {
		return value, err
	}
	if err := json.Unmarshal(data, &value); err != nil {
		return value, fmt.Errorf("invalid JSON in %s: %w", path, err)
	}
	return value, nil
}

func listJSON[T any](directory string) ([]T, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []T{}, nil
		}
		return nil, err
	}
	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		paths = append(paths, filepath.Join(directory, entry.Name()))
	}
	sort.Strings(paths)
	items := make([]T, 0, len(paths))
	for _, path := range paths {
		item, err := readJSON[T](path)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (s Store) Goal(id string) (model.Goal, error) {
	path, err := s.resourcePath("goal", id)
	if err != nil {
		return model.Goal{}, err
	}
	return readJSON[model.Goal](path)
}

func (s Store) Goals() ([]model.Goal, error) {
	return listJSON[model.Goal](filepath.Join(s.Root, "goals"))
}

func (s Store) SaveGoal(goal model.Goal, replace bool) error {
	if err := model.ValidateGoal(goal); err != nil {
		return err
	}
	path, err := s.resourcePath("goal", goal.ID)
	if err != nil {
		return err
	}
	return writeJSON(path, goal, replace)
}

func (s Store) Task(id string) (model.Task, error) {
	path, err := s.resourcePath("task", id)
	if err != nil {
		return model.Task{}, err
	}
	return readJSON[model.Task](path)
}

func (s Store) Tasks() ([]model.Task, error) {
	return listJSON[model.Task](filepath.Join(s.Root, "tasks"))
}

func (s Store) SaveTask(task model.Task, replace bool) error {
	if err := model.ValidateTask(task); err != nil {
		return err
	}
	path, err := s.resourcePath("task", task.ID)
	if err != nil {
		return err
	}
	return writeJSON(path, task, replace)
}

func (s Store) Run(id string) (model.Run, error) {
	path, err := s.resourcePath("run", id)
	if err != nil {
		return model.Run{}, err
	}
	return readJSON[model.Run](path)
}

func (s Store) Runs() ([]model.Run, error) {
	return listJSON[model.Run](filepath.Join(s.Root, "runs"))
}

func (s Store) SaveRun(run model.Run, replace bool) error {
	if err := model.ValidateRun(run); err != nil {
		return err
	}
	path, err := s.resourcePath("run", run.ID)
	if err != nil {
		return err
	}
	return writeJSON(path, run, replace)
}

func IsNotFound(err error) bool { return errors.Is(err, os.ErrNotExist) }
