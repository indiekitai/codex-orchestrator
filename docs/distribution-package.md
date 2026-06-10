# Distribution Package

This document is the beta distribution path for external users. It keeps the
human-facing setup simple: Codex App remains the primary entrypoint, while the
Go helper can be installed from source today. Release assets and Homebrew tap
installation are tracked separately because they depend on GitHub Release and
tap publishing permissions.

## Status

Current package: `v0.3.0-beta.2` source/tag smoke

Implemented:

- `v0.3.0-beta.2` git tag source install path.
- `scripts/install.sh` source install path for users with Go.
- Shell completion generation for bash, zsh, and fish.
- Homebrew formula draft at `Formula/codex-orchestrator.rb` that builds from
  the release tag.
- Release verifier routine for local tag and proxy GitHub release metadata.

Blocked or not yet implemented:

- GitHub Release publication for `v0.3.0-beta.2` assets is blocked by release
  API authentication in the current environment.
- Dedicated `homebrew-tap` repository.
- npm wrapper.
- Background daemon.

## Recommended User Flow

New users should still start by pasting the README bootstrap prompt into Codex
App. Codex App should explain which installation path it wants to use and why.
The human should not need to learn the helper CLI before the dry run.

## Install From Release Asset

The intended release-asset path is:

https://github.com/indiekitai/codex-orchestrator/releases/tag/v0.3.0-beta.2

This path is currently blocked for `v0.3.0-beta.2`: the tag exists and the
GitHub Actions build matrix passed, but the publish step could not create the
GitHub Release because the release API returned `401 Requires authentication`.
Use source install until the release is published.

## Install From Source

```bash
git clone https://github.com/indiekitai/codex-orchestrator.git
cd codex-orchestrator
git checkout v0.3.0-beta.2
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

## Homebrew Formula Draft

This repository includes a formula draft:

```text
Formula/codex-orchestrator.rb
```

Homebrew 5 rejects arbitrary local formula files outside a tap. Use the formula
as a tap-ready draft, not as a one-command local install:

```bash
brew tap-new indiekitai/codex-orchestrator-tap
cp Formula/codex-orchestrator.rb "$(brew --repository indiekitai/codex-orchestrator-tap)/Formula/"
brew install indiekitai/codex-orchestrator-tap/codex-orchestrator
```

The formula builds from the release tag and installs completions from the built
helper. This is not yet a dedicated Homebrew tap. Publishing a tap should be a
separate repository operation so updates, bottle policy, and release cadence
are clear.

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
scripts/build-release-assets.sh v0.3.0-beta.2 /tmp/codex-orchestrator-dist
scripts/publish-release.sh v0.3.0-beta.2 /tmp/codex-orchestrator-dist
```

`scripts/publish-release.sh` intentionally checks
`gh api repos/indiekitai/codex-orchestrator` before trying to publish. The API
account must have write, maintain, or admin permission. This is separate from
git push access: this repository currently has an SSH remote that can push tags,
while the active `gh` account only has read API access.

If the permission check fails, authenticate `gh` as an account that can publish
releases for `indiekitai/codex-orchestrator`, then rerun the script.

## Verification

Before announcing a distribution package:

```bash
go test ./...
go build -trimpath -ldflags='-s -w' -o /tmp/codex-orchestrator ./cmd/codex-orchestrator
/tmp/codex-orchestrator completion bash >/tmp/codex-orchestrator.bash
/tmp/codex-orchestrator completion zsh >/tmp/_codex-orchestrator
/tmp/codex-orchestrator completion fish >/tmp/codex-orchestrator.fish
go run ./cmd/codex-orchestrator validate-routines --dir routines
go run ./cmd/codex-orchestrator run-routine release-verifier --tag v0.3.0-beta.2 --repo . --json
```

Evidence labels:

- `local`: source build, completion generation, formula syntax inspection, local
  tag inspection.
- `proxy`: GitHub Release metadata and asset names from `gh`.
- `failed`: local tag exists but the GitHub Release is missing or missing
  expected assets.
- `blocked`: unavailable Homebrew tap, GitHub Release API auth/network failures,
  or metadata that cannot be inspected.

Do not claim direct production, daemon, deployed runtime, payment, hardware, or
Codex App session-launch proof from this distribution package.
