package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/chenhg5/harness-projects/internal/coordinator"
	"github.com/chenhg5/harness-projects/internal/dispatch"
	"github.com/chenhg5/harness-projects/internal/participants"
	"github.com/chenhg5/harness-projects/internal/presence"
	"github.com/chenhg5/harness-projects/internal/store"
	"github.com/chenhg5/harness-projects/internal/wakeup"
)

func (a app) wakeupConfigure(args []string) int {
	fs := newFlags("wakeup configure")
	projectPath := fs.String("project", "", "project directory (required)")
	every := fs.String("every", "", "interval, for example 30m or 24h")
	cron := fs.String("cron", "", "five-field cron schedule")
	timeZone := fs.String("timezone", "", "IANA time zone for cron")
	worktree := fs.String("worktree", "", "project-relative Coordinator execution directory")
	promptFile := fs.String("prompt-file", "", "project-relative Markdown wakeup routine")
	enable := fs.Bool("enable", false, "enable the schedule")
	disable := fs.Bool("disable", false, "disable the schedule")
	dryRun := fs.Bool("dry-run", false, "preview without writing")
	format, jsonFlag := outputFlags(fs)
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	if *projectPath == "" {
		return a.parseError(fmt.Errorf("--project is required"))
	}
	if *every != "" && *cron != "" {
		return a.parseError(fmt.Errorf("use exactly one of --every or --cron"))
	}
	if *enable && *disable {
		return a.parseError(fmt.Errorf("--enable and --disable conflict"))
	}
	s, err := store.Open(*projectPath)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	release, err := dispatch.AcquireLock(s.Root, "wakeup-config", "project")
	if err != nil {
		return a.operationError(err, s.Root)
	}
	defer release()
	registry := wakeup.Registry{Root: s.Root}
	config, err := registry.GetConfig()
	if errors.Is(err, os.ErrNotExist) {
		config = wakeup.Config{SchemaVersion: wakeup.SchemaVersion, Enabled: true, CreatedAt: timestamp()}
	} else if err != nil {
		return a.operationError(err, s.Root)
	}
	if *every != "" {
		config.Every, config.Cron, config.TimeZone = *every, "", ""
	}
	if *cron != "" {
		config.Cron, config.Every = *cron, ""
	}
	if *timeZone != "" {
		config.TimeZone = *timeZone
	}
	if *worktree != "" {
		config.Worktree = *worktree
	}
	if *promptFile != "" {
		config.PromptFile = *promptFile
	}
	if *enable {
		config.Enabled = true
	}
	if *disable {
		config.Enabled = false
	}
	config.UpdatedAt = timestamp()
	if err := wakeup.Validate(config); err != nil {
		return a.parseError(err)
	}
	if _, err := wakeup.ResolveFile(s.Root, config.Worktree, true); err != nil {
		return a.parseError(fmt.Errorf("invalid worktree: %w", err))
	}
	if _, err := wakeup.ResolveFile(s.Root, config.PromptFile, false); err != nil {
		return a.parseError(fmt.Errorf("invalid prompt file: %w", err))
	}
	next, err := wakeup.Next(config, time.Now())
	if err != nil {
		return a.parseError(err)
	}
	if *dryRun {
		a.emit(map[string]any{"action": "wakeup.configure", "dryRun": true, "config": config, "firstDueAt": next.Format(time.RFC3339Nano)}, "Would configure Coordinator wakeup", "json")
		return 10
	}
	if err := registry.SaveConfig(config); err != nil {
		return a.operationError(err, s.Root)
	}
	a.emit(map[string]any{"action": "wakeup.configure", "config": config, "firstDueAt": next.Format(time.RFC3339Nano)}, "Configured Coordinator wakeup; run hp wakeup start to execute it", selectedFormat(*format, *jsonFlag, a.out))
	return 0
}

