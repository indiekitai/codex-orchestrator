# Routine Contracts

V2.5 adds the foundation for reusable verification routines. A routine is a
workflow contract, not a prompt alias and not a background agent by itself.

The current runtime can validate routine specs with:

```bash
codex-orchestrator validate-routines --dir routines
codex-orchestrator validate-routines --dir routines --json
```

It can also run the conservative routine MVPs:

```bash
codex-orchestrator run-routine pr-reviewer --ledger .codex-orchestrator/ledger.json --task-id TASK --write-report /tmp/pr-reviewer-report.json
codex-orchestrator run-routine stale-task-rescuer --ledger .codex-orchestrator/ledger.json --task-id TASK --write-report /tmp/stale-task-rescuer-report.json
codex-orchestrator run-routine ci-fixer --ledger .codex-orchestrator/ledger.json --task-id TASK --write-report /tmp/ci-fixer-report.json
codex-orchestrator run-routine release-verifier --tag v0.3.0-alpha.1 --write-report /tmp/release-verifier-report.json
codex-orchestrator run-routine docs-drift-checker --write-report /tmp/docs-drift-checker-report.json
codex-orchestrator run-routine evidence-label-auditor --write-report /tmp/evidence-label-auditor-report.json
codex-orchestrator run-routine orchestration-policy-auditor --write-report /tmp/orchestration-policy-auditor-report.json
codex-orchestrator run-routine roadmap-next-task-suggester --write-report /tmp/roadmap-next-task-suggester-report.json
codex-orchestrator run-routine budget-policy-report --write-report /tmp/budget-policy-report.json
codex-orchestrator policy check --write-report /tmp/policy-check-report.json
codex-orchestrator eval run --write-report /tmp/eval-run-report.json
codex-orchestrator eval add-failure --id dry-run-example --text "Dry run mode can dispatch workers immediately." --expect OPA001=1
codex-orchestrator record-routine-run --report-json /tmp/pr-reviewer-report.json
```

`run-routine pr-reviewer` is read-only against the task worktree. It loads the
ledger task, checks that the worktree exists, verifies the current branch when
the ledger records one, captures `git status --short --branch`, checks commits
after `baseCommit`, records `git diff --name-status baseCommit..HEAD`, and runs
`git diff --check baseCommit..HEAD`. It also emits a conservative automated
review checklist from local/static evidence: changed paths against ledger
`writeSet.allowed`/`writeSet.forbidden` when present, review artifact filename
signals, artifact/report filename signals, worker self-review or handoff
filename signals, evidence-label filename signals, and suggested narrow gates
from the ledger task. Forbidden-path hits and allowed-path misses fail the
routine; missing locally detectable review/self-review/artifact/evidence
signals are warnings that require human/orchestrator review. It does not merge,
push, delete branches, clean worktrees, run task-specific tests, or claim
runtime proof. Its report uses `local` evidence only unless a future routine
actually observes another surface.

`run-routine stale-task-rescuer` is also read-only against the task worktree. It
loads the ledger task by id, records ledger status, last observation, and recent
task history, verifies worktree existence and expected branch, captures `git
status --short --branch` and `git log --oneline -3`, and classifies stale
rescue readiness conservatively. Clean worktrees with commits after
`baseCommit` pass for orchestrator review of the committed diff. Useful
uncommitted work fails with local evidence and a same-worker or same-task
takeover recommendation. Missing worktrees, branch mismatches, missing
`baseCommit`, and git inspection failures block. It does not update ledger
status, stage, commit, merge, clean, dispatch, or claim direct/proxy runtime
proof; this MVP emits only `local` or `blocked` evidence.

`run-routine ci-fixer` is a CI/local gate classifier, not an auto-fixer. It
loads the ledger task by id, requires explicit trusted gates recorded on that
task, verifies worktree existence and expected branch, refuses dirty worktrees,
checks commits and changed files after `baseCommit`, and runs the recorded gate
commands in the task worktree with a local timeout. Because those gates are
shell commands, do not run ci-fixer against an untrusted repository or untrusted
ledger. Passing gates plus committed work after `baseCommit` pass for
orchestrator review/merge. Dirty worktrees or failing gates fail with local
evidence and a same-worker or same-task takeover recommendation. Missing gates,
missing `baseCommit`, branch mismatches, and git inspection failures block. It
does not edit files, stage, commit, merge, push, clean, dispatch, update ledger
status, or claim direct/proxy runtime proof; this MVP emits only `local` or
`blocked` evidence.

