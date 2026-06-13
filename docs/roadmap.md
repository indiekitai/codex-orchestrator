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
- Agentic Engineering / SASE 相关调研见
  `docs/research/agentic-engineering-feature-notes.md`。当前功能增强方向不是
  增加更多 worker，而是把 worker 结果标准化成可审查的工程交付物，例如
  merge-readiness pack、consultation request pack、transcript failure eval 和
  static status page。
- 模型能力平台期 / 多模型够用化的产品判断见
  `docs/research/model-plateau-loop-engineering.md`。如果 Codex、Claude、
  DeepSeek 或本地模型都能处理足够小的工程块，`codex-orchestrator` 的长期价值
  应该是模型无关的 review pack、acceptance report、evidence labels 和可迁移
  loop，而不是绑定某个模型的提示技巧。
- Developer-agent misalignment 调研见
  `docs/research/developer-agent-misalignment.md`。真实 coding-agent 失败里，
  最值得产品化处理的不只是代码错误，而是违反约束、越界行动、错误汇报完成、
  stale state 和缺乏证据的信任损耗。下一阶段应该把这些失败类型变成本地
  event、claim verifier、policy/eval fixture 和状态页 trust-risk 信号。

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

## 下一阶段功能包推进顺序

下一阶段不要继续按全局安全 backlog 随机抓小任务，而是围绕功能包闭环推进。
一个功能包至少要能回答：当前 lane 是什么、有哪些 worker、哪些已验收、
哪些 blocked、下一步为什么仍属于同一个闭环。

1. Roadmap scorer v2：已补第一轮收窄。
   - 目标：让 `roadmap score` 默认只从项目主计划文档提取候选，不再从所有
     review 文档和风险复盘里捞零散任务。
   - 当前边界：review 文档仍可通过 `--config` 显式加入，但默认不参与候选提取。
   - 下一步：继续让 scorer 输出 package lane / do-not-dispatch reason，而不只是
     单条 task score。

2. Package ledger / package status：已补第一轮本地闭环。
   - 目标：把 3-5 个相关 worker 归到一个 feature package 下，展示 package outcome、
     active/review/blocked/cleaned 状态、外部 review 状态和下一步闭环动作。
   - 价值：避免“多开几个 session”，让编排器真正推进一个功能或模块闭环。
   - 当前落地：`record-task` / `dispatch record` 支持 `--package-id`；ledger task、
     job rows、runtime status item、`observe` JSON、`status` JSON/HTML/Markdown 和
     heartbeat summary 都会保留或展示 package 信息；`packageSummary` 会按功能包汇总
     active、review-needed、blocked、cleanup-needed、cleaned/recent task 状态，并给出
     “继续同一 lane / 先验收 / 先处理 blocker”等下一步建议。`packageLaneGuard`
     现在会在 worker 未归包、多条 package lane 并行、或可用派发槽只能补当前 lane 时
     给出 local/static warning，防止无人值守时又从全局 backlog 抓无关任务。
   - 边界：local/static orchestration state；不自动派发、不自动 merge；routine-run
     只有 `packageId` 而没有 task 时会显示 review-only 包状态。

2.5 Package closeout / legacy ledger polish：已补第一轮本地闭环。
   - 目标：让功能包收口和旧 ledger 历史不再靠人读长 JSON 判断。
   - 当前落地：新增 `codex-orchestrator pack status --package-id PKG`，嵌入
     package acceptance 与 package summary，输出 ready / external-review-needed /
     not-ready / blocked / reject-for-fixup 等收口判断；旧的 cleaned/merged/rejected/
     abandoned 且没有 `packageId` 的历史任务仍保留在 `jobSummary.rows`，但从
     当前行动列表隐藏，并通过 `legacyTerminalUngrouped` 计数说明，不再触发
     package-lane warning。`dispatchRecommendation` 现在把 raw capacity 和动作建议
     分开：`availableSlots` 只是底层槽位，`recommended=false` 会明确说明当前应等待
     active/pending package worker、reconcile setup、review/cleanup，还是可以继续同一
     package lane。后续真实项目反馈又补强为 `capacityOnly` / `capacityWarning` 字段，
     让机器和人都明确 raw slot 不是派发许可。
   - 边界：local/static closeout guidance；不自动 merge/push/cleanup，不产生直接
     runtime/device/provider proof。

