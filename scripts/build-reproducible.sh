#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUTPUT="${1:-$ROOT_DIR/idena-indexer}"

if [[ "$OUTPUT" != /* ]]; then
  OUTPUT="$PWD/$OUTPUT"
fi
if [[ -L "$OUTPUT" ]]; then
  echo "Refusing symlinked output: $OUTPUT" >&2
  exit 1
fi

OUTPUT_DIR="$(dirname "$OUTPUT")"
mkdir -p "$OUTPUT_DIR"
if [[ ! -d "$OUTPUT_DIR" ]]; then
  echo "Output directory is unavailable: $OUTPUT_DIR" >&2
  exit 1
fi
TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/idena-indexer-build.XXXXXX")"
trap 'rm -rf "$TMP_DIR"' EXIT

cd "$ROOT_DIR"
GOTOOLCHAIN=go1.26.5 go mod verify
GOOS="$(GOTOOLCHAIN=go1.26.5 go env GOOS)"
case "$GOOS" in
  darwin) LINK_FLAGS="-buildid= -extldflags=-Wl,-no_uuid" ;;
  linux) LINK_FLAGS="-buildid= -extldflags=-Wl,--build-id=none" ;;
  *) LINK_FLAGS="-buildid=" ;;
esac
GOTOOLCHAIN=go1.26.5 go build \
  -trimpath \
  -buildvcs=false \
  -ldflags="$LINK_FLAGS" \
  -o "$TMP_DIR/idena-indexer" \
  .
install -m 0755 "$TMP_DIR/idena-indexer" "$OUTPUT"
