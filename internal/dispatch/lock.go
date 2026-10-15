package dispatch

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// AcquireLock blocks overlapping hp deliveries to one local provider session.
// A crash can leave a lock file; operators must inspect the session before
// removing such a file because the provider may still be working.
func AcquireLock(projectRoot, harness, sessionID string) (func(), error) {
	return acquireLock(projectRoot, harness, sessionID)
}

// AcquireTaskLock serializes local dispatch decisions for one task, including
// the duplicate-delivery check and assignment write.
func AcquireTaskLock(projectRoot, taskID string) (func(), error) {
	return acquireLock(projectRoot, "task", taskID)
}

func acquireLock(projectRoot, harness, sessionID string) (func(), error) {
	directory := filepath.Join(projectRoot, "artifacts", "dispatch-locks")
	if err := os.MkdirAll(directory, 0700); err != nil {
		return nil, err
	}
	sum := sha256.Sum256([]byte(harness + "\x00" + sessionID))
	path := filepath.Join(directory, hex.EncodeToString(sum[:])+".lock")
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return nil, fmt.Errorf("session delivery lock unavailable: %w", err)
	}
	_, writeErr := fmt.Fprintf(file, "pid=%d\ncreatedAt=%s\n", os.Getpid(), time.Now().UTC().Format(time.RFC3339Nano))
	closeErr := file.Close()
	if writeErr != nil || closeErr != nil {
		_ = os.Remove(path)
		if writeErr != nil {
			return nil, writeErr
		}
		return nil, closeErr
	}
	return func() { _ = os.Remove(path) }, nil
}
