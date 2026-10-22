// Package session finds the current provider session without asking the agent
// to search for it.
package session

import "os"

// Current is the provider session that launched this process.
type Current struct {
	Harness   string `json:"harness"`
	SessionID string `json:"sessionId"`
	Source    string `json:"source"`
}

// Detect reads the session from the environment. HP_HARNESS and HP_SESSION_ID
// are exported by the hp session-start hook (Claude Code) or set by hand;
// Codex exposes CODEX_THREAD_ID to commands it runs.
func Detect() (Current, bool) {
	if id := os.Getenv("HP_SESSION_ID"); id != "" {
		return Current{Harness: os.Getenv("HP_HARNESS"), SessionID: id, Source: "HP_SESSION_ID"}, true
	}
	if id := os.Getenv("CODEX_THREAD_ID"); id != "" {
		return Current{Harness: "codex", SessionID: id, Source: "CODEX_THREAD_ID"}, true
	}
	return Current{}, false
}

// Fill completes empty harness and session flags from the environment. It
// never replaces a value the caller passed explicitly, and it only fills the
// harness when it also supplies the session ID or the harness matches.
func Fill(harness, sessionID *string) (Current, bool) {
	current, ok := Detect()
	if !ok {
		return current, false
	}
	if *sessionID == "" {
		if *harness != "" && current.Harness != "" && *harness != current.Harness {
			return current, false
		}
		*sessionID = current.SessionID
		if *harness == "" {
			*harness = current.Harness
		}
		return current, true
	}
	if *harness == "" && *sessionID == current.SessionID {
		*harness = current.Harness
	}
	return current, false
}
