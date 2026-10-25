package cli

import (
	"context"
	"fmt"

	"github.com/chenhg5/harness-projects/internal/doctor"
	"github.com/chenhg5/harness-projects/internal/git"
	"strings"
	"time"

	"github.com/chenhg5/harness-projects/internal/project"
)

func (a app) projectDoctor(args []string) int {
	fs := newFlags("project doctor")
	path := fs.String("path", ".", "project directory")
	format, jsonFlag := outputFlags(fs)
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	root, err := project.Find(*path)
	if err != nil {
		return a.operationError(err, *path)
	}
	report, err := doctor.Inspect(root)
	if err != nil {
		return a.operationError(err, *path)
	}
	human := "Project is healthy"
	if !report.Healthy {
		human = fmt.Sprintf("Project doctor found %d issue(s)", len(report.Findings))
		for _, finding := range report.Findings {
			human += fmt.Sprintf("\n- [%s] %s", finding.Code, finding.Message)
			if finding.Path != "" {
				human += " (" + finding.Path + ")"
			}
			if finding.Suggestion != "" {
				human += " — " + finding.Suggestion
			}
		}
	}
	a.emit(report, human, selectedFormat(*format, *jsonFlag, a.out))
	if !report.Healthy {
		return 1
	}
	return 0
}

func (a app) projectUpgrade(args []string) int {
	fs := newFlags("project upgrade")
	path := fs.String("path", ".", "project directory")
	format, jsonFlag := outputFlags(fs)
	dryRun := fs.Bool("dry-run", false, "preview without writing")
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	root, err := project.Find(*path)
	if err != nil {
		return a.operationError(err, *path)
	}
	plan, err := project.PlanUpgrade(root)
	if err != nil {
		return a.operationError(err, *path)
	}
	if *dryRun {
		a.emit(plan, fmt.Sprintf("Would apply %d project upgrade change(s)", len(plan.Changes)), "json")
		return 10
	}
	if err := project.ApplyUpgrade(plan); err != nil {
		return a.operationError(err, *path)
	}
	a.emit(plan, fmt.Sprintf("Applied %d project upgrade change(s)", len(plan.Changes)), selectedFormat(*format, *jsonFlag, a.out))
	return 0
}

func (a app) projectInit(args []string) int {
	fs := newFlags("project init")
	path := fs.String("path", "", "target project directory (required)")
	name := fs.String("name", "", "project name (default: directory name)")
	format, jsonFlag := outputFlags(fs)
	dryRun := fs.Bool("dry-run", false, "preview changes without writing")
	ifNotExists := fs.Bool("if-not-exists", false, "succeed if a valid project already exists")
	noGit := fs.Bool("no-git", false, "do not create a Git repository and initial commit")
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	if *path == "" {
		return a.fail(failure{2, "invalid_arguments", "--path is required", "Run hp project init --path <directory>", "--path"})
	}
	result, files, err := project.PlanInit(*path, *name, *ifNotExists, time.Now())
	if err != nil {
		return a.operationError(err, *path)
	}
	if *dryRun {
		a.emit(result, "Would initialize project at "+result.Path, "json")
		return 10
	}
	if err := project.ApplyInit(result, files); err != nil {
		return a.operationError(err, *path)
	}
	output := struct {
		project.InitResult
		Git string `json:"git"`
	}{InitResult: result, Git: "skipped"}
	human := "Project ready at " + result.Path
	if result.Created && !*noGit {
		output.Git, human = initProjectGit(result.Path, human)
	}
	if result.Created {
		human += "\n\nNext:\n  cd " + result.Path + "\n  hp goal create --project . --id <goal> --title \"...\"   # what done looks like\n  hp brief                                          # what any agent sees first"
	}
	a.emit(output, human, selectedFormat(*format, *jsonFlag, a.out))
	return 0
}

// initProjectGit gives a new project its own history. Missing Git or a missing
// identity leaves a usable project and says what to do instead of failing.
func initProjectGit(root, human string) (string, string) {
	if !git.Available() {
		return "unavailable", human + "\nGit not found; run git init in the project to keep its history."
	}
	if git.IsRepositoryRoot(root) {
		return "existing", human
	}
	initialized, committed, err := git.InitAndCommit(context.Background(), root, "Create Harness Project")
	switch {
	case committed:
		return "committed", human + " (Git repository with initial commit)"
	case initialized:
		return "initialized", human + "\nGit repository created; the initial commit failed: " + err.Error()
	default:
		return "failed", human + "\nCould not create a Git repository: " + err.Error()
	}
}

func (a app) projectStatus(args []string, validate bool) int {
	command := "project status"
	if validate {
		command = "project validate"
	}
	fs := newFlags(command)
	path := fs.String("path", "", "project directory (required)")
	format, jsonFlag := outputFlags(fs)
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	if *path == "" {
		return a.fail(failure{2, "invalid_arguments", "--path is required", "Run hp " + command + " --path <directory>", "--path"})
	}
	status := project.Inspect(*path)
	message := "Project structure is valid; run hp project doctor for operational drift"
	if !status.Valid {
		message = "Project has structural problems: " + strings.Join(status.Problems, "; ")
	}
	a.emit(status, message, selectedFormat(*format, *jsonFlag, a.out))
	if validate && !status.Valid {
		return 1
	}
	return 0
}
