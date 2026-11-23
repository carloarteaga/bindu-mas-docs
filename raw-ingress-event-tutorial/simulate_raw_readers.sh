#!/usr/bin/env bash
# ==============================================================================
# 📡 Bindu Heterogeneous Multi-Reader Raw Ingress Simulator Runner
#
# Streams non-semantic raw RFID payloads to MQTT topic bindu/in/raw/...
# Supports both positional arguments and CLI flags with auto-detection of
# running cluster endpoints from .e2e-root/current or report.txt.
#
# Usage:
#   ./simulate_raw_readers.sh [reader] [count] [interval] [broker]
#   ./simulate_raw_readers.sh --reader hopeland --count 10 --interval 200ms
# ==============================================================================
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../../.." && pwd)"

READER=""
COUNT=""
INTERVAL=""
BROKER=""
USER=""
PASS=""
BASE_DIR_CLI=""
POSITIONAL=()

while [[ "$#" -gt 0 ]]; do
  case "$1" in
    -h|--help|help)
      echo "Usage: $0 [options] OR $0 [reader] [count] [interval] [broker]"
      echo ""
      echo "Streams non-semantic raw RFID payloads to MQTT topic bindu/in/raw/..."
      echo ""
      echo "Positional Arguments (optional):"
      echo "  1) Reader model (default: all; options: all, hopeland, zebra, chafon, marktrace)"
      echo "  2) Readings count (default: 20; 0 for continuous)"
      echo "  3) Interval (default: 500ms)"
      echo "  4) Broker URL (default: auto-detected from cluster or ssl://127.0.0.1:1885)"
      echo ""
      echo "Options / Flags:"
      echo "  -r, --reader <model>   Reader model (all, hopeland, zebra, chafon, marktrace)"
      echo "  -c, --count <num>      Number of readings to publish (0 for continuous)"
      echo "  -i, --interval <dur>   Interval between readings (e.g. 100ms, 500ms, 1s)"
      echo "  -b, --broker <url>     MQTT broker URL (e.g. ssl://127.0.0.1:38190)"
      echo "  -u, --user <username>  MQTT username (default: console)"
      echo "  -p, --pass <password>  MQTT password (default: StrongPass123)"
      echo "  --base-dir <dir>       Bindu base directory for configuration lookup"
      echo "  -h, --help             Show this help message"
      exit 0
      ;;
    -r|--reader)   READER="$2"; shift 2 ;;
    -c|--count)    COUNT="$2"; shift 2 ;;
    -i|--interval) INTERVAL="$2"; shift 2 ;;
    -b|--broker)   BROKER="$2"; shift 2 ;;
    -u|--user)     USER="$2"; shift 2 ;;
    -p|--pass)     PASS="$2"; shift 2 ;;
    --base-dir)    BASE_DIR_CLI="$2"; shift 2 ;;
    *)
      POSITIONAL+=("$1")
      shift
      ;;
  esac
done

# Map positional arguments if flags were not specified
if [ -z "$READER" ] && [ ${#POSITIONAL[@]} -ge 1 ]; then
  READER="${POSITIONAL[0]}"
fi
if [ -z "$COUNT" ] && [ ${#POSITIONAL[@]} -ge 2 ]; then
  COUNT="${POSITIONAL[1]}"
fi
if [ -z "$INTERVAL" ] && [ ${#POSITIONAL[@]} -ge 3 ]; then
  INTERVAL="${POSITIONAL[2]}"
fi
if [ -z "$BROKER" ] && [ ${#POSITIONAL[@]} -ge 4 ]; then
  BROKER="${POSITIONAL[3]}"
fi
if [ -z "$USER" ] && [ ${#POSITIONAL[@]} -ge 5 ]; then
  USER="${POSITIONAL[4]}"
fi
if [ -z "$PASS" ] && [ ${#POSITIONAL[@]} -ge 6 ]; then
  PASS="${POSITIONAL[5]}"
fi

# Auto-detect Base Dir
BASE_DIR="${BASE_DIR_CLI:-${BASE_DIR:-}}"
if [ -z "${BASE_DIR}" ]; then
  if [ -d "${REPO_ROOT}/.e2e-root/current" ]; then
    BASE_DIR="${REPO_ROOT}/.e2e-root/current"
  elif [ -d "${REPO_ROOT}/.e2e-runs" ]; then
    LATEST_RUN="$(ls -td "${REPO_ROOT}/.e2e-runs"/* 2>/dev/null | head -n1)"
    if [ -n "$LATEST_RUN" ] && [ -d "$LATEST_RUN/root" ]; then
      BASE_DIR="$LATEST_RUN/root"
    fi
  fi
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

DEFAULT_BROKER=""
DEFAULT_USER="console"
DEFAULT_PASS="StrongPass123"

if [ -n "${REPORT_FILE}" ]; then
  REP_MQTT_INTEGRATION="$(report_value "${REPORT_FILE}" "root_mqtt_integration")"
  if [ -n "${REP_MQTT_INTEGRATION}" ]; then
    MQTT_PORT="${REP_MQTT_INTEGRATION##*:}"
    DEFAULT_BROKER="ssl://127.0.0.1:${MQTT_PORT}"
  fi
  REP_CONSOLE_LOGIN="$(report_value "${REPORT_FILE}" "root_console_login")"
  if [ -n "${REP_CONSOLE_LOGIN}" ]; then
    DEFAULT_USER="${REP_CONSOLE_LOGIN%%:*}"
    DEFAULT_PASS="${REP_CONSOLE_LOGIN#*:}"
  fi
fi

if [ -z "${DEFAULT_BROKER}" ]; then
  DEFAULT_BROKER="ssl://127.0.0.1:1885"
fi

# Apply defaults
READER="${READER:-all}"
COUNT="${COUNT:-20}"
INTERVAL="${INTERVAL:-500ms}"
BROKER="${BROKER:-$DEFAULT_BROKER}"
USER="${USER:-$DEFAULT_USER}"
PASS="${PASS:-$DEFAULT_PASS}"

echo "================================================================="
echo "📡 BINDU RAW INGRESS MULTI-READER SIMULATOR"
echo "================================================================="
echo "Reader Mode: ${READER} (options: all, hopeland, zebra, chafon, marktrace)"
echo "Readings:    ${COUNT} (0 for continuous)"
echo "Interval:    ${INTERVAL}"
echo "Broker:      ${BROKER}"
echo "Auth User:   ${USER}"
echo "================================================================="

cd "$SCRIPT_DIR"
GOCACHE=/tmp/gocache go run simulate_raw_readers.go \
    --reader "$READER" \
    --count "$COUNT" \
    --interval "$INTERVAL" \
    --broker "$BROKER" \
    --user "$USER" \
    --pass "$PASS"
