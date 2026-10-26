package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLayoutRegistrationClearsDoctorFindingAndAppearsInBrief(t *testing.T) {
	root := filepath.Join(t.TempDir(), "demo")
	if code, _, stderr := run(t, "project", "init", "--path", root, "--no-git"); code != 0 {
		t.Fatalf("init code=%d stderr=%s", code, stderr)
	}
	if _, err := os.Stat(filepath.Join(root, ".harness-project", "layout.json")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "lib"), 0755); err != nil {
		t.Fatal(err)
	}

	code, stdout, stderr := run(t, "project", "doctor", "--path", root, "--json")
	if code != 1 || stderr != "" || !strings.Contains(stdout, "unexpected_project_directory") {
		t.Fatalf("doctor should flag lib: code=%d stderr=%s stdout=%s", code, stderr, stdout)
	}
	if code, _, stderr := run(t, "layout", "add", "--project", root, "--name", "lib", "--description", "Third-party reference material"); code != 0 {
		t.Fatalf("layout add code=%d stderr=%s", code, stderr)
	}

	code, stdout, stderr = run(t, "layout", "list", "--project", root, "--json")
	if code != 0 || stderr != "" {
		t.Fatalf("layout list code=%d stderr=%s", code, stderr)
	}
	var view struct {
		Directories []struct {
			Path   string `json:"path"`
			Source string `json:"source"`
		} `json:"directories"`
	}
	if err := json.Unmarshal([]byte(stdout), &view); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, directory := range view.Directories {
		if directory.Path == "lib" && directory.Source == "project" {
			found = true
		}
	}
	if !found {
		t.Fatalf("custom directory missing from layout: %s", stdout)
	}

	if code, stdout, stderr = run(t, "project", "doctor", "--path", root, "--json"); code != 0 || stderr != "" {
		t.Fatalf("doctor should accept registered lib: code=%d stderr=%s stdout=%s", code, stderr, stdout)
	}
	if code, stdout, stderr = run(t, "brief", "--project", root, "--format", "text"); code != 0 || stderr != "" || !strings.Contains(stdout, "`lib/` — Third-party reference material") {
		t.Fatalf("brief should explain custom directory: code=%d stderr=%s stdout=%s", code, stderr, stdout)
	}
}

func TestLayoutAddDryRunDoesNotWrite(t *testing.T) {
	root := filepath.Join(t.TempDir(), "demo")
	if code, _, stderr := run(t, "project", "init", "--path", root, "--no-git"); code != 0 {
		t.Fatalf("init code=%d stderr=%s", code, stderr)
	}
	code, stdout, stderr := run(t, "layout", "add", "--project", root, "--name", "lib", "--description", "References", "--dry-run")
	if code != 10 || stderr != "" {
		t.Fatalf("dry run code=%d stderr=%s", code, stderr)
	}
	var result struct {
		DryRun    bool `json:"dryRun"`
		Changed   bool `json:"changed"`
		Directory struct {
			Path string `json:"path"`
		} `json:"directory"`
	}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("dry-run output is not JSON: %v\n%s", err, stdout)
	}
	if !result.DryRun || !result.Changed || result.Directory.Path != "lib" || strings.Contains(stdout, `"directories"`) {
		t.Fatalf("unexpected focused dry-run result: %s", stdout)
	}
	data, err := os.ReadFile(filepath.Join(root, ".harness-project", "layout.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "lib") {
		t.Fatalf("dry run changed contract: %s", data)
	}
}

func TestLayoutAddAgentFriendlyErrorsAndIdempotency(t *testing.T) {
	root := filepath.Join(t.TempDir(), "demo")
	if code, _, stderr := run(t, "project", "init", "--path", root, "--no-git"); code != 0 {
		t.Fatalf("init code=%d stderr=%s", code, stderr)
	}
	assertLayoutError(t, 2, "invalid_arguments", "--name", "layout", "add", "--project", root)
	assertLayoutError(t, 2, "invalid_arguments", "../lib", "layout", "add", "--project", root, "--name", "../lib", "--description", "References")
	assertLayoutError(t, 5, "conflict", "tasks", "layout", "add", "--project", root, "--name", "tasks", "--description", "Wrong place")

	if code, _, stderr := run(t, "layout", "add", "--project", root, "--name", "lib", "--description", "References"); code != 0 {
		t.Fatalf("first add code=%d stderr=%s", code, stderr)
	}
	assertLayoutError(t, 5, "conflict", "lib", "layout", "add", "--project", root, "--name", "lib", "--description", "References")

	code, stdout, stderr := run(t, "layout", "add", "--project", root, "--name", "lib", "--description", "Ignored replacement", "--if-not-exists", "--json")
	if code != 0 || stderr != "" {
		t.Fatalf("idempotent add code=%d stderr=%s", code, stderr)
	}
	var result struct {
		Changed   bool `json:"changed"`
		Directory struct {
			Description string `json:"description"`
		} `json:"directory"`
	}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatal(err)
	}
	if result.Changed || result.Directory.Description != "References" {
		t.Fatalf("idempotent add should preserve existing entry: %s", stdout)
	}
	assertLayoutError(t, 2, "invalid_arguments", "--replace,--if-not-exists", "layout", "add", "--project", root, "--name", "lib", "--description", "References", "--replace", "--if-not-exists")
}

func TestLayoutSchemaDocumentsDefaultsAndDescriptions(t *testing.T) {
	code, stdout, stderr := run(t, "schema", "get", "--command", "layout add", "--json")
	if code != 0 || stderr != "" {
		t.Fatalf("schema code=%d stderr=%s", code, stderr)
	}
	var schema struct {
		Flags []struct {
			Name        string `json:"name"`
			Default     any    `json:"default"`
			Description string `json:"description"`
		} `json:"flags"`
	}
	if err := json.Unmarshal([]byte(stdout), &schema); err != nil {
		t.Fatal(err)
	}
	foundProject := false
	foundName := false
	for _, flag := range schema.Flags {
		switch flag.Name {
		case "project":
			foundProject = flag.Default == "." && flag.Description != ""
		case "name":
			foundName = flag.Description != ""
		}
	}
	if !foundProject || !foundName {
		t.Fatalf("schema lacks defaults or descriptions: %s", stdout)
	}
}

func assertLayoutError(t *testing.T, wantCode int, wantKind, wantInput string, args ...string) {
	t.Helper()
	code, stdout, stderr := run(t, args...)
	if code != wantCode || stdout != "" {
		t.Fatalf("args=%v code=%d stdout=%q stderr=%s", args, code, stdout, stderr)
	}
	var failure struct {
		Error      string `json:"error"`
		Input      string `json:"input"`
		Message    string `json:"message"`
		Suggestion string `json:"suggestion"`
		Retryable  bool   `json:"retryable"`
	}
	if err := json.Unmarshal([]byte(stderr), &failure); err != nil {
		t.Fatalf("stderr is not JSON: %v\n%s", err, stderr)
	}
	if failure.Error != wantKind || failure.Input != wantInput || failure.Message == "" || failure.Suggestion == "" || failure.Retryable {
		t.Fatalf("unexpected failure: %#v", failure)
	}
}
