# Loop Acceptance Economics

Date: 2026-06-22

## Input

- X Article: <https://x.com/AnatoliKopadze/status/2068328135611822149>

## Summary

The article is broad and partly consumer-product promotion, so it did not
change the public positioning of `codex-orchestrator`. The useful engineering
points were narrower:

- a loop without a verifier is just the agent agreeing with itself;
- state must live outside the transient chat;
- every loop needs a stop condition;
- not every task deserves automation;
- the useful operator metric is accepted change, not prompt count, worker
  count, or loop iteration count.

## Changes

- `cmd/codex-orchestrator/main.go`: added a local/static `acceptance` summary
  under `jobSummary` with accepted, rejected, abandoned, blocked, reviewable,
  in-progress, cleanup-needed, terminal-decision, and accepted-change-rate
  fields.
- `cmd/codex-orchestrator/main.go`: surfaced acceptance counts in CLI, Markdown
  status, and HTML status.
- `cmd/codex-orchestrator/main_test.go`: covered the new status/JSON/Markdown
  acceptance summary.
- `README.md` and `README.zh-CN.md`: documented accepted change, verifier
  gates, durable state, and stop conditions.
- `SKILL.md`: required acceptance economics, objective verifiers, and package
  stop conditions in orchestration decisions.
- `docs/research/loop-engineering-alignment.md`: recorded the useful lessons
  from the article without adopting the consumer-product framing.
- `docs/roadmap.md`: noted the status-surface direction.

## Evidence Labels

- `proxy`: public X article used as product/research input.
- `local`: repository code, docs, and tests updated locally.
- `local/static`: the new acceptance summary is derived from local ledger and
  observe state. It is not runtime, Codex App, device, pre/prod, or provider
  proof.

## Verification

- `gofmt -w cmd/codex-orchestrator/main.go cmd/codex-orchestrator/main_test.go`
- `go test ./...`
- `go run ./cmd/codex-orchestrator run-routine evidence-label-auditor --repo . --json`
- `go run ./cmd/codex-orchestrator policy check --write-report /tmp/codex-orchestrator-policy-check.json --json`
- `git diff --check`

## Residual Risk

`acceptedChangeRate` is intentionally a simple ledger-derived signal:
accepted / (accepted + rejected + abandoned). It excludes active, reviewable,
cleanup-needed, and blocked tasks. It does not measure token spend, elapsed
time, or human review time. Those remain future budget/reporting work.
