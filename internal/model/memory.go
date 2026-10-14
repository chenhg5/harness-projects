package model

import (
	"fmt"
	"strings"
)

type Memory struct {
	SchemaVersion string   `json:"schemaVersion"`
	ID            string   `json:"id"`
	Kind          string   `json:"kind"`
	Title         string   `json:"title"`
	Content       string   `json:"content"`
	Tags          []string `json:"tags,omitempty"`
	Source        string   `json:"source,omitempty"`
	CreatedAt     string   `json:"createdAt"`
	UpdatedAt     string   `json:"updatedAt"`
}

var memoryKinds = map[string]bool{"decision": true, "lesson": true, "preference": true, "research": true}

func MemoryKinds() []string { return []string{"decision", "lesson", "preference", "research"} }

func ValidateMemory(memory Memory) error {
	if err := ValidateID(memory.ID); err != nil {
		return err
	}
	if !memoryKinds[memory.Kind] {
		return fmt.Errorf("invalid memory kind %q", memory.Kind)
	}
	if strings.TrimSpace(memory.Title) == "" {
		return fmt.Errorf("memory title is required")
	}
	if strings.TrimSpace(memory.Content) == "" {
		return fmt.Errorf("memory content is required")
	}
	for _, tag := range memory.Tags {
		if err := ValidateID(tag); err != nil {
			return fmt.Errorf("invalid memory tag: %w", err)
		}
	}
	return nil
}
