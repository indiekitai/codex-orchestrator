# Orchestrator Self-Failure Policy Fixtures

Date: 2026-06-11

Scope:

- `SKILL.md`
- `cmd/codex-orchestrator/main.go`
- `eval/orchestration-policy-auditor/`
- `README.md`
- `README.zh-CN.md`
- `docs/roadmap.md`

Trigger:

During live App-first orchestration, the orchestrator made two confirmed
process mistakes:

1. A heartbeat automation was created with `target_thread_id = "current"`.
   The automation file existed and was `ACTIVE`, but it was not correctly bound
   to the actual thread until the persisted automation TOML was inspected and
   recreated.
2. Two dispatched `pendingWorktreeId` values were initially kept in the
   heartbeat prompt/chat state instead of being recorded immediately in the
   durable project ledger.

Changes:

- Added `OPA006` heartbeat target binding guard.
- Added `OPA007` pending worktree ledger guard.
- Added eval fixtures for both failures:
  - `heartbeat-current-target-binding.json`
  - `pending-worktree-not-ledgered.json`
- Updated `SKILL.md` to require persisted heartbeat verification and immediate
  pending setup ledger recording.
- Updated README and roadmap references from `OPA001`-`OPA005` to
  `OPA001`-`OPA007`.

Evidence labels:

- `local`: failures were observed in local automation files, ledger behavior,
  and this live orchestration thread.
- `local`: policy/eval fixtures are deterministic local text fixtures.
- `blocked`: these checks do not prove Codex App scheduling delivery; they only
  prevent known unsafe orchestration instructions or persisted-state patterns.

Expected verification:

- `go test ./...`
- `go run ./cmd/codex-orchestrator eval run --repo . --json`
- `go run ./cmd/codex-orchestrator policy check --repo . --json`
- `go run ./cmd/codex-orchestrator run-routine docs-drift-checker --json`
- `go run ./cmd/codex-orchestrator run-routine evidence-label-auditor --json`
- `git diff --check`

Self-review notes:

- This change does not create or update automations.
- This change does not dispatch sessions.
- This change does not implement package-manager distribution.
- The new policy rules are conservative string/static checks, not semantic
  proof of runtime behavior.
