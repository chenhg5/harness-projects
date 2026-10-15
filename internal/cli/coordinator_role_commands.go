package cli

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/chenhg5/harness-projects/internal/coordinator"
	"github.com/chenhg5/harness-projects/internal/dispatch"
	"github.com/chenhg5/harness-projects/internal/participants"
	"github.com/chenhg5/harness-projects/internal/store"
)

func (a app) coordinatorAssign(args []string) int {
	fs := newFlags("coordinator assign")
	projectPath := fs.String("project", "", "project directory (required)")
	id := fs.String("registration-id", "", "registered participant ID (required)")
	replace := fs.Bool("replace", false, "explicitly hand the Coordinator role to another session")
	dryRun := fs.Bool("dry-run", false, "preview assignment without writing")
	format, jsonFlag := outputFlags(fs)
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	if *projectPath == "" || *id == "" {
		return a.parseError(fmt.Errorf("--project and --registration-id are required"))
	}
	s, err := store.Open(*projectPath)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	if !*dryRun {
		release, err := dispatch.AcquireLock(s.Root, "coordinator-role", "project")
		if err != nil {
			return a.operationError(err, s.Root)
		}
		defer release()
	}
	participant, err := (participants.Registry{Root: s.Root}).Get(*id)
	if err != nil {
		return a.operationError(err, *id)
	}
	roles := coordinator.RoleStore{Root: s.Root}
	current, err := roles.Get()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return a.operationError(err, s.Root)
	}
	if current.Current != nil && current.Current.RegistrationID == *id {
		a.emit(map[string]any{"action": "coordinator.assign", "changed": false, "coordinator": current, "agent": participant}, "Coordinator already assigned to "+participant.Harness+" / "+participant.SessionID, selectedFormat(*format, *jsonFlag, a.out))
		return 0
	}
	if current.Current != nil && !*replace {
		return a.fail(failure{5, "conflict", "another Coordinator is assigned", "Use --replace for an explicit handoff", current.Current.RegistrationID})
	}
	if *dryRun {
		a.emit(map[string]any{"action": "coordinator.assign", "dryRun": true, "registrationId": *id, "replaces": current.Current}, "Would assign Coordinator to "+participant.Harness+" / "+participant.SessionID, "json")
		return 10
	}
	role, _, err := roles.Assign(*id, *replace, time.Now())
	if errors.Is(err, coordinator.ErrCoordinatorAssigned) {
		return a.fail(failure{5, "conflict", "another Coordinator is assigned", "Use --replace for an explicit handoff", *id})
	}
	if err != nil {
		return a.operationError(err, *id)
	}
	a.emit(map[string]any{"action": "coordinator.assign", "changed": true, "coordinator": role, "agent": participant}, "Assigned Coordinator to "+participant.Harness+" / "+participant.SessionID, selectedFormat(*format, *jsonFlag, a.out))
	return 0
}

func (a app) coordinatorGet(args []string) int {
	fs := newFlags("coordinator get")
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
	role, err := (coordinator.RoleStore{Root: s.Root}).Get()
	if errors.Is(err, os.ErrNotExist) {
		a.emit(map[string]any{"coordinator": nil}, "No Coordinator assigned", selectedFormat(*format, *jsonFlag, a.out))
		return 0
	}
	if err != nil {
		return a.operationError(err, s.Root)
	}
	var agent any
	human := "No Coordinator assigned"
	if role.Current != nil {
		participant, err := (participants.Registry{Root: s.Root}).Get(role.Current.RegistrationID)
		if err != nil {
			return a.operationError(err, role.Current.RegistrationID)
		}
		agent = participant
		human = "Coordinator: " + participant.Harness + " / " + participant.SessionID
	}
	a.emit(map[string]any{"coordinator": role, "agent": agent}, human, selectedFormat(*format, *jsonFlag, a.out))
	return 0
}

func (a app) coordinatorClear(args []string) int {
	fs := newFlags("coordinator clear")
	projectPath := fs.String("project", "", "project directory (required)")
	id := fs.String("registration-id", "", "current Coordinator registration ID (required)")
	dryRun := fs.Bool("dry-run", false, "preview clearing without writing")
	format, jsonFlag := outputFlags(fs)
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	if *projectPath == "" || *id == "" {
		return a.parseError(fmt.Errorf("--project and --registration-id are required"))
	}
	s, err := store.Open(*projectPath)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	if !*dryRun {
		release, err := dispatch.AcquireLock(s.Root, "coordinator-role", "project")
		if err != nil {
			return a.operationError(err, s.Root)
		}
		defer release()
	}
	roles := coordinator.RoleStore{Root: s.Root}
	role, err := roles.Get()
	if err != nil {
		return a.operationError(err, s.Root)
	}
	if role.Current == nil {
		a.emit(map[string]any{"action": "coordinator.clear", "changed": false}, "No Coordinator assigned", selectedFormat(*format, *jsonFlag, a.out))
		return 0
	}
	if role.Current.RegistrationID != *id {
		return a.fail(failure{5, "conflict", "Coordinator assignment changed", "Read hp coordinator get and use its current registration ID", *id})
	}
	if *dryRun {
		a.emit(map[string]any{"action": "coordinator.clear", "dryRun": true, "coordinator": role}, "Would clear Coordinator", "json")
		return 10
	}
	role, _, err = roles.Clear(*id, time.Now())
	if errors.Is(err, coordinator.ErrCoordinatorChanged) {
		return a.fail(failure{5, "conflict", "Coordinator assignment changed", "Read hp coordinator get and use its current registration ID", *id})
	}
	if err != nil {
		return a.operationError(err, *id)
	}
	a.emit(map[string]any{"action": "coordinator.clear", "changed": true, "coordinator": role}, "Cleared Coordinator; participant remains registered", selectedFormat(*format, *jsonFlag, a.out))
	return 0
}
