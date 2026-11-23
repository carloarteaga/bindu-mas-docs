#!/usr/bin/env bash
# Helper script to run the Bindu BLE iBeacon Radio Simulator on Root
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ACTION="${1:-interactive}"
BACKEND="${2:-ble}"

echo "================================================================="
echo "📡 STARTING BINDU BLE IBEACON RADIO SIMULATOR"
echo "   Action:  ${ACTION}"
echo "   Backend: ${BACKEND} (use 'dry-run' if running on a host without BLE)"
echo "================================================================="

cd "$SCRIPT_DIR"

# Check if bluetoothctl / hciconfig exists or if running under Linux
if [ "$BACKEND" = "ble" ]; then
    if ! command -v hciconfig >/dev/null 2>&1 && ! command -v bluetoothctl >/dev/null 2>&1; then
        echo "⚠️  Warning: Bluetooth tools not detected in PATH."
        echo "   Falling back to --backend dry-run if physical radio is unavailable."
    fi
fi

go run simulate_ble_beacons.go --action "$ACTION" --backend "$BACKEND" "${@:3}"
