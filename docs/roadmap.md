# codex-orchestrator 路线图

这份路线图用于说明 `codex-orchestrator` 从当前 Codex App-first harness
形态，逐步演进到持久化 loop、routine library、安全评估层和更可靠的
Loop Engineering 控制面。

核心判断：

- 当前能力是 **Codex App-first**，因为只有 Codex App 具备创建/继续多个 session、管理 worktree session、设置 heartbeat automation 的能力。
- CLI/helper 不替代 Codex App 调度。它负责持久状态、git/worktree 观察、heartbeat report、policy/eval 检查。
- 后续演进不应该把 skill 硬撑成所有东西。更合理的是分层：skill 教 Agent 怎么工作，CLI/helper 管状态和检查，未来 watcher/UI 如果出现，也应先保持只读和可审计。
- 调研结论见 `docs/research/loop-engineering-alignment.md` 和
  `docs/research/harness-reading-notes.md`：当前项目更准确地说是
  **Codex App-first harness / outer orchestration loop**，下一阶段要先补
  recovery classification、inner verification routines 和 safety/eval。Agent OS
  不放进本项目路线图。

## 当前定位

`codex-orchestrator` 现在是一个 Codex App-first outer-loop harness：
skill 负责给 Codex App 编排 session 的工作方式，helper 负责持久状态、
git/worktree 观察、heartbeat report、routine/policy/eval 检查。

它不是完整的 Loop Engineering runtime，也不是 agent OS。agent OS 路线不在本项目范围内。worker session
内部的改代码、跑测试、修复是内层循环；`codex-orchestrator` 管理的是外层工程循环：选任务、隔离 session、巡检、审查、合并、清理和继续推进。

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
Codex App outer-loop orchestrator skill
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

关键点：用户不需要同时打开一个 Codex CLI AI 和一个 Codex App AI 让它们互相聊天。实际使用应该是 **Codex App 编排 session 调用本地 helper 命令**，helper 输出事实报告，App 继续调度。

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
- direct / proxy / local / blocked 证据标签；
- maturity model；
- Codex App worktree setup 注意事项；
- fallback worker 必须先有隔离 worktree，不能污染 orchestrator checkout；
- 单任务 heartbeat 删除前必须先判断整体队列是否继续，不能完成一个 child task 就让编排器静默停止。

仍需改进：

- README 里可以再补一段更短的 “App-first” 快速解释；
- 示例可以从 REST API 再扩展一个“真实长任务队列”例子；
- worker prompt 模板可以拆成可复制的 snippets。

## v2：Persistent task ledger + heartbeat helper

状态：alpha-plus。可用于 App-first orchestration 的持久 ledger 和保守 heartbeat，已补 release hardening、pending worktree setup ID 记录、review-pressure 输出；还不是自动创建 session / 自动合并的后台系统。

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
- 可在 Codex App 只返回 `pendingWorktreeId` 时先记录 `pending-setup`
  task，之后通过事件补入实际 worktree/branch；
- Go 单测覆盖核心状态机、dirty integration、bad ledger、unknown task、stale timeout、cleanup-needed 和 review queue saturation；
- `scripts/install.sh` 本地安装入口；
- GitHub Actions release binary workflow。

下一步建议：

1. 发布第一个 tag，验证 GitHub release artifacts。已完成到
   `v0.3.0-beta.4`。

2. 继续打磨 Codex App-first install UX。Homebrew、npm wrapper、tap 或其他
   package-manager 分发路线不在当前产品范围内；helper binary 只能作为 Codex App
   需要持久 ledger/routine 支持时的高级辅助路径。

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

状态：alpha foundation。已经有 routine contract 目录、首批 JSON specs、harness map、Go helper 的 `validate-routines` 校验命令、`record-routine-run` ledger 记录命令、JSON report 输入、heartbeat recent routine run 摘要、只读 task budget summary，以及多个只读 `run-routine` MVP；还没有后台 daemon、自动调度器或会主动派发 session 的 routine runtime。

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
  api-proof.json
  browser-runtime-proof.json
  stale-task-rescuer.json
  pr-reviewer.json
  ci-fixer.json
  release-verifier.json
  docs-drift-checker.json
  evidence-label-auditor.json
  orchestration-policy-auditor.json
  roadmap-next-task-suggester.json
  database-proof.json
  device-proof.json
  log-proof.json
cmd/codex-orchestrator validate-routines --dir routines
cmd/codex-orchestrator run-routine pr-reviewer --task-id ...
cmd/codex-orchestrator run-routine stale-task-rescuer --task-id ...
cmd/codex-orchestrator run-routine ci-fixer --task-id ...
cmd/codex-orchestrator run-routine release-verifier --tag ...
cmd/codex-orchestrator run-routine docs-drift-checker
cmd/codex-orchestrator run-routine evidence-label-auditor
cmd/codex-orchestrator run-routine orchestration-policy-auditor
cmd/codex-orchestrator run-routine roadmap-next-task-suggester
cmd/codex-orchestrator policy check
cmd/codex-orchestrator eval run
cmd/codex-orchestrator eval add-failure
cmd/codex-orchestrator rules propose
cmd/codex-orchestrator record-routine-run --routine ... --status ...
cmd/codex-orchestrator record-task --max-runtime-minutes ... --review-budget-minutes ...
cmd/codex-orchestrator observe / heartbeat budgetSummary
examples/routine-reports/
  pr-reviewer.passed.json
  api-proof.blocked.json
