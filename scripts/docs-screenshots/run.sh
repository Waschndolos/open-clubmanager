#!/usr/bin/env bash
# Regenerates all screenshots of the user documentation.
#
# Starts a throwaway backend and Vite dev server on their own ports, with their
# own config and database (your real ~/.config/open-clubmanager stays untouched),
# walks through the first-run setup with demo data and captures every page,
# once per documentation language (DOCS_LANGUAGES, default "de en").
#
#   scripts/docs-screenshots/run.sh        (or: npm run docs:screenshots)
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$HERE/../.." && pwd)"
API_PORT="${DOCS_API_PORT:-3101}"
UI_PORT="${DOCS_UI_PORT:-5273}"
LANGUAGES="${DOCS_LANGUAGES:-de en}"
SERVER_PID=""
VITE_PID=""
BIN_DIR="$(mktemp -d -t ocm-docs-bin-XXXXXX)"
DATA_DIR=""

cleanup() {
  kill $SERVER_PID $VITE_PID 2>/dev/null || true
  rm -rf "$BIN_DIR" "$DATA_DIR"
}
trap cleanup EXIT

wait_for() {
  for _ in $(seq 1 90); do
    curl -sf "$1" >/dev/null && return 0
    sleep 2
  done
  echo "Timed out waiting for $1" >&2
  return 1
}

(cd "$HERE" && [ -d node_modules ] || npm install --no-audit --no-fund)

# Build once; every language gets a fresh database and walks through the setup.
(cd "$ROOT/server" && go build -o "$BIN_DIR/open-clubmanager-docs-server" ./cmd/server)

echo "Starting frontend on :$UI_PORT …"
(cd "$ROOT/frontend" && VITE_BACKEND_ORIGIN="http://localhost:$API_PORT" \
  exec node_modules/.bin/vite --port "$UI_PORT" --strictPort >"$BIN_DIR/vite.log" 2>&1) &
VITE_PID=$!
wait_for "http://localhost:$UI_PORT/"

for LANG_CODE in $LANGUAGES; do
  DATA_DIR="$(mktemp -d -t ocm-docs-data-XXXXXX)"
  echo "[$LANG_CODE] Starting backend on :$API_PORT …"
  SERVER_CONFIG_FILE="$DATA_DIR/config.json" DATABASE_PATH="$DATA_DIR/clubmanager.db" PORT="$API_PORT" \
    "$BIN_DIR/open-clubmanager-docs-server" >"$DATA_DIR/server.log" 2>&1 &
  SERVER_PID=$!
  wait_for "http://localhost:$API_PORT/api/v2/system/health"

  (cd "$HERE" && DOCS_LANG="$LANG_CODE" \
    DOCS_BASE_URL="http://localhost:$UI_PORT" \
    DOCS_API_URL="http://localhost:$API_PORT/api/v2" \
    DOCS_DB_PATH="$DATA_DIR/clubmanager.db" \
    node capture.mjs)

  kill "$SERVER_PID"
  wait "$SERVER_PID" 2>/dev/null || true
  SERVER_PID=""
  rm -rf "$DATA_DIR"
done
