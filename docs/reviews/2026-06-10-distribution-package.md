# Distribution Package Verification

Date: 2026-06-10
Scope: beta distribution package for `codex-orchestrator`

## Summary

The distribution package now has concrete beta source/tag artifacts beyond the
App-first README prompt:

- shell completion generation for bash, zsh, and fish,
- an optional Homebrew formula draft that builds from the `v0.3.0-beta.2` tag,
- a distribution guide covering release assets, source install, completions,
  formula usage, release notes, and evidence boundaries,
- release workflow wiring to use `docs/beta-release-notes-draft.md` as the
  GitHub Release body for future tags.

This is still a beta distribution package, not a full package ecosystem.
`v0.3.0-beta.2` now has a git tag, GitHub prerelease, release assets, source
install proof, release-asset download smoke, and a real Codex App demo proof.
There is no dedicated Homebrew tap repository, npm wrapper, or daemon. The
missing tap is not a beta blocker because the product is Codex App-first.

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
- Added release helper scripts:
  - `scripts/build-release-assets.sh`
  - `scripts/publish-release.sh`
- `scripts/publish-release.sh v0.3.0-beta.2 /tmp/codex-orch-script-dist-check`
  passed after switching release creation/upload to direct `gh api` endpoints
  and retrying intermittent `401 Requires authentication` responses.

### proxy

- GitHub prerelease is published:
  https://github.com/indiekitai/codex-orchestrator/releases/tag/v0.3.0-beta.2
- `gh release view v0.3.0-beta.2 --repo indiekitai/codex-orchestrator`
  reports `isDraft=false`, `isPrerelease=true`, and all 10 expected assets.
- `release-verifier --tag v0.3.0-beta.2` passed against local tag plus GitHub
  release metadata.
- Downloaded `codex-orchestrator_darwin_arm64.tar.gz` from GitHub Release,
  extracted it, and verified `--help` plus bash/zsh/fish completions.
- GitHub Actions build jobs for `v0.3.0-beta.2` passed for darwin/linux/windows
  targets.

### blocked

- GitHub Actions publish job for `v0.3.0-beta.2` failed in
  `softprops/action-gh-release@v2` with `401 Requires authentication`.
- Before GitHub CLI account switching, `gh api repos/indiekitai/codex-orchestrator`
  reported no `push`, `maintain`, or `admin` permission, while the SSH remote
  could push git tags. Release publishing needs API write permission, not only
  git push permission.
- After authenticating `gh` as `indiekitai`, direct `gh api` release creation
  and asset uploads succeeded. Local `gh release create v0.3.0-beta.2 ...`
  still returned `401 Requires authentication`, so `scripts/publish-release.sh`
  now uses direct release API endpoints.
- Direct delete/upload API calls also intermittently returned `401` during
  same-release asset replacement; retrying the authenticated call succeeded.
- `v0.3.0-beta.1` release assets do not include the new `completion` command.
- A dedicated Homebrew tap repository was not created. This is an optional
  package-manager convenience, not a blocker for the Codex App-first beta.
- Homebrew 5 rejects direct local formula installs outside a tap:
  `Homebrew requires formulae to be in a tap`.
- The package was not tested on Linux or Windows machines in this pass.

## Boundaries

No Codex App sessions were dispatched. No release tag was moved. GitHub Release
assets were published for `v0.3.0-beta.2`; no daemon, production, payment,
hardware, deployed runtime, or Codex App session-launch proof is claimed.

## Verdict

Accepted as a beta distribution package with GitHub prerelease assets, source
install proof, and real App-first demo proof. The next packaging step should be
clearer App-first install UX, not a Homebrew tap unless users explicitly ask for
package-manager-managed helper binaries.
