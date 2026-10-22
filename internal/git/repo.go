package git

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
)

// Available reports whether a git executable is on PATH.
func Available() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// IsRepositoryRoot reports whether dir has its own .git entry.
func IsRepositoryRoot(dir string) bool {
	_, err := os.Lstat(filepath.Join(dir, ".git"))
	return err == nil
}

// InitAndCommit creates a repository in dir and commits everything in it.
// The commit step is reported separately because it needs a Git identity.
func InitAndCommit(ctx context.Context, dir, message string) (initialized, committed bool, err error) {
	if err := run(ctx, "-C", dir, "init", "--quiet"); err != nil {
		return false, false, err
	}
	if err := run(ctx, "-C", dir, "add", "--all"); err != nil {
		return true, false, err
	}
	if err := run(ctx, "-C", dir, "commit", "--quiet", "--message", message); err != nil {
		return true, false, err
	}
	return true, true, nil
}

// Fingerprint summarizes HEAD and the working tree state of a repository so a
// later call can tell whether anything changed. It returns "" outside a repository.
func Fingerprint(ctx context.Context, dir string) string {
	head, _ := exec.CommandContext(ctx, "git", "-C", dir, "rev-parse", "HEAD").Output()
	status, err := exec.CommandContext(ctx, "git", "-C", dir, "status", "--porcelain", "--untracked-files=normal").Output()
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(append(head, status...))
	return hex.EncodeToString(sum[:8])
}
