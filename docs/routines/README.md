# Routine Contracts

V2.5 adds the foundation for reusable verification routines. A routine is a
workflow contract, not a prompt alias and not a background agent by itself.

The current runtime can validate routine specs with:

```bash
codex-orchestrator validate-routines --dir routines
codex-orchestrator validate-routines --dir routines --json
```

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