3. Human-friendly status page：已补第二轮本地闭环。
   - 目标：让每轮 heartbeat/status 默认输出用户能看懂的一页摘要：当前功能包、
     当前 worker、未验收、blocked、missed heartbeat、下一步动作。
   - 价值：用户不用记命令，也不用从英文 task id 判断进度。
   - 当前落地：`status --html` 和 `status --write-summary` 顶部会先输出
     “当前进度”，用人话回答当前状态、当前功能包、已经完成、正在跑、
     是否需要用户处理、下一步和风险边界；详细 runtime/package/job 表保留在下方，
     并把并发槽、pendingWorktreeId、baseCommit 等机器字段后置。第二轮新增
     `Preflight`、`Lane Guard` 和 `Timeline` 区块，让状态页直接回答：
     “能不能放心离开电脑”、“是否还沿着同一个产品模块推进”、“最近发生了什么”。
     `status --write-html` / `--write-summary` 现在也会同步写或显式支持
     `.codex-orchestrator/status.json`，避免旧 JSON 快照继续显示过时 active worker。
   - 边界：local/static status evidence；不启动 server/daemon，不调度、不 merge、不
     push、不清理 worker。

4. Watchdog install/status polish：已完成。
   - 目标：把 macOS watchdog fallback 的安装、状态检查、last wake/missed count
     做成更清楚的 helper 输出。
   - 当前落地：新增 `codex-orchestrator watchdog status --repo .`，只读检查
     LaunchAgent plist、launchctl loaded 状态、watchdog heartbeat report/summary
     和最近 stdout/stderr/error log，并把 missed heartbeat 和未安装状态转成
     recommended actions。
   - 边界：仍然只做本地提醒和报告，不替代 Codex App session 调度。

4.5 Hands-off Preflight：已完成。
   - 目标：把“睡前/离开电脑前预检”变成 helper 一等命令，而不是靠聊天提醒。
   - 当前落地：`codex-orchestrator preflight --repo .` 会检查 repo cleanliness、ledger
     shape、dispatch mode、heartbeat gap、watchdog、project map、package lane guard
     和缺失的 external-review evidence，并输出 JSON/Markdown。
   - 边界：local/static readiness evidence；不能证明 Codex App automation、OS sleep、
     runtime、pre/prod、device 或 provider 状态。

4.6 Project onboarding templates：已完成。
   - 目标：让新项目第一次接入时不再全靠聊天 prompt 记住 project map、package
     plan 和 orchestration policy。
   - 当前落地：`codex-orchestrator init --write-templates` 会非破坏性写入
     `.codex-orchestrator/orchestration-policy.md`、`package-plan.md` 和
     `project-map.md`；已有文件默认不覆盖，只有显式 `--force` 才覆盖。
   - 边界：这些模板是本地编排状态，不等于项目代码或 direct proof。

5. v0.3.5 release closeout：已完成。
   - 目标：在 preflight、watchdog、status UX、package closeout、legacy ledger 降噪、
     onboarding templates、self-update、公开 README 和 real-run hardening 稳定后发下一个
     release。
   - 当前落地：`v0.3.5` 已发布；覆盖 self-update、README 拆分、公开案例匿名化、
     install surface 简化、drain 终态展示和 state-dir-only merge-readiness handling。

6. v0.3.6 trust-loop release closeout：已完成。
   - 目标：把 developer-agent misalignment reduction alpha 发成公开 release。
   - 当前落地：`v0.3.6` 已发布；覆盖 misalignment event log、constraint stack、
     `claimVerification`、`trustRisk`、`OPA010` 和 helper version reporting。

7. v0.3.7 real-run reliability closeout：已完成。
   - 目标：把真实项目长跑反馈里的 ledger 滞后、heartbeat 误删、path matcher 误报
     和 evidence scan 噪音沉淀成工具/规则/eval。
   - 当前落地：新增 `observe --reconcile --write`、active run-mode heartbeat
     deletion guard、`**` 跨目录 path matcher、matched-rule 输出，以及 cleanup/retry/
     event/outbox/lifecycle/API 幂等性自审规则。

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

状态：已具备，并已补上 App-first 首屏说明和真实项目案例入口。

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
- README 首屏已明确 App-first Loop Engineering 定位、非 daemon/非 package-manager-first/非 agent OS 边界；
- 中英文 README 已提供“把 GitHub 仓库交给 Codex App”的 dry-run 入口；
- 已补 restaurant POS rewrite 脱敏案例和案例文章，展示 worktree 隔离、repo-truth heartbeat、
  completed-unreviewed review、evidence label、review/merge/cleanup 和 blocked 边界。

仍需改进：

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
- observations/runtimeStatus 现在暴露结构化 `state` 字段，分别记录
  setup/worktree/branch/diff/review/cleanup 的本地静态状态；
- `overallStatus`、recommended actions、counts 和 reviewPressure；
- JSON report 和 Markdown summary；
- 可在 Codex App 只返回 `pendingWorktreeId` 时先记录 `pending-setup`
  task，之后通过事件补入实际 worktree/branch；
- Go 单测覆盖核心状态机、dirty integration、bad ledger、unknown task、stale timeout、cleanup-needed 和 review queue saturation；
- `scripts/install.sh` 本地安装入口；
- GitHub Actions release binary workflow。

下一步建议：

