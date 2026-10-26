package fileops

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/chenhg5/harness-projects/internal/apperror"
	"github.com/chenhg5/harness-projects/internal/project"
)

const MaxContentBytes = 4 << 20

var allowedSections = map[string]bool{
	"spec": true, "goals": true, "tasks": true, "runs": true, "agents": true,
	"context": true, "handoffs": true, "events": true, "verification": true, "worklogs": true, "archive": true,
	"skills": true, "scripts": true, "state": true, "artifacts": true,
	"integrations": true,
}

type WritePlan struct {
	Action  string `json:"action"`
	Project string `json:"project"`
	Section string `json:"section"`
	Path    string `json:"path"`
	Bytes   int    `json:"bytes"`
	SHA256  string `json:"sha256"`
	Exists  bool   `json:"exists"`
	Changes []struct {
		Operation string `json:"operation"`
		Path      string `json:"path"`
	} `json:"changes"`
}

func Target(root, section, relative string) (string, error) {
	if !allowedSections[section] {
		return "", fmt.Errorf("invalid section %q; use a canonical project section", section)
	}
	if relative == "" || filepath.IsAbs(relative) || relative == "." || strings.Contains(relative, "\\") {
		return "", fmt.Errorf("invalid relative path %q", relative)
	}
	for _, part := range strings.Split(filepath.ToSlash(relative), "/") {
		if part == "" || part == "." || part == ".." || part == ".ssh" || part == ".gnupg" {
			return "", fmt.Errorf("unsafe relative path %q", relative)
		}
	}
	projectRoot, err := project.ResolvePath(root)
	if err != nil {
		return "", err
	}
	if !project.Inspect(projectRoot).Valid {
		return "", fmt.Errorf("not a valid Harness Project: %s", projectRoot)
	}
	path := filepath.Join(projectRoot, section, relative)
	current := filepath.Join(projectRoot, section)
	for _, part := range strings.Split(relative, string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("symlink in target path: %s", current)
		}
	}
	return path, nil
}

func ReadSource(source string, stdin io.Reader) ([]byte, error) {
	if source == "" {
		return nil, errors.New("--from is required; use --from - for stdin")
	}
	var reader io.Reader = stdin
	if source != "-" {
		file, err := os.Open(source)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		reader = file
	}
	content, err := io.ReadAll(io.LimitReader(reader, MaxContentBytes+1))
	if err != nil {
		return nil, err
	}
	if len(content) > MaxContentBytes {
		return nil, fmt.Errorf("content exceeds %d bytes", MaxContentBytes)
	}
	return content, nil
}

func PlanWrite(root, section, relative string, content []byte, replace, ifNotExists bool) (WritePlan, error) {
	path, err := Target(root, section, relative)
	if err != nil {
		return WritePlan{}, err
	}
	info, err := os.Lstat(path)
	exists := err == nil
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return WritePlan{}, err
	}
	if exists && !info.Mode().IsRegular() {
		return WritePlan{}, fmt.Errorf("target is not a regular file: %s", path)
	}
	if exists && !replace && !ifNotExists {
		return WritePlan{}, apperror.Conflict("file already exists: %s; use --replace to update it", path)
	}
	hash := sha256.Sum256(content)
	plan := WritePlan{Action: "file.write", Project: root, Section: section, Path: path, Bytes: len(content), SHA256: hex.EncodeToString(hash[:]), Exists: exists}
	if !(exists && ifNotExists) {
		operation := "create_file"
		if exists {
			operation = "replace_file"
		}
		plan.Changes = append(plan.Changes, struct {
			Operation string `json:"operation"`
			Path      string `json:"path"`
		}{operation, path})
	}
	return plan, nil
}

func ApplyWrite(plan WritePlan, content []byte) error {
	if len(plan.Changes) == 0 {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(plan.Path), 0755); err != nil {
		return err
	}
	// Recheck the path immediately before writing to catch ordinary concurrent edits.
	info, err := os.Lstat(plan.Path)
	if plan.Exists {
		if err != nil || !info.Mode().IsRegular() {
			return apperror.Conflict("target changed since preview: %s", plan.Path)
		}
	} else if err == nil {
		return apperror.Conflict("target appeared since preview: %s", plan.Path)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if !plan.Exists {
		file, err := os.OpenFile(plan.Path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err != nil {
			return err
		}
		if _, err := file.Write(content); err != nil {
			_ = file.Close()
			_ = os.Remove(plan.Path)
			return err
		}
		return file.Close()
	}
	temp, err := os.CreateTemp(filepath.Dir(plan.Path), ".hp-write-*")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	if err := temp.Chmod(info.Mode().Perm()); err != nil {
		_ = temp.Close()
		return err
	}
	if _, err := temp.Write(content); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(temp.Name(), plan.Path)
}

func Read(root, section, relative string) (string, []byte, error) {
	path, err := Target(root, section, relative)
	if err != nil {
		return "", nil, err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return "", nil, err
	}
	if !info.Mode().IsRegular() {
		return "", nil, fmt.Errorf("not a regular file: %s", path)
	}
	data, err := os.ReadFile(path)
	return path, data, err
}