func (a app) wakeupStatus(args []string) int {
	fs := newFlags("wakeup status")
	projectPath := fs.String("project", "", "project directory (required)")
	format, jsonFlag := outputFlags(fs)
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	if *projectPath == "" {
		return a.parseError(fmt.Errorf("--project is required"))
	}
	s, err := store.Open(*projectPath)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	registry := wakeup.Registry{Root: s.Root}
	var config any
	if item, err := registry.GetConfig(); err == nil {
		config = item
	} else if !errors.Is(err, os.ErrNotExist) {
		return a.operationError(err, s.Root)
	}
	var role any
	if item, err := (coordinator.RoleStore{Root: s.Root}).Get(); err == nil {
		role = item
	} else if !errors.Is(err, os.ErrNotExist) {
		return a.operationError(err, s.Root)
	}
	state, err := registry.LoadStateOrEmpty()
	if err != nil {
		return a.operationError(err, s.Root)
	}
	human := "Coordinator wakeup is not configured"
	if item, ok := config.(wakeup.Config); ok {
		human = "Coordinator wakeup is configured but disabled"
		if item.Enabled {
			human = "Coordinator wakeup enabled; run hp wakeup start to run the foreground scheduler"
		}
	}
	if state.LastStatus != "" {
		human += "; last status: " + state.LastStatus
	}
	if state.NextDueAt != "" {
		human += "; next due: " + state.NextDueAt
	}
	a.emit(map[string]any{"config": config, "coordinator": role, "state": state}, human, selectedFormat(*format, *jsonFlag, a.out))
	return 0
}

func (a app) wakeupOnce(args []string) int {
	fs := newFlags("wakeup once")
	projectPath := fs.String("project", "", "project directory (required)")
	retry := fs.Bool("retry", false, "retry after inspecting an uncertain delivery")
	dryRun := fs.Bool("dry-run", false, "preview without starting a provider")
	timeoutText := fs.String("timeout", "30m", "provider CLI timeout")
	format, jsonFlag := outputFlags(fs)
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	if *projectPath == "" {
		return a.parseError(fmt.Errorf("--project is required"))
	}
	timeout, err := time.ParseDuration(*timeoutText)
	if err != nil || timeout < time.Second || timeout > 24*time.Hour {
		return a.parseError(fmt.Errorf("--timeout must be between 1s and 24h"))
	}
	s, err := store.Open(*projectPath)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	if *dryRun {
		config, role, participant, worktree, prompt, err := wakeupTarget(s.Root)
		if err != nil {
			return a.operationError(err, s.Root)
		}
		a.emit(map[string]any{"action": "wakeup.once", "dryRun": true, "config": config, "coordinator": role.Current, "agent": participant, "worktree": worktree, "prompt": prompt}, "Would wake Coordinator "+participant.Harness+" / "+participant.SessionID, "json")
		return 10
	}
	release, err := dispatch.AcquireLock(s.Root, "wakeup-scheduler", "project")
	if err != nil {
		return a.fail(failure{5, "conflict", "another wakeup scheduler or delivery is running", "Inspect its status before retrying", s.Root})
	}
	defer release()
	state, err := executeWakeup(context.Background(), s.Root, timeout, *retry)
	if err != nil {
		return a.operationError(err, s.Root)
	}
	a.emit(map[string]any{"action": "wakeup.once", "state": state}, "Coordinator wakeup: "+state.LastStatus+" — "+state.LastReason, selectedFormat(*format, *jsonFlag, a.out))
	if state.LastStatus == "uncertain" {
		return 1
	}
	return 0
}

