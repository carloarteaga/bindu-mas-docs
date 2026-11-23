#!/usr/bin/env bash
# public/console/docs/media-storage-tutorial/simulate_patio_media.sh
# Simulates patio camera media events by publishing PatioMonitorMedia metadata directly to MQTT.

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
PROVIDER="minio_local"
BUCKET="raw"

# Options:
#   --host <ip>         MQTT broker host (default: 192.168.12.160)
#   --port <port>       MQTT broker port (default: 1885)
#   --auth <u:p>        MQTT credentials (user:password)
#   --s3-endpoint <url> MinIO S3 endpoint URL (default: http://192.168.12.160:9000)
#   --access-key <key>  MinIO Access key (default: bindu-core-agent)
#   --secret-key <sec>  MinIO Secret key (default: bindu-media-123)
#   --provider <id>     Storage provider ID (default: minio_local)
#   --bucket <name>     Storage bucket name (default: raw)
#   --src-dir <path>    Source JPEG directory (default: script-relative pictures/ directory)
#   -h, --help          Show this help message

S3_ENDPOINT="http://192.168.12.160:9000"
ACCESS_KEY="bindu-core-agent"
SECRET_KEY="bindu-media-123"
SRC_DIR="${SCRIPT_DIR}/pictures"

# Parse CLI parameters
while [[ "$#" -gt 0 ]]; do
  case "$1" in
    -h|--help|help)
      echo "Usage: $0 [options]"
      echo "Options:"
      echo "  --host <ip>         MQTT broker host (default: 192.168.12.160)"
      echo "  --port <port>       MQTT broker port (default: 1885)"
      echo "  --auth <u:p>        MQTT credentials (user:password)"
      echo "  --s3-endpoint <url> MinIO S3 endpoint URL (default: http://192.168.12.160:9000)"
      echo "  --access-key <key>  MinIO Access Key (default: bindu-core-agent)"
      echo "  --secret-key <sec>  MinIO Secret Key (default: bindu-media-123)"
      echo "  --provider <id>     Storage provider ID (default: minio_local)"
      echo "  --bucket <name>     Storage bucket name (default: raw)"
      echo "  --src-dir <path>    Source JPEG directory (default: script-relative pictures/ directory)"
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
    --s3-endpoint)
      S3_ENDPOINT="$2"
      shift 2
      ;;
    --access-key)
      ACCESS_KEY="$2"
      shift 2
      ;;
    --secret-key)
      SECRET_KEY="$2"
      shift 2
      ;;
    --provider)
      PROVIDER="$2"
      shift 2
      ;;
    --bucket)
      BUCKET="$2"
      shift 2
      ;;
    --src-dir)
      SRC_DIR="$2"
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
      if [[ "${target_node}" == "bindu-minis" ]]; then
        resolved_host="192.168.12.160"
      elif [[ "${target_node}" == "node-dell-1" || "${target_node}" == "dell-debian1" ]]; then
        resolved_host="192.168.12.185"
      elif [[ "${target_node}" == "bindu-pi3-uno" ]]; then
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
      if [[ "${target_node}" == "bindu-minis" ]]; then
        resolved_port="1885"
      else
        resolved_port="1883"
      fi
    fi
  fi

  echo "${resolved_host}:${resolved_port}"
}

PUBLISH_GO="${SCRIPT_DIR}/publish_media.go"

if [ ! -f "$PUBLISH_GO" ]; then
  echo "Error: Publisher Go script not found at ${PUBLISH_GO}" >&2
  exit 1
fi

TARGET_NODES=("bindu-minis" "node-dell-1")

if [[ -n "$MQTT_HOST_CLI" || -n "$MQTT_PORT_CLI" ]]; then
  HOST_VAL="$MQTT_HOST_CLI"
  [[ -z "$HOST_VAL" ]] && HOST_VAL="192.168.12.160"
  PORT_VAL="$MQTT_PORT_CLI"
  [[ -z "$PORT_VAL" ]] && PORT_VAL="1885"

  echo "🚀 Invoking Bindu Unified Media & Telemetry Publisher for CLI Target: ${HOST_VAL}:${PORT_VAL}..."
  GOCACHE=/tmp/gocache go run "$PUBLISH_GO" \
    -host "$HOST_VAL" \
    -port "$PORT_VAL" \
    -user "$MQTT_USER" \
    -pass "$MQTT_PASS" \
    -s3-endpoint "$S3_ENDPOINT" \
    -access-key "$ACCESS_KEY" \
    -secret-key "$SECRET_KEY" \
    -provider "$PROVIDER" \
    -bucket "$BUCKET" \
    -src-dir "$SRC_DIR"
else
  for node in "${TARGET_NODES[@]}"; do
    HP_VAL="$(resolve_node_host_port "$node")"
    HOST_VAL="${HP_VAL%%:*}"
    PORT_VAL="${HP_VAL##*:}"

    echo "🚀 Invoking Bindu Unified Media & Telemetry Publisher for Node: ${node} (${HOST_VAL}:${PORT_VAL})..."
    GOCACHE=/tmp/gocache go run "$PUBLISH_GO" \
      -host "$HOST_VAL" \
      -port "$PORT_VAL" \
      -user "$MQTT_USER" \
      -pass "$MQTT_PASS" \
      -s3-endpoint "$S3_ENDPOINT" \
      -access-key "$ACCESS_KEY" \
      -secret-key "$SECRET_KEY" \
      -provider "$PROVIDER" \
      -bucket "$BUCKET" \
      -src-dir "$SRC_DIR"
    echo ""
  done
fi

