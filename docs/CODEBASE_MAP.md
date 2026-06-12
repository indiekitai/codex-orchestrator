# codex-orchestrator Codebase Map

This is a concise orientation map for Codex App before it starts broad
orchestration work in this repository.

## Core Surfaces

| Path | Purpose |
|---|---|
| `SKILL.md` | Codex App skill entrypoint and orchestration runbook. Keep runtime behavior and safety rules here when Codex should follow them during work. |
| `cmd/codex-orchestrator/main.go` | Go helper CLI. Contains ledger commands, observe/status/heartbeat reports, routine runners, policy/eval helpers, and release verification helpers. |
| `cmd/codex-orchestrator/main_test.go` | Go test coverage for ledger lifecycle, observe/status state, routines, policy/eval fixtures, release checks, and CLI behavior. |
| `routines/*.json` | Routine contracts: inputs, allowed/forbidden actions, gates, evidence labels, and budget metadata. |
| `docs/routines/README.md` | Human-facing routine library documentation. Update when routine behavior or command surface changes. |
| `docs/v2-usage.md` | Practical helper CLI usage for App-first orchestration. Update when `observe`, `status`, `heartbeat`, ledger, or routine workflows change. |
| `docs/v2-persistent-ledger-and-heartbeat.md` | Durable ledger and heartbeat design. Update when state schema or heartbeat semantics change. |
| `docs/roadmap.md` | Current roadmap and phase status. Update when a feature slice completes or a route is intentionally de-scoped. |
| `docs/reviews/` | Review and proof notes. Add a dated note for non-trivial changes, especially CLI behavior, release, policy/eval, or safety changes. |
| `examples/` | Example ledgers, routine reports, heartbeat reports, and external-user prompt artifacts. Keep examples in sync with schema changes. |
| `scripts/` | Install, update, release, compatibility, macOS watchdog, and legacy helper scripts. Do not add package-manager distribution scripts unless explicitly re-scoped. |

## Common Verification Gates

Use the narrowest credible gate for a change:

- Go helper or schema behavior: `go test ./...`
- Routine/docs surface: `go run ./cmd/codex-orchestrator run-routine docs-drift-checker --repo . --json`
- Evidence wording: `go run ./cmd/codex-orchestrator run-routine evidence-label-auditor --repo . --json`
- Orchestration safety wording or policy fixtures: `go run ./cmd/codex-orchestrator policy check --repo . --json`
- Markdown/whitespace: `git diff --check`
- Release tags/assets: `go run ./cmd/codex-orchestrator run-routine release-verifier --repo . --tag TAG --json`

## High-Risk Boundaries

- Do not add Homebrew, npm, tap, or other package-manager distribution as a
  mainline path. The product path is Codex App reading/installing the skill.
- Do not make the helper claim it can create Codex App sessions, merge, push,
  clean worktrees, schedule workers, or enforce budgets. Those are orchestrator
  decisions, not helper actions.
- Keep direct/proxy/local/blocked evidence honest. Most helper checks are
  `local` or `local/static`.
- Avoid turning this project into a full agent operating system. The scoped
  target is a Codex App-first orchestration harness.
- Keep Go helper changes backward-compatible where possible by adding JSON
  fields rather than renaming existing fields.

## Typical Change Routing

| Change | Usually edit | Usually verify |
|---|---|---|
| App skill behavior | `SKILL.md`, maybe README/docs | `policy check`, `evidence-label-auditor`, `git diff --check` |
| CLI observe/status/heartbeat | `cmd/codex-orchestrator/main.go`, tests, `docs/v2-usage.md`, roadmap | `go test ./...`, docs drift, policy check |
| macOS external watchdog | `scripts/macos-watchdog-run.sh`, `scripts/install-macos-watchdog.sh`, `SKILL.md`, README/docs | `bash -n scripts/*.sh`, one-shot local watchdog smoke, evidence-label auditor |
| Routine behavior | `cmd/codex-orchestrator/main.go`, `routines/*.json`, `docs/routines/README.md`, tests | `go test ./...`, `validate-routines`, docs drift |
| Policy/eval rule | `cmd/codex-orchestrator/main.go`, `eval/`, review doc | `policy check`, `eval run`, `go test ./...` |
| Release packaging | `scripts/`, `docs/distribution-package.md`, release notes/review docs | release verifier, `go test ./...` |
| Public positioning | `README.md`, `README.zh-CN.md`, site/blog if needed | docs drift, evidence-label auditor |

## First-Run Orchestration Notes

Before dispatching broad work, Codex App should:

1. Read this file, `SKILL.md`, `README.md`, and `docs/roadmap.md`.
2. Run `git status --short --branch` and preserve unrelated dirty work.
3. Run `codex-orchestrator observe --repo . --json` when the helper is installed.
4. Treat `jobSummary`, `runtimeStatus`, `projectMap`, and routine reports as
   local/static evidence, not direct runtime proof.
5. Create bounded worker sessions only after the user approves mutating
   orchestration.
