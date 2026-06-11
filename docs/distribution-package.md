# Distribution Package

This document is the beta distribution path for external users. It keeps the
human-facing setup simple: Codex App remains the primary entrypoint, while the
Go helper can be installed from source or release assets when Codex App wants
durable ledger/routine support. Homebrew, npm wrappers, taps, and other
package-manager distribution routes are out of scope for this product path.

## Status

Current package: `v0.3.0-beta.5` GitHub prerelease

Implemented:

- `v0.3.0-beta.5` git tag source install path.
- GitHub prerelease with darwin/linux/windows assets:
  https://github.com/indiekitai/codex-orchestrator/releases/tag/v0.3.0-beta.5
- `scripts/install.sh` source install path for users with Go.
- Release asset download smoke for `darwin_arm64`.
- Shell completion generation for bash, zsh, and fish.
- Release verifier routine for local tag and proxy GitHub release metadata.
- Release publishing helper that creates/releases assets through `gh api`.

Blocked or not yet implemented:

- Background daemon.

Out of scope:

- Homebrew formula or tap distribution.
- npm wrapper distribution.
- Any other package-manager route for the helper binary.

## Recommended User Flow

New users should still start by pasting the README bootstrap prompt into Codex
App. Codex App should explain which installation path it wants to use and why.
The human should not need to learn the helper CLI before the dry run.

Release binaries are useful when Codex App decides durable helper state is
worth installing. They are not the product's primary entrypoint.

The expected mental model is: give the GitHub repository to Codex App; let
Codex read this repository, install the Codex App skill if needed, and install
or build the helper only when the run benefits from durable local state.

## Install From Release Asset

The intended release-asset path is:

https://github.com/indiekitai/codex-orchestrator/releases/tag/v0.3.0-beta.5

`v0.3.0-beta.5` is published as a GitHub prerelease with release assets for:

- `darwin_amd64`
- `darwin_arm64`
- `linux_amd64`
- `linux_arm64`
- `windows_amd64`

The `darwin_arm64` tarball was downloaded from GitHub Release, extracted, and
smoked with `--help` plus bash/zsh/fish completion generation.

## Install From Source

```bash
git clone https://github.com/indiekitai/codex-orchestrator.git
cd codex-orchestrator
git checkout v0.3.0-beta.5
scripts/install.sh
codex-orchestrator --help
```

Use `BIN_DIR` to install somewhere else:

```bash
BIN_DIR=/usr/local/bin scripts/install.sh
```

## Shell Completions

The helper can print shell completion scripts:

```bash
codex-orchestrator completion bash
codex-orchestrator completion zsh
codex-orchestrator completion fish
```

Example installation paths:

```bash
# bash
mkdir -p ~/.local/share/bash-completion/completions
codex-orchestrator completion bash > ~/.local/share/bash-completion/completions/codex-orchestrator

# zsh
mkdir -p ~/.zsh/completions
codex-orchestrator completion zsh > ~/.zsh/completions/_codex-orchestrator

# fish
mkdir -p ~/.config/fish/completions
codex-orchestrator completion fish > ~/.config/fish/completions/codex-orchestrator.fish
```

For zsh, ensure `~/.zsh/completions` is in `fpath` before `compinit`.

## No Package-Manager Distribution

Do not present Homebrew, npm, taps, or package-manager wrappers as planned or
desirable user routes. They are outside the current product scope.

Manual source install and GitHub release binaries may remain available as
advanced helper paths, but the primary distribution route is still Codex
App-first: the user gives Codex App this GitHub repository and lets Codex decide
whether installing the helper is useful.

## Release Notes

The release workflow should populate GitHub Release notes from:

```text
docs/beta-release-notes-draft.md
```

If GitHub API editing is blocked, the release can still be valid when the tag,
workflow, prerelease flag, and assets verify. Record the release-body failure as
`blocked` or `failed` evidence instead of treating it as source/tag install
failure.

## Publishing Release Assets

Use the helper scripts when GitHub Release API credentials are available:

```bash
scripts/build-release-assets.sh v0.3.0-beta.5 /tmp/codex-orchestrator-dist
scripts/publish-release.sh v0.3.0-beta.5 /tmp/codex-orchestrator-dist
```

`scripts/publish-release.sh` intentionally checks
`gh api repos/indiekitai/codex-orchestrator` before trying to publish. The API
account must have write, maintain, or admin permission. This is separate from
git push access: SSH may have write access even when the active `gh` account
only has read API access.

If the permission check fails, authenticate `gh` as an account that can publish
releases for `indiekitai/codex-orchestrator`, then rerun the script.

The script uses direct `gh api` release creation and upload endpoints instead of
`gh release create`. This is intentional: during `v0.3.0-beta.2` publication,
`gh release create` returned `401 Requires authentication` even after
`gh api repos/indiekitai/codex-orchestrator` showed admin API permission. Direct
API creation and uploads succeeded. The script also retries GitHub API calls
because delete/upload operations may intermittently return `401` even when the
same authenticated account has verified release permission.

## Verification

Before announcing a distribution package:

```bash
go test ./...
go build -trimpath -ldflags='-s -w' -o /tmp/codex-orchestrator ./cmd/codex-orchestrator
/tmp/codex-orchestrator completion bash >/tmp/codex-orchestrator.bash
/tmp/codex-orchestrator completion zsh >/tmp/_codex-orchestrator
/tmp/codex-orchestrator completion fish >/tmp/codex-orchestrator.fish
go run ./cmd/codex-orchestrator validate-routines --dir routines
go run ./cmd/codex-orchestrator run-routine release-verifier --tag v0.3.0-beta.5 --repo . --json
```

Evidence labels:

- `local`: source build, completion generation, release binary smoke, and local
  tag inspection.
- `proxy`: GitHub Release metadata, release asset names from `gh`, and release
  asset download smoke from GitHub.
- `failed`: local tag exists but the GitHub Release is missing or missing
  expected assets.
- `blocked`: GitHub Release API auth/network failures or metadata that cannot
  be inspected. Missing Homebrew, npm, tap, or package-manager distribution is
  not a blocker because those routes are out of scope.

Do not claim direct production, daemon, deployed runtime, payment, hardware, or
Codex App session-launch proof from this distribution package.
