# Ledger-Enforced Dispatch Closure

Date: 2026-06-11

## Scope

Implemented a local/static helper path for App-first dispatch closure:

- `codex-orchestrator dispatch record`
- `codex-orchestrator dispatch reconcile`

The feature records pending Codex App setup facts into the project ledger and
later reconciles them with local `git worktree list --porcelain` truth.

## Evidence Labels

- `local/static`: ledger writes, events, JSON/text command output, and local git
  worktree reconciliation.
- `proxy`: not used for this slice.
- `direct`: not claimed. The helper does not query Codex App runtime state.
- `blocked`: not currently blocked.

Important boundary: `pendingWorktreeId` is setup evidence only. It is not proof
that a worker is running. A resolved worktree/branch is setup evidence only. It
is not proof that the task implementation is correct.

## Boundaries

Changed files stayed within allowed paths:

- `cmd/codex-orchestrator/**`
- `docs/**`
- `README.md`
- `README.zh-CN.md`

No forbidden paths were edited:

- `.github/workflows/**`
- release artifacts
- Homebrew/npm/tap/package-manager distribution files
- unrelated repo metadata

The helper still does not create Codex App sessions, dispatch workers, merge,
push, tag, release, delete branches, or clean worktrees.

## Verification

Commands run:

```bash
gofmt -w cmd/codex-orchestrator/main.go cmd/codex-orchestrator/main_test.go
go test ./cmd/codex-orchestrator
go test ./...
go build ./cmd/codex-orchestrator
git diff --check
```

Results:

- `go test ./cmd/codex-orchestrator`: passed.
- `go test ./...`: passed.
- `go build ./cmd/codex-orchestrator`: passed.
- `git diff --check`: passed.

No policy/eval wording was changed, so no policy/eval check was required for
this slice.

## Residual Risks

- `dispatch reconcile` can only resolve a pending task after local git worktree
  truth exists and the ledger or command flags provide enough branch/worktree
  identity to match it.
- The helper cannot prove whether Codex App created or continued a live thread.
  It only preserves the setup facts the App orchestrator provides.
- Correctness still depends on later `observe`, focused gates, self-review, and
  orchestrator review.
