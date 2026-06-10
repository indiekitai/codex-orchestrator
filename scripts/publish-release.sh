#!/usr/bin/env sh
set -eu

TAG=${1:-}
if [ -z "$TAG" ]; then
  echo "usage: scripts/publish-release.sh TAG [DIST_DIR]" >&2
  exit 2
fi

DIST_DIR=${2:-"dist"}
REPO=${REPO:-indiekitai/codex-orchestrator}
ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
NOTES_FILE=${NOTES_FILE:-"$ROOT_DIR/docs/beta-release-notes-draft.md"}

if ! command -v gh >/dev/null 2>&1; then
  echo "gh is required to publish a GitHub Release" >&2
  exit 1
fi

PERMISSION_JSON=$(gh api "repos/$REPO" --jq '{push:.permissions.push,maintain:.permissions.maintain,admin:.permissions.admin}' 2>&1) || {
  echo "Could not inspect GitHub API permissions for $REPO:" >&2
  echo "$PERMISSION_JSON" >&2
  exit 1
}
case "$PERMISSION_JSON" in
  *'"push":true'*|*'"maintain":true'*|*'"admin":true'*)
    ;;
  *)
    echo "GitHub API permission for $REPO is insufficient: $PERMISSION_JSON" >&2
    echo "Release publishing requires API write, maintain, or admin permission." >&2
    echo "Run 'gh auth status' and authenticate as an account that can write releases for $REPO." >&2
    echo "This check is separate from git push: SSH may have write access even when gh has only read API access." >&2
    exit 1
    ;;
esac

if ! git -C "$ROOT_DIR" rev-parse "$TAG^{tag}" >/dev/null 2>&1; then
  echo "local annotated tag $TAG was not found" >&2
  exit 1
fi

if ! git -C "$ROOT_DIR" ls-remote --exit-code --tags origin "refs/tags/$TAG" >/dev/null 2>&1; then
  echo "remote tag origin/$TAG was not found; push the tag before publishing" >&2
  exit 1
fi

if [ ! -d "$DIST_DIR" ] || ! ls "$DIST_DIR"/codex-orchestrator_* >/dev/null 2>&1; then
  echo "release assets not found in $DIST_DIR; run scripts/build-release-assets.sh $TAG $DIST_DIR first" >&2
  exit 1
fi

if gh release view "$TAG" --repo "$REPO" >/dev/null 2>&1; then
  echo "release $TAG already exists; uploading/replacing assets"
  gh release upload "$TAG" "$DIST_DIR"/codex-orchestrator_* --repo "$REPO" --clobber
else
  echo "creating release $TAG"
  gh release create "$TAG" "$DIST_DIR"/codex-orchestrator_* \
    --repo "$REPO" \
    --title "$TAG" \
    --notes-file "$NOTES_FILE" \
    --prerelease
fi

gh release view "$TAG" --repo "$REPO" --json tagName,isDraft,isPrerelease,url,assets \
  --jq '{tagName,isDraft,isPrerelease,url,assetNames:[.assets[].name]}'
