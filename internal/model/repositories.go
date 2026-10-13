package model

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

type Repository struct {
	SchemaVersion string `json:"schemaVersion"`
	ID            string `json:"id"`
	URL           string `json:"url,omitempty"`
	Path          string `json:"path,omitempty"`
	DefaultBranch string `json:"defaultBranch,omitempty"`
	Provider      string `json:"provider,omitempty"`
	CreatedAt     string `json:"createdAt"`
	UpdatedAt     string `json:"updatedAt"`
}

type Worktree struct {
	SchemaVersion string `json:"schemaVersion"`
	ID            string `json:"id"`
	RepositoryID  string `json:"repositoryId"`
	Path          string `json:"path"`
	Branch        string `json:"branch"`
	Base          string `json:"base,omitempty"`
	Status        string `json:"status"`
	CreatedAt     string `json:"createdAt"`
	UpdatedAt     string `json:"updatedAt"`
}

var worktreeStatuses = map[string]bool{"planned": true, "active": true, "removed": true}

func ValidateRepository(repository Repository) error {
	if err := ValidateID(repository.ID); err != nil {
		return err
	}
	if strings.TrimSpace(repository.URL) == "" && strings.TrimSpace(repository.Path) == "" {
		return fmt.Errorf("repository url or path is required")
	}
	if repository.URL != "" {
		if strings.HasPrefix(repository.URL, "git@") {
			// SCP-style Git URLs such as git@github.com:owner/repo.git are valid.
		} else {
			parsed, err := url.Parse(repository.URL)
			if err != nil || parsed.Scheme == "" || parsed.Host == "" {
				return fmt.Errorf("invalid repository url %q", repository.URL)
			}
			if parsed.User != nil {
				return fmt.Errorf("repository url must not include user credentials")
			}
		}
	}
	if repository.Path != "" {
		if !filepath.IsAbs(repository.Path) || strings.ContainsRune(repository.Path, 0) {
			return fmt.Errorf("repository path must be an absolute path")
		}
	}
	return nil
}

func ValidateWorktree(worktree Worktree) error {
	if err := ValidateID(worktree.ID); err != nil {
		return err
	}
	if err := ValidateID(worktree.RepositoryID); err != nil {
		return fmt.Errorf("invalid repository id: %w", err)
	}
	if strings.TrimSpace(worktree.Path) == "" {
		return fmt.Errorf("worktree path is required")
	}
	if !filepath.IsAbs(worktree.Path) {
		return fmt.Errorf("worktree path must be an absolute path")
	}
	if strings.TrimSpace(worktree.Branch) == "" {
		return fmt.Errorf("worktree branch is required")
	}
	if strings.Contains(worktree.Branch, "..") || strings.ContainsAny(worktree.Branch, " ~^:?*[\\") {
		return fmt.Errorf("invalid worktree branch %q", worktree.Branch)
	}
	if !worktreeStatuses[worktree.Status] {
		return fmt.Errorf("invalid worktree status %q", worktree.Status)
	}
	return nil
}

func WorktreeStatuses() []string { return []string{"planned", "active", "removed"} }
