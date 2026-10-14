package dashboard

import (
	"net/http"
	"strconv"
	"strings"
)

type labels struct {
	SkipToOverview                  string
	LocalProjectView                string
	Language                        string
	ProjectSections                 string
	Overview                        string
	Goals                           string
	Tasks                           string
	NextActions                     string
	Repositories                    string
	Activity                        string
	RefreshView                     string
	ProjectOverview                 string
	Updated                         string
	ProjectSummary                  string
	CurrentPhase                    string
	TasksCompleted                  string
	TrackedTasksProgress            string
	InProgress                      string
	Blocked                         string
	ParticipatingSourceRepositories string
	Outcomes                        string
	NoGoals                         string
	DoneWhen                        string
	EvidenceRequired                string
	KeyResults                      string
	Current                         string
	Target                          string
	LinkedTasksCompleted            string
	ReadGoalDocument                string
	Execution                       string
	NoTasks                         string
	WhatsNext                       string
	Risks                           string
	NoNextAction                    string
	SourceCode                      string
	NoRepositories                  string
	AgentActivity                   string
	LatestWorklog                   string
	ReadWorklog                     string
	NoWorklog                       string
	LatestHandoff                   string
	ReadHandoff                     string
	ReadOnlyLocalView               string
	RefreshHint                     string
	JSONAPI                         string
}

var english = labels{
	SkipToOverview:                  "Skip to project overview",
	LocalProjectView:                "Local project view",
	Language:                        "Language",
	ProjectSections:                 "Project sections",
	Overview:                        "Overview",
	Goals:                           "Goals",
	Tasks:                           "Tasks",
	NextActions:                     "Next actions",
	Repositories:                    "Repositories",
	Activity:                        "Activity",
	RefreshView:                     "Refresh view",
	ProjectOverview:                 "Project overview",
	Updated:                         "updated",
	ProjectSummary:                  "Project summary",
	CurrentPhase:                    "Current phase",
	TasksCompleted:                  "Tasks completed",
	TrackedTasksProgress:            "Tracked task progress",
	InProgress:                      "In progress",
	Blocked:                         "blocked",
	ParticipatingSourceRepositories: "participating source repositories",
	Outcomes:                        "Outcomes",
	NoGoals:                         "No goals recorded yet.",
	DoneWhen:                        "Done when",
	EvidenceRequired:                "Evidence required:",
	KeyResults:                      "Key results",
	Current:                         "Current",
	Target:                          "Target",
	LinkedTasksCompleted:            "linked tasks completed",
	ReadGoalDocument:                "Read goal document",
	Execution:                       "Execution",
	NoTasks:                         "No tasks recorded yet.",
	WhatsNext:                       "What's next",
	Risks:                           "Risks",
	NoNextAction:                    "No next action recorded.",
	SourceCode:                      "Source code",
	NoRepositories:                  "No repositories registered.",
	AgentActivity:                   "Agent activity",
	LatestWorklog:                   "Latest work log",
	ReadWorklog:                     "Read work log",
	NoWorklog:                       "No work log yet.",
	LatestHandoff:                   "Latest handoff",
	ReadHandoff:                     "Read handoff",
	ReadOnlyLocalView:               "Read-only local view",
	RefreshHint:                     "Use Refresh view to load new changes",
	JSONAPI:                         "JSON API",
}

