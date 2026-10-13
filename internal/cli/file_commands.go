package cli

import (
	"strings"

	"github.com/chenhg5/harness-projects/internal/fileops"
)

func (a app) fileWrite(args []string) int {
	fs := newFlags("file write")
	root := fs.String("project", "", "project directory (required)")
	section := fs.String("section", "", "canonical project section (required)")
	relative := fs.String("path", "", "path relative to section (required)")
	source := fs.String("from", "", "source file or - for stdin (required)")
	format, jsonFlag := outputFlags(fs)
	dryRun := fs.Bool("dry-run", false, "preview changes without writing")
	replace := fs.Bool("replace", false, "replace an existing file")
	ifNotExists := fs.Bool("if-not-exists", false, "leave an existing file unchanged")
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	if *root == "" || *section == "" || *relative == "" || *source == "" {
		return a.fail(failure{2, "invalid_arguments", "--project, --section, --path, and --from are required", "Run hp file write --help", strings.Join(args, " ")})
	}
	if *replace && *ifNotExists {
		return a.fail(failure{2, "invalid_arguments", "--replace and --if-not-exists cannot be combined", "Choose one file conflict policy", strings.Join(args, " ")})
	}
	content, err := fileops.ReadSource(*source, a.in)
	if err != nil {
		return a.operationError(err, *source)
	}
	plan, err := fileops.PlanWrite(*root, *section, *relative, content, *replace, *ifNotExists)
	if err != nil {
		return a.operationError(err, *relative)
	}
	if *dryRun {
		a.emit(plan, "Would write "+plan.Path, "json")
		return 10
	}
	if err := fileops.ApplyWrite(plan, content); err != nil {
		return a.operationError(err, plan.Path)
	}
	a.emit(plan, "File ready at "+plan.Path, selectedFormat(*format, *jsonFlag, a.out))
	return 0
}

func (a app) fileRead(args []string) int {
	fs := newFlags("file read")
	root := fs.String("project", "", "project directory (required)")
	section := fs.String("section", "", "canonical project section (required)")
	relative := fs.String("path", "", "path relative to section (required)")
	format, jsonFlag := outputFlags(fs)
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	if *root == "" || *section == "" || *relative == "" {
		return a.fail(failure{2, "invalid_arguments", "--project, --section, and --path are required", "Run hp file read --help", strings.Join(args, " ")})
	}
	path, data, err := fileops.Read(*root, *section, *relative)
	if err != nil {
		return a.operationError(err, *relative)
	}
	if selectedFormat(*format, *jsonFlag, a.out) == "json" {
		a.emit(map[string]any{"path": path, "content": string(data)}, "", "json")
	} else {
		_, _ = a.out.Write(data)
	}
	return 0
}
