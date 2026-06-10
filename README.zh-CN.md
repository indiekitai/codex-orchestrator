[English](README.md) | [中文](README.zh-CN.md)

# codex-orchestrator

**OpenAI Codex App 的 Loop Engineering skill。** 它把单个写代码助手变成一个受监督的工程循环：从路线图中拆任务，派发到隔离 worktree 会话，用 heartbeat 巡检，审查并合并干净分支，恢复卡住的 session，并在安全时继续派发下一批。

## 🔥 痛点

单个 Codex 会话处理小任务没问题。但遇到大活——新建一套 API、重写一个模块、跨服务开发——就开始痛了：

- **来回切换**：手动检查"第 3 个会话跑完没"，同时第 1 个会话等着合并
- **会话卡死**：某个会话在 80% 的地方卡住了，你一个小时后才发现
- **合并冲突**：两个会话改了同一个 proto 文件，各自跑完，合并时互相打架
- **过夜值守**：你想睡前派 3 个任务，但不敢放着不管

## 🏗️ 工作原理

```
                    ┌─────────────────────┐
                    │   编排器            │
                    │   (主线程)          │
                    └──────┬──────────────┘
                           │
              ┌────────────┼────────────────┐
              ▼            ▼                ▼
     ┌──────────────┐ ┌──────────────┐ ┌──────────────┐
     │  会话 A      │ │  会话 B      │ │  会话 C      │
     │  worktree/a  │ │  worktree/b  │ │  worktree/c  │
     │  branch: a   │ │  branch: b   │ │  branch: c   │
     └──────┬───────┘ └──────┬───────┘ └──────┬───────┘
            │                │                │
            ▼                ▼                ▼
     ┌──────────────────────────────────────────────┐
     │            5 分钟心跳巡检                     │
     │  ┌─ 检查 git 状态 ───────────────────────┐   │
     │  │  已提交? → 审查 → 合并 → 清理          │   │
     │  │  卡住+有commit? → 审查 → 合并           │   │
     │  │  卡住+有diff?   → 补发prompt继续       │   │
     │  │  还在跑? → 继续等                      │   │
     │  └───────────────────────────────────────┘   │
     │  全部完成? → 派发下一批                      │
     └──────────────────────────────────────────────┘
```

## ✨ 核心能力

| 能力 | 说明 |
|------|------|
| **有界任务契约** | 每个会话拿到精确的范围：允许路径、禁止路径、基准 commit、验收命令、证据标签 |
| **自动并发控制** | 默认 2 个会话，写入集不相交时可开 3 个。共享契约（proto、迁移、API）串行化 |
| **5 分钟心跳** | 定期巡检，将 Codex 线程状态与实际 git 状态对账——杜绝过夜静默卡死 |
| **卡住会话恢复** | 会话空转 >15 分钟时：已有干净 commit → 直接审查合并；有未提交的有用改动 → 补发 prompt 让 session 继续；没有有用 diff → 标记放弃 |
| **反浅切片门禁** | 拒绝"又一个占位页面"类任务。强制要求纵向完成、运行时证明或移除阻断点 |
| **证据纪律** | 证明标签：`direct`（直接）、`proxy`（代理）、`blocked`（阻断）。不许把单元测试升级成生产证明 |
| **强制自审查** | 每个会话必须在交付前审查自己的 diff。编排器在合并前再审一遍 |
| **特性包规划** | 当某个领域有多个局部闭合时，升级为完整里程碑而非继续堆小切片 |
| **连续运转** | 不只做一个功能——读路线图、选下一个可做的功能、派发、重复。专为过夜/无人值守多功能运行设计 |
| **续跑保护** | 单个任务 heartbeat 只有在统领确认大队列不需要继续后，才可以停止 |

## ✅ 前置条件与安全边界

这个仓库是一个 Codex skill / runbook，不是独立后台守护进程。完整的自动循环依赖宿主环境提供对应能力，尤其是：

