<h1 align="center">Harness Projects</h1>
<p align="center"><strong>每个新的 Agent 会话都从零开始。你的项目不必如此。</strong></p>
<p align="center">保存在 Git 里的项目记忆。Claude Code、Codex、Cursor 每次开新会话都会先读它，接着上一个会话停下的地方继续。</p>

<p align="center">
  <a href="https://github.com/chenhg5/harness-projects/releases"><img alt="Release" src="https://img.shields.io/github/v/release/chenhg5/harness-projects?label=release"></a>
  <a href="https://github.com/chenhg5/harness-projects/actions/workflows/ci.yml"><img alt="CI" src="https://github.com/chenhg5/harness-projects/actions/workflows/ci.yml/badge.svg"></a>
  <a href="go.mod"><img alt="Go 1.23+" src="https://img.shields.io/badge/Go-1.23%2B-00ADD8?logo=go"></a>
  <a href="LICENSE"><img alt="License" src="https://img.shields.io/github/license/chenhg5/harness-projects"></a>
</p>

<p align="center">
  <a href="#快速开始">快速开始</a> ·
  <a href="#试试示例项目">示例</a> ·
  <a href="#工作方式">工作方式</a> ·
  <a href="docs/cli.md">CLI 文档</a> ·
  <a href="README.md">English</a>
</p>

---

只要你用 Coding Agent 做过超过一个会话的事，大概都遇到过：

- **新会话什么都不知道。** 每次都要重新讲一遍目标、已经做的决定和做到一半的事。
- **换个工具就全丢了。** Codex 昨天做了什么，今天的 Claude Code 完全看不到。
- **多个 Agent、多个仓库各干各的。** 没人掌握整体计划，工作重复甚至互相冲突。

Harness Projects 把目标、计划、任务、决定和会话记录写成普通文件，放进独立的 Git 仓库。会话开始时，hook 把一份简短的 **brief** 交给新 Agent；会话结束时，hook 确保它给下一个会话留下记录。

<p align="center">
  <img src="docs/assets/brief-demo.svg" alt="新的 Claude Code 会话在开始时收到 hp brief，并接着另一个 Agent 留下的工作继续" width="800">
</p>
<p align="center"><sub>节选自 <a href="examples/checkout-revamp">examples/checkout-revamp</a> 的一次真实运行。Agent 的回复原样保留；它没有任何聊天记录，只有 brief。</sub></p>

## 快速开始

```bash
curl -fsSL https://raw.githubusercontent.com/chenhg5/harness-projects/main/install.sh | sh
hp project init --path ./my-project --name "My Project"
cd my-project && claude
```

就这些。`hp project init` 会把项目建成独立的 Git 仓库，写好给各个 Harness 的说明，并配置好 Claude Code hooks。之后：

| 时机 | 发生什么 |
| --- | --- |
| 会话开始 | Agent 收到 `hp brief`：目标、下一步、进行中/可开始/被阻塞的任务、当前计划，以及上个会话没做完的事。会话自动登记，不需要手动复制 ID。 |
| Agent 工作 | 用 `hp task update` 更新任务，把决定写进 `spec/`。 |
| 会话结束 | 如果改了文件却没留记录，会被提醒一次写 `hp worklog create`（交给别人时写 `hp handoff create`）。 |

直接用自然语言告诉 Agent 你要什么（“目标是把结账时间减半，先拆任务”），生成的 `AGENTS.md` 和项目 skill 会教它用哪些命令。

**Codex 和 Cursor** 会读取生成的 `AGENTS.md` 和 Cursor 规则，每个会话先运行 `hp brief`；Codex 会话通过 `CODEX_THREAD_ID` 自动识别。**已有项目**可以用 `hp hook install` 补上 Claude Code hooks。其他安装方式见 [INSTALL.md](INSTALL.md)。

## 试试示例项目

```bash
git clone https://github.com/chenhg5/harness-projects && cd harness-projects/examples/checkout-revamp
hp brief
```