1. 发布可用的 beta tag，验证 GitHub release artifacts。已完成到 `v0.3.7`。
   当前 release 线已包含 missed heartbeat 检测、macOS watchdog fallback、
   reusable status snapshot、package/external review workflow、package closeout
   status、starter templates、更清晰的 current-action rows，以及
   misalignment/claim-verification/trust-risk 本地静态闭环。后续 release 文档只应声明
   已经发布并验证过的资产。

2. 继续打磨 Codex App-first install UX。Homebrew、npm wrapper、tap 或其他
   package-manager 分发路线不在当前产品范围内；helper binary 只能作为 Codex App
   需要持久 ledger/routine 支持时的高级辅助路径。

3. 继续扩展 heartbeat policy：

   - stale timeout 更丰富的 per-task 配置；
   - merged/cleanup ownership 检查；
   - forbidden-path 和 evidence-label audit；
   - routine library 接口。

4. 添加 launchd/cron 示例，但保持保守：已完成 macOS 用户级 LaunchAgent
   watchdog 脚本：

   ```text
   observe -> write heartbeat-report.json -> notify user/App
   ```

   当前实现为 `scripts/install-macos-watchdog.sh` 和
   `scripts/macos-watchdog-run.sh`，只写本地 watchdog heartbeat report/summary
   并在 missed heartbeat 时发 macOS 通知；不创建 session、不派发、不 merge/push、
   不 cleanup，也不证明 App/OS 层的漏跑原因。

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
  budget-policy-report.json
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
cmd/codex-orchestrator run-routine budget-policy-report
cmd/codex-orchestrator policy check
cmd/codex-orchestrator eval run
cmd/codex-orchestrator eval add-failure
cmd/codex-orchestrator rules propose
cmd/codex-orchestrator record-routine-run --routine ... --status ...
cmd/codex-orchestrator record-task --max-runtime-minutes ... --review-budget-minutes ...
cmd/codex-orchestrator observe / heartbeat budgetSummary / budgetPressure
examples/routine-reports/
  pr-reviewer.passed.json
  api-proof.blocked.json
  budget-policy-report.review-only.json