func (a app) wakeupStart(args []string) int {
	fs := newFlags("wakeup start")
	projectPath := fs.String("project", "", "project directory (required)")
	timeoutText := fs.String("timeout", "30m", "provider CLI timeout")
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	if *projectPath == "" {
		return a.parseError(fmt.Errorf("--project is required"))
	}
	timeout, err := time.ParseDuration(*timeoutText)
	if err != nil || timeout < time.Second || timeout > 24*time.Hour {
		return a.parseError(fmt.Errorf("--timeout must be between 1s and 24h"))
	}
	s, err := store.Open(*projectPath)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	release, err := dispatch.AcquireLock(s.Root, "wakeup-scheduler", "project")
	if err != nil {
		return a.fail(failure{5, "conflict", "another wakeup scheduler is running", "Inspect the existing scheduler before restarting", s.Root})
	}
	defer release()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	fmt.Fprintln(a.out, "Coordinator wakeup scheduler running; press Ctrl-C to stop")
	registry := wakeup.Registry{Root: s.Root}
	for ctx.Err() == nil {
		config, err := registry.GetConfig()
		if err != nil {
			return a.operationError(err, s.Root)
		}
		if !config.Enabled {
			return a.fail(failure{5, "conflict", "wakeup schedule is disabled", "Enable it with hp wakeup configure --enable", s.Root})
		}
		role, err := (coordinator.RoleStore{Root: s.Root}).Get()
		if err != nil {
			return a.operationError(err, s.Root)
		}
		if role.Current == nil {
			return a.fail(failure{5, "conflict", "no Coordinator is assigned", "Use hp coordinator assign", s.Root})
		}
		state, err := registry.LoadStateOrEmpty()
		if err != nil {
			return a.operationError(err, s.Root)
		}
		if state.LastStatus == "sending" || state.LastStatus == "uncertain" {
			return a.fail(failure{5, "conflict", "previous wakeup delivery is uncertain", "Inspect logs, then use hp wakeup once --retry", state.LastDeliveryID})
		}
		if state.RegistrationID != role.Current.RegistrationID || state.ConfigUpdatedAt != config.UpdatedAt {
			next, err := wakeup.Next(config, time.Now())
			if err != nil {
				return a.operationError(err, s.Root)
			}
			state = wakeup.State{SchemaVersion: wakeup.SchemaVersion, RegistrationID: role.Current.RegistrationID, ConfigUpdatedAt: config.UpdatedAt, NextDueAt: next.Format(time.RFC3339Nano)}
			if err := registry.SaveState(state); err != nil {
				return a.operationError(err, s.Root)
			}
		}
		due, err := time.Parse(time.RFC3339Nano, state.NextDueAt)
		if err != nil {
			return a.operationError(fmt.Errorf("invalid nextDueAt: %w", err), s.Root)
		}
		if !time.Now().Before(due) {
			state, err = executeWakeup(ctx, s.Root, timeout, false)
			if err != nil {
				return a.operationError(err, s.Root)
			}
			fmt.Fprintln(a.out, "Coordinator wakeup:", state.LastStatus, state.LastReason)
			continue
		}
		wait := time.Until(due)
		if wait > 30*time.Second {
			wait = 30 * time.Second
		}
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return 0
		case <-timer.C:
		}
	}
	return 0
}

func wakeupTarget(root string) (wakeup.Config, coordinator.Role, participants.Session, string, string, error) {
	config, err := (wakeup.Registry{Root: root}).GetConfig()
	if err != nil {
		return wakeup.Config{}, coordinator.Role{}, participants.Session{}, "", "", err
	}
	role, err := (coordinator.RoleStore{Root: root}).Get()
	if err != nil {
		return config, coordinator.Role{}, participants.Session{}, "", "", err
	}
	if role.Current == nil {
		return config, role, participants.Session{}, "", "", fmt.Errorf("no Coordinator is assigned")
	}
	participant, err := (participants.Registry{Root: root}).Get(role.Current.RegistrationID)
	if err != nil {
		return config, role, participants.Session{}, "", "", err
	}
	worktree, err := wakeup.ResolveFile(root, config.Worktree, true)
	if err != nil {
		return config, role, participant, "", "", err
	}
	if participant.Worktree != config.Worktree {
		return config, role, participant, "", "", fmt.Errorf("Coordinator registration is for worktree %q; schedule uses %q", participant.Worktree, config.Worktree)
	}
	file, err := wakeup.ResolveFile(root, config.PromptFile, false)
	if err != nil {
		return config, role, participant, "", "", err
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return config, role, participant, "", "", err
	}
	if strings.TrimSpace(string(data)) == "" {
		return config, role, participant, "", "", fmt.Errorf("wakeup prompt file is empty")
	}
	prompt := fmt.Sprintf("You are the Coordinator for the Harness Project at %s. Read PROJECT.md, state/current-plan.md, the latest work log, handoffs, and your assigned task inbox. Follow this scheduled routine, update the project work log when you perform meaningful work, and report any blocker:\n\n%s", root, string(data))
	return config, role, participant, worktree, prompt, nil
}

