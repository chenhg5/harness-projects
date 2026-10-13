package cli

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"
)

func run(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := Run(args, bytes.NewBuffer(nil), &out, &errOut)
	return code, out.String(), errOut.String()
}

func TestProjectLifecycle(t *testing.T) {
	root := filepath.Join(t.TempDir(), "demo")
	code, _, stderr := run(t, "project", "init", "--path", root, "--name", "Demo")
	if code != 0 {
		t.Fatalf("init code=%d stderr=%s", code, stderr)
	}
	code, stdout, stderr := run(t, "project", "validate", "--path", root, "--json")
	if code != 0 {
		t.Fatalf("validate code=%d stderr=%s", code, stderr)
	}
	var result struct {
		Valid bool `json:"valid"`
	}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatal(err)
	}
	if !result.Valid {
		t.Fatalf("expected valid project: %s", stdout)
	}
}

func TestDryRunExitCode(t *testing.T) {
	root := filepath.Join(t.TempDir(), "demo")
	code, _, stderr := run(t, "project", "init", "--path", root, "--dry-run")
	if code != 10 {
		t.Fatalf("expected exit 10, got %d stderr=%s", code, stderr)
	}
}
