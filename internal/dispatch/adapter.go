package dispatch

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const maxLogBytes = 4 << 20

// Prepared contains an exact-session provider command. No shell interprets Args.
type Prepared struct {
	Executable string
	Args       []string
	Transport  string
	Harness    string
}

type Result struct {
	Started       bool
	ExitCode      int
	SessionID     string
	Summary       string
	Terminal      bool
	TerminalError bool
	Truncated     bool
	StdoutLog     string
	StderrLog     string
	Err           error
}

func Prepare(harness, sessionID, prompt string) (Prepared, error) {
	if sessionID == "" || prompt == "" {
		return Prepared{}, fmt.Errorf("session id and prompt are required")
	}
	switch harness {
	case "codex":
		path, err := exec.LookPath("codex")
		if err != nil {
			return Prepared{}, fmt.Errorf("codex CLI is unavailable: %w", err)
		}
		return Prepared{Executable: path, Args: []string{"queue", "--thread", sessionID, "--message", prompt}, Transport: "codex-queue", Harness: harness}, nil
	case "claude-code":
		path, err := exec.LookPath("claude")
		if err != nil {
			return Prepared{}, fmt.Errorf("Claude Code CLI is unavailable: %w", err)
		}
		return Prepared{Executable: path, Args: []string{"--print", "--resume", sessionID, "--output-format", "stream-json", "--verbose", prompt}, Transport: "claude-cli", Harness: harness}, nil
	case "cursor":
		path, err := exec.LookPath("cursor-agent")
		args := []string{}
		if err != nil {
			path, err = exec.LookPath("cursor")
			if err != nil {
				return Prepared{}, fmt.Errorf("Cursor Agent CLI is unavailable: %w", err)
			}
			args = append(args, "agent")
		}
		args = append(args, "--print", "--resume="+sessionID, "--output-format", "stream-json", prompt)
		return Prepared{Executable: path, Args: args, Transport: "cursor-cli", Harness: harness}, nil
	default:
		return Prepared{}, fmt.Errorf("no direct CLI adapter for harness %q", harness)
	}
}

// ClaudeSessionActive checks the provider's own active-session list before a
// separate CLI process attempts to resume an existing conversation.
func ClaudeSessionActive(ctx context.Context, executable, sessionID string) (bool, error) {
	command := exec.CommandContext(ctx, executable, "agents", "--json")
	var output bytes.Buffer
	command.Stdout = &boundedWriter{writer: &output, limit: 1 << 20}
	command.Stderr = &boundedWriter{writer: &bytes.Buffer{}, limit: 1 << 16}
	err := command.Run()
	if err != nil {
		return false, fmt.Errorf("cannot inspect active Claude sessions: %w", err)
	}
	var sessions []struct {
		SessionID string `json:"sessionId"`
	}
	if err := json.Unmarshal(output.Bytes(), &sessions); err != nil {
		return false, fmt.Errorf("invalid Claude session list: %w", err)
	}
	for _, session := range sessions {
		if session.SessionID == sessionID {
			return true, nil
		}
	}
	return false, nil
}

type boundedWriter struct {
	writer  io.Writer
	limit   int
	written int
}

func (w *boundedWriter) Write(p []byte) (int, error) {
	remaining := w.limit - w.written
	if remaining <= 0 {
		return 0, fmt.Errorf("provider output exceeds %d bytes", w.limit)
	}
	if len(p) > remaining {
		return 0, fmt.Errorf("provider output exceeds %d bytes", w.limit)
	}
	n, err := w.writer.Write(p)
	w.written += n
	return n, err
}

// Execute starts exactly the prepared command and captures bounded local logs.
// Started distinguishes safe pre-start failures from potentially delivered prompts.
func Execute(ctx context.Context, prepared Prepared, worktree, projectRoot, runID string) Result {
	result := Result{ExitCode: -1}
	logDir := filepath.Join(projectRoot, "artifacts", "dispatch")
	if err := os.MkdirAll(logDir, 0700); err != nil {
		result.Err = err
		return result
	}
	stdoutPath := filepath.Join(logDir, runID+".stdout.log")
	stderrPath := filepath.Join(logDir, runID+".stderr.log")
	stdout, err := os.OpenFile(stdoutPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		result.Err = err
		return result
	}
	defer stdout.Close()
	stderr, err := os.OpenFile(stderrPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		result.Err = err
		return result
	}
	defer stderr.Close()
	result.StdoutLog = filepath.ToSlash(filepath.Join("artifacts", "dispatch", filepath.Base(stdoutPath)))
	result.StderrLog = filepath.ToSlash(filepath.Join("artifacts", "dispatch", filepath.Base(stderrPath)))
	outLog := &boundedLog{file: stdout, limit: maxLogBytes}
	errLog := &boundedLog{file: stderr, limit: maxLogBytes}
	cmd := exec.CommandContext(ctx, prepared.Executable, prepared.Args...)
	cmd.Dir = worktree
	cmd.Stdin = bytes.NewReader(nil)
	cmd.Stdout = outLog
	cmd.Stderr = errLog
	if err := cmd.Start(); err != nil {
		result.Err = err
		return result
	}
	result.Started = true
	err = cmd.Wait()
	if err == nil {
		result.ExitCode = 0
	} else {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			result.ExitCode = exitErr.ExitCode()
		}
		result.Err = err
	}
	if ctx.Err() != nil {
		result.Err = ctx.Err()
	}
	result.Truncated = outLog.truncated || errLog.truncated
	if prepared.Transport == "codex-queue" {
		result.SessionID = strings.TrimSpace(prepared.Args[2])
		result.Summary = trimSummary(outLog.buffer.String())
		return result
	}
	result.SessionID, result.Summary, result.Terminal, result.TerminalError = parseResult(outLog.buffer.Bytes())
	if result.Summary == "" {
		result.Summary = trimSummary(errLog.buffer.String())
	}
	return result
}

type boundedLog struct {
	file      *os.File
	buffer    bytes.Buffer
	limit     int
	truncated bool
}

func (w *boundedLog) Write(p []byte) (int, error) {
	remaining := w.limit - w.buffer.Len()
	if remaining <= 0 {
		w.truncated = true
		return len(p), nil
	}
	part := p
	if len(part) > remaining {
		part = part[:remaining]
		w.truncated = true
	}
	if _, err := w.file.Write(part); err != nil {
		return 0, err
	}
	_, _ = w.buffer.Write(part)
	return len(p), nil
}

func parseResult(data []byte) (sessionID, summary string, terminal, terminalError bool) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 4096), maxLogBytes)
	for scanner.Scan() {
		var event struct {
			Type      string `json:"type"`
			SessionID string `json:"session_id"`
			Result    string `json:"result"`
			IsError   bool   `json:"is_error"`
		}
		if json.Unmarshal(scanner.Bytes(), &event) != nil {
			continue
		}
		if event.SessionID != "" {
			sessionID = event.SessionID
		}
		if event.Type == "result" {
			terminal, terminalError = true, event.IsError
			summary = trimSummary(event.Result)
		}
	}
	return
}

func trimSummary(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 2048 {
		return value[:2048]
	}
	return value
}
