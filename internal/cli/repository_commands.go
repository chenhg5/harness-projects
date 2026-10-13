package cli

import (
	"fmt"
	"path/filepath"

	"github.com/chenhg5/harness-projects/internal/model"
	"github.com/chenhg5/harness-projects/internal/project"
	"github.com/chenhg5/harness-projects/internal/store"
)

func (a app) repositoryAdd(args []string) int {
	fs := newFlags("repository add")
	projectPath := fs.String("project", "", "project directory (required)")
	id := fs.String("id", "", "repository id (required)")
	url := fs.String("url", "", "repository URL")
	localPath := fs.String("path", "", "local repository path")
	defaultBranch := fs.String("default-branch", "", "default branch or revision")
	provider := fs.String("provider", "", "repository provider")
	format, jsonFlag := outputFlags(fs)
	dryRun := fs.Bool("dry-run", false, "preview changes without writing")
	ifNotExists := fs.Bool("if-not-exists", false, "succeed if the repository already exists")
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	if *projectPath == "" || *id == "" {
		return a.fail(failure{2, "invalid_arguments", "--project and --id are required", "Run hp repository add --help", ""})
	}
	resolvedPath := *localPath
	if resolvedPath != "" {
		var err error
		resolvedPath, err = filepath.Abs(resolvedPath)
		if err != nil {
			return a.operationError(err, *localPath)
		}
	}
	now := timestamp()
	repository := model.Repository{SchemaVersion: model.ResourceSchemaVersion, ID: *id, URL: *url, Path: resolvedPath, DefaultBranch: *defaultBranch, Provider: *provider, CreatedAt: now, UpdatedAt: now}
	if err := model.ValidateRepository(repository); err != nil {
		return a.operationError(err, *id)
	}
	s, err := store.Open(*projectPath)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	existing, existingErr := s.Repository(*id)
	if existingErr == nil {
		if !*ifNotExists {
			return a.fail(failure{5, "conflict", fmt.Sprintf("repository already exists: %s", *id), "Use --if-not-exists or choose another id", *id})
		}
		repository = existing
	} else if !store.IsNotFound(existingErr) {
		return a.operationError(existingErr, *id)
	}
	result := map[string]any{"action": "repository.add", "created": existingErr != nil, "repository": repository}
	if *dryRun {
		a.emit(result, "Would register repository "+repository.ID, "json")
		return 10
	}
	if existingErr != nil {
		if err := s.SaveRepository(repository, false); err != nil {
			return a.operationError(err, *id)
		}
	}
	a.emit(result, "Repository ready: "+repository.ID, selectedFormat(*format, *jsonFlag, a.out))
	return 0
}

func (a app) repositoryList(args []string) int {
	fs := newFlags("repository list")
	projectPath := fs.String("project", "", "project directory (required)")
	format, jsonFlag := outputFlags(fs)
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	if *projectPath == "" {
		return a.fail(failure{2, "invalid_arguments", "--project is required", "Run hp repository list --help", "--project"})
	}
	s, err := store.Open(*projectPath)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	repositories, err := s.Repositories()
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	a.emit(repositories, fmt.Sprintf("%d repository(ies)", len(repositories)), selectedFormat(*format, *jsonFlag, a.out))
	return 0
}

func (a app) repositoryGet(args []string) int {
	fs := newFlags("repository get")
	projectPath := fs.String("project", "", "project directory (required)")
	id := fs.String("id", "", "repository id (required)")
	format, jsonFlag := outputFlags(fs)
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	if err := requiredResourceLookup(*projectPath, *id); err != nil {
		return a.fail(failure{2, "invalid_arguments", err.Error(), "Run hp repository get --help", ""})
	}
	s, err := store.Open(*projectPath)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	repository, err := s.Repository(*id)
	if err != nil {
		return a.operationError(err, *id)
	}
	a.emit(repository, repository.ID, selectedFormat(*format, *jsonFlag, a.out))
	return 0
}

func projectRoot(path string) (string, error) { return project.ResolvePath(path) }