`run-routine release-verifier` is a read-only release-state checker. It does
not load or update the ledger. It requires a supplied `--tag`, verifies the
local git tag resolves to a commit, records the local tag object type, and when
`gh` is available reads GitHub release metadata with `gh release view`. It
checks alpha/beta/rc tags are marked as prereleases, stable tags are not marked
as prereleases, the release is not a draft, and the release asset names contain
the expected set. By default, the expected assets match this repo's current Go
CLI release workflow: darwin amd64/arm64, linux amd64/arm64, and windows amd64,
including both raw binaries and archive files. Override the set with repeated
`--expected-asset NAME` flags. Missing tags, missing releases, prerelease
mismatches, drafts, or missing assets fail. Missing `gh`, auth/network errors,
or unparseable release metadata block. It does not create or edit releases,
move tags, upload assets, stage, commit, merge, push, clean, dispatch, mutate
the ledger, or claim production/runtime proof; this MVP emits `local`, `proxy`,
or `blocked` evidence.

`run-routine docs-drift-checker` is a read-only local docs drift checker. It
does not load or update the ledger. It parses runnable routine IDs from
`cmd/codex-orchestrator/main.go`, checks that each runnable routine has a JSON
spec under `routines/`, and scans `README.md`, `README.zh-CN.md`, `SKILL.md`,
`docs/routines/README.md`, `docs/v2-usage.md`, and `docs/roadmap.md` when
present for obvious missing routine references or stale status text. It also
scans `docs/reviews/*.md` for accepted or merged central-impact task notes that
mention command/routine/source changes but do not record a central docs update
or explicit docs-drift decision. Missing specs, docs references, or
post-merge docs-drift guard warnings fail; missing repository/source/spec or
review-doc access blocks. It does not stage, commit, merge, push, tag, release,
clean worktrees, dispatch sessions, mutate the ledger, or claim runtime proof;
this MVP emits `local` or `blocked` evidence.

`run-routine evidence-label-auditor` is a read-only local/static evidence-label
checker. It does not load or update the ledger. It scans explicit repo-local
docs, review/handoff notes, routine specs, routine report JSON files, and
ledger-shaped JSON for obvious evidence-label issues: weak evidence labels near
overstated proof wording, weak evidence promoted to direct/pre/prod/device/
runtime/payment proof without explicit direct evidence wording,
RoutineRunReport JSON missing the required evidence buckets, and direct
evidence recorded for routines whose specs explicitly reserve direct evidence.
It applies deterministic named policy/eval rules (`ELA001`-`ELA010`), skips
glossary/prohibition/blocked-definition/rule-description wording that should
stay allowed, and includes local rule-hit summaries when findings are present.
Findings are heuristics and are reported as local/static suspicions, not
semantic proof. It does not stage, commit, merge, push, tag, release, clean
worktrees, dispatch sessions, mutate the ledger, or claim runtime proof; this
MVP emits `local` or `blocked` evidence.

`run-routine orchestration-policy-auditor` is a read-only local/static
policy/eval checker. It does not load or update the ledger. It scans
repo-local orchestration docs, prompts, routine specs, routine report JSON, and
ledger/event files for deterministic orchestration policy rules (`OPA001`-
`OPA009`): dry-run dispatch barrier, no-main-checkout fallback guard, heartbeat
continuation guard, push-confirmation stop guard, delegated worker boundary,
evidence promotion boundary, heartbeat target binding guard, pending worktree
ledger guard, heartbeat lifecycle misuse such as foreground sleep or duplicate
creation, repeated generic heartbeat prompt updates, budget-policy
evidence/control boundary drift, and unrelated safe-backlog dispatch that
breaks feature-package continuity.
Findings are heuristics and are reported as local/static suspicions, not
semantic proof. It does not stage, commit, merge, push, tag, release, clean
worktrees, dispatch sessions, mutate the ledger, or claim runtime proof; this
MVP emits `local` or `blocked` evidence.

`policy check` is the product-facing V4 policy/eval wrapper for the
orchestration policy auditor. It runs the same read-only local/static scan and
then checks JSON eval fixtures from `eval/orchestration-policy-auditor/`. A
fixture defines synthetic files and expected `OPAxxx` rule-hit counts, which
turns repeated orchestration failures into deterministic regression checks.
The wrapper emits a normal `RoutineRunReport` with `routineId=policy-check`.
It does not stage, commit, merge, push, tag, release, clean worktrees, dispatch
sessions, mutate the ledger, or claim runtime proof; this command emits
`local` or `blocked` evidence.

`eval run` runs the fixture suite without scanning the current repository text.
Use it while changing policy rules to catch regressions in known good and bad
cases. The default suite is `orchestration-policy-auditor`, backed by
`eval/orchestration-policy-auditor/`.

`eval add-failure` writes a new fixture into the suite after validating that
the provided text actually produces the declared `--expect RULE=N` hit counts.
It is intentionally manual in this MVP: it accepts `--text` or `--text-file`,
does not parse review documents automatically, and refuses to overwrite
existing fixtures unless `--force` is supplied.