[`examples/checkout-revamp`](examples/checkout-revamp) 是一个进行到一半的完整项目：一个目标，横跨两个仓库的六个任务（已完成、进行中、可开始、被阻塞），一条设计决定，一份 Codex 工作日志，以及一次 Codex → Claude Code 的交接。用任意 Agent 打开它，让它继续即可。

## 工作方式

```text
my-project/              # 独立的 Git 仓库：共享的项目记忆
├── AGENTS.md            # 所有 Agent 在这里怎么工作（CLAUDE.md 和 Cursor 规则指向它）
├── PROJECT.md           # 目的、范围、涉及的仓库
├── goals/               # 什么算完成
├── state/               # 当前计划、下一步、健康度
├── tasks/               # 带依赖和状态的工作项
├── spec/                # 决定和需求
├── worklogs/            # 每个会话做了什么、剩下什么
├── handoffs/            # Agent 或人之间的明确交接
├── context/             # 调研和参考资料
└── repos/               # 你的源码仓库和 worktree（各自保留 Git 历史）
```

- **是文件，不是服务。** 全部是 Markdown 和 JSON，任何 Agent 都能读，任何人都能在 PR 里审。没有服务器、账号，也不绑定某个工具。
- **一份 brief，任意 Harness。** 在项目里任何位置（包括嵌套的仓库 worktree）运行 `hp brief`，Claude Code、Codex、Cursor 和人看到的都是同一份信息。
- **多个仓库，一个计划。** 同时涉及 `web` 和 `api` 的需求只有一个目标、一张任务图；各仓库保留自己的分支。
- **对 Agent 友好的 CLI。** 每个 `hp` 命令都有 JSON 输出、dry run 和稳定的退出码，Agent 不需要猜。

用 `hp project serve --path . --open` 打开只读看板。

### 为什么不直接用……

| | 跨会话保留进度 | 跨工具共享 | 在 Git 中审查 | 跨仓库 |
| --- | --- | --- | --- | --- |
| 只用 `CLAUDE.md` / `AGENTS.md` | 只有静态说明 | 部分 | 是 | 否 |
| 单个工具内置的 Projects 或记忆 | 是 | 否 | 否 | 否 |
| Issue 跟踪系统 | 要有人手动搬进来 | 会话开始时不会读 | 否 | 是 |
| **Harness Projects** | 是 | 是 | 是 | 是 |

## 进阶：协调多个 Agent

同时运行多个会话时，`hp` 还能在它们之间分派工作：

- `hp agent discover` 列出已登记的会话；`hp task inbox` 显示分给当前会话的任务。
- 可移交的 **Coordinator** 角色能分配任务，并用 `hp coordinator dispatch` 投递给某个确切的已有会话；结果不确定的投递不会被自动重试。
- `hp wakeup` 按固定间隔或 cron 唤醒 Coordinator。

这些命令和安全规则见 [CLI 文档](docs/cli.md)，规划中的能力（远程运行、事件触发、审批）见 [能力地图](docs/capability-map.md)。

## 更多文档

| 文档 | 内容 |
| --- | --- |
| [安装](INSTALL.md) | 发布包、Go 安装和源码构建 |
| [CLI 文档](docs/cli.md) | 全部命令、hooks、会话识别、分派和看板 |
| [产品模型](docs/product.md) | Project、Goal、Task、Run、Agent、Thread、Coordinator 等概念 |
| [规范](spec/README.md) 与 [RFC](rfcs/README.md) | 可移植的文件约定和设计提案 |
| [更新日志](CHANGELOG.md) | 每个版本的变化 |

## 参与贡献

欢迎提交 Issue 和 Pull Request，见 [CONTRIBUTING.md](CONTRIBUTING.md)。CLI 遵循 [Agent CLI Guide](https://github.com/Johnixr/agent-cli-guide)：显式参数、结构化输出、dry run 和稳定的退出码。

## 许可证

Apache License 2.0，见 [LICENSE](LICENSE)。
