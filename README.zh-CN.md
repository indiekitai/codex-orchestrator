[English](README.md) | [中文](README.zh-CN.md)

# codex-orchestrator

**面向真实代码仓库的 Codex App 优先 Loop Engineering 工程控制层。**

Loop Engineering 是一套工程方法：把目标、状态、反馈、审查和退出条件放进 Agent
工作流里。`codex-orchestrator` 是 Codex App 里的那层工程控制层，也就是 harness：
先把工作拆成边界清楚的任务，再启动隔离的 worktree 会话；用本地账本（ledger）记录状态，用心跳
（heartbeat）定时唤醒检查；完成的分支先审查，确认可接受后再 merge/push，最后清理
worktree，并继续推进路线图。

它的目标不是“让 Agent 一直自动写代码”，而是让每一个 worker 分支都能被审查、拒绝、
合并和清理。

![codex-orchestrator overview](docs/assets/codex-orchestrator-social-card.svg)

## 为什么需要它

一个 Codex 聊天处理小改动通常够用。任务变大以后，麻烦会集中出现：

- 多个 worker 会话完成时间不同；
- 待创建的 worktree 或卡住的会话很容易漏掉；
- 本地检查结果常常被说得比实际证据更强；
- 完成分支需要审查、合并、推送和清理；
- 长时间循环容易变成随机扫小任务，而不是持续推进一个功能包。

`codex-orchestrator` 解决的是这层控制系统：它不是 Loop Engineering 这个概念本身，
而是让循环变得可观察、可验收、可恢复的一套实践工具。

## 它包含什么

- **Codex skill**：安装到 `~/.codex/skills/codex-orchestrator`，作为 Codex App 的
  编排手册。
- **可选 Go 辅助命令**：`codex-orchestrator`，用于维护账本、生成状态页和心跳报告、
  准备审查包、运行策略检查，以及更新本地安装。
- **文档和模板**：项目地图、功能包计划、编排策略、线程地图、概念库/收件箱、
  Pulse/Inbox/Router 提示词、案例和 routine 规范。

辅助命令现在也会显式跟踪信任边界：它可以记录 developer-agent misalignment 事件，
保存 worker 的约束栈快照，把完成声明和本地证据绑定起来，先生成“失败案例 -> 回归
fixture”的草案供审查，并在状态页里显示 `trustRisk` 风险块。

它不是后台守护进程，不是以包管理器为中心的产品，也不是完整的 Agent 操作系统；更不是
不经审查就自动写代码的机器人。Codex App 仍然负责创建和运行 worker 会话。

## 适用位置

