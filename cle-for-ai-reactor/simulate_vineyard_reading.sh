#!/usr/bin/env bash
# public/console/docs/cle-for-ai-reactor/simulate_vineyard_reading.sh
# Publishes a SensorVineyardReading event via MQTT to trigger the CEL-powered AI Reactor.

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

# Default parameters
TARGET_NODE="node-1"
VINEYARD_ID="BODEGA-ALBARINO-01"
PARCEL="Val-Do-Salnes-Finca-Abalos"
WHERE_LOCAGRAM="/rias-baixas/salnes"
PRODUCER_ID="station-salnes-01"
SCENARIO="mildew_risk"

MQTT_HOST_CLI=""
MQTT_PORT_CLI=""
MQTT_USER="console"
MQTT_PASS="StrongPass123"

while [[ "$#" -gt 0 ]]; do
  case "$1" in
    -h|--help|help)
      echo "Usage: $0 [options]"
      echo "Options:"
      echo "  --node <id>        Specify target gateway node (default: node-1)"
      echo "  --scenario <name>  mildew_risk | frost_risk | nominal (default: mildew_risk)"
      echo "  --vineyard <id>    Specify vineyard ID (default: BODEGA-ALBARINO-01)"
      echo "  --parcel <name>    Specify parcel name (default: Val-Do-Salnes-Finca-Abalos)"
      echo "  --where <path>     Specify locagram path (default: /rias-baixas/salnes)"
      echo "  --producer <id>    Specify producer ID (default: station-salnes-01)"
      echo "  --host <ip>        MQTT broker host"
      echo "  --port <port>      MQTT broker port"
      echo "  --auth <u:p>       MQTT credentials (user:password)"
      echo "  -h, --help         Show this help message"
      exit 0
      ;;
    --node)
      TARGET_NODE="$2"
      shift 2
      ;;
    --scenario)
      SCENARIO="$2"
      shift 2
      ;;
    --vineyard)
      VINEYARD_ID="$2"
      shift 2
      ;;
    --parcel)
      PARCEL="$2"
      shift 2
      ;;
    --where)
      WHERE_LOCAGRAM="$2"
      shift 2
      ;;
    --producer)
      PRODUCER_ID="$2"
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

MQTT_HOST="$MQTT_HOST_CLI"
MQTT_PORT="$MQTT_PORT_CLI"

if [[ -z "$MQTT_HOST" ]]; then
  INVENTORY_FILE="${ROOT_DIR}/.e2e-ssh/nodes.tsv"
  if [[ -f "${INVENTORY_FILE}" ]]; then
    INV_HOST="$(awk -F'\t' -v node="${TARGET_NODE}" '$1 == node {print $2}' "${INVENTORY_FILE}")"
    if [[ -n "$INV_HOST" ]]; then
      MQTT_HOST="$INV_HOST"
    fi
  fi
  if [[ -z "$MQTT_HOST" ]]; then
    if [[ "$TARGET_NODE" == "node-dell-2" || "$TARGET_NODE" == "dell-debian1" ]]; then
      MQTT_HOST="192.168.12.185"
    elif [[ "$TARGET_NODE" == "bindu-minis" ]]; then
      MQTT_HOST="192.168.12.160"
    elif [[ "$TARGET_NODE" == "bindu-pi3-uno" || "$TARGET_NODE" == "bindu-pi3" ]]; then
      MQTT_HOST="192.168.12.108"
    else
      MQTT_HOST="127.0.0.1"
    fi
  fi
fi

if [[ -z "$MQTT_PORT" ]]; then
  # 1. Check mesh.json first for actual assigned listener ports
  MESH_FILE="${CURRENT_ROOT_RUN_DIR}/runtime/config/mesh.json"
  if [[ -f "$MESH_FILE" ]]; then
    P_FOUND="$(jq -r --arg n "$TARGET_NODE" '.. | objects | select(.id == $n or .bindu?.node_id == $n) | .bindu.channels.mqtt_integration.port // empty' "$MESH_FILE" 2>/dev/null | head -n 1 || true)"
    if [[ -z "$P_FOUND" || "$P_FOUND" == "null" ]]; then
      P_FOUND="$(jq -r --arg n "$TARGET_NODE" '.. | objects | select(.id == $n or .bindu?.node_id == $n) | .bindu.channels.mesh_mqtt.port // empty' "$MESH_FILE" 2>/dev/null | head -n 1 || true)"
    fi
    if [[ -n "$P_FOUND" && "$P_FOUND" != "null" ]]; then
      MQTT_PORT="$P_FOUND"
    fi
  fi

  # 2. Check report.txt
  if [[ -z "$MQTT_PORT" && -f "${REPORT_FILE}" ]]; then
    if [[ "$TARGET_NODE" == "node-1" || "$TARGET_NODE" == "node1" ]]; then
      INTEG_VAL="$(report_value "${REPORT_FILE}" "node_mqtt_integration")"
      [[ -z "$INTEG_VAL" ]] && INTEG_VAL="$(report_value "${REPORT_FILE}" "node_mesh_mqtt")"
      if [[ -n "$INTEG_VAL" ]]; then
        MQTT_PORT="${INTEG_VAL##*:}"
      fi
    elif [[ "$TARGET_NODE" == "root" ]]; then
      INTEG_VAL="$(report_value "${REPORT_FILE}" "root_mqtt_integration")"
      [[ -z "$INTEG_VAL" ]] && INTEG_VAL="$(report_value "${REPORT_FILE}" "root_mesh_mqtt")"
      if [[ -n "$INTEG_VAL" ]]; then
        MQTT_PORT="${INTEG_VAL##*:}"
      fi
    fi
  fi

  # 3. Fallbacks
  if [[ -z "$MQTT_PORT" ]]; then
    if [[ "$TARGET_NODE" == "node-1" || "$TARGET_NODE" == "node1" ]]; then
      MQTT_PORT="39188"
    elif [[ "$TARGET_NODE" == "root" ]]; then
      MQTT_PORT="38190"
    elif [[ "$TARGET_NODE" == "bindu-minis" || "$TARGET_NODE" == "bindu-pi3-uno" ]]; then
      MQTT_PORT="1885"
    else
      MQTT_PORT="1883"
    fi
  fi
fi

echo "=========================================================================="
echo "🍇 Albariño Smart Vineyard Telemetry Simulation: Scenario [${SCENARIO}]"
echo "📍 Target Node: ${TARGET_NODE} | Vineyard: ${VINEYARD_ID} (${PARCEL})"
echo "📡 Broker: ssl://${MQTT_HOST}:${MQTT_PORT} | Auth User: ${MQTT_USER}"
echo "=========================================================================="

GOCACHE=/tmp/gocache go run "${SCRIPT_DIR}/publish_vineyard_reading.go" \
  -host "${MQTT_HOST}" \
  -port "${MQTT_PORT}" \
  -user "${MQTT_USER}" \
  -pass "${MQTT_PASS}" \
  -topic "bindu/in" \
  -scenario "${SCENARIO}" \
  -vineyard "${VINEYARD_ID}" \
  -parcel "${PARCEL}" \
  -where "${WHERE_LOCAGRAM}" \
  -producer "${PRODUCER_ID}"

echo ""
echo "🎉 Telemetry event sent successfully!"
echo "📡 If conditions met CEL gating, the AI Reactor on ${TARGET_NODE} is running inference..."
echo "🍷 Check 'VineyardHarvestAdvisory' in Bindu Console (http://127.0.0.1:38080/console/ or http://localhost:8080/console/)!"
