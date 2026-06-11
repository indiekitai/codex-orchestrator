# Loop Engineering Alignment

This note checks whether `codex-orchestrator` is aligned with the emerging
"loop engineering" framing, and where the roadmap should change before adding
more features.

## Sources

- Addy Osmani, [Loop Engineering](https://addyosmani.com/blog/loop-engineering/)
- YouTube, [Reflecting on a year of Claude Code](https://www.youtube.com/watch?v=Hth_tLaC2j8)
- Auto-generated transcript provided locally:
  `/Users/tf/Downloads/[English (auto-generated)] Reflecting on a year of Claude Code [DownSub.com].txt`
- Feiteng Li's Chinese summary of the same Claude Code year-review interview:
  <https://x.com/FeitengLi/status/2064594760195801538>
- Daniel Demmel, [Feedback loop engineering](https://www.danieldemmel.me/blog/feedback-loop-engineering)
- Latent.Space, [Claude Code: Anthropic's Agent in Your Terminal](https://www.latent.space/p/claude-code)
- Martin Fowler / Thoughtworks, [Harness engineering for coding agent users](https://martinfowler.com/articles/harness-engineering.html)
- Addy Osmani, [Agent Harness Engineering](https://addyosmani.com/blog/agent-harness-engineering/)
- Thoughtworks, [Harness engineering and agent feedback: Exploring AI coding sensors](https://www.thoughtworks.com/en-us/insights/blog/generative-ai/harness-engineering-agent-feedback-exploring-ai-coding-sensors)
- The Pragmatic Engineer, [The creator of Clawd: "I ship code I don't read"](https://newsletter.pragmaticengineer.com/p/the-creator-of-clawd-i-ship-code)
- WyeWorks, [The Workflow Is the Product](https://www.wyeworks.com/es/blog/2026/05/13/custom-agentic-workflows-for-coding-agents/)
- Lower-weight trend scans from Reddit, Substack, MindStudio, DX, and X search
  results around "loop engineering", used only to check terminology drift.
- User-provided Grok/X synthesis covering recent Loop Engineering discussions,
  examples, criticisms, and related handles. This is useful for trend discovery
  but treated as second-hand unless linked back to original posts.

## Working Definition

Loop engineering is not just "run an agent repeatedly." It is designing the
system that prompts, observes, checks, records, and improves agent work while
the human moves up to loop design and evidence review.

The common pattern across the sources:

1. A loop discovers or receives work.
2. It dispatches one or more agents into isolated execution environments.
3. It gives them enough project knowledge, but not an overloaded context dump.
4. It makes them run and verify their work in the real environment.
5. It records what happened outside the transient chat context.
6. It uses separate review, security, or eval checks before trusting results.
7. It turns repeated mistakes into durable rules, skills, evals, or routines.

That means a useful loop has both an outer orchestration layer and an inner
verification layer:

- Outer loop: task discovery, dispatch, worktree isolation, heartbeat,
  review/merge, cleanup, and next-task selection.
- Inner loop: the worker can run the product, inspect logs/browser/device/db,
  see failures, fix, and retest.
- Learning loop: failures become skills, project rules, evals, or policy checks
  so the same mistake is less likely next time.

## What The Claude Code Interview Adds

The Claude Code year-review interview is important because it describes actual
operating practice, not just terminology.

Key takeaways from the provided transcript:

- Repeated agent mistakes should not be fixed only by telling that one agent to
  behave differently. Boris describes writing the lesson into `CLAUDE.md`, a
  skill, or another durable mechanism.
- Verification means more than lint, typecheck, or unit tests. The agent needs
  to run the thing. The interview gives examples around simulators, desktop app
  workflows, local app launch, computer use, edge cases, fixing, and rechecking.
- Routines are a concrete next layer. The examples include watching GitHub
  issues or bug reports, proactively proposing fixes, babysitting PRs, fixing
  CI, rebasing, and routing work back to the owner for review.
- Safety moved from raw permission prompts to model/classifier checks, backed
  by collected trajectories, red-team attempts, prompt-injection tests, and
  evals.
- The "loop" leap is explicitly described as moving from talking to source
  code, to talking to an agent, to talking to a loop or routine that prompts the
  agent.
- Context strategy is minimalist: give the model a small system prompt, minimal
  tools, and a way to pull context when needed, rather than stuffing the whole
  world into the prompt.
- The future shape is many longer-running agents, not one short synchronous
  session. The user interface for coordinating them will likely change.

## What Addy's Article Adds

Addy's framing gives the practical 5+1 checklist:

- automations,
- worktrees,
- skills,
- plugins/connectors,
- sub-agents,
- memory/state outside the chat.

`codex-orchestrator` maps well to automations, worktrees, skills, and
memory/state. It is weaker on plugins/connectors and on a formal maker/checker
sub-agent split.

Addy also emphasizes that loops do not remove engineering judgment. They make
quality problems sharper because unattended loops can make unattended mistakes.
That matches this project's evidence-label discipline and review-before-merge
stance.

## What Feedback Loop Engineering Adds

Daniel Demmel's feedback-loop framing is a useful correction: orchestration is
not enough. A loop that cannot observe runtime behavior is mostly a task
manager.

Important implications:

- Inner-loop proof should be product-specific and runtime-specific.
- Good loop tooling should expose browser state, logs, crash traces, database
  state, traces, simulators, devices, and real command output in CLI-friendly
  ways.
- The best interfaces for current agents are small, pipeable, progressively
  explorable CLIs with structured output.
- The outer loop should reflect finished-session lessons into shared knowledge.

This is the biggest gap in `codex-orchestrator`: it has strong outer-loop
mechanics, but it does not yet provide a library of inner verification routines.

## What Harness Engineering Adds

Harness engineering is the broader frame around the loop. The useful mental
model is:

```text
agent = model + harness
```

The harness includes:

- feedforward guides that steer the agent before it acts, such as
  `AGENTS.md`, `CLAUDE.md`, skills, specs, types, linting, and architecture
  rules;
- feedback sensors that let the agent observe consequences, such as tests,
  browser state, logs, traces, database checks, simulator runs, screenshots,
  security checks, and maintainability sensors;
- control boundaries that decide what tools the agent may use, when it must
  stop, and when a human or verifier must take over.

This matters because `codex-orchestrator` is not just a loop. It is part of the
harness. The skill is a feedforward guide. The Go helper and heartbeat are
state/feedback sensors. The review-before-merge and evidence-label rules are
control boundaries.

The gap is that the current harness is stronger at orchestration than at
runtime sensing. It knows whether a worker branch exists, whether a worktree is
dirty, and whether a commit needs review. It does not yet know whether the
feature actually worked in a browser, mobile app, terminal, database, API,
device, or CI system unless the delegated task manually reports that evidence.

## What Peter Steinberger / Closed-Loop Workflows Add

Peter Steinberger's public workflow discussions reinforce a practical point:
parallel agents are not the hard part by themselves. The hard part is closing
the loop so agents can compile, lint, execute, inspect output, and validate
their own work before handing it back.

This adds two constraints to the roadmap:

1. Do not measure loop maturity by number of simultaneous sessions. A loop with
   three well-instrumented workers is more valuable than a loop with twenty
   blind workers.
2. Each routine should name its verification surface. "Fix CI" is a real loop
   because CI is an observable sensor. "Implement feature" is not a complete
   loop unless it includes runtime proof.

`codex-orchestrator` already handles the review bandwidth problem by defaulting
to low concurrency. That should remain a feature, not a limitation. The next
improvement is not more parallelism; it is better per-worker verification.

## What Workflow Engineering Adds

The WyeWorks workflow framing is useful because it shifts attention from a
single clever agent to a repeatable pipeline. Their pattern is roughly:

```text
plan -> analyze -> implement -> validate -> visual/runtime QA -> review
```

This is close to what `codex-orchestrator` needs for V2.5. A routine should not
only say "run tests." It should define the whole validation pipeline for a
specific task family:

- what context to inspect,
- what artifact to produce,
- what runtime to launch,
- what sensor to read,
- what evidence to store,
- what failure means,
- what human decision is needed.

That suggests the routine library should be written as workflow contracts, not
just command aliases.

## What The X/Grok Trend Scan Adds

The user-provided Grok/X synthesis mostly confirms the same pattern, with a few
extra trend signals:

- The phrase is very recent and practitioner-driven, with conversation peaking
  around early to mid June 2026.
- The commonly repeated origin points are Addy Osmani, Peter Steinberger, and
  Boris Cherny.
- Real-loop examples mentioned in the scan include:
  - morning triage loops that read CI/issues/commits and write to `LOOP.md` or
    Linear;
  - PR babysitting loops that address review comments, CI failures, and rebase
    work;
  - Felix Craft's "Ralph loop" as an earlier related wrapper around planning,
    execution, tmux/worktrees, validation, and restartable state;
  - sub-agent review loops with explicit scoring or verifiable stop criteria;
  - long-running `/goal` or scheduled `/loop` maintenance tasks.
- Operators often mention always-on machines or VPS-style setups for 24/7
  loops, but that should be treated as deployment preference, not a requirement
  for this project.
- Criticism clusters around:
  - "is this just rebranded harness/context engineering?";
  - token costs and token-rich vs token-poor workflows;
  - quality drift and slop accumulation;
  - comprehension debt and cognitive surrender;
  - naming fatigue around prompt/context/harness/loop engineering.

These are not all first-hand claims, but they are useful product-positioning
constraints. The open-source project should be careful to avoid hype terms like
"fully autonomous" and should emphasize bounded loops, verification, and human
review.

## Current Coverage

`codex-orchestrator` already covers:

- Worktree isolation.
- Bounded task contracts.
- Concurrency control.
- Heartbeat monitoring.
- Stale-session recovery.
- Review-before-merge.
- Persistent ledger and events in v2.
- Evidence labels such as `direct`, `proxy`, and `blocked`.
- Self-review requirements.
- Living runbook behavior: repeated mistakes should update skill/rules/docs.
- Human notification checkpoints for real-world actions.

This is a credible outer orchestration loop.

## Gaps

The main gaps are not more task states. They are the deeper loop layers:

1. **Inner verification routines**

   The project needs reusable routines for browser proof, log proof, database
   proof, mobile/device proof, API proof, and artifact proof. These should be
   CLI-friendly and report structured evidence.

2. **Maker/checker separation**

   Today a worker self-reviews and the orchestrator reviews. That is useful, but
   not the same as a reusable verifier routine or dedicated review agent shape.

3. **Plugins/connectors**

   The repo currently does not define connector patterns for issue trackers,
   CI systems, PRs, Slack/notifications, or external project state.

4. **Security classifier and eval loop**

   The roadmap mentions this, but the interview suggests it is not optional for
   higher autonomy. Permission spam does not scale; dangerous actions need
   policy/classifier/eval support.

5. **Routine library**

   V3 should not be just YAML files. It should describe triggers, inputs,
   allowed actions, forbidden actions, gates, output schema, and escalation, and
   should be backed by helper commands or examples.

6. **Outer learning loop**

   The skill says to update runbooks, but there is no command or template for
   converting a completed session or failure into a proposed skill/rule/eval.

7. **Harness boundary clarity**

   The project should explicitly distinguish feedforward guides, feedback
   sensors, and control boundaries. Today they are present, but mixed together
   across README, `SKILL.md`, docs, and helper behavior.

8. **Workflow contracts**

   Routine specs should describe the full pipeline for a family of work, not
   just a prompt or command. Without this, V3 could become a collection of
   shortcuts rather than robust loops.

9. **Cost and review-budget accounting**

   Loop Engineering discussions repeatedly warn about token burn and review
   bandwidth. `codex-orchestrator` should keep low default concurrency and
   eventually expose cost/review pressure as part of heartbeat status.

## Roadmap Adjustment

The previous staged roadmap is directionally right through v4, but the emphasis
should change.

Recommended structure:

| Layer | Status | Purpose |
|-------|--------|---------|
| v1: outer orchestrator skill | Mature enough | Codex App session dispatch, worktrees, review, merge, cleanup |
| v2: durable state + heartbeat | Alpha complete | Ledger, events, heartbeat report, resume from repo truth |
| v2.5: verification routine foundation | Next | Browser/log/db/device proof interfaces, evidence schemas, harness boundary model |
| v3: routine library | After v2.5 | Workflow contracts for CI fixer, PR reviewer, stale rescuer, docs drift, release verifier |
| v4: safety/eval layer | Needed before high autonomy | Classifiers, prompt-injection evals, dangerous action policy |
The key change is adding **v2.5 verification routine foundation** before a broad
routine library. Otherwise v3 risks becoming a list of task managers instead of
real loops.

An agent operating system should not be part of the current roadmap. The
project should stop at a Codex App-first harness with durable state, routine
contracts, policy/eval checks, and reviewable rule improvement.

## Product Positioning

The most accurate public positioning is:

> `codex-orchestrator` is a Codex App-first outer-loop orchestrator for coding
> agents. It turns roadmap work into isolated sessions, tracks them in a durable
> ledger, runs heartbeat checks against git/worktree truth, and keeps a human
> reviewer in control of merge and evidence quality.

Avoid claiming:

- full agent operating system,
- autonomous development platform,
- complete Loop Engineering runtime,
- automatic safety classifier,
- automatic PR/CI/rebase routine library,
- production verification layer.

Use "Loop Engineering" carefully:

> It implements the outer orchestration loop and v2 persistent heartbeat layer.
> The next work is to add inner verification routines and safety/eval gates.

## Recommended Next Work

Do not jump to a daemon UI or worker-pool platform. The highest-leverage next
work is:

1. Add `docs/routines/` with routine specs for:
   - stale task rescuer,
   - PR reviewer,
   - CI fixer,
   - browser/runtime proof,
   - docs drift checker.
2. Add a common routine output schema:
   - `status`,
   - `evidence`,
   - `actionsTaken`,
   - `needsHuman`,
   - `blockedReason`,
   - `nextSuggestedAction`.
3. Add an evidence schema that separates:
   - direct runtime proof,
   - proxy proof,
   - local-only proof,
   - blocked claims.
4. Add a first policy checker command for dangerous paths and evidence
   exaggeration before adding more automation.
5. Add a "reflect failure into rule" template before trying self-improving
   rules.
6. Add a short harness map:
   - feedforward guides,
   - feedback sensors,
   - control boundaries,
   - current implementation,
   - missing implementation.
7. For each routine spec, require a workflow contract:
   - trigger,
   - context inputs,
   - isolated execution surface,
   - verification sensors,
   - evidence output,
   - escalation rule.
8. Add lightweight cost/review-budget notes to the orchestrator status model:
   - active worker count,
   - review-needed queue length,
   - stale queue length,
   - whether dispatch should pause because review bandwidth is saturated.

## Conclusion

The project is aligned with Loop Engineering, but it currently covers the outer
loop more than the inner verification loop.

The broader web scan strengthened that conclusion rather than overturning it.
The repeated theme across Loop Engineering, Feedback Loop Engineering, Harness
Engineering, and closed-loop workflow discussions is the same: the leverage is
not just asking more agents to do more work. The leverage is designing a
recoverable, observable, bounded system where agents can verify themselves and
where failures become durable improvements.

That is a good starting point. The mistake would be to keep adding
orchestration features while postponing runtime verification and safety
classification. The next phase should make the loop more trustworthy, not
merely more autonomous.
