#!/usr/bin/env bash
# ==============================================================================
# BINDU AI-NATIVE TUTORIAL: Edge CV Multi-Camera Presence
# Dynamic Video Capture & Inference Control Simulator (SetCaptureStatus.sh)
#
# Publishes EdgeCVSetStatus message directly to Bindu MQTT ingress (bindu/in)
# to dynamically start or pause camera capture loops and YOLO ONNX inference.
# ==============================================================================

set -e

# Terminal Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
YELLOW='\033[1;33m'
BOLD='\033[1m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/../../../.." && pwd)"

# E2E current run directory resolution
CURRENT_ROOT_RUN_DIR="${ROOT_DIR}/.e2e-root/current"
RUN_DIR=""
if [ -d "${CURRENT_ROOT_RUN_DIR}" ]; then
  RUN_DIR="$(readlink -f "${CURRENT_ROOT_RUN_DIR}" | sed 's#/root$##')"
fi
REPORT_FILE="${RUN_DIR}/report.txt"

# Helper to read key=value from report.txt
report_value() {
  local file="$1"
  local key="$2"
  if [[ -f "$file" ]]; then
    awk -F= -v key="${key}" '$1 == key { sub(/^[^=]*=/, "", $0); value=$0 } END { print value }' "${file}"
  fi
}

MQTT_HOST_CLI=""
MQTT_PORT_CLI=""
MQTT_USER="console"
MQTT_PASS="StrongPass123"
AUTH_VAL=""
TARGET_NODE="node-dell-1"
CAPTURE_VAL=""
REASON_VAL=""

# Show usage help
show_help() {
  echo -e "${BOLD}Usage:${NC} $0 <true|false|on|off|start|stop> [options]"
  echo -e "       $0 --capture <true|false> [--node <node_id>] [options]"
  echo ""
  echo "Publishes an EdgeCVSetStatus message via MQTT to dynamically control video capture & inference."
  echo ""
  echo "Positional Arguments / Values:"
  echo "  true | on | start | enable    Start/resume camera frame capture & ONNX inference"
  echo "  false | off | stop | pause    Pause camera capture, close handles & save CPU"
  echo ""
  echo "Options:"
  echo "  --capture <true|false>        Set capture state explicitly"
  echo "  --node <node_id>              Target gateway node (default: root)"
  echo "  --host <ip>                   MQTT broker host (default: from report.txt or 127.0.0.1)"
  echo "  --port <port>                 MQTT broker port (default: from report.txt or 1883)"
  echo "  --auth <user:pass>            Basic auth credentials"
  echo "  --reason <string>             Optional reason for state transition"
  echo "  -h, --help                    Show this help message"
  echo ""
  echo "Examples:"
  echo "  $0 true                       # Enable capture on root node"
  echo "  $0 false                      # Pause capture on root node"
  echo "  $0 start --node node-dell-1   # Enable capture on node-dell-1"
  echo "  $0 stop --node node-dell-1    # Pause capture on node-dell-1"
  echo "  $0 --capture false --auth console:StrongPass123"
}

# Check first positional argument if provided
if [[ "$#" -gt 0 && ! "$1" =~ ^- ]]; then
  case "$1" in
    true|1|on|ON|start|START|enable|ENABLE|resume|RESUME)
      CAPTURE_VAL="true"
      shift
      ;;
    false|0|off|OFF|stop|STOP|disable|DISABLE|pause|PAUSE)
      CAPTURE_VAL="false"
      shift
      ;;
    help|-h|--help)
      show_help
      exit 0
      ;;
    *)
      echo -e "${RED}Error: Invalid capture parameter '$1'. Use 'true', 'false', 'start', or 'stop'.${NC}" >&2
      show_help
      exit 1
      ;;
  esac
fi

# Parse remaining CLI options
while [[ "$#" -gt 0 ]]; do
  case "$1" in
    -h|--help|help)
      show_help
      exit 0
      ;;
    --capture)
      case "$2" in
        true|1|on|ON|start|START|enable|ENABLE) CAPTURE_VAL="true" ;;
        false|0|off|OFF|stop|STOP|disable|DISABLE|pause|PAUSE) CAPTURE_VAL="false" ;;
        *)
          echo -e "${RED}Error: Invalid --capture argument '$2'. Must be true or false.${NC}" >&2
          exit 1
          ;;
      esac
      shift 2
      ;;
    --node)
      TARGET_NODE="$2"
      shift 2
      ;;
    --host)
      MQTT_HOST_CLI="$2"
      shift 2
      ;;
    --port)
      MQTT_PORT_CLI="$2"
      shift 2
      ;;
    --auth)
      AUTH_VAL="$2"
      MQTT_USER="${AUTH_VAL%%:*}"
      MQTT_PASS="${AUTH_VAL##*:}"
      shift 2
      ;;
    --reason)
      REASON_VAL="$2"
      shift 2
      ;;
    true|1|on|ON|start|START|enable|ENABLE)
      CAPTURE_VAL="true"
      shift
      ;;
    false|0|off|OFF|stop|STOP|disable|DISABLE|pause|PAUSE)
      CAPTURE_VAL="false"
      shift
      ;;
    *)
      echo -e "${RED}Unknown parameter: $1${NC}" >&2
      show_help
      exit 1
      ;;
  esac
done

# If capture parameter is still missing, default to true or prompt
if [[ -z "$CAPTURE_VAL" ]]; then
  echo -e "${YELLOW}No capture parameter provided. Defaulting to true (ACTIVE CAPTURE).${NC}"
  CAPTURE_VAL="true"
fi

# Resolve default credentials from report.txt if available
LOGIN_STR="$(report_value "${REPORT_FILE}" root_console_login)"
if [[ -n "$LOGIN_STR" && -z "$AUTH_VAL" ]]; then
  MQTT_USER="${LOGIN_STR%%:*}"
  MQTT_PASS="${LOGIN_STR##*:}"
fi

# Resolve target MQTT Host and Port
RESOLVED_HOST="$MQTT_HOST_CLI"
RESOLVED_PORT="$MQTT_PORT_CLI"

if [[ -z "$RESOLVED_HOST" || -z "$RESOLVED_PORT" ]]; then
  if [[ -f "${REPORT_FILE}" ]]; then
    PORT_VAL="$(report_value "${REPORT_FILE}" "root_mqtt_integration")"
    if [[ -n "${PORT_VAL}" ]]; then
      [[ -z "$RESOLVED_HOST" ]] && RESOLVED_HOST="${PORT_VAL%%:*}"
      [[ -z "$RESOLVED_PORT" ]] && RESOLVED_PORT="${PORT_VAL##*:}"
    fi
  fi
fi

# Fallback default values
[[ -z "$RESOLVED_HOST" ]] && RESOLVED_HOST="127.0.0.1"
[[ -z "$RESOLVED_PORT" ]] && RESOLVED_PORT="1883"

PUBLISHER_GO="${SCRIPT_DIR}/publish_capture_status.go"
if [[ ! -f "$PUBLISHER_GO" ]]; then
  echo -e "${RED}Error: Go publisher script not found at ${PUBLISHER_GO}${NC}" >&2
  exit 1
fi

GOCACHE=/tmp/gocache go run "$PUBLISHER_GO" \
  -host "$RESOLVED_HOST" \
  -port "$RESOLVED_PORT" \
  -user "$MQTT_USER" \
  -pass "$MQTT_PASS" \
  -node "$TARGET_NODE" \
  -capture="$CAPTURE_VAL" \
  -reason="$REASON_VAL"
