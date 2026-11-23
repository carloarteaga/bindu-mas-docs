#!/usr/bin/env bash
# ==============================================================================
# ⏰ BINDU POTATO PROCESSING PIPELINE: DIGITAL TWIN AWAKENING CRON JOB SIMULATOR
# ==============================================================================
# Simulates the Go Core Min-Heap scheduler firing a timer and publishing
# SystemCronJob to bindu/in to awaken the digital twin after silo quarantine.
#
# Usage:
#   ./simulate_cron_job.sh [optional_lot_id] [optional_oil_temp_c] [optional_belt_speed_mm_s] [options]
#
# Examples:
#   ./simulate_cron_job.sh                                       # Uses default lot-potato-cold-001 (178.5 °C, 32 mm/s)
#   ./simulate_cron_job.sh lot-potato-cold-002 176.0 30.0        # Custom lot and setpoints
#   ./simulate_cron_job.sh --host 127.0.0.1 --port 38190 --auth console:StrongPass123
# ==============================================================================

set -e
export GOCACHE=/tmp/gocache

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../../.." && pwd)"

# E2E runtime directory resolution
CURRENT_ROOT_RUN_DIR="${REPO_ROOT}/.e2e-root/current"
BASE_DIR=""
if [ -d "${CURRENT_ROOT_RUN_DIR}" ]; then
  BASE_DIR="${CURRENT_ROOT_RUN_DIR}"
fi

REPORT_FILE=""
if [ -n "${BASE_DIR}" ]; then
  RUN_PARENT="$(readlink -f "${BASE_DIR}" | sed 's#/root$##')"
  if [ -f "${RUN_PARENT}/report.txt" ]; then
    REPORT_FILE="${RUN_PARENT}/report.txt"
  fi
fi

report_value() {
  local file="$1"
  local key="$2"
  if [[ -n "$file" && -f "$file" ]]; then
    awk -F= -v key="${key}" '$1 == key { sub(/^[^=]*=/, "", $0); value=$0 } END { print value }' "${file}"
  fi
}

# CLI Overrides
LOT_ID_CLI=""
OIL_TEMP_CLI=""
BELT_SPEED_CLI=""
ACTION_CLI=""
HOST_CLI=""
PORT_CLI=""
USER_CLI=""
PASS_CLI=""
AUTH_CLI=""

# Parse arguments
while [[ "$#" -gt 0 ]]; do
  case "$1" in
    -h|--help|help)
      echo "================================================================="
      echo "⏰ BINDU DIGITAL TWIN AWAKENING: CRON JOB SIMULATOR"
      echo "================================================================="
      echo "Usage: $0 [optional_lot_id] [optional_oil_temp_c] [optional_belt_speed_mm_s] [options]"
      echo ""
      echo "Examples:"
      echo "  $0                                       # Default lot-potato-cold-001 (178.5 °C, 32 mm/s)"
      echo "  $0 lot-potato-cold-002 176.0 30.0        # Custom lot and setpoints"
      echo ""
      echo "Options:"
      echo "  --host <ip>          MQTT Broker Host (default: auto-detected or 127.0.0.1)"
      echo "  --port <port>        MQTT Broker Port (default: auto-detected or 38190/1885)"
      echo "  --auth <user:pass>   MQTT Basic Auth credentials"
      echo "  --user <user>        MQTT Username (default: console)"
      echo "  --pass <pass>        MQTT Password (default: StrongPass123)"
      echo "  --lot-id <id>        Target Digital Twin Lot ID ($producer)"
      echo "  --oil-temp <temp>    Target fryer oil temperature setpoint (°C)"
      echo "  --belt-speed <speed> Target conveyor belt speed (mm/s)"
      echo "  --action <action>    Scheduler action tag (default: recondition_wakeup)"
      echo "  -h, --help           Show this help message"
      exit 0
      ;;
    --host) HOST_CLI="$2"; shift 2 ;;
    --port) PORT_CLI="$2"; shift 2 ;;
    --auth) AUTH_CLI="$2"; shift 2 ;;
    --user) USER_CLI="$2"; shift 2 ;;
    --pass) PASS_CLI="$2"; shift 2 ;;
    --lot-id) LOT_ID_CLI="$2"; shift 2 ;;
    --oil-temp) OIL_TEMP_CLI="$2"; shift 2 ;;
    --belt-speed) BELT_SPEED_CLI="$2"; shift 2 ;;
    --action) ACTION_CLI="$2"; shift 2 ;;
    *)
      if [[ -z "$LOT_ID_CLI" ]]; then
        LOT_ID_CLI="$1"
      elif [[ -z "$OIL_TEMP_CLI" ]]; then
        OIL_TEMP_CLI="$1"
      elif [[ -z "$BELT_SPEED_CLI" ]]; then
        BELT_SPEED_CLI="$1"
      fi
      shift
      ;;
  esac