```

其中 `evidence-label-auditor` 现在已经有本地 evidence-label policy/eval：
命名规则 `ELA001`-`ELA010`、deterministic false-positive guard、review/handoff
文档扫描，以及按规则汇总的 rule-hit 统计；但它仍然是只读、本地、静态的保守检查器。

`orchestration-policy-auditor` 启动了 V4 policy/eval 层的第一块：命名规则
`OPA001`-`OPA010` 覆盖 dry-run 派发屏障、主工作区 fallback guard、heartbeat
continuation guard、push-confirmation stop guard、worker 边界、证据升级边界、
heartbeat target / lifecycle guard、pending worktree ledger guard，以及
budget-policy 证据/控制边界漂移、破坏 feature-package 主线的互不相关安全 backlog
派发。heartbeat lifecycle guard 还覆盖已验证通用 monitor 每轮反复写入当前
worker/task 状态的 prompt churn。它同样是只读、本地、静态的保守检查器，
输出的是可复核疑点，不是语义定罪。

`policy check` 把 `orchestration-policy-auditor` 和
`eval/orchestration-policy-auditor/` 下的 fixture eval 串起来，成为 V4 的第一个
产品化入口。第一批 fixture 覆盖真实编排失败类别：dry-run 未批准派发、setup
失败后回退主工作区、单个 child task 完成后停止总队列、worker prompt 缺少边界、
local/proxy/weak 证据升级为 direct、heartbeat target 绑定错误、pending setup
未写入 ledger、pending setup 被误当作 running worker、heartbeat 绑定 stale fixed
task id、已验收提交造成 default branch ahead 后删 heartbeat 等用户确认 push/继续、
前台 sleep/轮询替代 Codex App heartbeat、重复创建 heartbeat、创建后未验证
persisted automation truth、已验证通用 heartbeat prompt 被反复更新成当前 worker
状态、setup 失败后统领自己写 worker 实现代码，以及
budget-policy helper 控制或证据夸大，以及从全局安全 backlog 补两个互不相关任务
导致产品包主线断裂。

`eval run` 单独运行 fixture suite，不扫描当前仓库文本。它适合在修改
`OPA001`-`OPA010` 规则时做确定性回归检查。

`eval add-failure` 已有手动 MVP：通过 `--text`/`--text-file` 和 `--expect RULE=N`
写入新的 fixture。写入前会先用当前规则验证实际命中是否匹配期望；还没有从 review
文档自动解析失败案例。

`rules propose` 已有保守的 review-only MVP：通过 `--from-review`、`--text` 或
`--text-file` 读取本地证据文本，输出建议规则 title/body/source/evidence label/
needs human review 报告；只有 `--write-report` 会写出报告文件，不会自动修改
SKILL、README、AGENTS、CLAUDE、policy 文件或项目规则。

已完成一层保守的 budget pressure helper：`observe`、`status` 和 heartbeat report
会展示 task/routine spec budget summary；`observe` 和 heartbeat JSON/Markdown 会用
本地 ledger timestamp 输出缺失预算、runtime near/exceeded、review near/exceeded 或
review timestamp unknown warnings。这些都是 local/static helper evidence，不会调度、
排序、kill 进程或强制执行预算。

已完成 review-only budget policy design：
`docs/reviews/2026-06-11-budget-policy-review-only-design.md` 定义了 helper
可以报告的预算事实、App orchestrator/人类 reviewer 可以做的决策，以及 helper
仍然禁止触碰的调度、排序、worker control、dispatch 和预算强制执行边界。

已完成 docs/spec budget policy report/eval local slice：
`routines/budget-policy-report.json` 和
`examples/routine-reports/budget-policy-report.review-only.json` 定义了下一层
review-only 报告契约：预算 metadata coverage、local/static pressure warnings、
unknown timing state 和 human/App-layer recommendation 必须分开表达。后续 runner
实现沿用这个契约，不引入 scheduler、priority engine、worker kill、dispatch
enforcement、merge/push/delete 自动化或预算强制执行。

已完成只读 `run-routine budget-policy-report` runner：
读取 roadmap、routine docs、routine specs、可选 repo-local ledger 和可选
heartbeat report，输出上述契约形状的 local/static 报告。budget metadata 和
heartbeat `budgetPressure` 仍只作为 local/static visibility；live runtime / review
timing 不存在直接证据时写入 blocked/unknown。runner 不调度、不排序、不 pause/kill
worker、不做 dispatch enforcement、不 merge/push/delete/cleanup，也不修改 ledger。

已完成 budget policy static eval follow-up：
`OPA008` 和 3 个 local/static fixture 检查预算证据误用或边界漂移，例如把
local/static timestamp / ledger / heartbeat budget evidence 写成 direct runtime
proof，或把 budget warning 写成 helper 已经 pause/kill worker、强制 dispatch
eligibility、承担 scheduler/prioritizer/worker-control 行为。finding 只能作为
review prompt，不能成为自动调度决策。

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
- orchestration policy auditor follow-on eval fixtures：已补 transcript-style local review-note fixtures，覆盖 stale heartbeat binding、pending setup ledger、child completion without continuation proof，并补了一个 narrow `OPA004` forbidden-path worker-boundary case；后续又补了否定语义 false-positive guard 和 human-review transcript fixtures，覆盖被明确拒绝/警告的坏模式不应算作 action，以及 human-review transcript 中实际发生的 dry-run dispatch、main-checkout fallback、evidence promotion 仍应命中对应 `OPA001`、`OPA002`、`OPA005`；budget static eval follow-up 已补 `OPA008` 和 3 个 fixture，覆盖 budget local/static evidence promotion、helper pause/kill/dispatch-enforcement/scheduler overclaim，以及 review-only budget wording no-hit；Transcript / Heartbeat Failure Eval 已补 local/static fixtures，覆盖 stale fixed task id heartbeat、pendingWorktreeId 被误当 running worker、setup 失败后统领自己写 worker 实现代码；heartbeat prompt churn eval 已补 `OPA006` fixture，覆盖已验证通用 monitor 每轮反复写当前 worker 状态；feature-package continuity eval 已补 `OPA009`，覆盖无人值守从全局安全 backlog 抓互不相关任务导致日报/产品主线散乱；私有 transcript 解析仍未包含；

补充说明：

- `roadmap-next-task-suggester` 的第一版只读 MVP 已经具备；接下来剩余的是更深的
  policy/eval 约束，以及和 heartbeat / ledger budget 的 review-only 联动。预算
  联动必须先停留在报告和人工/App 层决策建议，不等于 helper 自动排序或调度。

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
eval；`eval run` 只运行 fixture eval；当前 orchestration-policy-auditor suite 有
22 个 local/static fixture；`eval add-failure` 能手动沉淀失败案例。
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

最现实的路线是把 **Codex App-first harness** 做扎实，并且从真实项目反馈倒推功能。
当前最有价值的反馈来自 restaurant POS rewrite 长编排，记录在
`docs/reviews/2026-06-11-restaurant-pos-orchestration-feedback.md`。

接下来不再为了“继续做小切片”而扩展 routine。只有当新工作能解决真实编排痛点，才进入
roadmap。上一批 V4 收口项已经完成，下一批应转向 **developer-agent
misalignment reduction**：把 worker 输出变成可审查、可拒绝、可升级为人工咨询的标准包，
同时把越界、误报完成、约束漂移和 heartbeat/worktree 状态误判变成可见、可复盘、
可回归测试的对象。

下一批应该优先做这些功能包，而不是继续补零散 routine。2026-06-13 已完成
misalignment reduction alpha，本节保留为能力边界和后续打磨入口：

0. Misalignment Event Log / Pushback Capture：已补 alpha local/static 闭环。
   - 目标：ledger 能记录 `constraint-violation`、`overreach`、
     `self-report-mismatch`、`setup-failure`、`heartbeat-gap`、`package-drift`
     这类本地 misalignment event。
   - 必须解决：现在很多真实问题只存在于聊天复盘里，例如 heartbeat 漏跑、
     pending setup 误判、可用槽位诱导跨 package filler dispatch。
   - 当前落地：`codex-orchestrator misalignment record --category CAT ...` 会把事件
     追加到 ledger 的 `misalignmentEvents` 和 events JSONL；`misalignment report`
     会按类别、状态、严重度、关联 task/package 输出只读 insight。
   - 输出：事件类别、来源、相关 task/package、证据标签、resolution、是否需要
     eval/rule proposal。
   - 边界：local/static state；不自动采集私有 transcript，不联网，不替代人工判断。

0.5 Claim Verifier / Evidence-Bound Self-Report：已补 alpha local/static 闭环。
   - 目标：merge-readiness、package acceptance 和 status 能列出关键完成声明，并检查
     是否有对应证据。
   - 检查声明：tests/build/gates passed、merged/pushed/cleaned、direct proof captured、
     no forbidden paths touched、no human action needed、heartbeat active。
   - 当前落地：`pack merge-readiness`、`pack acceptance` 和 orchestrator acceptance
     report 现在包含 `claimVerification`，把“有 task commit”“路径检查通过”“diff
     check 通过”“有 gate 记录”“有自审信号”“证据标签边界未越界”等声明逐条绑定到
     本地证据。
   - 必须解决：agent 过早说“完成/已验证/已推送/无需人工”时，用户很难判断它有没有证据。
   - 边界：只做 local/static evidence binding；缺证据时标 missing/blocked/proxy，
     不把本地静态检查写成 direct/pre/prod/device proof。

0.6 Misalignment Taxonomy Policy/Eval Fixtures：已补 alpha fixture。
   - 目标：把 developer-agent misalignment taxonomy 映射进 OPA/ELA/claim-verifier
     fixture，覆盖 constraint violation、overreach、inaccurate self-reporting、
     wrong diagnosis、setup failure 和 package drift。
   - 当前落地：新增 `OPA010` claim-verification guard 和
     `claim-self-report-without-verification` fixture，`policy check` / `eval run`
     会把无证据的完成声明作为回归项检查。
   - 示例：用户说 dry run 但发生 dispatch；用户说只解释但发生编辑；agent 声称 tests
     passed 但没有 command evidence；availableSlots>0 导致派无关 package worker；
     本地或间接证据被误表述成真实环境证明。
   - 边界：deterministic local/static eval；不解析私有 transcript，不做语义定罪。

0.7 Trust-Risk Status Block：已补 alpha status/observe 输出。
   - 目标：状态页在任务数量之前先告诉用户“当前能不能信”：未验证完成声明、缺失 gate
     evidence、未解决 pushback event、active constraints、当前 package lane 原因、
     heartbeat gap、watchdog 状态和 local/static-only 警告。
   - 当前落地：`observe --json`、`status --write-summary` 和 `status --html` 输出
     `trustRisk`，在普通任务数量之外列出未解决 misalignment、缺少 gate/self-review、
     stale worker、heartbeat gap、setup reconcile gap、constraint drift 等风险。
   - 必须解决：用户不应该从 availableSlots、英文 task id 或机器字段推断进度和风险。
   - 边界：status UX；不启动 server，不调度，不 merge/push/cleanup。

0.8 Constraint Stack / Worker Contract Snapshot：已补 alpha ledger 字段。
   - 目标：dispatch 时把最新用户指令、repo/skill 规则、allowed/forbidden paths、
     required gates、evidence labels、merge/push/cleanup authority、package lane 和
     package-switch reasons 固化进 ledger。
   - 当前落地：`record-task` 支持 `--constraint`、`--authority`、
     `--user-instruction`、`--evidence-boundary`、`--package-switch-reason`，并把快照写入
     `constraintStack`。
   - 必须解决：长 session 或上下文压缩后，worker 和 reviewer 应该按当时的约束快照
     验收，而不是按聊天记忆。
   - 边界：local/static contract snapshot；不是 runtime proof。

0.9 Misalignment Insights Report：已补 alpha local/static report。
   - 当前落地：`codex-orchestrator misalignment report --repo . --json` 会读取 ledger
     里的 misalignment events、当前 observe/trust risk、heartbeat local/static 状态，
     输出 grouped counts、open/high severity events、recommended actions 和 evidence
     boundaries。
   - 边界：不上传 transcript，不推断模型意图，不作为自动 merge authority。

下一批候选：

0. Model-Agnostic Review Pack：已补 package-level external review MVP。
   - 目标：从 ledger 和 git truth 生成可给 Codex / Claude / DeepSeek / local model
     或人类 reviewer 使用的标准审查包。
   - 当前落地：
     `codex-orchestrator pack review --package-id PKG --repo . --output review-pack/PKG`；
     `codex-orchestrator pack acceptance --package-id PKG --repo . --write-report package-acceptance.json`；
     `codex-orchestrator review policy check --package-id PKG --risk medium --task-count 4`；
     `codex-orchestrator review run --package-id PKG --reviewer pi|claude --pack review-pack/PKG`；
     `codex-orchestrator review import --package-id PKG --reviewer deepseek --file review.md --status passed`。
   - 输出：task contract、changed files、diff patch、allowed/forbidden paths、
     requested vs observed gates、docs drift、evidence labels、residual risks、
     reviewer prompt 和 blocked claims。
   - 已解决的第一层问题：外部模型 review 不再只能靠复制聊天或手工整理上下文；
     审查包可复用，Pi/Claude 可由 helper 只读调用，DeepSeek/人工等结果可用
     `review import` 记回 ledger。
   - 已解决的第二层问题：`review policy show/check` 会读取
     `.codex-orchestrator/review-policy.json` 或内置默认策略，检查本机 Pi/Claude/Codex
     reviewer 可用性，并在功能包边界推荐零个、一个或两个 reviewer。`pack review`
     和 `pack acceptance` 现在都能从 `packageId` 自动选择相关 worker，避免统领手工
     复制一串 task id。
   - 已解决的第三层问题：`pack acceptance` 支持 post-cleanup mode。已经
     accepted/merged/pushed/cleaned 的 terminal task 即使 worker worktree 被删除，也会
     用 ledger terminal state 和 recorded gates 作为 local/static closeout evidence，
     不再把正常 cleanup 误判成 package acceptance blocked。
   - 使用时机：不是每个小切片都跑；适合 3-5 个相关 worker 组成一个 feature package、
     或出现 shared contract / DB / API / security / payment / hardware / pre/prod 风险时。
   - 边界：local/static review material；不是 runtime proof，不自动 merge，不因为
     多模型同意就自动接受。外部 reviewer 输出只能作为 `proxy/advisory` evidence。
   - 已解决的第四层问题：`packageSummary` 现在会根据内置 review policy 标记
     `reviewRequired` / `reviewDecision` / `reviewNextAction`。当一个 package 已有多个
     worker 或命中高风险关键词时，状态页和 Markdown summary 会提示生成 review pack
     并导入 reviewer evidence，避免 feature package 到收口阶段才想起异模型 review。
   - 已解决的第五层问题：当 Claude 等 reviewer 因额度、auth 或本地安装不可用时，
     持续编排可使用可用 reviewer（例如 Pi）并记录 unavailable proxy/advisory context，
     不再把缺失 reviewer 当成必须等待用户确认的默认停止点。
   - 下一步：如需更强审计，可再把 package-level review state 从派生状态提升成 ledger
     一等字段；当前版本先保持从 routine run / policy 派生，避免过早扩 schema。

1. Ledger-Enforced Dispatch Closure：已补 local/static helper slice。
   - 目标：派发后立即记录 taskId、pendingWorktreeId、resolved thread、
     worktree、branch、baseCommit、allowed/forbidden paths、gates 和 status。
   - 必须解决：长跑中 pending ids、branch、gates、review 状态仍然散落在 heartbeat
     文本和聊天总结里。
   - 当前落地：
     `codex-orchestrator dispatch record --task-id TASK --pending-worktree-id ID`
     和 `codex-orchestrator dispatch reconcile --task-id TASK`。`dispatch record`
     会把 pending setup、预期 branch、base commit、allowed/forbidden paths、gates、
     optional title/thread id 写入 ledger；`dispatch reconcile` 会用本地
     `git worktree list --porcelain` 对账真实 worktree/branch，并写入
     `dispatch-reconcile` 事件。
   - 边界：local/static orchestration state；pendingWorktreeId 不是 running worker
     proof；resolved worktree 不是 task correctness proof。

2. Project-Aware Roadmap Scorer：已补 local/static helper slice。
   - 目标：支持 restaurant POS rewrite 这类项目自定义 source-of-truth docs，而不是只读
     `docs/roadmap.md`。
   - 输出候选任务的 `vertical-completion`、`runtime-proof`、`blocked-removal`、
     `owner-gated`、`shallow-risk` 分类，以及 write-set / external dependency
     风险。
   - 当前落地：
     `codex-orchestrator roadmap score --repo .`，可选
     `--config roadmap-score.json` 使用简单 JSON `sources` 列表覆盖默认 sources，
     可选 `--ledger PATH` 或默认 repo-local ledger 对已完成/已合并/已清理任务做只读降权。
     默认会读取存在的 `docs/roadmap.md`、`PROGRESS.md` 和
     `docs/整体开发计划与进度.md`；review docs 必须通过 `--config`
     显式加入，避免把复盘风险句误当成下一批任务。feature package / package
     status 候选会优先排在互不相关的安全小任务前面。
   - 边界：local/static planning evidence；不 dispatch、不修改 git/ledger、不联网、
     不声称 direct runtime/product proof；只建议 dispatch，不代表允许 merge。

3. Consultation Request Pack：已补 local/static helper slice。
   - 目标：当任务 blocked、需要产品决策或需要人做物理动作时，生成结构化求助包。
   - 输出 blocker、已尝试路径、证据、需要人的动作/决策、可选方案和后果、worktree/branch
     保留或清理建议。
   - 当前落地：
     `codex-orchestrator pack consultation --task-id TASK [--repo PATH] [--ledger PATH] [--write-report PATH] [--json]`。
     它只读加载本地 ledger task、task history、routine runs、recorded gates、
     evidence labels，并用已有本地 worktree metadata 分类 blocker/stale/setup 状态；
     输出 required human input、decision options/tradeoffs、next safe action 和
     branch/worktree keep/clean 建议。后续吸收 Peter Steinberger
     `maintainer-orchestrator` 的三条维护者纪律：`ownerDecisionBrief` 不让 owner
     面对裸 blocker 做决定；`authorizationMatrix` 明确 consultation 不授权
     implementation/merge/push/release；`liveProofGate` 明确真实 runtime/device/provider
     proof 或 item-specific waiver 仍在 pack 外部。
   - 边界：local/static consultation planning evidence；不 dispatch、不 merge/push、
     不 cleanup、不修改 ledger/git state、不联网、不声称 direct runtime/product/device
     proof；真正的人类决策或物理动作仍然是 pack 外部的 `blocked` 项。

4. Transcript / Heartbeat Failure Eval：已补 local/static fixture slice。
   - 目标：把真实编排失败变成 regression fixture。
   - 已覆盖 child 完成后忘记继续队列、heartbeat 绑定旧任务或 stale fixed task id、
     pendingWorktreeId 被误当 running worker、setup 失败后 fallback 主 checkout、
     证据升级、统领下场写 worker 代码等错误。
   - 推荐命令：
     `codex-orchestrator eval run --suite orchestration-policy-auditor`
   - 边界：local/static eval；当前不解析私有 transcript，使用脱敏 fixture；私有 transcript validation 标 blocked/not included。

5. Static Dashboard / Status Page：已补 local/static HTML 输出。
   - 目标：把 `observe` / `status` 的 JSON 变成可读 HTML/Markdown 状态页。
   - 输出 active、pending setup、dirty-uncommitted、completed-unreviewed、blocked、
     cleanup-needed、recent merged/cleaned、available slots、budget/review pressure 和
     next action。
   - 推荐命令：
     `codex-orchestrator status --html > orchestrator-status.html`
   - 当前落地：`status --html` 会把本地 ledger/observe 状态渲染成中文友好的静态
     HTML，包含 integration、队列压力、下一步建议、证据标签、各状态任务分组、任务列表和
     最近 routine run。
   - 边界：local/static status evidence；不引入 daemon/web server，不调度、不 merge、
     不 push、不 cleanup。

已完成的 V4 收口项：

1. Runtime status report：已补。
   - 目标：每次 heartbeat / observe 都能一眼说明当前到底在等什么。
   - 输出 active workers、pending setup、dirty-uncommitted、completed-unreviewed、
     merged-this-cycle、blockers、cleanup-needed 和 available dispatch slots。
   - 边界：只读状态面板；不创建 session、不 merge/push、不删除 worktree。
   - 当前落地：helper 已在 `status` / `observe --json` / heartbeat summary 中输出
     `runtimeStatus` 本地静态报告，并补了 jobs/status 风格的 `jobSummary` 与首次编排
     前的 `projectMap` 准备度提示；仍不代表 Codex App runtime/daemon direct proof。

2. First-class setup/worktree state model：已补。
   - 目标：把 `pendingWorktreeId`、真实 worktree、branch、dirty diff、clean commit、
     completed-unreviewed、blocked、merged、cleaned 作为工具级状态，而不是靠聊天记忆。
   - 必须解决：pending setup 被误当成 active worker、重复派发同一任务、线程状态和 git
     状态不一致。
   - 当前落地：`observe` / `status` / heartbeat JSON 的每个 observation 和
     `runtimeStatus` item 都包含 `state`，把 setup/worktree/branch/diff/review/cleanup
     拆开表达；detached `HEAD` 且 ledger 记录了 branch 时报告为 `blocked`；
     clean task commit 始终报告为 `completed-unreviewed`，直到 orchestrator review。
   - 边界：这仍然是 local/static helper evidence，不查询 Codex App runtime，不创建
     session，不 merge/push/delete/cleanup。

3. Automated review checklist：已补。
   - 目标：在 merge 前自动生成 reviewer checklist，而不是只靠统领手工记得查。
   - 检查 allowed/forbidden paths、diff name-status、`git diff --check`、review doc、
     artifact、worker self-review、evidence labels、docs drift 和 required gates。
   - 当前落地：`run-routine pr-reviewer` 复用已有 ledger/task/worktree/git 检查，并补
     local/static 自动 review checklist。它会检查 task/worktree/branch/dirty 状态、
     `baseCommit..HEAD` 提交、`git diff --name-status`、`git diff --check`、
     ledger `writeSet.allowed`/`writeSet.forbidden` 路径边界、本地可检测的
     review/self-review/artifact/evidence-label 文件名信号，以及 ledger 记录的窄
     gates 建议。明确 forbidden path 命中或 allowed path 越界会 fail；缺少本地可检测
     review/self-review/artifact/evidence 信号会作为 warning/needs-human 进入报告。
   - 边界：它可以 block/warn，但不自动 merge，不 push，不 cleanup，不 dispatch，也不把
     local/static checklist 当作 direct runtime proof。

4. Merge-Readiness Pack：已补 local/static helper slice。
   - 目标：把一个 completed-unreviewed task 转成标准验收包。
   - 输出 task id/title/status/thread/pendingWorktreeId/worktree/branch/baseCommit、
     本地 observed status、`git status --short --branch`、`baseCommit..HEAD`
     commit 数、`git diff --name-status`、ledger writeSet allowed/forbidden path
     检查、`git diff --check`、review doc/artifact/self-review/evidence-label/docs
     drift 信号、recorded gates、suggested gates、residual risks 和 `needsHuman`。
   - 当前落地：
     `codex-orchestrator pack merge-readiness --task-id TASK --write-report /tmp/merge-readiness-pack.json`。
     report status 会在 task/worktree/base/diff 证据缺失时标 `blocked`，在 dirty
     worktree、0 commit、writeSet 违规或 `git diff --check` 失败时标 `failed`，
     对缺失 review/self-review/artifact/evidence/docs-drift/gate 证据设置
     `needsHuman`。报告现在还包含 `authorizationMatrix`、`liveProofGate` 和
     `acceptanceReport` 草案，用来区分 review-ready、reject-for-fixup、blocked、
     merge/push/cleanup/release 授权边界，以及 live proof / waiver 缺口。
   - 边界：local/static review evidence；不自动 merge/push/cleanup/dispatch，不修改
     git state，不声称 runtime、production、device 或 direct worker proof。

5. Evidence-label linter：已补 local/static helper slice。
   - 目标：把 restaurant POS rewrite 里最危险的证据升级问题变成机器可检查规则。
   - 特别检查 review docs、progress/roadmap docs、handoff summaries 中是否把
     `local`/`proxy`/`weak` 写成 `direct`/`pre`/`prod`/`device`/payment proof。
   - 当前落地：`run-routine evidence-label-auditor` 会扫描 `docs/reviews/*.md`，
     并用 `ELA010` 标出没有 explicit direct evidence wording 的
     local/static/proxy 到 direct/pre/prod/device/runtime/payment proof 升级嫌疑。
   - 边界：仍然只输出 local/static suspicion，不检查真实 pre/prod/device/payment
     runtime，不产生 direct proof，不修改 ledger/git/worktree。

6. Post-merge docs drift guard：已补 local/static helper slice。
   - 目标：accepted merge 后明确提示 central docs 是否需要 orchestrator-owned update。
   - 当前落地：`docs-drift-checker` 会本地静态扫描 `docs/reviews/*.md`，当已接受或已合并
     的任务记录提到 command/routine/source 这类中央文档影响面，却没有记录中央文档更新
     或明确的 docs-drift 决策时，输出 `local` post-merge docs-drift guard 告警。
   - 允许结论：docs updated、docs-not-needed、central-docs-pending、docs-drift-blocked。
   - 边界：worker 默认不直接改中央 progress/roadmap；统领负责合并后的统一同步或显式说明。

7. Case study and bootstrap docs：已补。
   - 目标：把 restaurant POS rewrite 写成脱敏真实案例，展示 worktree 隔离、review/merge/cleanup、
     evidence label 和 blocked boundary。
   - README 继续保持 Codex App-first 入口清晰：用户可以把 GitHub 链接交给 Codex App
     dry-run，而不是先学习 CLI。
   - 当前进展：已补一版脱敏案例、案例文章和 bootstrap 文案收口；README 首屏已经把
     “Codex App-first workflow，而不是 daemon/package-manager/agent OS”讲清楚。
     继续观察真实用户是否还会把 helper 误解为主入口，或把本地证据误读成
     runtime/prod/payment/hardware proof。

暂不进入的方向：

- 重 daemon、自动 session scheduler、worker pool 或 agent OS；
- Homebrew、npm、tap 或其他 package-manager 分发；
- 自动 merge/push/delete worktree 的后台系统；
- 声称 helper 能证明 live Codex App runtime、production、payment、hardware 或设备行为。

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
