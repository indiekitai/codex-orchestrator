# codex-orchestrator 路线图

这份路线图用于说明 `codex-orchestrator` 从当前 skill/runbook 形态，逐步演进到持久化 loop、routine library、安全评估层，甚至更完整 agent orchestration runtime 的路径。

核心判断：

- 当前能力是 **Codex App-first**，因为只有 Codex App 具备创建/继续多个 session、管理 worktree session、设置 heartbeat automation 的能力。
- CLI/helper 不替代 Codex App 调度。它负责持久状态、git/worktree 观察、heartbeat report、policy/eval 检查。
- 后续演进不应该把 skill 硬撑成所有东西。更合理的是分层：skill 教 Agent 怎么工作，CLI/helper 管状态和检查，daemon/UI 负责长期运行和可视化。
- 调研结论见 `docs/research/loop-engineering-alignment.md`：当前项目更准确地说是 **outer orchestration loop**，下一阶段要先补 inner verification routines 和 safety/eval，而不是直接堆更大的 agent OS。

## 当前定位

`codex-orchestrator` 现在是一个 Codex App orchestrator skill。

它适合做：

- 拆分路线图任务；
- 创建隔离 worktree/session；
- 要求 worker 自审；
- 通过 heartbeat 动态巡检；
- 从 reviewer 角度验收 diff；
- 合并、push、清理分支和 worktree；
- 遇到需要人工动作时停下来通知；
- 把经验沉淀进 skill / AGENTS / docs，而不是只停留在聊天里。

它不适合假装自己已经是：

- 后台 daemon；
- 独立 agent runtime；
- 能自动创建 Codex CLI 多 session 的工具；
- 无需人工 review 的自动合并系统；
- 安全分类器或 eval 平台。

## 分层架构

推荐长期结构：

```text
Codex App orchestrator skill
  - 拆任务
  - 创建/继续 Codex App sessions
  - 派发 worker prompt
  - 设置 heartbeat/checkback
  - 做 review / merge / cleanup 决策

Local CLI/helper
  - 初始化和更新 ledger
  - 扫描 git/worktree
  - 生成 heartbeat report
  - 做 stale detection
  - 做 evidence/policy checks
  - 给 App orchestrator 生成下一步建议

Project-local state
  - .codex-orchestrator/ledger.json
  - .codex-orchestrator/events.jsonl
  - git branches / worktrees / commits
  - docs/reviews / progress / roadmap

Future daemon/UI
  - 定时运行 observe
  - 展示任务状态
  - 发通知
  - 提供 routine registry
  - 对高风险动作做 policy/eval gate
```

关键点：用户不需要同时打开一个 Codex CLI AI 和一个 Codex App AI 让它们互相聊天。实际使用应该是 **Codex App 统领 session 调用本地 helper 命令**，helper 输出事实报告，App 继续调度。

## v1：Codex App supervised orchestrator skill

状态：当前已基本具备。

目标：

- 把单个 Codex 写代码助手变成可监督的多 session 工程循环；
- 支持 worktree 隔离、bounded task contract、heartbeat 巡检、review/merge 纪律；
- 让人从“反复提示 Agent”提升到“设计 loop、审查证据、决定合并”。

已具备：

- `SKILL.md` 作为可安装 skill；
- 中英文 README；
- 有界任务契约；
- max concurrency 规则；
- stale session 处理规则；
- anti-shallow-slice gate；
- direct / proxy / blocked 证据标签；
- maturity model；
- Codex App worktree setup 注意事项。

仍需改进：

- README 里可以再补一段更短的 “App-first” 快速解释；
- 示例可以从 REST API 再扩展一个“真实长任务队列”例子；
- worker prompt 模板可以拆成可复制的 snippets。

## v2：Persistent task ledger + heartbeat helper

状态：alpha-plus。可用于 App-first orchestration 的持久 ledger 和保守 heartbeat，已补 release hardening 测试与 review-pressure 输出；还不是自动创建 session / 自动合并的后台系统。

目标：

- 让 orchestration 状态不只存在于长聊天上下文；
- 新 orchestrator session 能通过 ledger + git truth 恢复现场；
- heartbeat 不再依赖写死的旧 task ID；
- 支持只读、可重复、可审计的状态观察。

已具备：

- `docs/v2-persistent-ledger-and-heartbeat.md`
- `docs/v2-usage.md`
- `examples/ledger.example.json`
- `cmd/codex-orchestrator` Go helper CLI
- `scripts/ledger_heartbeat.py`
- helper CLI 子命令：
  - `init`
  - `observe`
  - `status`
  - `record-task`
  - `append-event`
  - `heartbeat`
- integration checkout dirty/error 检查；
- per-task `pending-setup` / `stale-needs-inspection` / `completed-unreviewed` / `blocked` / `cleanup-needed` 分类；
- `overallStatus`、recommended actions、counts 和 reviewPressure；
- JSON report 和 Markdown summary；
- Go 单测覆盖核心状态机、dirty integration、bad ledger、unknown task、stale timeout、cleanup-needed 和 review queue saturation；
- `scripts/install.sh` 本地安装入口；
- GitHub Actions release binary workflow。

下一步建议：

1. 发布第一个 tag，验证 GitHub release artifacts。

2. 视用户反馈补 Homebrew tap 或 npm wrapper。

3. 继续扩展 heartbeat policy：

   - stale timeout 更丰富的 per-task 配置；
   - merged/cleanup ownership 检查；
   - forbidden-path 和 evidence-label audit；
   - routine library 接口。

4. 添加 launchd/cron 示例，但保持保守：

   ```text
   observe -> write heartbeat-report.json -> notify user/App
   ```

边界：

