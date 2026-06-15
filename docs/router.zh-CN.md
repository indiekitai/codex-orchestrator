# Router 指南

`codex-orchestrator` 跑久以后，最容易乱的不是代码，而是输入都挤在一个长聊天里。
Router 的作用很小：判断一条新输入应该交给哪个线程。

Router **不**写代码、不派 worker、不 merge、不 push、不清理 worktree。它只读取线程地图，
分类输入，然后把内容转给正确的 owner。

## 线程职责

| 线程 | 负责什么 | 不负责什么 |
|---|---|---|
| Project Orchestrator | repo truth、ledger、功能包计划、worker 派发、验收、merge、push、cleanup、状态收口 | 原始 issue 收集、零散笔记、无关研究 |
| Pulse | 定时状态检查、heartbeat 漏跑报告、有实质变化时汇报 | 实现、merge 决策、凭空发明新任务 |
| Inbox | GitHub issue、用户反馈、外部 review、真实使用体验、模型/研究链接 | 在 triage 之前直接派 worker |
| Router | 分类新输入，判断目标线程 | 代码修改、worker 控制、merge/push/cleanup |
| Log | 人能看懂的决策记录、功能包里程碑、为什么切换路线 | 权威 ledger 状态 |

## 路由规则

1. 先读 `.codex-orchestrator/thread-map.md`。
2. 如果输入和当前 worker 状态有关，交给 Project Orchestrator。
3. 如果输入是定时检查或 missed heartbeat，交给 Pulse。
4. 如果输入是新反馈、issue、外部 review 或研究材料，交给 Inbox。
5. 如果输入是已经确定的决策，写入 Log，或提醒 Project Orchestrator 记录到 ledger/status。
6. 如果输入需要人操作设备、pre/prod、支付、短信/provider、DNS/SSL 或部署窗口，标成
   blocked/owner-gated，不要派 worker。
7. 如果不确定，优先放 Inbox，加一条简短分类说明，不要让 Project Orchestrator 直接下场实现。

## 最小 Router Prompt

```text
你是这个 codex-orchestrator 项目的 Router。

请读取 .codex-orchestrator/thread-map.md、.codex-orchestrator/inbox.md、
.codex-orchestrator/status.md，以及当前 ledger/status truth（如果存在）。

把新的输入分类到：
- Project Orchestrator
- Pulse
- Inbox
- Log
- Human/owner-gated

不要改代码、不要派 worker、不要 merge、不要 push、不要 cleanup、不要部署、不要创建
新的 worker session。只输出目标线程、理由、证据标签（local/proxy/direct/blocked），
以及应该转发的原文或摘要。
```

## 输出格式

```text
Target: Inbox
Reason: 新的外部反馈，还不是任务契约。
Evidence: local
Forward:
请先 triage 这条反馈，尽量关联到已有 feature package；只有检查当前 package lane 和
ledger 状态后，才能建议是否拆 worker。
```

## 反模式

- 把 Router 当成第二个 Project Orchestrator。
- 因为 availableSlots 有空，就让 Router 派 worker。
- 把当前 task id 写死在 Router prompt，而不是读取 ledger/status。
- 把 Inbox 里的反馈或社交评论当成已经批准的 roadmap 变更。
- 把 Pulse 发现的 missed heartbeat 当成 root cause 的 direct proof。

Router 应该很无聊。它让整个循环更清楚：新输入进 Inbox，当前工作归 Project
Orchestrator，定时检查归 Pulse，长期决策进入 Log/ledger。
