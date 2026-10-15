package wakeup

import (
	"testing"
	"time"
)

func TestSchedulesRequireValidIntervalOrFiveFieldCron(t *testing.T) {
	now := time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)
	config := Config{SchemaVersion: SchemaVersion, Enabled: true, Every: "30m", Worktree: ".", PromptFile: "agents/coordinator-wakeup.md", CreatedAt: now.Format(time.RFC3339), UpdatedAt: now.Format(time.RFC3339)}
	if next, err := Next(config, now); err != nil || !next.Equal(now.Add(30*time.Minute)) {
		t.Fatalf("interval: %s %v", next, err)
	}
	config.Every = ""
	config.Cron = "0 9 * * 1-5"
	config.TimeZone = "Asia/Shanghai"
	if next, err := Next(config, now); err != nil || next.Hour() != 1 {
		t.Fatalf("cron: %s %v", next, err)
	}
	config.Cron = "@every 1m"
	if err := Validate(config); err == nil {
		t.Fatal("non-five-field cron was accepted")
	}
}
