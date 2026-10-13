package cli

import (
	"context"
	"fmt"
	"path/filepath"

	hpgit "github.com/chenhg5/harness-projects/internal/git"
	"github.com/chenhg5/harness-projects/internal/model"
	"github.com/chenhg5/harness-projects/internal/store"
)

func (a app) worktreeCreate(args []string) int {
	fs := newFlags("worktree create")
	projectPath := fs.String("project", "", "project directory (required)")
	id := fs.String("id", "", "worktree id (required)")
	repositoryID := fs.String("repository-id", "", "repository id (required)")
	targetPath := fs.String("path", "", "worktree path")
	branch := fs.String("branch", "", "branch name")
	base := fs.String("base", "", "base branch or revision")
	format, jsonFlag := outputFlags(fs)
	dryRun := fs.Bool("dry-run", false, "preview changes without writing")
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	if *projectPath == "" || *id == "" || *repositoryID == "" {
		return a.fail(failure{2, "invalid_arguments", "--project, --id, and --repository-id are required", "Run hp worktree create --help", ""})
	}
	root, err := projectRoot(*projectPath)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	s, err := store.Open(root)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	repository, err := s.Repository(*repositoryID)
	if err != nil {
		return a.operationError(err, *repositoryID)
	}
	if repository.Path == "" {
		return a.fail(failure{2, "invalid_arguments", "repository path is required to create a worktree", "Register the repository with --path <local-git-repository>", *repositoryID})
	}
	target := *targetPath
	if target == "" {
		target = filepath.Join(root, "repos", repository.ID, *id)
	}
	target, err = filepath.Abs(target)
	if err != nil {
		return a.operationError(err, target)
	}
	worktreeBranch := *branch
	if worktreeBranch == "" {
		worktreeBranch = "hp/" + repository.ID + "/" + *id
	}
	worktreeBase := *base
	if worktreeBase == "" {
		worktreeBase = repository.DefaultBranch
	}
	now := timestamp()
	worktree := model.Worktree{SchemaVersion: model.ResourceSchemaVersion, ID: *id, RepositoryID: *repositoryID, Path: target, Branch: worktreeBranch, Base: worktreeBase, Status: "planned", CreatedAt: now, UpdatedAt: now}
	if err := model.ValidateWorktree(worktree); err != nil {
		return a.operationError(err, *id)
	}
	_, existingErr := s.Worktree(*id)
	if existingErr == nil {
		return a.fail(failure{5, "conflict", fmt.Sprintf("worktree already exists: %s", *id), "Remove it or choose another id", *id})
	}
	if !store.IsNotFound(existingErr) {
		return a.operationError(existingErr, *id)
	}
	command := hpgit.Command{Action: "worktree.create", Args: []string{"git", "-C", repository.Path, "worktree", "add", "-b", worktree.Branch, worktree.Path}, Dir: repository.Path}
	if worktree.Base != "" {
		command.Args = append(command.Args, worktree.Base)
	}
	result := map[string]any{"action": "worktree.create", "worktree": worktree, "command": command}
	if *dryRun {
		a.emit(result, "Would create worktree "+worktree.ID, "json")
		return 10
	}
	if _, err := hpgit.Add(context.Background(), repository.Path, worktree.Path, worktree.Branch, worktree.Base); err != nil {
		return a.operationError(err, worktree.ID)
	}
	worktree.Status = "active"
	worktree.UpdatedAt = timestamp()
	if err := s.SaveWorktree(worktree, false); err != nil {
		_, _ = hpgit.Remove(context.Background(), repository.Path, worktree.Path, true)
		return a.operationError(err, worktree.ID)
	}
	result["worktree"] = worktree
	a.emit(result, "Worktree active: "+worktree.ID, selectedFormat(*format, *jsonFlag, a.out))
	return 0
}

func (a app) worktreeList(args []string) int {
	fs := newFlags("worktree list")
	projectPath := fs.String("project", "", "project directory (required)")
	repositoryID := fs.String("repository-id", "", "filter by repository id")
	status := fs.String("status", "", "filter by status")
	format, jsonFlag := outputFlags(fs)
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	if *projectPath == "" {
		return a.fail(failure{2, "invalid_arguments", "--project is required", "Run hp worktree list --help", "--project"})
	}
	s, err := store.Open(*projectPath)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	worktrees, err := s.Worktrees()
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	filtered := worktrees[:0]
	for _, worktree := range worktrees {
		if *repositoryID != "" && worktree.RepositoryID != *repositoryID {
			continue
		}
		if *status != "" && worktree.Status != *status {
			continue
		}
		filtered = append(filtered, worktree)
	}
	a.emit(filtered, fmt.Sprintf("%d worktree(s)", len(filtered)), selectedFormat(*format, *jsonFlag, a.out))
	return 0
}

func (a app) worktreeGet(args []string) int {
	fs := newFlags("worktree get")
	projectPath := fs.String("project", "", "project directory (required)")
	id := fs.String("id", "", "worktree id (required)")
	format, jsonFlag := outputFlags(fs)
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	if err := requiredResourceLookup(*projectPath, *id); err != nil {
		return a.fail(failure{2, "invalid_arguments", err.Error(), "Run hp worktree get --help", ""})
	}
	s, err := store.Open(*projectPath)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	worktree, err := s.Worktree(*id)
	if err != nil {
		return a.operationError(err, *id)
	}
	a.emit(worktree, worktree.ID, selectedFormat(*format, *jsonFlag, a.out))
	return 0
}

func (a app) worktreeRemove(args []string) int {
	fs := newFlags("worktree remove")
	projectPath := fs.String("project", "", "project directory (required)")
	id := fs.String("id", "", "worktree id (required)")
	format, jsonFlag := outputFlags(fs)
	force := fs.Bool("force", false, "remove a dirty or locked worktree")
	dryRun := fs.Bool("dry-run", false, "preview changes without writing")
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	if err := requiredResourceLookup(*projectPath, *id); err != nil {
		return a.fail(failure{2, "invalid_arguments", err.Error(), "Run hp worktree remove --help", ""})
	}
	s, err := store.Open(*projectPath)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	worktree, err := s.Worktree(*id)
	if err != nil {
		return a.operationError(err, *id)
	}
	repository, err := s.Repository(worktree.RepositoryID)
	if err != nil {
		return a.operationError(err, worktree.RepositoryID)
	}
	if repository.Path == "" {
		return a.fail(failure{2, "invalid_arguments", "repository path is required to remove a worktree", "Update the repository metadata with a local path", worktree.RepositoryID})
	}
	commandArgs := []string{"git", "-C", repository.Path, "worktree", "remove"}
	if *force {
		commandArgs = append(commandArgs, "--force")
	}
	commandArgs = append(commandArgs, worktree.Path)
	result := map[string]any{"action": "worktree.remove", "worktree": worktree, "command": hpgit.Command{Action: "worktree.remove", Args: commandArgs, Dir: repository.Path}}
	if *dryRun {
		a.emit(result, "Would remove worktree "+worktree.ID, "json")
		return 10
	}
	if _, err := hpgit.Remove(context.Background(), repository.Path, worktree.Path, *force); err != nil {
		return a.operationError(err, worktree.ID)
	}
	worktree.Status = "removed"
	worktree.UpdatedAt = timestamp()
	if err := s.SaveWorktree(worktree, true); err != nil {
		return a.operationError(err, worktree.ID)
	}
	result["worktree"] = worktree
	a.emit(result, "Worktree removed: "+worktree.ID, selectedFormat(*format, *jsonFlag, a.out))
	return 0
}
