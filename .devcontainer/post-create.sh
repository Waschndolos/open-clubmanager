#!/usr/bin/env bash
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

cd "${PROJECT_ROOT}"

echo "[devcontainer] Installing root dependencies..."
npm ci

echo "[devcontainer] Installing frontend dependencies..."
npm --prefix frontend ci

echo "[devcontainer] Installing Prisma/DB dependencies..."
npm --prefix db ci

echo "[devcontainer] Downloading Go modules..."
(
  cd server
  go mod download
)

echo "[devcontainer] Fetching Rust crates..."
(
  cd src-tauri
  cargo fetch
)

echo "[devcontainer] Running quick validations..."
npm run prisma:validate
npm run tauri:check

echo

echo "[devcontainer] Done. If no local env files exist yet, create them following the instructions in README.md."

