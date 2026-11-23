#!/usr/bin/env bash
# public/console/docs/mas-complete-lifecycle-tutorial/simulate_trigger.sh
# Emits a MASclTrigger message to test all 4 agent paradigms across the mesh.

set -e
export GOCACHE=/tmp/gocache

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/../../../.." && pwd)"

CURRENT_ROOT_RUN_DIR="${ROOT_DIR}/.e2e-root/current"
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

HOST_CLI=""
PORT_CLI=""
USER_CLI=""
PASS_CLI=""
SOURCE_CLI="operator-manual-trigger"
NOTES_CLI="Verification trigger for MAS complete lifecycle"

while [[ "$#" -gt 0 ]]; do
  case "$1" in
    --host) HOST_CLI="$2"; shift 2 ;;
    --port) PORT_CLI="$2"; shift 2 ;;
    --user) USER_CLI="$2"; shift 2 ;;
    --pass) PASS_CLI="$2"; shift 2 ;;
    --source) SOURCE_CLI="$2"; shift 2 ;;
    --notes) NOTES_CLI="$2"; shift 2 ;;
    -h|--help)
      echo "Usage: $0 [--host <host>] [--port <port>] [--user <user>] [--pass <pass>] [--source <source>] [--notes <notes>]"
      exit 0
      ;;
    *)
      echo "Unknown argument: $1"
      exit 1
      ;;
  esac
done

USER="${USER_CLI}"
PASS="${PASS_CLI}"
if [ -z "${USER}" ] && [ -n "${REPORT_FILE}" ]; then
  USER="$(report_value "${REPORT_FILE}" "console_user")"
  PASS="$(report_value "${REPORT_FILE}" "console_password")"
fi
if [ -z "${USER}" ]; then
  USER="console"
  PASS="StrongPass123"
fi

HOST="${HOST_CLI}"
if [ -z "${HOST}" ]; then
  HOST="127.0.0.1"
fi

PORT="${PORT_CLI}"
if [ -z "${PORT}" ] && [ -n "${BASE_DIR}" ] && [ -f "${BASE_DIR}/runtime/config/mesh.json" ]; then
  ROOT_PORT=$(grep -A 2 '"mqtt_integration":' "${BASE_DIR}/runtime/config/mesh.json" | grep '"port":' | tail -n 1 | tr -dc '0-9')
  if [ -n "${ROOT_PORT}" ]; then
    PORT="${ROOT_PORT}"
  fi
fi
if [ -z "${PORT}" ] && [ -n "${REPORT_FILE}" ]; then
  PORT="$(report_value "${REPORT_FILE}" "root_tls_port")"
fi
if [ -z "${PORT}" ]; then
  PORT="8883"
fi

cd "${ROOT_DIR}"
go run "${SCRIPT_DIR}/publish_trigger.go" \
  --host "${HOST}" \
  --port "${PORT}" \
  --user "${USER}" \
  --pass "${PASS}" \
  --source "${SOURCE_CLI}" \
  --notes "${NOTES_CLI}"
