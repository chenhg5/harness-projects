package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/chenhg5/harness-projects/internal/project"
	"github.com/chenhg5/harness-projects/internal/skillmount"
)

func (a app) skillSync(args []string) int {
	fs := newFlags("skill sync")
	projectPath := fs.String("project", ".", "project directory")
	targetValue := fs.String("targets", "all", "comma-separated codex,claude,cursor,agents or all")
	prune := fs.Bool("prune", false, "remove stale symlinks that point into project skills")
	dryRun := fs.Bool("dry-run", false, "preview changes without writing")
	format, jsonFlag := outputFlags(fs)
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	targets, err := skillmount.ParseTargets(*targetValue)
	if err != nil {
		return a.parseError(err)
	}
	root, err := project.Find(*projectPath)
	if err != nil {
		return a.fail(failure{3, "not_found", err.Error(), "Run inside a Harness Project or pass --project", *projectPath})
	}
	plan, err := skillmount.BuildPlan(root, targets, *prune)
	if err != nil {
		var conflict *skillmount.ConflictError
		if errors.As(err, &conflict) {
			return a.fail(failure{5, "conflict", err.Error(), "Move or rename the conflicting entry, then retry", conflict.Path})
		}
		return a.operationError(err, root)
	}
	if *dryRun {
		a.emit(plan, fmt.Sprintf("Would apply %d skill mount change(s)", len(plan.Changes)), "json")
		return 10
	}
	if err := skillmount.Apply(plan); err != nil {
		return a.operationError(err, root)
	}
	message := fmt.Sprintf("Skill mounts are synchronized for %s (%d change(s), %d unchanged)", strings.Join(plan.Targets, ", "), len(plan.Changes), plan.Unchanged)
	a.emit(plan, message, selectedFormat(*format, *jsonFlag, a.out))
	return 0
}
