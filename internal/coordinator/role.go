package coordinator

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/chenhg5/harness-projects/internal/participants"
)

// Role records the one current Coordinator for a project. Participation
// remains independent: changing this role never deletes a session record.
type Role struct {
	SchemaVersion string `json:"schemaVersion"`
	Current       *Term  `json:"current,omitempty"`
	History       []Term `json:"history,omitempty"`
	UpdatedAt     string `json:"updatedAt"`
}

type Term struct {
	RegistrationID string `json:"registrationId"`
	StartedAt      string `json:"startedAt"`
	EndedAt        string `json:"endedAt,omitempty"`
}

type RoleStore struct{ Root string }

var ErrCoordinatorAssigned = errors.New("another Coordinator is assigned")
var ErrCoordinatorChanged = errors.New("Coordinator assignment changed")

func (s RoleStore) path() string { return filepath.Join(s.Root, "agents", "coordinator.json") }

func (s RoleStore) Get() (Role, error) {
	data, err := os.ReadFile(s.path())
	if err != nil {
		return Role{}, err
	}
	var role Role
	if err := json.Unmarshal(data, &role); err != nil {
		return Role{}, fmt.Errorf("invalid coordinator assignment: %w", err)
	}
	if err := validateRole(role); err != nil {
		return Role{}, err
	}
	return role, nil
}

func (s RoleStore) Assign(registrationID string, replace bool, now time.Time) (Role, bool, error) {
	if _, err := (participants.Registry{Root: s.Root}).Get(registrationID); err != nil {
		return Role{}, false, fmt.Errorf("coordinator must be a registered participant: %w", err)
	}
	role, err := s.Get()
	if errors.Is(err, os.ErrNotExist) {
		role = Role{SchemaVersion: "1"}
	} else if err != nil {
		return Role{}, false, err
	}
	if role.Current != nil && role.Current.RegistrationID == registrationID {
		return role, false, nil
	}
	if role.Current != nil && !replace {
		return Role{}, false, ErrCoordinatorAssigned
	}
	stamp := now.UTC().Format(time.RFC3339Nano)
	if role.Current != nil {
		previous := *role.Current
		previous.EndedAt = stamp
		role.History = append(role.History, previous)
	}
	role.Current = &Term{RegistrationID: registrationID, StartedAt: stamp}
	role.UpdatedAt = stamp
	if err := s.save(role); err != nil {
		return Role{}, false, err
	}
	return role, true, nil
}

func (s RoleStore) Clear(expectedID string, now time.Time) (Role, bool, error) {
	role, err := s.Get()
	if err != nil {
		return Role{}, false, err
	}
	if role.Current == nil {
		return role, false, nil
	}
	if expectedID == "" || role.Current.RegistrationID != expectedID {
		return Role{}, false, ErrCoordinatorChanged
	}
	stamp := now.UTC().Format(time.RFC3339Nano)
	previous := *role.Current
	previous.EndedAt = stamp
	role.History = append(role.History, previous)
	role.Current = nil
	role.UpdatedAt = stamp
	if err := s.save(role); err != nil {
		return Role{}, false, err
	}
	return role, true, nil
}

func (s RoleStore) save(role Role) error {
	if err := validateRole(role); err != nil {
		return err
	}
	dir := filepath.Dir(s.path())
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(role, "", "  ")
	if err != nil {
		return err
	}
	temp, err := os.CreateTemp(dir, ".hp-coordinator-*")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	if _, err := temp.Write(append(data, '\n')); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(temp.Name(), s.path())
}

func validateRole(role Role) error {
	if role.SchemaVersion != "1" {
		return fmt.Errorf("unsupported coordinator assignment schemaVersion %q", role.SchemaVersion)
	}
	if _, err := time.Parse(time.RFC3339Nano, role.UpdatedAt); err != nil {
		return fmt.Errorf("invalid coordinator updatedAt: %w", err)
	}
	terms := append([]Term{}, role.History...)
	if role.Current != nil {
		if role.Current.EndedAt != "" {
			return fmt.Errorf("current Coordinator must not have endedAt")
		}
		terms = append(terms, *role.Current)
	}
	for _, term := range terms {
		if len(term.RegistrationID) != 64 {
			return fmt.Errorf("invalid coordinator registration ID")
		}
		if _, err := hex.DecodeString(term.RegistrationID); err != nil {
			return fmt.Errorf("invalid coordinator registration ID")
		}
		if _, err := time.Parse(time.RFC3339Nano, term.StartedAt); err != nil {
			return fmt.Errorf("invalid coordinator startedAt: %w", err)
		}
		if term.EndedAt != "" {
			if _, err := time.Parse(time.RFC3339Nano, term.EndedAt); err != nil {
				return fmt.Errorf("invalid coordinator endedAt: %w", err)
			}
		}
	}
	return nil
}
