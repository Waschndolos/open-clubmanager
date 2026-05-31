#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DIST_DIR="$ROOT_DIR/dist"
mkdir -p "$DIST_DIR"

build_target() {
  local goos="$1"
  local goarch="$2"
  local ext=""
  if [[ "$goos" == "windows" ]]; then
    ext=".exe"
  fi

  local output="$DIST_DIR/open-clubmanager-server-${goos}-${goarch}${ext}"
  GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o "$output" ./cmd/server
}

mode="${1:-all}"
if [[ "$mode" == "local" ]]; then
  CGO_ENABLED=0 go build -o "$DIST_DIR/open-clubmanager-server" ./cmd/server
  exit 0
fi

build_target linux amd64
build_target windows amd64
build_target darwin amd64
build_target darwin arm64