- 创建或继续隔离的 Codex 会话
- 创建独立 git worktree，或等价的隔离 worker 环境
- 读取线程状态并检查 worktree 的 git 状态
- 创建/更新定时巡检 automation 或 heartbeat reminder
- 按项目正常 git 策略执行 merge / push

如果这些工具不可用，这个 skill 应降级为手动编排清单：少开会话，直接检查 git 状态，并且不要假装已经完成监控、合并、推送或清理。

开源场景下建议先在可丢弃仓库或功能分支上 dry run。自动 push 应保持关闭，直到你确认 review gate 和项目分支保护策略可靠。

核心 skill 本身不依赖 Python。v2 helper 现在是 Go CLI，可以构建成单文件二进制。Python helper 会先保留，作为开发原型和兼容参考。

## 🚫 这不是什么

它不是工程判断、代码审查或生产验收的替代品。它的目标是让 AI 辅助开发更结构化：有界任务、隔离 worktree、明确证据标签、合并前审查。

重点不是让 agent 永远无人值守地写代码，而是把人放在更高杠杆的位置：设计循环、审查证据、决定什么可以发布。

## 🚀 快速开始

### 1. 安装 skill

```bash
# 复制到 Codex skills 目录
cp -r codex-orchestrator ~/.codex/skills/delegated-session-orchestrator
```

可选：安装 v2 Go helper：

```bash
scripts/install.sh
codex-orchestrator init
```

发布第一个 GitHub release 后，也可以直接下载预构建的
`codex-orchestrator_<os>_<arch>` 二进制文件并放到 `PATH` 里。

### 2. 在 Codex 中使用

打开 Codex 会话，告诉它编排：

```
用 $delegated-session-orchestrator 把这个特性拆成有界的 worktree 会话，
审查合并完成的分支，然后派发下一批。
```

或者更具体：

```
我需要构建一套 REST API，包含用户认证、CRUD、分页和限流。
用 $delegated-session-orchestrator 今晚并行跑。
```

### 3. 去睡觉

编排器会自动：
1. 将工作分解为有界任务契约
2. 将会话派发到独立的 worktree
3. 每 5 分钟跑一次心跳巡检
4. 审查并合并完成的会话
5. 收割卡住会话的可用 commit
6. 有空位时派发下一批任务

安装 v2 helper 后，它还可以把任务状态持久化到
`.codex-orchestrator/ledger.json`，并写出 heartbeat report，让新的统领
session 能从 repo/ledger truth 恢复现场。

## 📋 使用示例

**目标**：构建一套包含 4 个主要组件的 REST API。

编排器分解为并行会话：

```
会话 A: codex/api-auth
  允许: src/auth/**, src/middleware/auth.ts, tests/auth/**
  禁止: src/db/migrations/**, src/api/products/**
  验收: npm test -- --grep auth

会话 B: codex/api-products
  允许: src/api/products/**, src/models/product.ts, tests/products/**
  禁止: src/auth/**, src/db/migrations/**
  验收: npm test -- --grep products
```

A 和 B 并行运行（写入集不相交）。两者合并后，编排器派发：

```
会话 C: codex/api-pagination
  允许: src/middleware/pagination.ts, src/api/**/router.ts, tests/pagination/**
  验收: npm test -- --grep pagination

会话 D: codex/api-rate-limit
  允许: src/middleware/rateLimit.ts, src/config/limits.ts, tests/rateLimit/**
  验收: npm test -- --grep rateLimit
```

半夜，心跳发现会话 C 在第 22 分钟卡住了，但有一个干净的 commit。编排器直接审查该 commit，合并，继续——无需人工干预。

## 🪜 Loop Engineering 成熟度模型

`codex-orchestrator` 是一个实用的 **v1 loop**，不是 Agentic 软件开发的终局形态。它处在“人工逐条 prompt”和“完整持久化 Agent 操作系统”之间。

