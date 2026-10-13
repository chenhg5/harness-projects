package git

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type Command struct {
	Action string   `json:"action"`
	Args   []string `json:"args"`
	Dir    string   `json:"dir"`
}

func Add(ctx context.Context, source, target, branch, base string) (Command, error) {
	if strings.TrimSpace(source) == "" || strings.TrimSpace(target) == "" {
		return Command{}, fmt.Errorf("git worktree source and target are required")
	}
	if strings.TrimSpace(branch) == "" {
		return Command{}, fmt.Errorf("git worktree branch is required")
	}
	args := []string{"-C", source, "worktree", "add", "-b", branch, target}
	if base != "" {
		args = append(args, base)
	}
	command := Command{Action: "worktree.create", Args: append([]string{"git"}, args...), Dir: source}
	if err := run(ctx, args...); err != nil {
		return command, err
	}
	return command, nil
}

func Remove(ctx context.Context, source, target string, force bool) (Command, error) {
	if strings.TrimSpace(source) == "" || strings.TrimSpace(target) == "" {
		return Command{}, fmt.Errorf("git worktree source and target are required")
	}
	args := []string{"-C", source, "worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, target)
	command := Command{Action: "worktree.remove", Args: append([]string{"git"}, args...), Dir: source}
	if err := run(ctx, args...); err != nil {
		return command, err
	}
	return command, nil
}

func run(ctx context.Context, args ...string) error {
	output, err := exec.CommandContext(ctx, "git", args...).CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			return fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
		}
		return fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, message)
	}
	return nil
}
