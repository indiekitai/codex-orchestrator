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

## ✅ 前置条件与安全边界

这个仓库是一个 Codex skill / runbook，不是独立后台守护进程。完整的自动循环依赖宿主环境提供对应能力，尤其是：

- 创建或继续隔离的 Codex 会话
- 创建独立 git worktree，或等价的隔离 worker 环境
- 读取线程状态并检查 worktree 的 git 状态
- 创建/更新定时巡检 automation 或 heartbeat reminder
- 按项目正常 git 策略执行 merge / push

如果这些工具不可用，这个 skill 应降级为手动编排清单：少开会话，直接检查 git 状态，并且不要假装已经完成监控、合并、推送或清理。

开源场景下建议先在可丢弃仓库或功能分支上 dry run。自动 push 应保持关闭，直到你确认 review gate 和项目分支保护策略可靠。

## 🚫 这不是什么

它不是工程判断、代码审查或生产验收的替代品。它的目标是让 AI 辅助开发更结构化：有界任务、隔离 worktree、明确证据标签、合并前审查。

重点不是让 agent 永远无人值守地写代码，而是把人放在更高杠杆的位置：设计循环、审查证据、决定什么可以发布。

## 🚀 快速开始

### 1. 安装 skill

```bash
# 复制到 Codex skills 目录
cp -r codex-orchestrator ~/.codex/skills/delegated-session-orchestrator
```

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
| **v3：Routine 库** | 可复用的后台 routine | PR reviewer、CI fixer、stale-session rescuer、rebase helper、docs drift checker、release verifier |
| **v4：Eval 与安全层** | 失败案例沉淀成测试和策略 | prompt injection 样本、危险操作分类器、权限检查、证据质量 eval |
| **v5：Agent Operating System** | 多个 routine 持续协作 | 人和 loop/routine 对话，由专门 Agent 执行、审查、安全检查和汇报 |

这个仓库刻意从 v1 开始，因为这是大多数团队今天就能落地的一层：不需要先写 daemon，也不需要重做整个研发平台。后续真正难的是持久状态、routine 组合、安全分类和 eval 驱动改进。

它不宣称一个 Codex skill 已经等于完整 loop runtime。它要先把第一个有用的 loop 做具体：有界任务、隔离执行、心跳巡检、诚实证据标签，以及合并前审查。

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
├── README.md             # 英文说明
├── README.zh-CN.md       # 本文件
└── LICENSE               # MIT
```

## 📄 许可证

MIT

---

由 [IndieKit.ai](https://indiekit.ai) 构建 — 面向 AI 原生工作流的开源开发者工具。