| 阶段 | 形态 | 变化 |
|------|------|------|
| **v0：人工 Prompt** | 人一次提示一个 Agent | 人负责调度、审查、恢复和合并 |
| **v1：受监督的 orchestrator skill** | 现在的 `codex-orchestrator` | worktree 隔离、有界任务契约、heartbeat 巡检、review/merge 纪律、证据标签 |
| **v2：持久任务账本** | loop 背后有真正的状态存储 | task、attempt、worker 状态、gate、blocker、结果能跨 thread 和重启保留 |
| **v2.5：验证 routine 基础** | routine contract 可检查 | 共享输出 schema、证据标签、harness map、routine validator |
| **v3：Routine 库** | 可复用的后台 routine | PR reviewer、CI fixer、stale-session rescuer、rebase helper、docs drift checker、release verifier |
| **v4：Eval 与安全层** | 失败案例沉淀成测试和策略 | prompt injection 样本、危险操作分类器、权限检查、证据质量 eval |
| **v5：Agent Operating System** | 多个 routine 持续协作 | 人和 loop/routine 对话，由专门 Agent 执行、审查、安全检查和汇报 |

这个仓库刻意从 v1 开始，因为这是大多数团队今天就能落地的一层：不需要先写 daemon，也不需要重做整个研发平台。后续真正难的是持久状态、routine 组合、安全分类和 eval 驱动改进。

它不宣称一个 Codex skill 已经等于完整 loop runtime。它要先把第一个有用的 loop 做具体：有界任务、隔离执行、心跳巡检、诚实证据标签，以及合并前审查。

V2 持久化状态层见
[docs/v2-persistent-ledger-and-heartbeat.md](docs/v2-persistent-ledger-and-heartbeat.md)：持久 ledger 格式和保守 heartbeat helper。
V2.5 routine contract 见 [docs/routines/README.md](docs/routines/README.md)，
feedback-loop harness map 见 [docs/routines/harness-map.md](docs/routines/harness-map.md)。
Loop Engineering 对齐调研见
[docs/research/loop-engineering-alignment.md](docs/research/loop-engineering-alignment.md)。
完整 v2-v5 演进路线见 [docs/roadmap.md](docs/roadmap.md)。

当前 v2 helper CLI 已支持：

```bash
go build -o codex-orchestrator ./cmd/codex-orchestrator
./codex-orchestrator init
./codex-orchestrator record-task --id TASK --worktree /path/to/wt --branch codex/task
./codex-orchestrator observe
./codex-orchestrator heartbeat --count 1 --write-report .codex-orchestrator/heartbeat-report.json
./codex-orchestrator status
./codex-orchestrator append-event --type review --task-id TASK --status completed-unreviewed
./codex-orchestrator validate-routines --dir routines
./codex-orchestrator run-routine pr-reviewer --task-id TASK --write-report /tmp/pr-reviewer-report.json
./codex-orchestrator run-routine stale-task-rescuer --task-id TASK --write-report /tmp/stale-task-rescuer-report.json
./codex-orchestrator run-routine ci-fixer --task-id TASK --write-report /tmp/ci-fixer-report.json
./codex-orchestrator run-routine release-verifier --tag v0.3.0-alpha.1 --write-report /tmp/release-verifier-report.json
./codex-orchestrator record-routine-run --routine pr-reviewer --status passed --evidence-local "go test ./..." --action "reviewed diff" --next "merge branch"
./codex-orchestrator record-routine-run --report-json examples/routine-reports/pr-reviewer.passed.json
```

JSON heartbeat report 会包含 `overallStatus`、按状态聚合的 `counts`，以及
`reviewPressure`，让统领在 review / stale / blocked / cleanup 队列堆积时暂停派发。

