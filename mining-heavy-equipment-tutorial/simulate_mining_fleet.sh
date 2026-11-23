#!/usr/bin/env bash
# Convenience runner for Mining Fleet Simulator
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../../.." && pwd)"

BASE_DIR="${BASE_DIR:-}"
if [ -z "${BASE_DIR}" ]; then
  if [ -d "${REPO_ROOT}/.e2e-root/current" ]; then
    BASE_DIR="${REPO_ROOT}/.e2e-root/current"
  fi
fi

EXTRA_ARGS=()
if [ -n "${BASE_DIR}" ]; then
  EXTRA_ARGS+=("--base-dir" "${BASE_DIR}")
fi

cd "$REPO_ROOT"
go run "$SCRIPT_DIR/simulate_mining_fleet.go" "${EXTRA_ARGS[@]}" "$@"
