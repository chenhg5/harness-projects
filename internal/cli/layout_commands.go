package cli

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/chenhg5/harness-projects/internal/layout"
	"github.com/chenhg5/harness-projects/internal/project"
)

type layoutListResult struct {
	Action         string         `json:"action"`
	Project        string         `json:"project"`
	ContractPath   string         `json:"contractPath"`
	SchemaVersion  string         `json:"schemaVersion"`
	ContractExists bool           `json:"contractExists"`
	Directories    []layout.Entry `json:"directories"`
}

type layoutAddResult struct {
	Action              string       `json:"action"`
	Project             string       `json:"project"`
	ContractPath        string       `json:"contractPath"`
	DryRun              bool         `json:"dryRun"`
	Changed             bool         `json:"changed"`
	ContractCreated     bool         `json:"contractCreated"`
	Replaced            bool         `json:"replaced"`
	PreviousDescription string       `json:"previousDescription,omitempty"`
	Directory           layout.Entry `json:"directory"`
}

func (a app) layoutList(args []string) int {
	fs := newFlags("layout list")
	projectPath := fs.String("project", ".", "project directory or any directory inside it")
	format, jsonFlag := outputFlags(fs)
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	root, err := project.Find(*projectPath)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	contract, exists, err := layout.Load(root)
	if err != nil {
		return a.operationError(err, layout.RelativePath)
	}
	result := layoutListResult{
		Action:         "layout.list",
		Project:        root,
		ContractPath:   filepath.Join(root, filepath.FromSlash(layout.RelativePath)),
		SchemaVersion:  contract.SchemaVersion,
		ContractExists: exists,
		Directories:    layout.Entries(contract),
	}
	a.emit(result, layoutListText(result), selectedFormat(*format, *jsonFlag, a.out))
	return 0
}

func (a app) layoutAdd(args []string) int {
	fs := newFlags("layout add")
	projectPath := fs.String("project", ".", "project directory or any directory inside it")
	name := fs.String("name", "", "top-level directory name (required)")
	description := fs.String("description", "", "what belongs in the directory (required)")
	replace := fs.Bool("replace", false, "replace an existing project-specific entry")
	ifNotExists := fs.Bool("if-not-exists", false, "succeed without changes when the entry already exists")
	dryRun := fs.Bool("dry-run", false, "preview without writing")
	format, jsonFlag := outputFlags(fs)
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	directoryName := strings.TrimSpace(*name)
	directoryDescription := strings.TrimSpace(*description)
	if directoryName == "" {
		return a.fail(failure{2, "invalid_arguments", "--name is required", "Run hp layout add --name lib --description \"Third-party reference material\" --dry-run", "--name"})
	}
	if directoryDescription == "" {
		return a.fail(failure{2, "invalid_arguments", "--description is required", "Describe what belongs in the directory and retry", "--description"})
	}
	if *replace && *ifNotExists {
		return a.fail(failure{2, "invalid_arguments", "--replace and --if-not-exists cannot be used together", "Choose --replace to update the entry or --if-not-exists to leave it unchanged", "--replace,--if-not-exists"})
	}
	if err := layout.ValidateName(directoryName); err != nil {
		return a.fail(failure{2, "invalid_arguments", err.Error(), "Pass one top-level directory name without slashes, such as --name lib", directoryName})
	}
	root, err := project.Find(*projectPath)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	contract, contractExists, err := layout.Load(root)
	if err != nil {
		return a.operationError(err, layout.RelativePath)
	}
	previousDescription, entryExists := contract.Directories[directoryName]
	entry := layout.Entry{Path: directoryName, Description: directoryDescription, Source: "project"}
	result := layoutAddResult{
		Action:          "layout.add",
		Project:         root,
		ContractPath:    filepath.Join(root, filepath.FromSlash(layout.RelativePath)),
		DryRun:          *dryRun,
		Changed:         true,
		ContractCreated: !contractExists,
		Directory:       entry,
	}
	if entryExists && *ifNotExists {
		result.Changed = false
		result.ContractCreated = false
		result.Directory.Description = previousDescription
		return a.emitLayoutAdd(result, "Directory already registered; no changes needed", *format, *jsonFlag, *dryRun)
	}
	if entryExists && *replace && previousDescription == directoryDescription {
		result.Changed = false
		result.ContractCreated = false
		return a.emitLayoutAdd(result, "Directory already has the requested description; no changes needed", *format, *jsonFlag, *dryRun)
	}
	if entryExists && *replace {
		result.Replaced = true
		result.PreviousDescription = previousDescription
	}
	contract, err = layout.Add(contract, directoryName, directoryDescription, *replace)
	if err != nil {
		switch {
		case errors.Is(err, layout.ErrBuiltIn):
			return a.fail(failure{5, "conflict", err.Error(), "Run hp layout list to use the built-in directory without registering it", directoryName})
		case errors.Is(err, layout.ErrExists):
			return a.fail(failure{5, "conflict", err.Error(), "Use --if-not-exists to keep it or --replace to update its description", directoryName})
		default:
			return a.operationError(err, directoryName)
		}
	}
	if *dryRun {
		return a.emitLayoutAdd(result, fmt.Sprintf("Would register %s/: %s", entry.Path, entry.Description), *format, *jsonFlag, true)
	}
	if err := layout.Save(root, contract); err != nil {
		return a.operationError(err, layout.RelativePath)
	}
	message := fmt.Sprintf("Registered %s/: %s", entry.Path, entry.Description)
	if result.Replaced {
		message = fmt.Sprintf("Updated %s/: %s", entry.Path, entry.Description)
	}
	return a.emitLayoutAdd(result, message, *format, *jsonFlag, false)
}

func (a app) emitLayoutAdd(result layoutAddResult, human, format string, jsonFlag, dryRun bool) int {
	if dryRun {
		a.emit(result, human, "json")
		return 10
	}
	a.emit(result, human, selectedFormat(format, jsonFlag, a.out))
	return 0
}

func layoutListText(result layoutListResult) string {
	var output strings.Builder
	fmt.Fprintf(&output, "Project directories (%s):", result.ContractPath)
	for _, entry := range result.Directories {
		fmt.Fprintf(&output, "\n- %s/ [%s] %s", entry.Path, entry.Source, entry.Description)
	}
	if !result.ContractExists {
		output.WriteString("\n\nNo project-specific contract file yet; run hp project upgrade or hp layout add.")
	}
	return output.String()
}
