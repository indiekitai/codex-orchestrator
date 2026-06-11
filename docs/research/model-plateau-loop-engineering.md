# Model Plateau And Loop Engineering

This note records a product judgment behind `codex-orchestrator`: as coding
models become "good enough" for many ordinary engineering tasks, the leverage
shifts from picking the latest model to designing a better engineering loop.

## Context

Recent practitioner discussions increasingly say that, for many software
engineering workflows, the difference between strong current models is harder
to feel in blind daily use. The user also reported a similar experience:
DeepSeek is already useful for review, and the practical gap between model
families is often smaller than the gap between a good workflow and a weak one.

This is not a benchmark claim. It is an operating observation:

- models still hallucinate;
- architecture and system-specific knowledge still matter;
- direct runtime proof still matters;
- review quality depends heavily on context, artifacts, and stop criteria;
- the human still needs to understand the system.

The conclusion is not "models are all the same." The conclusion is that
model choice is becoming one variable inside a larger harness.

## Core Product Judgment

If many models are competent enough to implement or review bounded chunks, the
next advantage is not a larger prompt or a model-specific trick. The advantage
is a repeatable loop:

```text
bounded task contract
  -> isolated worktree/session
  -> scoped implementation
  -> self-review
  -> machine-readable review pack
  -> independent reviewer(s)
  -> evidence labels and gates
  -> merge or reject
  -> ledger update
  -> next product-package decision
```

This matches the direction of `codex-orchestrator`. The project should stay
model-agnostic and make the loop portable across Codex, Claude Code, DeepSeek,
local models, or future reviewer tools.

## What This Means For codex-orchestrator

### 1. The Loop Is The Product

`codex-orchestrator` should not position itself as a better coding model. It is
the outer engineering loop around coding agents:

- task selection;
- worktree isolation;
- durable ledger state;
- heartbeat observation;
- review/merge/cleanup discipline;
- evidence labeling;
- policy/eval fixtures;
- product-package continuity.

The durable process matters because the model can change without rewriting the
workflow.

### 2. Artifacts Matter More Than Chat

If DeepSeek, Claude, Codex, or a local model can all review a bounded diff, the
handoff artifact becomes the stable interface. A reviewer should not need the
original long thread to understand the task.

The next useful object is a model-agnostic review pack:

```text
review-pack/
  task.md
  diff.patch
  changed-files.txt
  allowed-forbidden-paths.md
  gates.md
  evidence.md
  docs-drift.md
  residual-risks.md
  reviewer-prompt.md
```

That pack can be sent to Codex, Claude Code, DeepSeek, a local model, or a
human reviewer. The acceptance decision still belongs to the orchestrator or
human owner.

### 3. Maker / Checker Separation Becomes Cheaper

When multiple models are competent, using a second model as a checker becomes
more practical:

- Codex worker implements in an isolated worktree.
- Codex orchestrator generates a review pack.
- DeepSeek or Claude reviews the pack.
- The orchestrator records agreement, disagreement, and unresolved risks.

This does not mean "two models agree, therefore merge." It means disagreement
is a useful sensor, and agreement is only one input into the acceptance report.

### 4. Evidence Labels Stay Non-Negotiable

Better models do not remove the need for evidence labels. In fact, more agents
can create more confident but unsupported claims.

The loop must keep these boundaries explicit:

- `local`: static, unit, build, fixture, or local-only proof;
- `proxy`: indirect signal that suggests a result but is not the real system;
- `direct`: observed on the real intended surface;
- `blocked`: not proven, with the missing input or condition named.

No model should be allowed to promote local/static/proxy evidence into
direct/pre/prod/device proof.

### 5. Smaller Or Cheaper Models Can Be Useful Sensors

If the review pack is well-structured, cheaper models can help with:

- changed-file summaries;
- forbidden-path scan;
- docs drift review;
- evidence-label review;
- missing-test review;
- consistency checks between task contract and diff;
- release-note or case-study copy review.

High-end models remain useful for:

- architecture decisions;
- ambiguous product tradeoffs;
- cross-module integration reasoning;
- diagnosing unclear runtime failures;
- turning repeated failures into better policy/eval rules.

The product should support this routing without hard-coding any vendor.

## Recommended Feature Direction

### P0: Model-Agnostic Review Pack

Add or extend a command that generates a portable review pack from ledger and
git truth.

Possible command:

```bash
codex-orchestrator pack review --task-id TASK --repo . --output review-pack/TASK
```

Required contents:

- task id, title, branch, worktree, base commit, head commit;
- task contract and scope;
- changed-file list;
- full patch or bounded patch reference;
- allowed/forbidden path result;
- requested gates vs observed gates;
- docs/reviews/artifact presence;
- evidence labels found;
- residual risks and blocked claims;
- reviewer prompt that can be copied into another model.

Evidence boundary:

- this is `local/static` review material;
- it does not prove runtime correctness;
- it must not claim consensus or acceptance by itself.

### P1: External Reviewer Import

Add a lightweight way to record external reviews against a task.

Possible command:

```bash
codex-orchestrator review import --task-id TASK --reviewer deepseek --file review.md
```

The ledger should record:

- reviewer name/type;
- review timestamp;
- verdict: pass, concerns, reject, or inconclusive;
- findings count;
- whether findings were addressed;
- whether the orchestrator accepted or rejected the review.

This keeps external model feedback from living only in chat.

### P2: Orchestrator Acceptance Report

After review, generate a stable acceptance artifact:

```text
accepted / rejected / blocked
why
what evidence was reviewed
which gates passed
which external reviews were considered
what risks remain
what was merged or intentionally left unmerged
```

This report is more important than a chat summary because it survives context
compression, thread changes, and model changes.

### P3: Reviewer Quality Eval Fixtures

Turn repeated reviewer failures into fixtures:

- reviewer missed forbidden path;
- reviewer accepted local evidence as direct;
- reviewer over-focused on style and missed contract drift;
- reviewer rejected correct code because it misunderstood project rules;
- reviewer ignored missing docs/gates;
- reviewer claimed a test ran when no command evidence exists.

These fixtures should test the review loop, not the coding model.

## What Not To Do

Do not build model-specific assumptions into the core:

- no "Codex-only" review artifact format;
- no "Claude-only" reviewer prompt shape;
- no DeepSeek-specific evidence rules;
- no automatic trust because two models agree;
- no claim that local models can replace direct runtime proof;
- no agent operating system detour.

The repository should remain an App-first harness with portable artifacts,
durable state, and explicit control boundaries.

## Public Positioning

Useful phrasing:

> Model selection still matters, but the durable advantage is the loop around
> the model: task contracts, isolated execution, review packs, evidence labels,
> policy fixtures, and product-package continuity.

Short version:

> The model is a replaceable worker. The loop is the engineering system.

Avoid saying:

- all models are equal;
- models have hit a final ceiling;
- automated review is enough;
- consensus means correctness;
- `codex-orchestrator` is an agent OS.

## Practical Implication

The next product work should prioritize portable review artifacts before adding
more routines. A good review pack makes the current Codex App workflow better,
helps external reviewers like DeepSeek or Claude Code, and creates a future
interface for local models or UI layers without changing the core workflow.

