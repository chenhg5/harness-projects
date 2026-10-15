package cli

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chenhg5/harness-projects/internal/dispatch"
	"github.com/chenhg5/harness-projects/internal/model"
	"github.com/chenhg5/harness-projects/internal/presence"
	"github.com/chenhg5/harness-projects/internal/runtime"
	"github.com/chenhg5/harness-projects/internal/store"
)

func (a app) coordinatorDispatch(args []string) int {
	fs := newFlags("coordinator dispatch")
	projectPath := fs.String("project", "", "project directory (required)")
	taskID := fs.String("task-id", "", "ready task id (required)")
	harnessText := fs.String("harness", "", "codex|claude-code|cursor (required)")
	sessionID := fs.String("session-id", "", "actual existing provider session ID (required)")
	worktreeText := fs.String("worktree", "", "execution directory, relative to project root if not absolute (required)")
	message := fs.String("message", "", "additional task instructions")
	delivery := fs.String("delivery", "auto", "auto|direct|inbox")
	runID := fs.String("run-id", "", "execution run id; generated when omitted")
	timeoutText := fs.String("timeout", "30m", "provider CLI timeout, 1s..24h")
	replace := fs.Bool("replace", false, "replace a different task assignment")
	retry := fs.Bool("retry", false, "explicitly allow another delivery attempt after an earlier attempt")
	dryRun := fs.Bool("dry-run", false, "preview without assigning or starting a provider")
	format, jsonFlag := outputFlags(fs)
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	if *projectPath == "" || *taskID == "" || *harnessText == "" || *sessionID == "" || *worktreeText == "" {
		return a.parseError(fmt.Errorf("--project, --task-id, --harness, --session-id, and --worktree are required"))
	}
	if *delivery != "auto" && *delivery != "direct" && *delivery != "inbox" {
		return a.parseError(fmt.Errorf("--delivery must be auto, direct, or inbox"))
	}
	harness, err := presence.CanonicalHarness(*harnessText)
	if err != nil {
		return a.parseError(err)
	}
	if err := model.ValidateProviderSessionID(*sessionID); err != nil {
		return a.parseError(err)
	}
	if *sessionID == "new" {
		return a.parseError(fmt.Errorf("this dispatch command requires an existing session ID; new-session dispatch is not implemented"))
	}
	timeout, err := time.ParseDuration(*timeoutText)
	if err != nil || timeout < time.Second || timeout > 24*time.Hour {
		return a.parseError(fmt.Errorf("--timeout must be between 1s and 24h"))
	}
	if *runID == "" {
		*runID, err = newDispatchRunID()
		if err != nil {
			return a.operationError(err, "run id")
		}
	}
	if err := model.ValidateID(*runID); err != nil {
		return a.parseError(err)
	}
	s, err := store.Open(*projectPath)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	if !*dryRun {
		releaseTask, err := dispatch.AcquireTaskLock(s.Root, *taskID)
		if err != nil {
			return a.fail(failure{5, "conflict", "task dispatch lock unavailable", "Inspect an existing dispatch before retrying", *taskID})
		}
		defer releaseTask()
	}
	worktree, err := resolveDispatchWorktree(s.Root, *worktreeText)
	if err != nil {
		return a.parseError(err)
	}
	tasks, err := s.Tasks()
	if err != nil {
		return a.operationError(err, s.Root)
	}
	readiness, err := runtime.ReadyTasks(tasks)
	if err != nil {
		return a.operationError(err, s.Root)
	}
	var task model.Task
	ready := false
	for _, item := range readiness {
		if item.Task.ID == *taskID {
			task, ready = item.Task, item.Ready
			break
		}
	}
	if task.ID == "" {
		return a.operationError(os.ErrNotExist, *taskID)
	}
	if !ready {
		return a.fail(failure{5, "conflict", "task is not ready to start", "Complete its dependencies or resolve its status first", *taskID})
	}
	if _, err := s.Run(*runID); err == nil {
		return a.fail(failure{5, "conflict", "run id already exists", "Use a different --run-id", *runID})
	} else if !store.IsNotFound(err) {
		return a.operationError(err, *runID)
	}
	if task.Assignment != nil && (task.Assignment.Harness != harness || task.Assignment.SessionID != *sessionID) && !*replace {
		return a.fail(failure{5, "conflict", "task is assigned to another session", "Use --replace to reassign", *taskID})
	}
	if *delivery != "inbox" && !*retry {
		runs, err := s.Runs()
		if err != nil {
			return a.operationError(err, s.Root)
		}
		for _, previous := range runs {
			if previous.TaskID == task.ID && previous.RequestedSessionID == *sessionID && previous.Harness == harness &&
				(previous.DeliveryStatus == "accepted" || previous.DeliveryStatus == "completed" || previous.DeliveryStatus == "failed" || previous.DeliveryStatus == "uncertain" || previous.DeliveryStatus == "sending") {
				return a.fail(failure{5, "conflict", "an earlier delivery may already have reached this session", "Inspect the earlier run before using --retry", previous.ID})
			}
		}
	}
	now := time.Now()
	registration, registrationErr := (presence.Registry{Root: s.Root}).Get(presence.RegistrationID(harness, *sessionID), now)
	if registrationErr != nil && !errors.Is(registrationErr, os.ErrNotExist) {
		return a.operationError(registrationErr, *sessionID)
	}
	if task.Assignment == nil || task.Assignment.Harness != harness || task.Assignment.SessionID != *sessionID {
		if task.Assignment != nil {
			previous := *task.Assignment
			previous.EndedAt = timestamp()
			task.AssignmentHistory = append(task.AssignmentHistory, previous)
		}
		task.Assignment = &model.TaskAssignment{Harness: harness, SessionID: *sessionID, AssignedAt: timestamp()}
	}
	if registrationErr == nil {
		task.Assignment.RegistrationID = registration.ID
	}
	task.UpdatedAt = timestamp()
	if !*dryRun {
		if err := s.SaveTask(task, true); err != nil {
			return a.operationError(err, task.ID)
		}
	}
	prompt := dispatchPrompt(s.Root, worktree, task, *message)
	prepared, reason := dispatchPreflight(*delivery, harness, *sessionID, worktree, prompt, registration, registrationErr, !*dryRun)
	run := model.Run{
		SchemaVersion: model.ResourceSchemaVersion, ID: *runID, TaskID: task.ID,
		Adapter: "cli", Harness: harness, RequestedSessionID: *sessionID,
		Objective: task.Title, Status: "planned", Worktree: worktree,
		DeliveryStatus: "queued", DeliveryReason: reason,
		CreatedAt: timestamp(), UpdatedAt: timestamp(),
	}
	if prepared.Transport != "" {
		run.Transport = prepared.Transport
		run.Command = prepared.Executable
	}
	outputFormat := selectedFormat(*format, *jsonFlag, a.out)
	if *dryRun {
		if reason == "" {
			run.DeliveryStatus = ""
		}
		a.emit(map[string]any{"action": "coordinator.dispatch", "dryRun": true, "wouldAttemptDirect": reason == "", "providerProbePending": reason == "" && harness == "claude-code", "task": task, "run": run}, "Would dispatch task "+task.ID, "json")
		return 10
	}
	if err := s.SaveRun(run, false); err != nil {
		return a.operationError(err, run.ID)
	}
	if reason != "" {
		return a.finishQueuedDispatch(s, run, reason, *delivery, outputFormat)
	}
	release, err := dispatch.AcquireLock(s.Root, harness, *sessionID)
	if err != nil {
		return a.finishQueuedDispatch(s, run, err.Error(), *delivery, outputFormat)
	}
	defer release()
	run.Status, run.DeliveryStatus, run.DeliveryReason, run.StartedAt, run.UpdatedAt = "running", "sending", "", timestamp(), timestamp()
	if err := s.SaveRun(run, true); err != nil {
		return a.operationError(err, run.ID)
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	result := dispatch.Execute(ctx, prepared, worktree, s.Root, run.ID)
	if result.StdoutLog != "" {
		run.Logs = append(run.Logs, result.StdoutLog)
	}
	if result.StderrLog != "" {
		run.Logs = append(run.Logs, result.StderrLog)
	}
	if result.ExitCode >= 0 {
		run.ExitCode = &result.ExitCode
	}
	if result.SessionID != "" {
		run.EffectiveSessionID = result.SessionID
	}
	run.Summary = result.Summary
	run.UpdatedAt = timestamp()
	if !result.Started {
		run.Status, run.StartedAt = "planned", ""
		reason := "provider CLI could not start: " + result.Err.Error()
		return a.finishQueuedDispatch(s, run, reason, *delivery, outputFormat)
	}
	run.FinishedAt = timestamp()
	switch {
	case result.SessionID != "" && result.SessionID != *sessionID:
		run.Status, run.DeliveryStatus, run.DeliveryReason = "paused", "uncertain", "provider returned a different session ID; inspect before retrying"
	case prepared.Transport == "codex-queue" && result.Err == nil && result.ExitCode == 0:
		run.Status, run.DeliveryStatus, run.FinishedAt = "running", "accepted", ""
	case prepared.Transport != "codex-queue" && result.Terminal && result.TerminalError:
		run.Status, run.DeliveryStatus, run.DeliveryReason = "failed", "failed", "provider returned a terminal error"
	case prepared.Transport != "codex-queue" && result.Terminal && result.Err == nil && result.ExitCode == 0 && !result.Truncated:
		run.Status, run.DeliveryStatus = "succeeded", "completed"
		run.Verification = []string{"provider terminal success event", result.StdoutLog}
	default:
		run.Status, run.DeliveryStatus, run.DeliveryReason = "paused", "uncertain", "provider process started but delivery outcome is unclear; inspect logs before retrying"
	}
	if result.Err != nil {
		run.Error = result.Err.Error()
	}
	if err := s.SaveRun(run, true); err != nil {
		return a.operationError(err, run.ID)
	}
	a.emit(map[string]any{"action": "coordinator.dispatch", "task": task, "run": run}, dispatchHuman(run), outputFormat)
	if run.DeliveryStatus == "accepted" || run.DeliveryStatus == "completed" {
		return 0
	}
	return 1
}

func (a app) finishQueuedDispatch(s store.Store, run model.Run, reason, delivery, format string) int {
	run.Status, run.DeliveryStatus, run.DeliveryReason, run.StartedAt, run.FinishedAt, run.UpdatedAt = "planned", "queued", reason, "", "", timestamp()
	if err := s.SaveRun(run, true); err != nil {
		return a.operationError(err, run.ID)
	}
	a.emit(map[string]any{"action": "coordinator.dispatch", "run": run}, "Task remains assigned in the inbox: "+reason, format)
	if delivery == "direct" {
		return 5
	}
	return 0
}

func dispatchPreflight(delivery, harness, sessionID, worktree, prompt string, registration presence.View, registrationErr error, probeProvider bool) (dispatch.Prepared, string) {
	if delivery == "inbox" {
		return dispatch.Prepared{}, "inbox delivery selected"
	}
	if registrationErr != nil {
		return dispatch.Prepared{}, "target session has no local registration; cannot verify its worktree or availability"
	}
	if !registration.Live {
		return dispatch.Prepared{}, "target session registration expired; renew it before direct CLI delivery"
	}
	registeredWorktree, err := filepath.EvalSymlinks(registration.Worktree)
	if err != nil || registeredWorktree != worktree {
		return dispatch.Prepared{}, "target session is registered for a different worktree"
	}
	if harness != "codex" && registration.Availability == "busy" {
		return dispatch.Prepared{}, "target session is busy; separate CLI resume may duplicate active work"
	}
	prepared, err := dispatch.Prepare(harness, sessionID, prompt)
	if err != nil {
		return dispatch.Prepared{}, err.Error()
	}
	if harness == "claude-code" && probeProvider {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		active, err := dispatch.ClaudeSessionActive(ctx, prepared.Executable, sessionID)
		if err != nil {
			return dispatch.Prepared{}, err.Error()
		}
		if active {
			return dispatch.Prepared{}, "target Claude session is still active; separate CLI resume may fork it"
		}
	}
	return prepared, ""
}

func resolveDispatchWorktree(root, value string) (string, error) {
	path := value
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	path, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("invalid worktree: %w", err)
	}
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("worktree is not a directory: %s", path)
	}
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(canonicalRoot, path)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("worktree must be inside the project: %s", path)
	}
	return path, nil
}

