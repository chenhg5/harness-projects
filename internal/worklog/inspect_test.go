package worklog

import (
	"reflect"
	"testing"
)

func TestMissingCoreFields(t *testing.T) {
	cliHeader := "# Work log: api\n- Agent: codex\n- Harness: codex\n- Created: 2026-10-26T00:00:00Z\n- Task: api\n- Run: None recorded.\n"
	cliFooter := "## Changes\nNone recorded.\n## Verification\nNone recorded.\n## Blockers\nNone recorded.\n"
	cases := []struct {
		name string
		log  string
		want []string
	}{
		{"canonical without verification evidence", cliHeader + "## Objective\nShip API\n## Completed\nAdded route\n## Remaining\nTests\n## Next action\nRun tests\n" + cliFooter, nil},
		{"legacy headings and metadata", "# Work log\n- **Objective:** Fix search\n## Changes\nDebounced input.\n## Verification\nTests passed.\n## Follow-up\nPreview still needed; review the PR.\n", nil},
		{"translated headings", "# 工作日志\n## 目标\n减少延迟\n## 已完成\n加入缓存\n## 剩余工作与下一步\n还需压测，然后提交结果\n", nil},
		{"freeform narrative", "# Work log\nFixed the search delay by adding a debounce; tests passed and the next agent can review the PR.\n", nil},
		{"combined future work", cliHeader + "## Objective\nShip API\n## Completed\nAdded route\n## Remaining\nRun the tests next.\n" + cliFooter, nil},
		{"issue reference is content", cliHeader + "## Objective\nShip API\n## Completed\nAdded route\n## Remaining\n#2034 needs review.\n" + cliFooter, nil},
		{"colon in prose is content", cliHeader + "## Objective\nShip API\n## Completed\nAdded route\n## Remaining\nMaintainer decisions: choose release version.\n" + cliFooter, nil},
		{"missing future work", cliHeader + "## Objective\nShip API\n## Completed\nAdded route\n" + cliFooter, []string{"remaining", "next action"}},
		{"empty required section", cliHeader + "## Objective\nShip API\n## Completed\n...\n## Remaining\nTests\n## Next action\nRun tests\n" + cliFooter, []string{"completed"}},
		{"empty document", "# Work log\n", []string{"objective", "completed", "remaining", "next action"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := MissingCoreFields(tc.log); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("MissingCoreFields() = %v, want %v", got, tc.want)
			}
		})
	}
}