done

LOT_ID="${LOT_ID_CLI:-lot-potato-cold-001}"
OIL_TEMP="${OIL_TEMP_CLI:-178.5}"
BELT_SPEED="${BELT_SPEED_CLI:-32.0}"
ACTION="${ACTION_CLI:-recondition_wakeup}"

# Resolve Credentials
USER="${USER_CLI}"
PASS="${PASS_CLI}"
if [[ -n "$AUTH_CLI" ]]; then
  USER="${AUTH_CLI%%:*}"
  PASS="${AUTH_CLI##*:}"
elif [[ -z "$USER" && -n "$REPORT_FILE" ]]; then
  USER="$(report_value "$REPORT_FILE" "console_user")"
  PASS="$(report_value "$REPORT_FILE" "console_password")"
  if [[ -z "$USER" ]]; then
    REP_LOGIN="$(report_value "$REPORT_FILE" "root_console_login")"
    if [[ -n "$REP_LOGIN" ]]; then
      USER="${REP_LOGIN%%:*}"
      PASS="${REP_LOGIN##*:}"
    fi
  fi
fi
if [[ -z "$USER" ]]; then
  USER="console"
  PASS="StrongPass123"
fi

# Resolve Host
HOST="${HOST_CLI}"
if [[ -z "$HOST" && -n "$REPORT_FILE" ]]; then
  REP_MQTT="$(report_value "$REPORT_FILE" "root_mqtt_integration")"
  if [[ -n "$REP_MQTT" ]]; then
    HOST="${REP_MQTT%%:*}"
  fi
fi
if [[ -z "$HOST" ]]; then
  HOST="127.0.0.1"
fi

# Resolve Port
PORT="${PORT_CLI}"
if [[ -z "$PORT" && -n "$REPORT_FILE" ]]; then
  REP_MQTT="$(report_value "$REPORT_FILE" "root_mqtt_integration")"
  if [[ -n "$REP_MQTT" ]]; then
    PORT="${REP_MQTT##*:}"
  fi
fi
if [[ -z "$PORT" && -n "$BASE_DIR" && -f "${BASE_DIR}/runtime/config/mesh.json" ]]; then
  PORT_VAL="$(jq -r '.. | objects | select(.bindu?.node_id == "root") | .bindu.channels.mqtt_integration.port' "${BASE_DIR}/runtime/config/mesh.json" 2>/dev/null || true)"
  if [[ "$PORT_VAL" != "null" && -n "$PORT_VAL" ]]; then
    PORT="$PORT_VAL"
  fi
fi
if [[ -z "$PORT" ]]; then
  PORT="38190"
fi

echo "================================================================="
echo "⏰ BINDU DIGITAL TWIN AWAKENING: CRON JOB SIMULATOR"
echo "================================================================="
echo "Target Broker:     ssl://${HOST}:${PORT}"
echo "Auth Principal:    ${USER}"
echo "Target Lot ID:     $LOT_ID"
echo "Target Fryer Temp: $OIL_TEMP °C (Modbus Reg 200 = $(awk -v t="$OIL_TEMP" 'BEGIN{printf "%d", t*10}'))"
echo "Target Belt Speed: $BELT_SPEED mm/s (Modbus Reg 201 = $(awk -v s="$BELT_SPEED" 'BEGIN{printf "%d", s}'))"
echo "Scheduler Action:  $ACTION"
echo "-----------------------------------------------------------------"

cd "$REPO_ROOT"
go run "$SCRIPT_DIR/simulate_cron_job.go" \
  -host "$HOST" \
  -port "$PORT" \
  -user "$USER" \
  -pass "$PASS" \
  -lot-id "$LOT_ID" \
  -oil-temp "$OIL_TEMP" \
  -belt-speed "$BELT_SPEED" \
  -action "$ACTION"
