#!/usr/bin/env bash
# Runner for the Amazon Suzano Airborne Precision Forestry & Fly-By Carbon Telemetry Simulator
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../../.." && pwd)"

cd "$REPO_ROOT"
go run "$SCRIPT_DIR/simulate_amazon_flyby.go" "$@"