| 入口 | 怎么用 |
|---|---|
| **Codex App** | 主入口。让 Codex App 阅读本仓库，安装/更新 skill，并先给出只读计划。只有用户明确批准后，才创建隔离的 worktree 会话，或验收、合并、清理已接受的分支。 |
| **Go 辅助命令** | 可选的本地静态状态层，用来维护 ledger、状态页、health 检查、review pack、routine 和 self-update。它不会创建 Codex App 会话，也不能替代统领判断。 |
| **Codex CLI** | 可以读取已安装 skill、运行辅助命令，但不能自己创建 Codex App worktree session。 |
| **Claude Code** | 使用兄弟项目 [claude-orchestrator](https://github.com/indiekitai/claude-orchestrator)，它把同一套 Loop Engineering 思路适配到 Claude Code 的终端优先工作流。 |
| **其他审查模型** | Pi、DeepSeek、Claude 或其他模型可以通过 review pack 参与，作为 proxy/advisory 证据，而不是自动 merge 授权。 |

## 快速开始

在你想编排的项目里打开 Codex App，把下面这段作为第一条消息粘贴进去：

```text
我想在这个仓库里试用 codex-orchestrator。

请阅读 https://github.com/indiekitai/codex-orchestrator，并把它作为
以 Codex App 为主入口的 Loop Engineering 工程控制层来使用。

如果这个仓库提供的 Codex App skill 还没安装，请安装到
~/.codex/skills/codex-orchestrator。

如果 Go 辅助命令对持久化任务状态有帮助，请先解释它的作用，然后在安全的情况下安装或构建。

先做只读演练：
- 检查 git status、worktree 和项目文档；
- 说明你会如何把工作拆成隔离的 Codex worktree 会话；
- 说明你会监控、审查、合并、推送和清理哪些内容；
- 把证据标为 direct、proxy、local 或 blocked。

除非我明确批准，不要推送、部署、删除 worktree，也不要执行破坏性操作。
```

Codex 应该先读取本仓库，按需安装或更新 skill，判断是否需要辅助命令，然后给出只读
计划。

## 更新方式

更新需要用户触发，不会在后台自动发生。推荐方式仍然是交给 Codex App：

```text
请从 https://github.com/indiekitai/codex-orchestrator 更新我的本地 codex-orchestrator。

检查 ~/.codex/skills/codex-orchestrator 里的已安装 skill，以及 PATH 里的辅助命令。
需要时 fetch 或 clone 最新仓库，更新 Codex App skill；只有在辅助命令已经安装或明确有用时
才重建 Go helper；不要触碰任何项目里的 .codex-orchestrator/ledger.json。更新后跑一次
烟测，并告诉我改了什么。
```

如果你已经安装了辅助命令，也可以直接运行：

```bash
codex-orchestrator self-update
codex-orchestrator self-update --from-github
codex-orchestrator self-update --with-helper
```

`self-update` 只刷新本地 skill 和辅助命令。它不会派发会话，不会修改项目账本，也不会
合并、推送、部署或清理 worktree。

## 工作方式

```mermaid
flowchart LR
    plan["规划功能包"] --> dispatch["派发有界 worker"]
    dispatch --> worktree["Codex App worktree 会话"]
    worktree --> ledger["账本 + 状态"]
    ledger --> review["审查 diff、检查项、证据"]
    review --> merge["合并 / 推送 / 清理"]
    merge --> next["继续或停止"]
```

这套循环刻意保守：

- repo/worktree 的真实状态优先于聊天里的说法；
- worker 是 Maker；统领和 routine 报告是 Checker；
- 共享契约、迁移、API、设备、支付和部署串行处理；
- `direct`、`proxy`、`local`、`blocked` 证据不混写；
- 有空闲并发槽，不代表要派无关任务；
- worker 只提交自己的分支，统领负责审查和合并。

## 线程布局

长期使用 Codex App 时，一个线程通常不够。推荐把职责分开：

- **Project Orchestrator**：负责 repo truth、ledger、worker 派发、验收、合并、
  推送、清理和功能包收口。
- **Pulse**：按计划醒来，只报告有意义的变化，比如 heartbeat 漏跑、worker 阻塞、
  等待验收的 commit。
- **Inbox**：收集 GitHub issue、用户反馈、外部 review、真实项目使用体验，先归类，
  不直接派任务。
- **Router**：读取线程地图，判断新输入应该交给哪个线程。它负责路由，不负责写代码、
  派 worker、merge 或 push。
- **Log**：记录人能看懂的决策和功能包进展。

`codex-orchestrator init --write-templates` 现在会生成
`.codex-orchestrator/thread-map.md` 和
`.codex-orchestrator/pulse-threads.md`，把这些长期线程关系落到文件里，而不是靠聊天记忆。

它也会生成一个很轻的本地知识层：`.codex-orchestrator/concepts.md` 用来记录术语、
稳定规则、历史决策和踩坑记录；`.codex-orchestrator/inbox.md` 用来收集 issue、反馈、
外部 review 和 Pulse 输出，先归类，再变成任务。

## 文档

- [完整指南](docs/full-guide.zh-CN.md)：原来的长 README，包含详细流程、routine、
  配置和示例。
- [v2 辅助命令用法](docs/v2-usage.md)：账本、状态、心跳、审查包、self-update
  和 CLI 细节。
- [Router 指南](docs/router.zh-CN.md)：如何拆分 Project Orchestrator、Pulse、
  Inbox、Router 和 Log 线程，同时避免让 Router 下场实现。
- [Routine 库](docs/routines/README.md)：包括 `pr-reviewer`、
  `stale-task-rescuer`、`ci-fixer`、`release-verifier`、
  `docs-drift-checker`、`evidence-label-auditor`、
  `orchestration-policy-auditor`、`roadmap-next-task-suggester` 和
  `budget-policy-report`。
- [路线图](docs/roadmap.md)：当前产品方向和已完成阶段。
- [餐厅 POS 重写案例](docs/case-studies/restaurant-pos-orchestration.md)：真实项目编排案例。
- [Loop Engineering 对齐笔记](docs/research/loop-engineering-alignment.md)：研究背景和设计取舍。
- [Developer-agent misalignment 笔记](docs/research/developer-agent-misalignment.md)：
  为什么工具需要记录约束、完成声明和信任风险。
- [分发包说明](docs/distribution-package.md)：release 资产和辅助命令打包细节。

## 相关项目

- [indiekitai/claude-orchestrator](https://github.com/indiekitai/claude-orchestrator)：
  面向 Claude Code 用户的兄弟项目。它把同一套 Loop Engineering 工程控制层思路，
  适配到 Claude Code 的终端优先工作流和 Claude 侧的 skill / runtime 习惯。

## 同名项目说明

现在已经有其他项目也叫 `codex-orchestrator`。这个仓库专注于 Codex App 优先的
worktree 会话编排；它不管理机器集群、API 代理或凭证，也不负责启动 tmux 里的 Codex
CLI agent。

## 许可证

MIT
