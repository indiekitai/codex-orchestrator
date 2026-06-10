# Routine Contracts

V2.5 adds the foundation for reusable verification routines. A routine is a
workflow contract, not a prompt alias and not a background agent by itself.

The current runtime can validate routine specs with:

```bash
codex-orchestrator validate-routines --dir routines
codex-orchestrator validate-routines --dir routines --json
```

It can also run the first conservative routine MVP:

```bash
codex-orchestrator run-routine pr-reviewer --ledger .codex-orchestrator/ledger.json --task-id TASK --write-report /tmp/pr-reviewer-report.json
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
- `ci-fixer`: diagnose and fix a failing CI or local gate.
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
