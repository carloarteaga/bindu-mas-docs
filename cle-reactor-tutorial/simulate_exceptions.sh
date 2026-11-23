#!/usr/bin/env bash
# public/console/docs/cle-reactor-tutorial/simulate_exceptions.sh
# Simulates telemetry exceptions by publishing MessageState and SystemVersion metadata to MQTT.

set -e

# Base directory lookup (4 levels up from this script's folder)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/../../../.." && pwd)"

# E2E current run directory resolution
CURRENT_ROOT_RUN_DIR="${ROOT_DIR}/.e2e-root/current"
RUN_DIR=""
if [ -d "${CURRENT_ROOT_RUN_DIR}" ]; then
  RUN_DIR="$(readlink -f "${CURRENT_ROOT_RUN_DIR}" | sed 's#/root$##')"
fi
REPORT_FILE="${RUN_DIR}/report.txt"

# Helper to read from report.txt
report_value() {
  local file="$1"
  local key="$2"
  if [[ -f "$file" ]]; then
    awk -F= -v key="${key}" '$1 == key { sub(/^[^=]*=/, "", $0); value=$0 } END { print value }' "${file}"
  fi
}

# Default Target parameters
MQTT_HOST_CLI=""
MQTT_PORT_CLI=""
MQTT_USER="console"
MQTT_PASS="StrongPass123"

# Parse CLI parameters
while [[ "$#" -gt 0 ]]; do
  case "$1" in
    -h|--help|help)
      echo "Usage: $0 [options]"
      echo "Options:"
      echo "  --host <ip>         MQTT broker host"
      echo "  --port <port>       MQTT broker port"
      echo "  --auth <u:p>        MQTT credentials (user:password)"
      echo "  -h, --help          Show this help message"
      exit 0
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
    *)
      echo "Unknown parameter: $1" >&2
      exit 1
      ;;
  esac
done

# Resolve default credentials from report.txt if available
LOGIN_STR="$(report_value "${REPORT_FILE}" root_console_login)"
if [[ -n "$LOGIN_STR" && -z "$AUTH_VAL" ]]; then
  MQTT_USER="${LOGIN_STR%%:*}"
  MQTT_PASS="${LOGIN_STR##*:}"
fi

resolve_node_host_port() {
  local target_node="$1"
  local resolved_host="$MQTT_HOST_CLI"
  local resolved_port="$MQTT_PORT_CLI"

  # Resolve Host if not CLI-specified
  if [[ -z "$resolved_host" ]]; then
    local inventory_file="${ROOT_DIR}/.e2e-ssh/nodes.tsv"
    if [[ -f "${inventory_file}" ]]; then
      local inv_host
      inv_host="$(awk -F'\t' -v node="${target_node}" '$1 == node {print $2}' "${inventory_file}")"
      if [[ -n "$inv_host" ]]; then
        resolved_host="$inv_host"
      fi
    fi
    if [[ -z "$resolved_host" ]]; then
      if [[ "$target_node" == "bindu-minis" ]]; then
        resolved_host="192.168.12.160"
      elif [[ "$target_node" == "node-dell-1" || "$target_node" == "dell-debian1" ]]; then
        resolved_host="192.168.12.185"
      elif [[ "$target_node" == "bindu-pi3-uno" ]]; then
        resolved_host="192.168.12.108"
      else
        resolved_host="127.0.0.1"
      fi
    fi
  fi

  # Resolve Port if not CLI-specified
  if [[ -z "$resolved_port" ]]; then
    local mesh_file="${CURRENT_ROOT_RUN_DIR}/runtime/config/mesh.json"
    if [[ -f "$mesh_file" ]]; then
      local port_val
      port_val="$(jq -r --arg node "${target_node}" '.. | objects | select(.bindu?.node_id == $node) | .bindu.channels.mqtt_integration.port' "$mesh_file" 2>/dev/null || true)"
      if [[ "$port_val" != "null" && -n "$port_val" ]]; then
        resolved_port="$port_val"
      fi
    fi
    if [[ -z "$resolved_port" ]]; then
      resolved_port="1883"
    fi
  fi

  echo "${resolved_host}:${resolved_port}"
}

PUBLISH_GO="${SCRIPT_DIR}/publish_telemetry_exceptions.go"

if [ ! -f "$PUBLISH_GO" ]; then
  echo "Error: Publisher Go script not found at ${PUBLISH_GO}" >&2
  exit 1
fi

TARGET_NODES=("root" "node-1" "bindu-minis")

if [[ -n "$MQTT_HOST_CLI" || -n "$MQTT_PORT_CLI" ]]; then
  HOST_VAL="$MQTT_HOST_CLI"
  [[ -z "$HOST_VAL" ]] && HOST_VAL="127.0.0.1"
  PORT_VAL="$MQTT_PORT_CLI"
  [[ -z "$PORT_VAL" ]] && PORT_VAL="1883"

  echo "🚀 Invoking Watchdog Exception Publisher for CLI Target: ${HOST_VAL}:${PORT_VAL}..."
  GOCACHE=/tmp/gocache go run "$PUBLISH_GO" \
    -host "$HOST_VAL" \
    -port "$PORT_VAL" \
    -user "$MQTT_USER" \
    -pass "$MQTT_PASS"
else
  # Target the active mesh nodes
  for node in "${TARGET_NODES[@]}"; do
    HP_VAL="$(resolve_node_host_port "$node")"
    HOST_VAL="${HP_VAL%%:*}"
    PORT_VAL="${HP_VAL##*:}"

    echo "🚀 Invoking Watchdog Exception Publisher for Node: ${node} (${HOST_VAL}:${PORT_VAL})..."
    GOCACHE=/tmp/gocache go run "$PUBLISH_GO" \
      -host "$HOST_VAL" \
      -port "$PORT_VAL" \
      -user "$MQTT_USER" \
      -pass "$MQTT_PASS"
    echo ""
  done
fi