- v2 CLI/helper 不创建 Codex App session；
- 不 merge；
- 不 push；
- 不删除 worktree；
- 不把 local/proxy evidence 升级成 direct。

## v2.5：Verification routine foundation

状态：alpha foundation。已经有 routine contract 目录、首批 JSON specs、harness map，以及 Go helper 的 `validate-routines` 校验命令；还没有自动执行 routine 或后台 daemon。

原因：Loop Engineering 不只是调度任务。Claude Code 访谈和 feedback-loop
engineering 都强调 agent 必须能运行产品、观察结果、修复并复测。否则
routine library 容易变成任务管理器，而不是可靠 loop。

目标：

- 定义 routine output schema；
- 定义 evidence schema；
- 定义 harness map：feedforward guides、feedback sensors、control boundaries；
- 给高频 engineering loop 写最小 routine spec；
- 要求每个 routine 都是 workflow contract，而不只是 prompt 或命令别名；
- 把 cost/review budget 作为 heartbeat 状态的一部分，避免盲目扩大并发；
- 让 routine 输出可以被 ledger/heartbeat/report 消费；
- 保持 helper 保守，不自动 merge/push/删除。

当前交付物：

```text
docs/routines/
  README.md
  harness-map.md
routines/
  stale-task-rescuer.json
  pr-reviewer.json
  ci-fixer.json
cmd/codex-orchestrator validate-routines --dir routines
```

剩余：

- browser/log/db/device/API proof routine specs；
- routine report examples；
- ledger event 中记录 routine run outcome；
- per-routine runtime budget / review budget 与 heartbeat 更深集成。

## v3：Routine library

目标：

把常见的工程 loop 拆成可复用 routine，而不是每次都靠统领 session 临场推理。

候选 routine：

- stale task rescuer；
- PR reviewer；
- CI fixer；
- docs drift checker；
- rebase helper；
- release verifier；
- evidence label auditor；
- roadmap next-task suggester。

推荐形式：

```text
routines/
  stale-rescuer.yaml
  pr-reviewer.yaml
  ci-fixer.yaml
  docs-drift-checker.yaml
```

每个 routine 至少定义：

- trigger：什么时候运行；
- inputs：需要哪些 git/thread/ledger/doc 信息；
- allowed actions：允许做什么；
- forbidden actions：禁止做什么；
- gates：完成前必须跑什么；
- output schema：输出给 orchestrator 的格式；
- escalation：什么时候必须交给人。

注意：

- v3 routine 仍然不等于后台 Agent。
- routine 可以建议 “派一个 worker session 修 CI”，但真正创建 session 仍由 Codex App orchestrator 做，除非未来有公开 session API。

## v4：Eval / security classifier + self-improving rules

目标：

把 repeated failures 变成 eval / policy，而不是每次靠人提醒。

候选能力：

- 高风险路径检查；
- destructive command 检查；
- secrets/provider/payment/pre/prod 操作检查；
- evidence exaggeration 检查；
- prompt injection 样本库；
- failed orchestration case -> eval fixture；
- rule proposal：根据失败案例建议更新 skill/AGENTS。

推荐命令形态：

```bash
codex-orchestrator policy check --task TASK_ID
codex-orchestrator eval add-failure --from-review docs/reviews/...
codex-orchestrator eval run
codex-orchestrator rules propose
```

边界：

- policy/eval 可以 block 或 warn；
- 不应该自动改规则并立即启用；
- 自改规则至少需要 review；
- 对生产、支付、硬件、真实用户数据相关操作必须偏保守。

## v5：Multi-agent operating system

目标：

从 App-first skill + helper，演进成更完整的 agent orchestration runtime。

它可能包含：

- task queue；
- persistent DB；
- routine registry；
- worker pool；
- project adapters；
- App/Codex/Claude/GitHub Actions adapters；
- web UI；
- notification center；
- audit log；
- policy/eval gate；
- human-in-the-loop approval。

但 v5 的前提是有可靠的 worker control surface：

- Codex App/CLI 提供可编程 session API；
- 或者系统自己实现 worker runtime；
- 或者接入 Claude Code / GitHub Actions / shell workers 等可控执行面。

在这之前，不应该把 `codex-orchestrator` 描述成完整 multi-agent OS。

## 近期优先级

最现实的短期路线：

1. 保持 v1 skill 简洁可安装。
2. 把 v2 helper CLI 做完整：
   - `init`
   - `observe`
   - `status`
   - `record-task`
   - `append-event`
3. 加测试和 fixture。
4. 写 App-first 使用文档：
   - “用户只开 Codex App 统领 session”
   - “统领 session 调用 helper CLI”
   - “helper CLI 不替代 App 派 session”
5. 再做 1 个 routine：
   - stale task rescuer 最适合，因为它已经来自真实痛点。

## 成功标准

v2 完成时，应该能做到：

- 新 orchestrator session 不依赖旧聊天上下文，也能看懂当前所有 active/pending/completed tasks；
- `observe` 能稳定指出哪些任务需要 review、哪些 stale、哪些 blocked；
- heartbeat prompt 不再需要写死 task ID；
- 所有动作都有 ledger/event 可追溯；
- 用户不用同时开 Codex CLI AI 和 Codex App AI，只需要让 Codex App 调用本地 helper。

v3 完成时，应该能做到：

- routine 不只是文档，而是可配置、可运行、可审计；
- 至少一个 routine 能把真实高频问题稳定处理掉；
- routine 输出能被 App orchestrator 直接消费。

v4 完成时，应该能做到：

- 失败案例能沉淀成 eval；
- 高风险动作有 policy gate；
- skill/rules 的更新有 evidence-backed proposal，而不是拍脑袋改。
