# V3 Alpha E2E Smoke

Date: 2026-06-10
Branch: `codex/tf-codex-orch-v3-alpha-e2e-smoke-local`
Scope: local/static validation plus proxy GitHub release metadata for `v0.3.0-alpha.10`.

## Summary

The V3 alpha helper path works end to end locally from a user perspective:
build, initialize a ledger, record a task with worktree/branch and budget
metadata, observe ledger state, write a heartbeat report and summary, validate
routine specs, run read-only task routines against a fixture task, and run the
repo-local read-only checker/auditor/suggester routines.

No README, `SKILL.md`, routine-doc, roadmap, or code drift was found during the
smoke. This report is the only readiness artifact added.

## Evidence

### Local Build And Test

- `GOMAXPROCS=1 go test ./...`
  - Result: passed.
  - Evidence label: `local`.
- `go build -trimpath -ldflags='-s -w' -o /tmp/codex-orchestrator-go ./cmd/codex-orchestrator`
  - Result: passed.
  - Evidence label: `local`.

### Temporary Ledger Flow

Temporary root: `/tmp/codex-orch-e2e.WMiGPq`

Fixture shape:

- Git repository: `/tmp/codex-orch-e2e.WMiGPq/repo`
- Task worktree: `/tmp/codex-orch-e2e.WMiGPq/task-wt`
- Base commit: `5773629370e38f71110dee12a140aa967ca41dde`
- Task commit: `81d65f55a95b78767bad9f2afb155b340389b042`
- Task branch: `codex/e2e-task`
- Task ID: `TF-SMOKE-TASK`

Commands:

```bash
/tmp/codex-orchestrator-go init --ledger "$REPO/.codex-orchestrator/ledger.json" --project-root "$REPO" --default-branch main --max-concurrency 2
/tmp/codex-orchestrator-go record-task --ledger "$REPO/.codex-orchestrator/ledger.json" --id TF-SMOKE-TASK --title "Local e2e smoke task" --worktree "$TASK_WT" --branch codex/e2e-task --base-commit "$BASE" --allowed task.txt --forbidden secrets --gate "git diff --check HEAD~1..HEAD" --max-runtime-minutes 30 --review-budget-minutes 10 --budget-note "local smoke budget" --evidence local --evidence-note "local fixture smoke"
/tmp/codex-orchestrator-go observe --ledger "$REPO/.codex-orchestrator/ledger.json" --json --write-report "$REPORT_DIR/observe.json" --write-summary "$REPORT_DIR/observe.md"
/tmp/codex-orchestrator-go heartbeat --ledger "$REPO/.codex-orchestrator/ledger.json" --count 1 --write-report "$REPORT_DIR/heartbeat.json" --write-summary "$REPORT_DIR/heartbeat.md" --json
/tmp/codex-orchestrator-go run-routine pr-reviewer --ledger "$REPO/.codex-orchestrator/ledger.json" --task-id TF-SMOKE-TASK --write-report "$REPORT_DIR/pr-reviewer.json" --json
/tmp/codex-orchestrator-go run-routine stale-task-rescuer --ledger "$REPO/.codex-orchestrator/ledger.json" --task-id TF-SMOKE-TASK --write-report "$REPORT_DIR/stale-task-rescuer.json" --json
```

Results:

- `observe`: `review-needed`
- `heartbeat`: `review-needed`
- `pr-reviewer`: `passed`
- `stale-task-rescuer`: `passed`

Evidence label: `local`. The fixture proves local CLI behavior only; it does
not prove Codex App session dispatch, merge, push, cleanup, production runtime,
or daemon behavior.

### Routine Validation And Read-Only Runners

- `/tmp/codex-orchestrator-go validate-routines --dir routines`
  - Result: passed, all 12 JSON specs valid.
  - Evidence label: `local`.
- `go run ./cmd/codex-orchestrator validate-routines --dir routines`
  - Result: passed, all 12 JSON specs valid.
  - Evidence label: `local`.
- `go run ./cmd/codex-orchestrator run-routine docs-drift-checker --repo . --json`
  - Result: `passed`; `README.md`, `README.zh-CN.md`, `SKILL.md`,
    `docs/routines/README.md`, and `docs/roadmap.md` mention all runnable
    routines.
  - Evidence label: `local`.
- `go run ./cmd/codex-orchestrator run-routine evidence-label-auditor --repo . --json`
  - Result: `passed`; scanned 21 repo-local evidence-label input files and
    found no rule hits.
  - Evidence label: `local`.
- `go run ./cmd/codex-orchestrator run-routine roadmap-next-task-suggester --repo . --json`
  - Result: `passed`; suggested `evidence label auditor follow-on policy/eval
    expansion beyond the current named-rule local layer`.
  - Evidence label: `local`.
- `go run ./cmd/codex-orchestrator run-routine release-verifier --tag v0.3.0-alpha.10 --repo . --json`
  - Result: `passed`; local tag resolves to
    `2d5fbe571113acdcecafe9c47b79b3a6cddfe9f3`, GitHub release is
    non-draft prerelease, and expected Go CLI assets are present.
  - Evidence label: `local` for tag inspection and `proxy` for GitHub release
    metadata through `gh`.

### Diff Hygiene

- `git diff --check`
  - Result: passed.
  - Evidence label: `local`.
- `git diff --name-status main..HEAD`
  - Result: only this report is added.
  - Evidence label: `local`.

## Docs Readiness

The checked docs are clear enough for the alpha helper flow:

- `README.md` and `README.zh-CN.md` list install/build usage, ledger commands,
  heartbeat reports, budget visibility, App-first boundaries, and all runnable
  read-only routines.
- `SKILL.md` states that the helper is not a session launcher and documents the
  read-only routine boundaries.
- `docs/routines/README.md` documents the routine output schema, evidence
  labels, and the current runnable MVPs.

No minimal docs correction was needed.

## Residual Risks

- This smoke did not create a real Codex App worktree session. App dispatch
  remains outside the local CLI proof surface.
- This smoke did not merge, push, clean worktrees, or run a daemon/automation
  loop, by design.
- Release verification used `gh release view` as proxy metadata. It did not
  download every release asset or execute the released binaries.
- The temporary fixture proves routine behavior on a simple committed task
  branch, not on a complex production repository with conflicts or dirty state.

## Self-Review

- Diff reread: only this concise smoke report was added.
- Allowed paths: change is under `docs/reviews/**`, which is allowed.
- Forbidden paths: no `.github/workflows/**`, `dist/**`, secrets, credentials,
  generated binaries, feature code, runner implementation, daemon behavior, or
  rebase helper files were changed.
- Docs drift: `docs-drift-checker` passed after the smoke; no misleading docs
  drift was found.
- Verification gaps: all requested local gates passed; remaining gaps are
  explicitly local/proxy-only boundaries listed above.
