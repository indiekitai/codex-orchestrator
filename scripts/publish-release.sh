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
PRERELEASE=false
case "$TAG" in
  *-alpha*|*-beta*|*-rc*)
    PRERELEASE=true
    ;;
esac

gh_api_retry() {
  ATTEMPT=1
  while :; do
    if gh api "$@"; then
      return 0
    fi
    if [ "$ATTEMPT" -ge 3 ]; then
      return 1
    fi
    echo "gh api failed; retrying attempt $((ATTEMPT + 1))/3" >&2
    sleep "$ATTEMPT"
    ATTEMPT=$((ATTEMPT + 1))
  done
}

if ! command -v gh >/dev/null 2>&1; then
  echo "gh is required to publish a GitHub Release" >&2
  exit 1
fi

PERMISSION_JSON=$(gh_api_retry "repos/$REPO" --jq '{push:.permissions.push,maintain:.permissions.maintain,admin:.permissions.admin}' 2>&1) || {
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

if RELEASE_ID=$(gh_api_retry "repos/$REPO/releases/tags/$TAG" --jq '.id' 2>/dev/null); then
  :
else
  RELEASE_ID=""
fi
if [ -n "$RELEASE_ID" ]; then
  echo "release $TAG already exists; replacing matching assets"
else
  if [ ! -f "$NOTES_FILE" ]; then
    echo "release notes file not found: $NOTES_FILE" >&2
    exit 1
  fi

  echo "creating release $TAG"
  RELEASE_BODY=$(cat "$NOTES_FILE")
  RELEASE_ID=$(gh_api_retry -X POST "repos/$REPO/releases" \
    -f tag_name="$TAG" \
    -f name="$TAG" \
    -f body="$RELEASE_BODY" \
    -F draft=false \
    -F prerelease="$PRERELEASE" \
    --jq '.id')
fi

for ASSET in "$DIST_DIR"/codex-orchestrator_*; do
  ASSET_NAME=$(basename "$ASSET")
  EXISTING_IDS=$(gh_api_retry "repos/$REPO/releases/$RELEASE_ID/assets" \
    --jq ".[] | select(.name == \"$ASSET_NAME\") | .id")
  for ASSET_ID in $EXISTING_IDS; do
    echo "deleting existing asset $ASSET_NAME ($ASSET_ID)"
    gh_api_retry -X DELETE "repos/$REPO/releases/assets/$ASSET_ID" >/dev/null
  done

  echo "uploading $ASSET_NAME"
  gh_api_retry -X POST \
    -H "Content-Type: application/octet-stream" \
    --input "$ASSET" \
    "https://uploads.github.com/repos/$REPO/releases/$RELEASE_ID/assets?name=$ASSET_NAME" \
    --jq '{name,state,size}'
done

gh release view "$TAG" --repo "$REPO" --json tagName,isDraft,isPrerelease,url,assets \
  --jq '{tagName,isDraft,isPrerelease,url,assetNames:[.assets[].name]}'