```

其中 `evidence-label-auditor` 现在已经有第一层本地 policy/eval：命名规则
`ELA001`-`ELA009`、deterministic false-positive guard，以及按规则汇总的
rule-hit 统计；但它仍然是只读、本地、静态的保守检查器。

`orchestration-policy-auditor` 启动了 V4 policy/eval 层的第一块：命名规则
`OPA001`-`OPA007` 覆盖 dry-run 派发屏障、主工作区 fallback guard、heartbeat
continuation guard、worker 边界、证据升级边界、heartbeat target 绑定 guard，
以及 pending worktree ledger guard。它同样是只读、本地、静态的保守检查器，
输出的是可复核疑点，不是语义定罪。

`policy check` 把 `orchestration-policy-auditor` 和
`eval/orchestration-policy-auditor/` 下的 fixture eval 串起来，成为 V4 的第一个
产品化入口。第一批 fixture 覆盖真实编排失败类别：dry-run 未批准派发、setup
失败后回退主工作区、单个 child task 完成后停止总队列、worker prompt 缺少边界、
local/proxy/weak 证据升级为 direct。

`eval run` 单独运行 fixture suite，不扫描当前仓库文本。它适合在修改
`OPA001`-`OPA007` 规则时做确定性回归检查。

`eval add-failure` 已有手动 MVP：通过 `--text`/`--text-file` 和 `--expect RULE=N`
写入新的 fixture。写入前会先用当前规则验证实际命中是否匹配期望；还没有从 review
文档自动解析失败案例。

`rules propose` 已有保守的 review-only MVP：通过 `--from-review`、`--text` 或
`--text-file` 读取本地证据文本，输出建议规则 title/body/source/evidence label/
needs human review 报告；只有 `--write-report` 会写出报告文件，不会自动修改
SKILL、README、AGENTS、CLAUDE、policy 文件或项目规则。

剩余：

- 更深的 per-routine budget 策略、排序、告警或调度联动；当前 helper 只记录和展示预算元数据，不执行预算。

## v3：Routine library

目标：

把常见的工程 loop 拆成可复用 routine，而不是每次都靠编排 session 临场推理。

候选 routine：

- stale task rescuer；
- PR reviewer；
- CI fixer；
- docs drift checker；
- rebase helper；
- release verifier；
- orchestration policy auditor follow-on eval fixtures and transcript-backed cases beyond the current named-rule local layer；

补充说明：

- `roadmap-next-task-suggester` 的第一版只读 MVP 已经具备；接下来剩余的是更深的排序、policy/eval 约束，以及和 heartbeat / ledger budget 的联动。

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
codex-orchestrator rules propose --from-review docs/reviews/...
```

当前已落地的第一步是：

```bash
codex-orchestrator policy check --repo .
codex-orchestrator eval run --repo .
codex-orchestrator eval add-failure --id dry-run-example --text "Dry run mode can dispatch workers immediately." --expect OPA001=1
codex-orchestrator rules propose --from-review docs/reviews/example.md --write-report /tmp/rules-proposal-report.json
```

`policy check` 会先运行本地 orchestration policy auditor，再运行仓库内置 fixture
eval；`eval run` 只运行 fixture eval；`eval add-failure` 能手动沉淀失败案例。
`rules propose` 能从本地 review/text 输入生成只供人工 review 的规则建议报告；还没有
自动修改 live 规则，也不应自动修改。

边界：

- policy/eval 可以 block 或 warn；
- 不应该自动改规则并立即启用；
- 自改规则至少需要 review；
- 对生产、支付、硬件、真实用户数据相关操作必须偏保守。

## 不做：Agent operating system

`codex-orchestrator` 不把 Agent OS 作为路线图阶段。

原因：

- 当前最有价值的用户入口是 Codex App-first 编排，不是另起一套 worker runtime；
- helper CLI 没有可靠的 Codex App session control surface，不能也不应该假装自己能直接管理 worker pool；
- 现在的主要风险不是“agent 不够多”，而是 evidence over-claim、stale recovery、unsafe fallback、review bandwidth 和规则漂移；
- daemon/UI/worker pool 会显著增加使用成本，但不能自动解决上述风险。

因此本项目的路线图终点先收在 **V4：policy/eval + reviewable rule
improvement**。如果未来另开 agent runtime 项目，应作为新产品重新论证，而不是塞进
`codex-orchestrator`。

## 下一阶段优先级

最现实的路线是把 **Codex App-first harness** 做扎实：

1. 保持 skill / README / bootstrap prompt 简洁，让新用户能把 GitHub 链接交给
   Codex App 自己 dry-run。
2. 持续维护 real Codex App demo proof，证明 App dispatch / review / merge /
   push / cleanup 这条链路真实可用。
3. 扩展 V4 policy/eval：
   - 从真实编排失败沉淀 fixture；
   - 检查 setup-failed 后污染主工作区、单任务结束后停止总队列、证据升级、缺少 worker 边界、heartbeat target 绑定错误、pending setup 未写入 ledger 等问题；
   - 增加 review-only `rules propose`，但不自动启用新规则。
4. 补 recovery-state classifier：
   - pending setup；
   - setup failed；
   - active；
   - stale with clean commit；
   - stale with useful diff；
   - completed-unreviewed；
   - blocked；
   - cleanup-needed；
   - abandoned。
5. 继续保守扩展 routine：
   - 只有当某个 routine 能覆盖真实高频工作流或重复失败时才添加；
   - 每个 routine 必须声明 verification surface、evidence labels、allowed actions、escalation rule。
6. 保持 daemon/watch 方向只读：
   - 可以运行 `observe` 并写报告；
   - 不创建 session；
   - 不 merge/push；
   - 不删除 worktree。

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