func dispatchPrompt(projectRoot, worktree string, task model.Task, message string) string {
	var prompt strings.Builder
	fmt.Fprintf(&prompt, "Work on Harness Project task %q in %s. Use worktree %s.\n", task.ID, projectRoot, worktree)
	fmt.Fprintf(&prompt, "Read the project AGENTS.md, PROJECT.md, state/project-state.json, and tasks/%s.json before editing. Follow relevant repository instructions.\n", task.ID)
	fmt.Fprintf(&prompt, "Task: %s\nDescription: %s\n", task.Title, task.Description)
	if len(task.Deliverables) > 0 {
		fmt.Fprintf(&prompt, "Deliverables: %s\n", strings.Join(task.Deliverables, "; "))
	}
	if len(task.Acceptance) > 0 {
		fmt.Fprintf(&prompt, "Acceptance: %s\n", strings.Join(task.Acceptance, "; "))
	}
	if message != "" {
		fmt.Fprintf(&prompt, "Additional instructions: %s\n", message)
	}
	prompt.WriteString("Record progress and verification in the project files. Do not mark the task complete until its acceptance criteria have evidence.")
	return prompt.String()
}

func newDispatchRunID() (string, error) {
	var random [4]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", err
	}
	return "dispatch-" + time.Now().UTC().Format("20060102t150405") + "-" + hex.EncodeToString(random[:]), nil
}

func dispatchHuman(run model.Run) string {
	switch run.DeliveryStatus {
	case "accepted":
		return "Provider accepted task " + run.TaskID + " for session " + run.RequestedSessionID
	case "completed":
		return "Provider completed CLI run " + run.ID + "; verify task acceptance separately"
	case "queued":
		return "Task remains in the session inbox: " + run.DeliveryReason
	default:
		return "Delivery " + run.DeliveryStatus + " for run " + run.ID + ": " + run.DeliveryReason
	}
}