func executeWakeup(ctx context.Context, root string, timeout time.Duration, retry bool) (wakeup.State, error) {
	registry := wakeup.Registry{Root: root}
	config, role, participant, worktree, prompt, err := wakeupTarget(root)
	if err != nil {
		return wakeup.State{}, err
	}
	state, err := registry.LoadStateOrEmpty()
	if err != nil {
		return state, err
	}
	if (state.LastStatus == "sending" || state.LastStatus == "uncertain") && !retry {
		return state, fmt.Errorf("previous wakeup delivery is uncertain; inspect logs before --retry")
	}
	now := time.Now()
	state.SchemaVersion, state.RegistrationID, state.ConfigUpdatedAt = wakeup.SchemaVersion, role.Current.RegistrationID, config.UpdatedAt
	state.LastAttemptAt = now.UTC().Format(time.RFC3339Nano)
	state.LastDeliveryID = ""
	state.Logs = nil
	finish := func(status, reason string) (wakeup.State, error) {
		state.LastStatus, state.LastReason = status, reason
		if status == "uncertain" {
			state.NextDueAt = ""
		} else if status == "queued" {
			state.NextDueAt = time.Now().Add(time.Minute).UTC().Format(time.RFC3339Nano)
		} else {
			next, err := wakeup.Next(config, time.Now())
			if err != nil {
				return state, err
			}
			state.NextDueAt = next.Format(time.RFC3339Nano)
		}
		return state, registry.SaveState(state)
	}
	prepared, err := dispatch.Prepare(participant.Harness, participant.SessionID, prompt)
	if err != nil {
		return finish("queued", err.Error())
	}
	if participant.Harness == "claude-code" {
		probeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		active, probeErr := dispatch.ClaudeSessionActive(probeCtx, prepared.Executable, participant.SessionID)
		cancel()
		if probeErr != nil {
			return finish("queued", probeErr.Error())
		}
		if active {
			return finish("queued", "Claude session is active; separate CLI resume may fork it")
		}
	}
	if participant.Harness == "cursor" {
		lease, leaseErr := (presence.Registry{Root: root}).Get(participant.ID, now)
		if leaseErr != nil || !lease.Live || lease.Worktree != worktree {
			return finish("queued", "Cursor requires a current local delivery lease for this worktree")
		}
		if lease.Availability == "busy" {
			return finish("queued", "Cursor session is busy")
		}
	}
	release, err := dispatch.AcquireLock(root, participant.Harness, participant.SessionID)
	if err != nil {
		return finish("queued", err.Error())
	}
	defer release()
	id, err := newDispatchRunID()
	if err != nil {
		return state, err
	}
	state.LastStatus, state.LastReason, state.LastDeliveryID, state.NextDueAt = "sending", "provider CLI started or about to start", id, ""
	if err := registry.SaveState(state); err != nil {
		return state, err
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	result := dispatch.Execute(runCtx, prepared, worktree, root, id)
	for _, log := range []string{result.StdoutLog, result.StderrLog} {
		if log != "" {
			state.Logs = append(state.Logs, log)
		}
	}
	if !result.Started {
		reason := "provider CLI could not start"
		if result.Err != nil {
			reason += ": " + result.Err.Error()
		}
		return finish("queued", reason)
	}
	if result.SessionID != "" && result.SessionID != participant.SessionID {
		return finish("uncertain", "provider returned a different session ID; inspect logs before retrying")
	}
	if prepared.Transport == "codex-queue" && result.Err == nil && result.ExitCode == 0 {
		return finish("accepted", "message queued to the exact Codex session")
	}
	if prepared.Transport != "codex-queue" && result.Terminal && result.TerminalError {
		return finish("failed", "provider returned a terminal error")
	}
	if prepared.Transport != "codex-queue" && result.Terminal && result.Err == nil && result.ExitCode == 0 && !result.Truncated {
		return finish("completed", "provider returned terminal success")
	}
	return finish("uncertain", "provider process started but delivery outcome is unclear; inspect logs before retrying")
}
