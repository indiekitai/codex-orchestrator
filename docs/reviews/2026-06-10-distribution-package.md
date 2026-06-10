# Distribution Package Verification

Date: 2026-06-10
Scope: beta distribution package for `codex-orchestrator`

## Summary

The distribution package now has concrete beta artifacts beyond the App-first
README prompt:

- shell completion generation for bash, zsh, and fish,
- a Homebrew formula draft for the `v0.3.0-beta.1` release assets,
- a distribution guide covering release assets, source install, completions,
  formula usage, release notes, and evidence boundaries,
- release workflow wiring to use `docs/beta-release-notes-draft.md` as the
  GitHub Release body for future tags.

This is still a beta distribution package, not a full package ecosystem. There
is no dedicated Homebrew tap repository, npm wrapper, or daemon.

## Evidence

### local

- `GOMAXPROCS=1 go test ./...` passed.
- `go build -trimpath -ldflags='-s -w' -o /tmp/codex-orchestrator ./cmd/codex-orchestrator`
  passed.
- Generated shell completions from the built helper:
  - `/tmp/codex-orchestrator completion bash` produced 53 lines.
  - `/tmp/codex-orchestrator completion zsh` produced 70 lines.
  - `/tmp/codex-orchestrator completion fish` produced 22 lines.
- `go run ./cmd/codex-orchestrator validate-routines --dir routines` passed.
- `go run ./cmd/codex-orchestrator run-routine docs-drift-checker --repo . --json`
  passed.
- `go run ./cmd/codex-orchestrator run-routine evidence-label-auditor --repo . --json`
  passed.
- `ruby -c Formula/codex-orchestrator.rb` returned `Syntax OK`.
- `git diff --check` passed.

### proxy

- Formula URLs and checksums were taken from the published
  `v0.3.0-beta.1` GitHub Release assets and asset digests.
- The release workflow body path points at the repository release notes draft
  for future release publication.

### blocked

- A dedicated Homebrew tap repository was not created.
- `brew audit --formula Formula/codex-orchestrator.rb` did not complete within
  the local run window and was stopped to avoid leaving a long-running process.
- The formula was not installed into the user's global Homebrew prefix during
  this verification.
- The package was not tested on Linux or Windows machines in this pass.

## Boundaries

No Codex App sessions were dispatched. No release tag was moved. No GitHub
Release assets were edited. No daemon, production, payment, hardware, deployed
runtime, or Codex App session-launch proof is claimed.

## Verdict

Accepted as a beta distribution package foundation. The next packaging step
should be either a real tap repository or a clean machine install/download smoke,
not more internal routine work.
