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
codex-orchestrator record-routine-run --report-json /tmp/pr-reviewer-report.json
```

`run-routine pr-reviewer` is read-only against the task worktree. It loads the
ledger task, checks that the worktree exists, verifies the current branch when
the ledger records one, captures `git status --short --branch`, checks commits
after `baseCommit`, records `git diff --name-status baseCommit..HEAD`, and runs
`git diff --check baseCommit..HEAD`. It does not merge, push, delete branches,
clean worktrees, run task-specific tests, or claim runtime proof. Its report
uses `local` evidence only unless a future routine actually observes another
surface.

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

`run-routine ci-fixer` is a read-only CI/local gate classifier, not an
auto-fixer. It loads the ledger task by id, requires explicit recorded task
gates, verifies worktree existence and expected branch, refuses dirty
worktrees, checks commits and changed files after `baseCommit`, and runs the
recorded gate commands in the task worktree with a local timeout. Passing gates
plus committed work after `baseCommit` pass for orchestrator review/merge.
Dirty worktrees or failing gates fail with local evidence and a same-worker or
same-task takeover recommendation. Missing gates, missing `baseCommit`, branch
mismatches, and git inspection failures block. It does not stage, commit,
merge, push, clean, dispatch, update ledger status, or claim direct/proxy
runtime proof; this MVP emits only `local` or `blocked` evidence.

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
`docs/routines/README.md`, and `docs/roadmap.md` when present for obvious
missing routine references or stale status text. Missing specs or docs
references fail; missing repository/source/spec access blocks. It does not
stage, commit, merge, push, tag, release, clean worktrees, dispatch sessions,
mutate the ledger, or claim runtime proof; this MVP emits `local` or `blocked`
evidence.

`run-routine evidence-label-auditor` is a read-only local/static evidence-label
checker. It does not load or update the ledger. It scans explicit repo-local
docs, routine specs, routine report JSON files, and ledger-shaped JSON for
obvious evidence-label issues: weak evidence labels near overstated proof
wording, RoutineRunReport JSON missing the required evidence buckets, and
direct evidence recorded for routines whose specs explicitly reserve direct
evidence. Findings are heuristics and are reported as local/static suspicions,
not semantic proof. It does not stage, commit, merge, push, tag, release, clean
worktrees, dispatch sessions, mutate the ledger, or claim runtime proof; this
MVP emits `local` or `blocked` evidence.

Routine specs live in [`../../routines`](../../routines). They are JSON so the
Go helper can validate them without a Python, YAML, or Node dependency.

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
