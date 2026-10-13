package cli

import (
	"fmt"

	"github.com/chenhg5/harness-projects/internal/coordinator"
	"github.com/chenhg5/harness-projects/internal/store"
)

func (a app) coordinatorPlan(args []string) int {
	fs := newFlags("coordinator plan")
	projectPath := fs.String("project", "", "project directory (required)")
	format, jsonFlag := outputFlags(fs)
	if err := parse(fs, args); err != nil {
		return a.parseError(err)
	}
	if *projectPath == "" {
		return a.fail(failure{2, "invalid_arguments", "--project is required", "Run hp coordinator plan --help", "--project"})
	}
	s, err := store.Open(*projectPath)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	plan, err := coordinator.Build(s)
	if err != nil {
		return a.operationError(err, *projectPath)
	}
	a.emit(plan, fmt.Sprintf("%d ready task(s), %d waiting task(s)", plan.ReadyCount, plan.WaitingCount), selectedFormat(*format, *jsonFlag, a.out))
	return 0
}
