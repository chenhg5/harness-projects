package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/chenhg5/harness-projects/internal/model"
)

func requiredResourceFlags(projectPath, id, title string) error {
	if projectPath == "" || id == "" || title == "" {
		return fmt.Errorf("--project, --id, and --title are required")
	}
	return nil
}

func parseKeyResults(value string) ([]model.KeyResult, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	var results []model.KeyResult
	if err := json.Unmarshal([]byte(value), &results); err != nil {
		return nil, fmt.Errorf("invalid --key-results JSON: %w", err)
	}
	for _, result := range results {
		if err := model.ValidateKeyResult(result); err != nil {
			return nil, err
		}
	}
	return results, nil
}

func requiredResourceLookup(projectPath, id string) error {
	if projectPath == "" || id == "" {
		return fmt.Errorf("--project and --id are required")
	}
	return nil
}

func commaList(value string) ([]string, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item == "" {
			return nil, fmt.Errorf("list contains an empty item")
		}
		if seen[item] {
			return nil, fmt.Errorf("list contains duplicate item %q", item)
		}
		seen[item] = true
		result = append(result, item)
	}
	return result, nil
}

func flagSet(fs *flag.FlagSet, name string) bool {
	set := false
	fs.Visit(func(flag *flag.Flag) {
		if flag.Name == name {
			set = true
		}
	})
	return set
}

func timestamp() string { return time.Now().UTC().Format(time.RFC3339) }

func parsePositiveInt(value string) (int, error) {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return 0, fmt.Errorf("expected a positive integer, got %q", value)
	}
	return parsed, nil
}
