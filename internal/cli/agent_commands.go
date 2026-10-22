package cli

import (
	"errors"
	"fmt"
	"github.com/chenhg5/harness-projects/internal/session"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chenhg5/harness-projects/internal/cleanup"
	"github.com/chenhg5/harness-projects/internal/coordinator"
	"github.com/chenhg5/harness-projects/internal/participants"
	"github.com/chenhg5/harness-projects/internal/presence"
	"github.com/chenhg5/harness-projects/internal/store"
)

type participantView struct {
	participants.Session
	Role string `json:"role,omitempty"`
}

func currentCoordinatorID(root string) (string, error) {
	role, err := (coordinator.RoleStore{Root: root}).Get()
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if role.Current == nil {
		return "", nil
	}
	return role.Current.RegistrationID, nil
}

func (a app) agentRegister(args []string) int {
	fs := newFlags("agent register")
	projectPath := fs.String("project", "", "project directory (required)")
	harness := fs.String("harness", "", "codex|claude-code|cursor|other harness name (detected when omitted)")
	sessionID := fs.String("session-id", "", "current provider session ID (detected when omitted)")
	worktree := fs.String("worktree", "", "working directory, relative to project root if not absolute")
	taskID := fs.String("task-id", "", "current project task ID")
	availability := fs.String("availability", "", "local delivery hint: available|busy|paused (defaults to busy)")
	focus := fs.String("focus", "", "current responsibility")
	modules := fs.String("modules", "", "comma-separated modules or repository areas")
	capabilities := fs.String("capabilities", "", "comma-separated skills or capabilities")
	ttlText := fs.String("ttl", presence.DefaultTTL.String(), "local delivery hint lease, 1m..168h")
	format, jsonFlag := outputFlags(fs)
	dryRun := fs.Bool("dry-run", false, "preview registration without writing")
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	session.Fill(harness, sessionID)
	if *projectPath == "" {
		return a.parseError(fmt.Errorf("--project is required"))
	}
	if *harness == "" || *sessionID == "" {
		return a.fail(failure{2, "invalid_arguments", "could not detect the provider session", sessionHint, "--session-id"})
	}
	ttl, err := parseAgentTTL(*ttlText)
	if err != nil {
		return a.parseError(err)
	}
	moduleList, err := commaList(*modules)
	if err != nil {
		return a.parseError(err)
	}
	capabilityList, err := commaList(*capabilities)
	if err != nil {
		return a.parseError(err)
	}
	s, err := store.Open(*projectPath)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	if *taskID != "" {
		if _, err := s.Task(*taskID); err != nil {
			return a.operationError(fmt.Errorf("task %q: %w", *taskID, err), *taskID)
		}
	}
	resolvedWorktree := ""
	if *worktree != "" {
		resolvedWorktree = *worktree
		if !filepath.IsAbs(resolvedWorktree) {
			resolvedWorktree = filepath.Join(s.Root, resolvedWorktree)
		}
	}
	registry := presence.Registry{Root: s.Root}
	requested := presence.Registration{
		Harness: *harness, SessionID: *sessionID, Worktree: resolvedWorktree,
		TaskID: *taskID, Availability: *availability, Focus: *focus,
		Modules: moduleList, Capabilities: capabilityList,
	}
	now := time.Now()
	lease, err := registry.PrepareRegister(requested, ttl, now)
	if err != nil {
		return a.operationError(err, *sessionID)
	}
	participantRegistry := participants.Registry{Root: s.Root}
	requestedParticipant := participants.Session{Harness: lease.Harness, SessionID: lease.SessionID,
		Focus: lease.Focus, Modules: lease.Modules, Capabilities: lease.Capabilities,
		Worktree: participants.ProjectRelativeWorktree(s.Root, lease.Worktree), TaskID: lease.TaskID}
	var item participants.Session
	if *dryRun {
		item, err = participantRegistry.PrepareRegister(requestedParticipant, now)
	} else {
		item, err = participantRegistry.Register(requestedParticipant, now)
	}
	if err != nil {
		return a.operationError(err, *sessionID)
	}
	result := map[string]any{"action": "agent.register", "agent": item}
	if *dryRun {
		result["deliveryLease"] = lease
		a.emit(result, "Would record participant "+item.ID, "json")
		return 10
	}
	if localLease, leaseErr := registry.Register(requested, ttl, now); leaseErr == nil {
		result["deliveryLease"] = localLease
	} else {
		result["deliveryLeaseError"] = leaseErr.Error()
	}
	a.emit(result, "Recorded participant "+item.ID+" ("+item.Harness+" / "+item.SessionID+")", selectedFormat(*format, *jsonFlag, a.out))
	return 0
}

