package worklog

import "strings"

// MissingCoreFields checks explicit fielded logs without imposing the CLI's
// English headings on historical freeform prose. A deterministic parser cannot
// prove the semantic completeness of a narrative, so freeform logs are only
// rejected when they are empty.
func MissingCoreFields(markdown string) []string {
	required := []string{"objective", "completed", "remaining", "next action"}
	canonical := strings.HasPrefix(strings.TrimSpace(markdown), "# Work log:") &&
		strings.Contains(markdown, "- Agent:") && strings.Contains(markdown, "- Harness:") &&
		strings.Contains(markdown, "- Created:") && strings.Contains(markdown, "- Task:") &&
		strings.Contains(markdown, "- Run:") && strings.Contains(markdown, "## Changes") &&
		strings.Contains(markdown, "## Verification") && strings.Contains(markdown, "## Blockers")
	seen := map[string]bool{}
	filled := map[string]bool{}
	var section []string
	prose := false
	for _, raw := range strings.Split(markdown, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "# ") || strings.HasPrefix(line, "## ") || strings.HasPrefix(line, "### ") {
			section = nil
			if strings.HasPrefix(line, "## ") {
				section = coreFields(strings.TrimSpace(strings.TrimPrefix(line, "## ")))
				for _, field := range section {
					seen[field] = true
				}
			}
			continue
		}
		if label, value, ok := labeledValue(line); ok {
			if fields := coreFields(label); len(fields) > 0 {
				for _, field := range fields {
					seen[field] = true
					filled[field] = filled[field] || meaningful(value)
				}
				continue
			}
			if identityMetadata(label) {
				continue // identity metadata is not a work summary
			}
		}
		if meaningful(line) {
			prose = true
			for _, field := range section {
				filled[field] = true
			}
		}
	}
	if !canonical && prose {
		return nil // Legacy narrative: no safe way to infer missing semantic fields.
	}
	// Existing fielded logs often combine unfinished work and the next step in
	// one section. Both are future-facing context for the next agent.
	if filled["remaining"] && !seen["next action"] {
		seen["next action"], filled["next action"] = true, true
	}
	if filled["next action"] && !seen["remaining"] {
		seen["remaining"], filled["remaining"] = true, true
	}
	var missing []string
	for _, field := range required {
		if !seen[field] || !filled[field] {
			missing = append(missing, field)
		}
	}
	return missing
}

func labeledValue(line string) (string, string, bool) {
	if !strings.HasPrefix(line, "- ") && !strings.HasPrefix(line, "* ") {
		return "", "", false
	}
	line = strings.TrimSpace(strings.TrimLeft(line, "-* "))
	colon := strings.IndexAny(line, ":：")
	if colon < 0 {
		return "", "", false
	}
	label := strings.Trim(strings.TrimSpace(line[:colon]), "* ")
	value := strings.Trim(strings.TrimSpace(line[colon+1:]), "* ")
	// A long list item with a colon later in its sentence is not metadata.
	return label, value, label != "" && len(label) <= 40
}

func coreFields(label string) []string {
	label = strings.ToLower(strings.TrimSpace(label))
	fields := []string{}
	if hasAny(label, "objective", "goal", "intent", "request", "目标", "目的", "本次要做") {
		fields = append(fields, "objective")
	}
	if hasAny(label, "completed", "done", "changes", "work done", "findings", "review", "decision", "实现", "已完成", "完成", "改动", "变更", "结果") {
		fields = append(fields, "completed")
	}
	if hasAny(label, "remaining", "follow-up", "follow up", "未完成", "剩余", "待处理", "待办", "后续", "交接") {
		fields = append(fields, "remaining")
	}
	if hasAny(label, "next", "follow-up", "follow up", "下一步", "后续", "待办", "交接") {
		fields = append(fields, "next action")
	}
	return fields
}

func hasAny(value string, candidates ...string) bool {
	for _, candidate := range candidates {
		if strings.Contains(value, candidate) {
			return true
		}
	}
	return false
}

func identityMetadata(label string) bool {
	switch strings.ToLower(strings.TrimSpace(label)) {
	case "agent", "harness", "session", "session id", "provider session", "created", "task", "run", "date":
		return true
	}
	return false
}

func meaningful(value string) bool {
	value = strings.ToLower(strings.TrimSpace(strings.Trim(value, "-* `>")))
	switch value {
	case "", ".", "...", "todo", "tbd", "none recorded.", "none recorded", "待填写", "待补充":
		return false
	}
	return true
}
