package cli

import (
	"context"
	"encoding/json"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
)

func TestEveryCommandHasCompleteGeneratedContract(t *testing.T) {
	for name, schema := range commandSchemas {
		t.Run(name, func(t *testing.T) {
			if schema.Command != name {
				t.Fatalf("schema command=%q, map key=%q", schema.Command, name)
			}
			if strings.TrimSpace(schema.Description) == "" {
				t.Fatal("missing description")
			}
			if len(schema.Examples) == 0 {
				t.Fatal("missing examples")
			}

			enriched := enrichCommandSchema(schema)
			if len(enriched.Output.Formats) == 0 || enriched.Output.Default == "" {
				t.Fatal("missing output contract")
			}
			if hasSchemaFlag(schema.Flags, "format") && enriched.Output.JSONShape == "" {
				t.Fatal("formatted command missing JSON shape")
			}
			if len(enriched.Errors) == 0 {
				t.Fatal("missing error contract")
			}
			seen := map[string]bool{}
			for _, flag := range enriched.Flags {
				if seen[flag.Name] {
					t.Fatalf("duplicate flag --%s", flag.Name)
				}
				seen[flag.Name] = true
				if strings.TrimSpace(flag.Description) == "" {
					t.Fatalf("--%s has no description", flag.Name)
				}
				commonDescription := commonFlagDetails[flag.Name].Description
				commandDescription := commandFlagDetails[name][flag.Name].Description
				if commonDescription == "" && commandDescription == "" {
					t.Fatalf("--%s relies on a generated placeholder description", flag.Name)
				}
				if flag.Required && flag.Default != nil {
					t.Fatalf("required --%s unexpectedly has default %#v", flag.Name, flag.Default)
				}
				if !flag.Required && flag.Default == nil {
					t.Fatalf("optional --%s has no default", flag.Name)
				}
			}
			if seen["format"] && !seen["json"] {
				t.Fatal("output command exposes --format without --json")
			}

			help := renderCommandHelp(schema)
			for _, section := range []string{"Usage:", "Examples:", "Description:", "Output:", "Options:"} {
				if !strings.Contains(help, section) {
					t.Fatalf("generated help missing %s", section)
				}
			}
			if strings.Index(help, "Examples:") > strings.Index(help, "Description:") {
				t.Fatal("examples must appear before prose description")
			}
		})
	}
}

func TestCommandSchemasMatchRegisteredFlags(t *testing.T) {
	fileSet := token.NewFileSet()
	packages, err := parser.ParseDir(fileSet, ".", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	pkg := packages["cli"]
	methods := map[string]*ast.FuncDecl{}
	var run *ast.FuncDecl
	for _, file := range pkg.Files {
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok {
				continue
			}
			if function.Name.Name == "Run" {
				run = function
			}
			if function.Recv != nil {
				methods[function.Name.Name] = function
			}
		}
	}
	if run == nil {
		t.Fatal("Run function not found")
	}

	handlers := handlersFromRun(run)
	handlers["brief"] = "brief"
	handlers["finish"] = "finish"
	for command, schema := range commandSchemas {
		handler := handlers[command]
		if handler == "" {
			t.Errorf("%s: no handler found in Run", command)
			continue
		}
		actual := collectMethodFlags(handler, methods, map[string]bool{})
		expected := map[string]bool{}
		for _, flag := range enrichCommandSchema(schema).Flags {
			expected[flag.Name] = true
		}
		if !reflect.DeepEqual(sortedKeys(actual), sortedKeys(expected)) {
			t.Errorf("%s: registered flags %v, schema flags %v", command, sortedKeys(actual), sortedKeys(expected))
		}
	}
}

func TestOperationErrorUsesTypedClassification(t *testing.T) {
	for _, test := range []struct {
		name      string
		err       error
		wantCode  int
		wantKind  string
		retryable bool
	}{
		{name: "untyped text is not conflict", err: errors.New("already exists"), wantCode: exitOperationFailed, wantKind: "operation_failed"},
		{name: "typed conflict", err: os.ErrExist, wantCode: exitConflict, wantKind: "conflict"},
		{name: "not found", err: os.ErrNotExist, wantCode: exitNotFound, wantKind: "not_found"},
		{name: "permission", err: os.ErrPermission, wantCode: exitPermissionDenied, wantKind: "permission_denied"},
		{name: "transient", err: context.DeadlineExceeded, wantCode: exitOperationFailed, wantKind: "transient_failure", retryable: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			var stderr strings.Builder
			application := app{err: &stderr}
			if code := application.operationError(test.err, "input"); code != test.wantCode {
				t.Fatalf("code=%d want=%d", code, test.wantCode)
			}
			var result struct {
				Error     string `json:"error"`
				ExitCode  int    `json:"exitCode"`
				Retryable bool   `json:"retryable"`
			}
			if err := json.Unmarshal([]byte(stderr.String()), &result); err != nil {
				t.Fatal(err)
			}
			if result.Error != test.wantKind || result.ExitCode != test.wantCode || result.Retryable != test.retryable {
				t.Fatalf("unexpected failure: %+v", result)
			}
		})
	}
}

func handlersFromRun(run *ast.FuncDecl) map[string]string {
	result := map[string]string{}
	ast.Inspect(run.Body, func(node ast.Node) bool {
		clause, ok := node.(*ast.CaseClause)
		if !ok || len(clause.List) != 1 {
			return true
		}
		literal, ok := clause.List[0].(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			return true
		}
		command, err := strconv.Unquote(literal.Value)
		if err != nil {
			return true
		}
		ast.Inspect(&ast.BlockStmt{List: clause.Body}, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			receiver, ok := selector.X.(*ast.Ident)
			if ok && receiver.Name == "a" {
				result[command] = selector.Sel.Name
				return false
			}
			return true
		})
		return true
	})
	return result
}

func collectMethodFlags(name string, methods map[string]*ast.FuncDecl, visited map[string]bool) map[string]bool {
	result := map[string]bool{}
	if visited[name] || methods[name] == nil {
		return result
	}
	visited[name] = true
	ast.Inspect(methods[name].Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		if identifier, ok := call.Fun.(*ast.Ident); ok && identifier.Name == "outputFlags" {
			result["format"] = true
			result["json"] = true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		receiver, ok := selector.X.(*ast.Ident)
		if !ok {
			return true
		}
		if receiver.Name == "fs" && (selector.Sel.Name == "String" || selector.Sel.Name == "Bool" || selector.Sel.Name == "Int" || selector.Sel.Name == "Duration") && len(call.Args) > 0 {
			if literal, ok := call.Args[0].(*ast.BasicLit); ok && literal.Kind == token.STRING {
				if flagName, err := strconv.Unquote(literal.Value); err == nil {
					result[flagName] = true
				}
			}
		}
		if receiver.Name == "a" {
			for flagName := range collectMethodFlags(selector.Sel.Name, methods, visited) {
				result[flagName] = true
			}
		}
		return true
	})
	return result
}

func sortedKeys(values map[string]bool) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