var simplifiedChinese = labels{
	SkipToOverview:                  "跳转到项目概览",
	LocalProjectView:                "本地项目看板",
	Language:                        "语言",
	ProjectSections:                 "项目导航",
	Overview:                        "概览",
	Goals:                           "目标",
	Tasks:                           "任务",
	NextActions:                     "下一步",
	Repositories:                    "仓库",
	Activity:                        "动态",
	RefreshView:                     "刷新看板",
	ProjectOverview:                 "项目概览",
	Updated:                         "更新于",
	ProjectSummary:                  "项目摘要",
	CurrentPhase:                    "当前阶段",
	TasksCompleted:                  "已完成任务",
	TrackedTasksProgress:            "已登记任务完成率",
	InProgress:                      "进行中",
	Blocked:                         "个受阻",
	ParticipatingSourceRepositories: "个关联代码仓库",
	Outcomes:                        "成果",
	NoGoals:                         "暂无目标。",
	DoneWhen:                        "完成条件",
	EvidenceRequired:                "所需证据：",
	KeyResults:                      "关键结果",
	Current:                         "当前",
	Target:                          "目标",
	LinkedTasksCompleted:            "个关联任务已完成",
	ReadGoalDocument:                "查看目标文档",
	Execution:                       "执行",
	NoTasks:                         "暂无任务。",
	WhatsNext:                       "接下来",
	Risks:                           "风险",
	NoNextAction:                    "暂无下一步行动。",
	SourceCode:                      "源代码",
	NoRepositories:                  "暂无关联仓库。",
	AgentActivity:                   "Agent 动态",
	LatestWorklog:                   "最新工作日志",
	ReadWorklog:                     "查看工作日志",
	NoWorklog:                       "暂无工作日志。",
	LatestHandoff:                   "最新交接记录",
	ReadHandoff:                     "查看交接记录",
	ReadOnlyLocalView:               "本地只读看板",
	RefreshHint:                     "点击“刷新看板”加载最新改动",
	JSONAPI:                         "JSON 接口",
}

var statusLabels = map[string]map[string]string{
	"en": {
		"unknown": "Unknown", "on_track": "On track", "at_risk": "At risk", "blocked": "Blocked", "complete": "Complete",
		"draft": "Draft", "active": "Active", "completed": "Completed", "archived": "Archived", "pending": "Pending",
		"ready": "Ready", "in_progress": "In progress", "failed": "Failed", "cancelled": "Cancelled", "planned": "Planned",
		"achieved": "Achieved", "missed": "Missed", "succeeded": "Succeeded", "paused": "Paused", "running": "Running",
	},
	"zh-CN": {
		"unknown": "未知", "on_track": "正常", "at_risk": "有风险", "blocked": "受阻", "complete": "已完成",
		"draft": "草稿", "active": "进行中", "completed": "已完成", "archived": "已归档", "pending": "待处理",
		"ready": "就绪", "in_progress": "进行中", "failed": "失败", "cancelled": "已取消", "planned": "已计划",
		"achieved": "已达成", "missed": "未达成", "succeeded": "成功", "paused": "已暂停", "running": "运行中",
	},
}

type pageView struct {
	Summary
	Locale string
	L      labels
}

func (p pageView) Status(value string) string {
	if translated, ok := statusLabels[p.Locale][value]; ok {
		return translated
	}
	return value
}

func newPageView(summary Summary, locale string) pageView {
	view := pageView{Summary: summary, Locale: locale, L: english}
	if locale == "zh-CN" {
		view.L = simplifiedChinese
	}
	return view
}

func selectLocale(r *http.Request) string {
	if values, explicit := r.URL.Query()["lang"]; explicit {
		if len(values) > 0 && isChinese(values[0]) {
			return "zh-CN"
		}
		return "en"
	}
	selected, quality := "en", 0.0
	for _, preference := range strings.Split(r.Header.Get("Accept-Language"), ",") {
		parts := strings.Split(preference, ";")
		language := strings.TrimSpace(parts[0])
		locale := ""
		if isChinese(language) {
			locale = "zh-CN"
		} else if strings.HasPrefix(strings.ToLower(language), "en") {
			locale = "en"
		}
		if locale == "" {
			continue
		}
		weight := 1.0
		for _, parameter := range parts[1:] {
			key, value, ok := strings.Cut(strings.TrimSpace(parameter), "=")
			if !ok || !strings.EqualFold(key, "q") {
				continue
			}
			parsed, err := strconv.ParseFloat(value, 64)
			if err != nil || parsed < 0 || parsed > 1 {
				weight = 0
			} else {
				weight = parsed
			}
		}
		if weight > quality {
			selected, quality = locale, weight
		}
	}
	return selected
}

func isChinese(language string) bool {
	return strings.HasPrefix(strings.ToLower(language), "zh")
}
