package cli

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/chenhg5/harness-projects/internal/cleanup"
	"github.com/chenhg5/harness-projects/internal/presence"
	"github.com/chenhg5/harness-projects/internal/store"
)

func (a app) agentRegister(args []string) int {
	fs := newFlags("agent register")
	projectPath := fs.String("project", "", "project directory (required)")
	harness := fs.String("harness", "", "codex|claude-code|cursor|other harness name (required)")
	sessionID := fs.String("session-id", "", "current provider session ID (required)")
	worktree := fs.String("worktree", "", "working directory, relative to project root if not absolute")
	taskID := fs.String("task-id", "", "current project task ID")
	availability := fs.String("availability", "", "available|busy|paused (defaults to busy for a new registration)")
	focus := fs.String("focus", "", "current responsibility")
	modules := fs.String("modules", "", "comma-separated modules or repository areas")
	capabilities := fs.String("capabilities", "", "comma-separated skills or capabilities")
	ttlText := fs.String("ttl", presence.DefaultTTL.String(), "registration lease, 1m..168h")
	format, jsonFlag := outputFlags(fs)
	dryRun := fs.Bool("dry-run", false, "preview registration without writing")
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	if *projectPath == "" || *harness == "" || *sessionID == "" {
		return a.parseError(fmt.Errorf("--project, --harness, and --session-id are required"))
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
	var item presence.View
	if *dryRun {
		item, err = registry.PrepareRegister(requested, ttl, now)
	} else {
		item, err = registry.Register(requested, ttl, now)
	}
	if err != nil {
		return a.operationError(err, *sessionID)
	}
	result := map[string]any{"action": "agent.register", "agent": item}
	if *dryRun {
		a.emit(result, "Would register agent "+item.ID, "json")
		return 10
	}
	a.emit(result, "Registered agent "+item.ID+" ("+item.Harness+" / "+item.SessionID+")", selectedFormat(*format, *jsonFlag, a.out))
	return 0
}

func (a app) agentDiscover(args []string) int {
	fs := newFlags("agent discover")
	projectPath := fs.String("project", "", "project directory (required)")
	harness := fs.String("harness", "", "filter by harness")
	module := fs.String("module", "", "filter by module")
	capability := fs.String("capability", "", "filter by capability")
	includeStale := fs.Bool("include-stale", false, "include expired registrations")
	format, jsonFlag := outputFlags(fs)
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	if *projectPath == "" {
		return a.parseError(fmt.Errorf("--project is required"))
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
	items, err := (presence.Registry{Root: s.Root}).Discover(presence.Filter{
		Harness: selectedHarness, Module: *module, Capability: *capability,
		IncludeStale: *includeStale,
	}, time.Now())
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	lines := make([]string, 0, len(items))
	for _, item := range items {
		state := item.Availability
		if !item.Live {
			state = "stale"
		}
		lines = append(lines, fmt.Sprintf("%s %s %s %s — %s", item.ID, item.Harness, item.SessionID, state, item.Focus))
	}
	human := fmt.Sprintf("%d agent session(s)", len(items))
	if len(lines) > 0 {
		human += "\n" + strings.Join(lines, "\n")
	}
	a.emit(items, human, selectedFormat(*format, *jsonFlag, a.out))
	return 0
}

func (a app) agentGet(args []string) int {
	fs := newFlags("agent get")
	projectPath := fs.String("project", "", "project directory (required)")
	id := fs.String("id", "", "registration ID (required)")
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
	item, err := (presence.Registry{Root: s.Root}).Get(*id, time.Now())
	if err != nil {
		return a.operationError(err, *id)
	}
	a.emit(item, item.Harness+" / "+item.SessionID+" / "+item.Availability, selectedFormat(*format, *jsonFlag, a.out))
	return 0
}

func (a app) agentHeartbeat(args []string) int {
	fs := newFlags("agent heartbeat")
	projectPath := fs.String("project", "", "project directory (required)")
	id := fs.String("id", "", "registration ID (required)")
	availability := fs.String("availability", "", "available|busy|paused")
	focus := fs.String("focus", "", "updated responsibility")
	ttlText := fs.String("ttl", presence.DefaultTTL.String(), "registration lease, 1m..168h")
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
	dryRun := fs.Bool("dry-run", false, "preview removal without writing")
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
	item, err := registry.Get(*id, time.Now())
	if err != nil {
		return a.operationError(err, *id)
	}
	if !*dryRun {
		if err := registry.Unregister(*id); err != nil {
			return a.operationError(err, *id)
		}
	}
	result := map[string]any{"action": "agent.unregister", "agent": item, "removed": !*dryRun}
	if *dryRun {
		a.emit(result, "Would unregister agent "+item.ID, "json")
		return 10
	}
	a.emit(result, "Unregistered agent "+item.ID, selectedFormat(*format, *jsonFlag, a.out))
	return 0
}

func (a app) agentPrune(args []string) int {
	fs := newFlags("agent prune")
	projectPath := fs.String("project", "", "project directory (required)")
	olderThan := fs.String("older-than", "7d", "remove registrations expired this long ago")
	format, jsonFlag := outputFlags(fs)
	dryRun := fs.Bool("dry-run", false, "preview stale registrations without removing them")
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