`run-routine roadmap-next-task-suggester` is a read-only local planning
assistant. It reads `docs/roadmap.md`, compares the remaining v3 and explicit
remaining-task candidates against the local runnable routine ids and
`routines/*.json`, and optionally filters duplicate active/pending/merged
matches from a repo-local `.codex-orchestrator/ledger.json` when present. It
prefers conservative read-only local tasks such as checkers, auditors, and
suggester-style work, and marks the queue drained when only mutating,
release-scoped, or otherwise unsafe items remain. It does not stage, commit,
merge, push, tag, release, clean worktrees, dispatch sessions, mutate the
ledger, or claim runtime proof; this MVP emits `local` or `blocked` evidence.

Routine specs live in [`../../routines`](../../routines). They are JSON so the
Go helper can validate them without a Python, YAML, or Node dependency.

Routine specs may include `maxRuntimeMinutes` and `reviewBudgetMinutes` as
non-negative budget metadata. These fields are validated and documented for
planning visibility only; the local helper does not enforce runtime limits or
review budgets. Task-level budget metadata recorded in the ledger is surfaced
through `observe` and heartbeat summaries when present.

Budget-policy follow-up work is review-only by default. A routine may report
budget metadata coverage, local/static pressure warnings, and unknown timing
states, but it must not start, stop, prioritize, reschedule, or kill workers, and
must not make dispatch eligibility decisions without Codex App or human review.

`run-routine budget-policy-report` implements the local/static report surface
for budget policy work. It reads `docs/roadmap.md`, this routine README,
`routines/*.json`, and optional repo-local `.codex-orchestrator/ledger.json`
and `.codex-orchestrator/heartbeat-report.json` files when present. It
summarizes routine/task budget metadata coverage, copies recorded heartbeat
budget warnings only as `local` evidence, separates unknown runtime or review
timing into `blocked` evidence, and returns advisory recommendations for the
Codex App orchestrator or a human reviewer. It stays read-only: it does not
schedule, prioritize, pause, kill, dispatch, merge, push, delete, clean
worktrees, mutate the ledger, or enforce budgets.

## Required Output Shape

Every routine must produce a report with these fields:

```json
{
  "status": "passed | failed | blocked",
  "evidence": {
    "direct": [],
    "proxy": [],
    "local": [],
    "blocked": []
  },
  "actionsTaken": [],
  "needsHuman": false,
  "blockedReason": "",
  "nextSuggestedAction": ""
}
```

Example reports live in [`../../examples/routine-reports`](../../examples/routine-reports)
and can be recorded with:

```bash
codex-orchestrator record-routine-run --report-json examples/routine-reports/pr-reviewer.passed.json
```

Evidence labels are intentionally strict:

- `direct`: the routine observed the real target surface itself.
- `proxy`: the routine used indirect but relevant evidence.
- `local`: local static checks, unit tests, or fixture-only proof.
- `blocked`: the claim could not be proven safely.

Do not turn `local` or `proxy` evidence into `direct` proof in the final report.

## Current Specs

- `stale-task-rescuer`: classify stale delegated tasks and decide nudge,
  same-task takeover, blocked, or abandon.
- `pr-reviewer`: review a completed task branch before merge.
- `ci-fixer`: classify failing CI/local gates and route the task back for a
  same-worker fix; the runnable MVP does not edit code.
- `release-verifier`: verify local tag, GitHub release metadata, prerelease
  flag, and expected Go CLI assets without mutating release state.
- `docs-drift-checker`: compare runnable routine IDs, JSON specs, key docs,
  and roadmap status text without mutating repository state.
- `evidence-label-auditor`: scan local docs, routine specs, routine reports,
  and ledger-shaped JSON for obvious evidence-label misuse without mutating
  repository state.
- `orchestration-policy-auditor`: scan local orchestration docs, prompts,
  routine reports, and ledger/event files for dry-run barrier, fallback,
  continuation, worker-boundary, and evidence-promotion policy regressions.
- `roadmap-next-task-suggester`: suggest the next safe bounded roadmap task
  from repo-local roadmap, runnable routine, routine-spec, and optional ledger
  state without mutating repository state.
- `budget-policy-report`: report review-only budget metadata coverage,
  local/static heartbeat pressure warnings, unknown timing states, and
  human/App-layer recommendations without mutating orchestration state.
- `browser-runtime-proof`: verify browser-visible behavior through a browser
  harness.
- `log-proof`: verify behavior through current runtime logs.
- `database-proof`: verify persisted state through read-only queries or
  fixtures.
- `device-proof`: verify device-visible behavior while preserving hardware
  evidence labels.
- `api-proof`: verify endpoint behavior through request/response proof.

## Boundary

These specs can recommend a new Codex App worker session, but they do not create
one. Codex App orchestration remains the layer that creates sessions, merges,
pushes, and cleans worktrees.
