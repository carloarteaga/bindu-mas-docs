#!/usr/bin/env bash
# Helper script to compile and run the LLRP Mock Reader & 4-Antenna Dock Portal Simulator
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PORT="${1:-5084}"
HOST="${2:-0.0.0.0}"

echo "================================================================="
echo "📡 STARTING BINDU RFID DOCK PORTAL SIMULATOR (LLRP Port: ${PORT})"
echo "================================================================="

cd "$SCRIPT_DIR"
go run simulate_dock_portal.go --port "$PORT" --host "$HOST"