Codex App worktree 派发是 App-first。要用 project worktree session 前，先确认
这个仓库已经保存为 Codex App project。如果因为 unknown project、没有 saved
project，或 pending setup 长时间没有变成真实 worktree/thread，就把它当作 setup
blocker。不要让 fallback worker 直接修改统领自己的 checkout；必须先创建并验证
隔离 fallback worktree，或者停止并报告 blocker。

`run-routine pr-reviewer` 是第一个可运行的 routine MVP。它只读检查任务
worktree：读取 ledger task、确认 worktree 和 branch 状态、记录
`git status --short --branch`、比较 `baseCommit..HEAD`、输出
`git diff --name-status`，并运行 `git diff --check`。它会写出标准
`RoutineRunReport` JSON，之后可用 `record-routine-run --report-json` 记录。
它不会 merge、push、删除 branch、清理 worktree、运行任务专用测试 gate，也不会把
本地静态证据说成 runtime proof。

`run-routine stale-task-rescuer` 是第二个可运行 routine MVP。它同样只读检查
任务 worktree：按 id 读取 ledger task，记录 ledger status、last observation
和近期 task history，确认 worktree 和 branch 状态，采集
`git status --short --branch` 与 `git log --oneline -3`，再根据本地 git 状态
保守判断是否可救回。干净 worktree 且 `baseCommit` 之后有 commit 时返回
`passed`，下一步是统领 review 已提交 diff；有未提交但可能有用的改动时返回
`failed` 并建议回到同一个 worker 或同任务 takeover；worktree 缺失、分支不匹配、
缺少 `baseCommit` 或 git 检查失败时返回 `blocked`。它不会修改 ledger status、
stage、commit、merge、清理 worktree、派发新任务，也不会把证据说成
direct/proxy runtime proof；这个 MVP 只使用 `local` 或 `blocked` 证据。

`run-routine ci-fixer` 是第三个可运行 routine MVP。虽然名字里有 fixer，
它不会自动改代码或修 CI；它是只读的 CI/local gate 分类器。它按 id 读取
ledger task，确认任务 worktree 和预期 branch，拒绝 dirty worktree，比较
`baseCommit..HEAD`，记录已提交文件列表，并只在任务 worktree 里运行 ledger
task 已记录的 gate 命令，且带本地超时。gate 通过且 `baseCommit` 之后有提交时
返回 `passed`，下一步是统领 review/merge；dirty worktree 或 gate 失败时返回
`failed`，建议回到同一个 worker 或同任务 takeover；缺少 gate、缺少
`baseCommit`、分支不匹配或 git 检查失败时返回 `blocked`。它不会 stage、
commit、merge、push、清理 worktree、修改 ledger status，也不会把证据说成
direct/proxy runtime proof；这个 MVP 只使用 `local` 或 `blocked` 证据。

`run-routine release-verifier` 是第四个可运行 routine MVP。它只读检查 release
状态，不读取或修改 ledger。它验证传入的本地 git tag，通过 `gh release view`
读取 GitHub release 元数据（如果 `gh` 可用），检查 alpha/beta/rc tag 是否标为
prerelease，并把 release asset 名称与本仓库默认 Go CLI 资产集合或重复传入的
`--expected-asset` 覆盖项对比。缺少 tag、缺少 release、draft、prerelease
不匹配或缺少 asset 时返回 `failed`；`gh` 不可用、认证/网络失败或 release 元数据
无法解析时返回 `blocked`。它不会创建或编辑 release、移动 tag、上传 asset、stage、
commit、merge、push、清理、派发任务、修改 ledger，也不会声称 production/runtime
proof；这个 MVP 使用 `local`、`proxy` 或 `blocked` 证据。

一个 delegated task 完成 merge、push、release、cleanup，并不等于整个 loop
结束。删除任务专属 heartbeat 前，统领必须先检查 ledger / repo truth 和 roadmap
queue。如果还有安全可做的任务，应继续派发下一个有界任务，或把 heartbeat 替换成
下一任务 monitor。只有在队列耗尽，或下一步被明确 blocker 卡住并已报告后，才删除
heartbeat。

