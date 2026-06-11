# Project-Aware Roadmap Scorer

## Scope

- Implemented `codex-orchestrator roadmap score --repo . [--config PATH] [--json] [--write-report PATH]`.
- The command reads local roadmap/progress/review docs only.
- Default sources are existing `docs/roadmap.md`, `PROGRESS.md`, `docs/TastyFuture-整体开发计划与进度.md`, and `docs/reviews/*.md`.
- Optional config is a small JSON file with a `sources` array.
- Rework added optional read-only ledger awareness via `--ledger PATH`, defaulting to repo-local `.codex-orchestrator/ledger.json` when present.

## Local Evidence

- The scorer extracts local candidate lines from configured docs.
- It classifies candidates as `vertical-completion`, `runtime-proof`, `blocked-removal`, `owner-gated`, or `shallow-risk`.
- It reports static write-set hints and external-dependency hints when local wording makes them inferable.
- It demotes candidates that match completed/merged/cleaned ledger task id/title/branch/history values, so stale review-doc follow-ups do not outrank current roadmap pending work.
- It records that human project judgement, runtime/product proof, provider credentials, deployment state, and device evidence remain blocked/out of scope.

## Reviewer Finding Rework

Reviewer finding: the first implementation could rank stale/completed review-doc follow-ups, such as `Budget-policy static eval remains a follow-up for detecting future wording`, above the current pending `docs/roadmap.md` item `Consultation Request Pack：待做`.

Fix: `roadmap score` now loads the optional/default ledger read-only and demotes candidates matching terminal ledger tasks (`completed-unreviewed`, `merged`, `released`, `cleaned`, `rejected`, or `abandoned`). Same-score ties also prefer current roadmap/progress sources over old review docs. The focused regression test covers a cleaned `Budget-policy static eval` ledger task and verifies the current roadmap pending item ranks first.

## Commands Run

- `go test ./cmd/codex-orchestrator -run 'TestRoadmapScore|TestRunRoadmapNextTaskSuggester'`
- `go test ./...`
- `go build ./cmd/codex-orchestrator`
- `go run ./cmd/codex-orchestrator run-routine docs-drift-checker --repo . --json`
- `go run ./cmd/codex-orchestrator run-routine evidence-label-auditor --repo . --json`
- `go run ./cmd/codex-orchestrator roadmap score --repo . --json`
- `git diff --check`

All commands passed locally.

## Boundaries

- `local`: static docs/config parsing, local tests, and generated local reports.
- `proxy`: none.
- `direct`: none; no runtime, production, device, network, deployment, Codex App automation, or external-provider proof.
- `blocked`: deciding whether a scored candidate is truly the right next project task still requires human review.

## Self-Review

- Diff reread: completed after implementation and verification.
- Allowed paths: intended changes stay in `cmd/codex-orchestrator/**`, `README.md`, `README.zh-CN.md`, and `docs/**`.
- Forbidden paths: no intended edits to `.github/**`, `Formula/**`, `dist/**`, package-manager distribution files, release notes/tags, credentials, or unrelated project files.
- Docs drift: README and Chinese README document the user-facing command; roadmap marks the slice complete only for the local/static helper.
- Verification gaps: no direct runtime/product proof is claimed; the scorer remains a static planning aid.
- Residual risks: keyword scoring is conservative and approximate; project-specific judgement and source quality determine whether the suggested next action is useful.
