# Distribution Package Verification

Date: 2026-06-10
Scope: beta distribution package for `codex-orchestrator`

## Summary

The distribution package now has concrete beta source/tag artifacts beyond the
App-first README prompt:

- shell completion generation for bash, zsh, and fish,
- a Homebrew formula draft that builds from the `v0.3.0-beta.2` tag,
- a distribution guide covering release assets, source install, completions,
  formula usage, release notes, and evidence boundaries,
- release workflow wiring to use `docs/beta-release-notes-draft.md` as the
  GitHub Release body for future tags.

This is still a beta distribution package, not a full package ecosystem.
`v0.3.0-beta.2` has a git tag and source install proof, but GitHub Release
publication is blocked by release API authentication. There is no dedicated
Homebrew tap repository, npm wrapper, or daemon.

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
- Clean clone from `v0.3.0-beta.2` tag passed source install:
  - `BIN_DIR=/tmp/codex-orch-beta2-tag-smoke/bin ./scripts/install.sh`
  - `codex-orchestrator completion bash|zsh|fish`
  - `codex-orchestrator init`
  - `codex-orchestrator observe --json`

### proxy

- The release workflow body path points at the repository release notes draft
  for future release publication.
- GitHub Actions build jobs for `v0.3.0-beta.2` passed for darwin/linux/windows
  targets.

### blocked

- GitHub Actions publish job for `v0.3.0-beta.2` failed in
  `softprops/action-gh-release@v2` with `401 Requires authentication`.
- Local `gh release create v0.3.0-beta.2 ...` also failed with
  `401 Requires authentication`.
- `release-verifier --tag v0.3.0-beta.2` returned `failed` because the local tag
  is present but the GitHub Release is not found.
- `v0.3.0-beta.1` release assets do not include the new `completion` command.
- A dedicated Homebrew tap repository was not created.
- Homebrew 5 rejects direct local formula installs outside a tap:
  `Homebrew requires formulae to be in a tap`.
- The package was not tested on Linux or Windows machines in this pass.

## Boundaries

No Codex App sessions were dispatched. No release tag was moved. No GitHub
Release assets were published or edited. No daemon, production, payment,
hardware, deployed runtime, or Codex App session-launch proof is claimed.

## Verdict

Accepted as a beta source/tag distribution foundation with blocked GitHub
Release publication. The next packaging step should be fixing release API
permissions or publishing a real tap repository, not more internal routine work.