## 🧱 架构

编排器作为一个**状态机**管理所有委派会话：

```
派发 → 活跃 → 完成待审查 → 已合并
           ↘ 陈旧待检查 → 救回/放弃
           ↘ 阻断 → 等待人工输入
```

**核心组件：**

- **状态账本**：记录每个会话的任务 ID、线程 ID、worktree、分支、基准 commit、写入集、状态和验收门禁
- **心跳循环**：每 5 分钟对账 Codex 线程状态与实际 git 状态
- **审查流水线**：diff 边界检查、自审查验证、契约冲突检测、证据标签验证
- **反浅切片门禁**：每个任务必须分类为 `vertical-completion`、`runtime-proof`、`blocked-removal` 或 `owner-gated`

## ⚖️ 对比手动编排

| | 手动 | codex-orchestrator |
|---|------|-------------------|
| **会话监控** | 手动切 tab 逐个检查 | 5 分钟心跳自动对账 |
| **会话卡死** | 你（终于）注意到了 | 15 分钟自动检测，收割 commit |
| **合并冲突** | 合并时才发现 | 不相交写入集提前预防 |
| **浅层工作** | 会话产出一堆占位页面 | 反浅切片门禁拒绝或重写 |
| **证据诚信** | 信任会话自述 | `direct`/`proxy`/`blocked` 标签强制执行 |
| **过夜运行** | 醒来面对一团乱麻 | 醒来看到合并好的分支 |
| **并发** | 随缘并行 | 契约串行化，最多 2-3 个有规则 |

## ⚙️ 配置参数

以下参数可在 skill 中或按次派发时调整：

| 参数 | 默认值 | 说明 |
|------|--------|------|
| 最大并发数 | 2 | 活跃会话数。仅在写入集不相交且无活跃共享契约时可升至 3 |
| 陈旧阈值 | 15 分钟 | 无进展超过此时间标记为待检查 |
| 心跳间隔 | 5 分钟 | 编排器检查所有会话的频率 |
| 分支前缀 | `codex/` | 任务分支的命名空间 |
| Push 策略 | 项目自定 | 仅在项目正常流程或用户明确要求时 push |
| 证据标签 | `direct`, `proxy`, `blocked` | 硬件/部署/支付证明的必填分类 |
| 反浅切片 | 强制 | 任务派发前必须分类 |

## 📂 文件结构

```
codex-orchestrator/
├── SKILL.md              # 编排器 skill（复制到 ~/.codex/skills/）
├── agents/
│   └── openai.yaml       # Agent 接口定义
├── .github/workflows/
│   └── release.yml       # 多平台 release binary workflow
├── cmd/
│   └── codex-orchestrator/
│       ├── main.go       # Go helper CLI
│       └── main_test.go  # CLI 状态机测试
├── docs/
│   ├── roadmap.md
│   ├── routines/
│   │   ├── README.md
│   │   └── harness-map.md
│   ├── v2-usage.md
│   └── v2-persistent-ledger-and-heartbeat.md
├── routines/
│   ├── api-proof.json
│   ├── browser-runtime-proof.json
│   ├── ci-fixer.json
│   ├── database-proof.json
│   ├── device-proof.json
│   ├── log-proof.json
│   ├── pr-reviewer.json
│   └── stale-task-rescuer.json
├── examples/
│   ├── ledger.example.json
│   └── routine-reports/
│       ├── api-proof.blocked.json
│       └── pr-reviewer.passed.json
├── scripts/
│   ├── install.sh
│   └── ledger_heartbeat.py
├── go.mod
├── README.md             # 英文说明
├── README.zh-CN.md       # 本文件
└── LICENSE               # MIT
```

## 📄 许可证

MIT

---

由 [IndieKit.ai](https://indiekit.ai) 构建 — 面向 AI 原生工作流的开源开发者工具。
