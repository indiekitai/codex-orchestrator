# Real Codex App Demo Proof

Date: 2026-06-10
Task ID: `TF-CODEX-ORCH-BETA-REAL-APP-DEMO-WORKER-LOCAL`
Source thread: `019e9036-3c4d-7bf3-b72c-3ff929a81882`
Worker thread: `019eb1e5-d407-7ff2-95a6-0858eeb32b25`
Worker worktree: `/Users/tf/.codex/worktrees/49a9/codex-orchestrator`
Worker branch: `codex/tf-codex-orch-beta-real-app-demo-worker-local`
Worker commit: `2053b481c5da432fc21700571a40fd3bd60f3c2a`
Merge commit: `10197c5ffa17986ebb77a618aaa3093867003852`

## Result

The real Codex App demo checklist was completed for one small delegated worker
task:

1. A Codex App worktree session was created from the orchestrator.
2. The pending worktree resolved to a real worker thread and worktree.
3. The worker attached the detached worktree to a task branch.
4. The worker created and committed a single allowed file.
5. The worker produced a final handoff and did not merge, push, or clean.
6. The orchestrator reviewed the worker diff and routine report.
7. The orchestrator merged the worker branch into `main`.
8. The orchestrator ran post-merge local gates.
9. The orchestrator pushed `main` to `origin/main`.
10. The orchestrator removed the worker worktree and deleted the local worker
    branch.
11. The helper ledger now records the task as `cleaned`.

This closes the helper-only smoke gap for App dispatch / review / merge / push
/ cleanup at the local workflow level.

## Evidence

### Direct

None. This demo does not claim production, daemon, hardware, payment, deployed
runtime, or external direct proof.

### Proxy

None used as success proof.

### Local

- Codex App created worker thread
  `019eb1e5-d407-7ff2-95a6-0858eeb32b25`.
- Codex App created worker worktree
  `/Users/tf/.codex/worktrees/49a9/codex-orchestrator`.
- Worker branch existed:
  `codex/tf-codex-orch-beta-real-app-demo-worker-local`.
- Worker commit existed:
  `2053b481c5da432fc21700571a40fd3bd60f3c2a`.
- Worker changed exactly one file:
  `docs/reviews/2026-06-10-real-app-demo-worker-note.md`.
- `run-routine pr-reviewer` passed and recorded:
  - worktree exists,
  - branch matches ledger,
  - worktree clean,
  - one commit after base,
  - committed diff contains only the worker note,
  - `git diff --check` passed.
- Merge commit exists:
  `10197c5ffa17986ebb77a618aaa3093867003852`.
- `git push origin main` completed successfully.
- `git worktree remove /Users/tf/.codex/worktrees/49a9/codex-orchestrator`
  completed successfully.
- `git branch -d codex/tf-codex-orch-beta-real-app-demo-worker-local`
  completed successfully.
- Final `go run ./cmd/codex-orchestrator observe --json` reported this task as
  `cleaned`, with no active, pending, stale, blocked, or cleanup-needed work.

### Blocked / Not Proven

- No long-running daemon behavior was exercised.
- No Homebrew, npm wrapper, or binary release asset installation was exercised.
- No production runtime, external service, hardware, payment, or deployed
  environment proof was attempted.
- The helper still does not create Codex App sessions by itself; the App
  orchestrator created the worker session.

## Commands And Gates

Worker side:

```bash
git diff --check
git status --short --branch
```

Orchestrator review and merge side:

```bash
go run ./cmd/codex-orchestrator run-routine pr-reviewer \
  --task-id TF-CODEX-ORCH-BETA-REAL-APP-DEMO-WORKER-LOCAL \
  --write-report /tmp/real-app-demo-pr-reviewer.json --json
git diff main..codex/tf-codex-orch-beta-real-app-demo-worker-local -- docs/reviews/2026-06-10-real-app-demo-worker-note.md
git diff --name-status main..codex/tf-codex-orch-beta-real-app-demo-worker-local
git diff --check main..codex/tf-codex-orch-beta-real-app-demo-worker-local
git merge --no-ff codex/tf-codex-orch-beta-real-app-demo-worker-local -m "merge: real app demo worker note"
GOMAXPROCS=1 go test ./...
go run ./cmd/codex-orchestrator validate-routines --dir routines
go run ./cmd/codex-orchestrator run-routine docs-drift-checker --repo . --json
go run ./cmd/codex-orchestrator run-routine evidence-label-auditor --repo . --json
git diff --check HEAD~1..HEAD
git push origin main
git worktree remove /Users/tf/.codex/worktrees/49a9/codex-orchestrator
git branch -d codex/tf-codex-orch-beta-real-app-demo-worker-local
go run ./cmd/codex-orchestrator observe --json
```

All listed orchestrator gates passed.

## Self-Review

- Diff boundary: worker diff was one allowed `docs/reviews` file. This proof
  report is also under `docs/reviews`.
- Forbidden paths: no code, routine specs, release workflow, skill file, README,
  hidden config, secrets, or release assets were changed by the worker.
- Evidence labels: this report uses local evidence only and explicitly avoids
  claiming direct runtime/prod/hardware/payment/daemon proof.
- Docs drift: `docs-drift-checker` passed after merge.
- Evidence policy: `evidence-label-auditor` passed after merge.
- Residual risk: this proves a real App-orchestrated local workflow, not a
  standalone daemon or CLI-driven session launcher.
