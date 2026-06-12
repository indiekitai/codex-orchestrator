# codex-orchestrator progress audit

Date: 2026-06-12

Scope: read the current `main` checkout, release state, README/distribution docs,
roadmap docs, helper status output, routine validation, policy/evidence/docs
auditors, and roadmap scorer output.

## Current state

- Repo state before this review: `main...origin/main`, no active worktrees.
- Latest published GitHub Release: `v0.3.2`.
- Source state: `main` contains post-`v0.3.2` changes:
  - missed heartbeat detection;
  - macOS watchdog fallback;
  - reusable status snapshot outputs;
  - package/external review workflows;
  - run-mode guardrails.
- Ledger/status state: 38 historical tasks, 30 cleaned and 8 merged, with no
  active, pending, review-needed, blocked, or cleanup-needed task at audit time.

## What is solid

The project is no longer just a prompt skill. It now has five real layers:

1. App-first orchestration runbook in `SKILL.md`.
2. Durable local state and visibility through ledger, `observe`, `status`,
   heartbeat reports, reusable status snapshots, and watchdog reports.
3. Acceptance and review packs through merge-readiness, consultation,
   decision-ready fields, package review, and external review pack generation.
4. Policy/eval/routine library, including docs drift, evidence label, stale task,
   CI fixer, release verifier, and orchestration policy routines.
5. Product-facing docs and case-study material for the App-first usage model.

The strongest areas are state visibility, review/acceptance discipline, and
policy/eval checks. These directly address repeated real failures from
TastyFuture: pending setup confusion, child-task completion stopping the larger
queue, local/proxy proof being overstated, and heartbeat prompt churn.

## Findings fixed in this audit

1. README still described heartbeat as fixed "5-minute" behavior. The current
   product model is configurable and project-specific, so the README now says
   configurable heartbeat instead of hard-coding 5 minutes.
2. `docs/distribution-package.md` still described `v0.3.0-beta.5` as the current
   package. It now points at the current published `v0.3.2` release and clearly
   separates `v0.3.2` artifacts from post-`v0.3.2` changes on `main`.
3. `docs/roadmap.md` still described the first stable tag as only `v0.3.0` and
   framed `v0.3.1` as the next visibility release. It now records that stable
   release artifacts have reached `v0.3.2`, while newer watchdog/status/review
   work remains post-release.

## Main product gap

The biggest remaining gap is planning quality, not another routine runner.

`roadmap score` currently produces noisy candidates from review prose and risk
sentences, such as "Review doc writes local/proxy proof as direct/pre/prod
proof" or "Project-specific live proof rules still need project docs...".
Those are useful warnings, but they are not clean feature-package backlog items.

This explains why downstream project orchestration can feel scattered: the
helper can track and verify tasks well, but its "what should we do next" layer
still needs stronger source filtering and feature-package grouping.

## Recommended next work

1. Roadmap scorer v2:
   - read only explicit backlog/task/next-action sections by default;
   - separate warnings, eval fixtures, and review findings from dispatchable
     feature candidates;
   - group candidates under product-package lanes;
   - output "do not dispatch" reasons when the candidate is only a warning.
2. Status surface polish:
   - make the human-readable status summary the default thing an orchestrator
     prints every cycle;
   - include package lane, active tasks, review queue, missed heartbeat count,
     current blocker, and next dispatch decision.
3. Release closeout:
   - after deciding whether post-`v0.3.2` watchdog/status/review work is stable
     enough, cut the next release and update release notes.
4. Real-app proof:
   - continue using TastyFuture as the main case study, but measure by coherent
     feature-package lanes rather than by count of small cleaned tasks.

## Verification

- `git diff --check`: passed.
- `go test ./...`: passed.
- `go run ./cmd/codex-orchestrator validate-routines --dir routines --json`:
  passed, 14 specs valid.
- `go run ./cmd/codex-orchestrator policy check --repo .`: passed.
- `go run ./cmd/codex-orchestrator run-routine docs-drift-checker --repo . --json`:
  passed.
- `go run ./cmd/codex-orchestrator run-routine evidence-label-auditor --repo . --json`:
  passed.
- `go run ./cmd/codex-orchestrator run-routine release-verifier --tag v0.3.2 --repo . --json`:
  passed with local tag evidence and proxy GitHub Release asset evidence.

## Evidence labels

- `local`: repo status, git log, docs scan, helper status output, routines,
  policy/eval checks, Go tests.
- `proxy`: GitHub Release metadata and release asset names from `gh`.
- `direct`: none. This audit did not run a live Codex App multi-session proof.
- `blocked`: none for this audit; next release decision remains a product
  choice, not a blocker.