func (a app) agentDiscover(args []string) int {
	fs := newFlags("agent discover")
	projectPath := fs.String("project", "", "project directory (required)")
	harness := fs.String("harness", "", "filter by harness")
	module := fs.String("module", "", "filter by module")
	capability := fs.String("capability", "", "filter by capability")
	role := fs.String("role", "", "filter by current project role: coordinator")
	_ = fs.Bool("include-stale", false, "deprecated; durable discovery always includes participants")
	format, jsonFlag := outputFlags(fs)
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	if *projectPath == "" {
		return a.parseError(fmt.Errorf("--project is required"))
	}
	if *role != "" && *role != "coordinator" {
		return a.parseError(fmt.Errorf("--role must be coordinator"))
	}
	selectedHarness := ""
	if *harness != "" {
		var err error
		selectedHarness, err = presence.CanonicalHarness(*harness)
		if err != nil {
			return a.parseError(err)
		}
	}
	s, err := store.Open(*projectPath)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	items, err := (participants.Registry{Root: s.Root}).Discover(participants.Filter{
		Harness: selectedHarness, Module: *module, Capability: *capability,
	})
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	coordinatorID, err := currentCoordinatorID(s.Root)
	if err != nil {
		return a.operationError(err, s.Root)
	}
	views := make([]participantView, 0, len(items))
	lines := make([]string, 0, len(items))
	for _, item := range items {
		view := participantView{Session: item}
		if item.ID == coordinatorID {
			view.Role = "coordinator"
		} else if *role != "" {
			continue
		}
		views = append(views, view)
		lines = append(lines, fmt.Sprintf("%s %s %s %s — %s", item.ID, item.Harness, item.SessionID, view.Role, item.Focus))
	}
	human := fmt.Sprintf("%d agent session(s)", len(views))
	if len(lines) > 0 {
		human += "\n" + strings.Join(lines, "\n")
	}
	a.emit(views, human, selectedFormat(*format, *jsonFlag, a.out))
	return 0
}

func (a app) agentGet(args []string) int {
	fs := newFlags("agent get")
	projectPath := fs.String("project", "", "project directory (required)")
	id := fs.String("id", "", "participant registration ID (required)")
	format, jsonFlag := outputFlags(fs)
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	if err := requiredResourceLookup(*projectPath, *id); err != nil {
		return a.parseError(err)
	}
	s, err := store.Open(*projectPath)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	item, err := (participants.Registry{Root: s.Root}).Get(*id)
	if err != nil {
		return a.operationError(err, *id)
	}
	coordinatorID, err := currentCoordinatorID(s.Root)
	if err != nil {
		return a.operationError(err, s.Root)
	}
	view := participantView{Session: item}
	if item.ID == coordinatorID {
		view.Role = "coordinator"
	}
	a.emit(view, item.Harness+" / "+item.SessionID+" / "+view.Role, selectedFormat(*format, *jsonFlag, a.out))
	return 0
}

