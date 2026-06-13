#!/usr/bin/env sh
set -eu

TAG=${1:-}
if [ -z "$TAG" ]; then
  echo "usage: scripts/build-release-assets.sh TAG [DIST_DIR]" >&2
  exit 2
fi

DIST_DIR=${2:-"dist"}
ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)

mkdir -p "$DIST_DIR"
rm -f "$DIST_DIR"/codex-orchestrator_*

cd "$ROOT_DIR"

for target in darwin/arm64 darwin/amd64 linux/amd64 linux/arm64 windows/amd64; do
  GOOS_VALUE=${target%/*}
  GOARCH_VALUE=${target#*/}
  EXT=""
  if [ "$GOOS_VALUE" = "windows" ]; then
    EXT=".exe"
  fi

  OUT="codex-orchestrator_${GOOS_VALUE}_${GOARCH_VALUE}${EXT}"
  echo "building $OUT for $TAG"
  GOOS=$GOOS_VALUE GOARCH=$GOARCH_VALUE CGO_ENABLED=0 \
    go build -trimpath -ldflags="-s -w -X main.helperVersion=$TAG" -o "$DIST_DIR/$OUT" ./cmd/codex-orchestrator

  (
    cd "$DIST_DIR"
    if [ "$GOOS_VALUE" = "windows" ]; then
      zip -q "$OUT.zip" "$OUT"
    else
      tar -czf "$OUT.tar.gz" "$OUT"
    fi
  )
done

echo
echo "built release assets in $DIST_DIR"
shasum -a 256 "$DIST_DIR"/codex-orchestrator_* | sort
