[English](README.md) | [中文](README.zh-CN.md)

# codex-orchestrator

**给真实代码仓库用的 Codex App-first 受监督工程循环。**

`codex-orchestrator` 帮一个主 Codex App session 跑更安全的外层循环：把工作拆成有界
任务，启动隔离的 worktree session，用本地 ledger 记录状态，用 heartbeat 唤醒检查，
审查完成分支，merge/push 已接受的工作，清理现场，然后继续推进路线图。

它的目标不是“让 Agent 一直自动写代码”，而是让每一个 worker branch 都可审查、可
拒绝、可合并、可清理。

## 为什么需要它

一个 Codex 聊天做小改动没问题。任务变大以后，问题会变多：

- worker session 完成时间不同；
- pending worktree 或卡住的 session 很容易漏看；
- 本地检查结果容易被描述得强于实际证据；
- 完成分支需要 review、merge、push、cleanup；
- 长循环容易变成随机扫小任务，而不是推进一个功能包。

`codex-orchestrator` 解决的是这套工程纪律。

## 它包含什么

- **Codex skill**：安装到 `~/.codex/skills/codex-orchestrator`，给 Codex App
  当编排 runbook。
- **可选 Go helper CLI**：`codex-orchestrator`，用于 ledger、status、heartbeat
  report、review pack、policy check 和本地更新。
- **文档和模板**：project map、package plan、orchestration policy、案例和 routine
  spec。

它不是 daemon，不是 package-manager-first 产品，不是完整 Agent Operating System，
也不是不经审查就自动写代码的 bot。Codex App 仍然负责创建和运行 worker session。

## 快速开始

在你想编排的项目里打开 Codex App，把下面这段作为第一条消息粘贴进去：

```text
我想在这个仓库里试用 codex-orchestrator。

请阅读 https://github.com/indiekitai/codex-orchestrator，并把它作为
Codex App-first 的工程编排工作流来使用。

如果这个仓库提供的 Codex App skill 还没安装，请安装到
~/.codex/skills/codex-orchestrator。

如果 Go helper CLI 对持久 ledger 状态有帮助，请先解释它的作用，然后在安全的情况下安装或构建。

先做 dry run：
- 检查 git status、worktree 和项目文档；
- 说明你会如何把工作拆成隔离的 Codex worktree session；
- 说明你会监控、审查、合并、push、清理哪些东西；
- 把证据标为 direct、proxy、local 或 blocked。

除非我明确批准，不要 push、deploy、删除 worktree，或执行破坏性操作。
```

Codex 应该读取本仓库，按需安装或更新 skill，判断 helper 是否有用，然后先给出只读
计划。

## 更新方式

更新需要用户触发，不会后台自动发生。推荐方式仍然是 Codex App-first：

```text
请从 https://github.com/indiekitai/codex-orchestrator 更新我的本地 codex-orchestrator。

检查 ~/.codex/skills/codex-orchestrator 里的已安装 skill，以及 PATH 上的 helper
binary。需要时 fetch 或 clone 最新仓库，更新 Codex App skill；只有在 helper 已经
安装或明确有用时才重建 Go helper；不要触碰任何项目里的
.codex-orchestrator/ledger.json。更新后跑一个 smoke check，并告诉我改了什么。
```

如果你已经安装了 helper，也可以运行：

```bash
codex-orchestrator self-update
codex-orchestrator self-update --from-github
codex-orchestrator self-update --with-helper
```

`self-update` 只刷新本地 skill/helper。它不会派发 session，不会修改项目 ledger，不会
merge、push、deploy 或 cleanup worktree。

## 工作方式

```mermaid
flowchart LR
    plan["规划功能包"] --> dispatch["派发有界 worker"]
    dispatch --> worktree["Codex App worktree session"]
    worktree --> ledger["Ledger + status"]
    ledger --> review["审查 diff、gate、证据"]
    review --> merge["Merge / push / cleanup"]
    merge --> next["继续或停止"]
```

这套循环刻意保守：

- repo/worktree truth 优先于聊天状态；
- shared contract、migration、API、设备、支付和部署串行处理；
- `direct`、`proxy`、`local`、`blocked` 证据不混写；
- 有空闲并发槽，不代表要派无关任务；
- worker 只提交自己的分支，统领负责审查和合并。

## 文档

- [完整指南](docs/full-guide.zh-CN.md)：原来的长 README，包含详细流程、routine、
  配置和示例。
- [v2 helper 用法](docs/v2-usage.md)：ledger、status、heartbeat、review pack、
  self-update 和 CLI 细节。
- [Routine 库](docs/routines/README.md)：包括 `pr-reviewer`、
  `stale-task-rescuer`、`ci-fixer`、`release-verifier`、
  `docs-drift-checker`、`evidence-label-auditor`、
  `orchestration-policy-auditor`、`roadmap-next-task-suggester` 和
  `budget-policy-report`。
- [路线图](docs/roadmap.md)：当前产品方向和已完成阶段。
- [餐厅 POS 重写案例](docs/case-studies/restaurant-pos-orchestration.md)：真实项目编排案例。
- [Loop Engineering 对齐笔记](docs/research/loop-engineering-alignment.md)：研究背景和设计取舍。
- [分发包说明](docs/distribution-package.md)：release assets 和 helper 打包细节。

## 同名项目说明

现在已经有其他叫 `codex-orchestrator` 的项目。这个仓库专注于 Codex App-first 的
worktree session 编排，不管理机器 fleet、API proxy、凭证，也不启动 tmux 里的 Codex
CLI agent。

## 许可证

MIT