func (a app) agentHeartbeat(args []string) int {
	fs := newFlags("agent heartbeat")
	projectPath := fs.String("project", "", "project directory (required)")
	id := fs.String("id", "", "registration ID (required)")
	availability := fs.String("availability", "", "available|busy|paused")
	focus := fs.String("focus", "", "updated responsibility")
	ttlText := fs.String("ttl", presence.DefaultTTL.String(), "local delivery hint lease, 1m..168h")
	format, jsonFlag := outputFlags(fs)
	dryRun := fs.Bool("dry-run", false, "preview heartbeat without writing")
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	if err := requiredResourceLookup(*projectPath, *id); err != nil {
		return a.parseError(err)
	}
	ttl, err := parseAgentTTL(*ttlText)
	if err != nil {
		return a.parseError(err)
	}
	s, err := store.Open(*projectPath)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	registry := presence.Registry{Root: s.Root}
	var item presence.View
	if *dryRun {
		item, err = registry.PrepareHeartbeat(*id, *availability, *focus, ttl, time.Now())
	} else {
		item, err = registry.Heartbeat(*id, *availability, *focus, ttl, time.Now())
	}
	if err != nil {
		return a.operationError(err, *id)
	}
	result := map[string]any{"action": "agent.heartbeat", "agent": item}
	if *dryRun {
		a.emit(result, "Would renew agent "+item.ID, "json")
		return 10
	}
	a.emit(result, "Renewed agent "+item.ID, selectedFormat(*format, *jsonFlag, a.out))
	return 0
}

func (a app) agentUnregister(args []string) int {
	fs := newFlags("agent unregister")
	projectPath := fs.String("project", "", "project directory (required)")
	id := fs.String("id", "", "registration ID (required)")
	format, jsonFlag := outputFlags(fs)
	dryRun := fs.Bool("dry-run", false, "preview ending the local delivery hint")
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	if err := requiredResourceLookup(*projectPath, *id); err != nil {
		return a.parseError(err)
	}
	s, err := store.Open(*projectPath)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	registry := presence.Registry{Root: s.Root}
	item, err := (participants.Registry{Root: s.Root}).Get(*id)
	if err != nil {
		return a.operationError(err, *id)
	}
	_, leaseErr := registry.Get(*id, time.Now())
	if leaseErr != nil && !store.IsNotFound(leaseErr) {
		return a.operationError(leaseErr, *id)
	}
	if !*dryRun {
		if leaseErr == nil {
			if err := registry.Unregister(*id); err != nil {
				return a.operationError(err, *id)
			}
		}
	}
	result := map[string]any{"action": "agent.unregister", "agent": item, "deliveryLeaseRemoved": !*dryRun && leaseErr == nil, "participantRetained": true}
	if *dryRun {
		a.emit(result, "Would end local delivery lease for "+item.ID, "json")
		return 10
	}
	a.emit(result, "Ended local delivery lease for "+item.ID+"; participant retained", selectedFormat(*format, *jsonFlag, a.out))
	return 0
}

func (a app) agentPrune(args []string) int {
	fs := newFlags("agent prune")
	projectPath := fs.String("project", "", "project directory (required)")
	olderThan := fs.String("older-than", "7d", "remove local delivery hints expired this long ago")
	format, jsonFlag := outputFlags(fs)
	dryRun := fs.Bool("dry-run", false, "preview stale local delivery hints without removing them")
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	if *projectPath == "" {
		return a.parseError(fmt.Errorf("--project is required"))
	}
	age, err := cleanup.ParseAge(*olderThan)
	if err != nil {
		return a.parseError(err)
	}
	s, err := store.Open(*projectPath)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	registry := presence.Registry{Root: s.Root}
	now := time.Now()
	candidates, err := registry.PruneCandidates(age, now)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	result := map[string]any{"action": "agent.prune", "candidates": candidates, "removed": []string{}}
	if *dryRun {
		a.emit(result, fmt.Sprintf("Would remove %d stale registration(s)", len(candidates)), "json")
		return 10
	}
	removed, err := registry.Prune(age, now)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	result["removed"] = removed
	a.emit(result, fmt.Sprintf("Removed %d stale registration(s)", len(removed)), selectedFormat(*format, *jsonFlag, a.out))
	return 0
}

func parseAgentTTL(value string) (time.Duration, error) {
	ttl, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("invalid --ttl %q: %w", value, err)
	}
	if err := presence.ValidateTTL(ttl); err != nil {
		return 0, err
	}
	return ttl, nil
}
