#!/usr/bin/env bash
# public/console/docs/semantic-ai-reactor-tutorial/simulate_cold_chain_anomaly.sh
# Simulates a progressive cold-chain failure at a facility to trigger AI reasoning.

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
CLI_NODE_SPECIFIED=false
FACILITY_CLI=""
PRODUCER_CLI=""
WHERE_CLI=""
SIM_MODE="telemetry"

# Default MQTT parameters (can be autodetchecked or overridden)
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
      echo "  --node <id>      Specify gateway node (e.g. node-1, bindu-minis)"
      echo "  --freezer <id>   Specify freezer producer ID"
      echo "  --where <path>   Specify locagram path"
      echo "  --mode <mode>    Simulation mode: telemetry | maintenance | all (default: telemetry)"
      echo "  --host <ip>      MQTT broker host"
      echo "  --port <port>    MQTT broker port"
      echo "  --auth <u:p>     MQTT credentials (user:password)"
      echo "  -h, --help       Show this help message"
      exit 0
      ;;
    --node)
      CLI_NODE_SPECIFIED=true
      FACILITY_CLI="$2"
      shift 2
      ;;
    --freezer)
      PRODUCER_CLI="$2"
      shift 2
      ;;
    --where)
      WHERE_CLI="$2"
      shift 2
      ;;
    --mode)
      SIM_MODE="$2"
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
if [[ -n "$LOGIN_STR" ]]; then
  MQTT_USER="${LOGIN_STR%%:*}"
  MQTT_PASS="${LOGIN_STR##*:}"
fi

# Resolve gateway nodes to target
NODES_TO_PROCESS=()
if [ "$CLI_NODE_SPECIFIED" = true ]; then
  NODES_TO_PROCESS+=("$FACILITY_CLI")
else
  # Read unique gateway node IDs from mesh.json
  MESH_FILE="${CURRENT_ROOT_RUN_DIR}/runtime/config/mesh.json"
  if [ -f "$MESH_FILE" ]; then
    while read -r node_id; do
      if [ -n "$node_id" ] && [ "$node_id" != "null" ]; then
        NODES_TO_PROCESS+=("$node_id")
      fi
    done < <(jq -r '.. | objects | select(.connectivity_kind == "gateway" and .id != "root") | .id' "$MESH_FILE" 2>/dev/null | sort -u || true)
  fi
  # Fallback if no nodes found or mesh.json not present
  if [ ${#NODES_TO_PROCESS[@]} -eq 0 ]; then
    NODES_TO_PROCESS=("node-1" "bindu-minis" "bindu-pi3-uno")
  fi
fi

echo "🚀 Starting simulation for configured gateway nodes: ${NODES_TO_PROCESS[*]}"
echo "--------------------------------------------------------"

for FACILITY in "${NODES_TO_PROCESS[@]}"; do
  # Reset variables per-node
  MQTT_HOST="$MQTT_HOST_CLI"
  MQTT_PORT="$MQTT_PORT_CLI"
  PRODUCER="$PRODUCER_CLI"
  WHERE="$WHERE_CLI"

  # Resolve default credentials/hosts
  if [[ -z "$MQTT_HOST" ]]; then
    # Look up in inventory first if it exists
    INVENTORY_FILE="${ROOT_DIR}/.e2e-ssh/nodes.tsv"
    if [[ -f "${INVENTORY_FILE}" ]]; then
      RESOLVED_HOST="$(awk -F'\t' -v node="$FACILITY" '$1 == node {print $2}' "${INVENTORY_FILE}")"
      if [[ -n "$RESOLVED_HOST" ]]; then
        MQTT_HOST="$RESOLVED_HOST"
      fi
    fi
    # Fallback/defaults
    if [[ -z "$MQTT_HOST" ]]; then
      if [[ "$FACILITY" == "node-1" || "$FACILITY" == "node1" || "$FACILITY" == "root" ]]; then
        MQTT_HOST="127.0.0.1"
      elif [[ "$FACILITY" == "bindu-minis" ]]; then
        MQTT_HOST="192.168.12.160"
      elif [[ "$FACILITY" == "bindu-pi3-uno" || "$FACILITY" == "bindu-pi3" ]]; then
        MQTT_HOST="192.168.12.108"
      else
        MQTT_HOST="127.0.0.1"
      fi
    fi
  fi

  if [[ -z "$MQTT_PORT" ]]; then
    MESH_FILE="${CURRENT_ROOT_RUN_DIR}/runtime/config/mesh.json"
    if [[ -f "$MESH_FILE" ]]; then
      PORT_VAL="$(jq '.. | objects | select(.bindu?.node_id == "'"$FACILITY"'") | .bindu.channels.mqtt_integration.port' "$MESH_FILE" 2>/dev/null || true)"
      if [[ "$PORT_VAL" != "null" && -n "$PORT_VAL" ]]; then
        MQTT_PORT="$PORT_VAL"
      fi
    fi
    # Fallbacks
    if [[ -z "$MQTT_PORT" ]]; then
      if [[ "$FACILITY" == "bindu-minis" || "$FACILITY" == "bindu-pi3-uno" || "$FACILITY" == "bindu-pi3" ]]; then
        MQTT_PORT="1885"
      elif [[ "$FACILITY" == "node-1" || "$FACILITY" == "node1" ]]; then
        MQTT_PORT="39188"
      elif [[ "$FACILITY" == "root" ]]; then
        MQTT_PORT="38190"
      else
        MQTT_PORT="1889"
      fi
    fi
  fi

  # Resolve defaults for Freezer ID and LocaGrama based on the target Node ID
  if [[ -z "${PRODUCER}" ]]; then
    if [[ "$FACILITY" == "node-1" || "$FACILITY" == "node1" ]]; then
      PRODUCER="freezer_madrid_01"
    elif [[ "$FACILITY" == "bindu-minis" ]]; then
      PRODUCER="freezer_barcelona_02"
    elif [[ "$FACILITY" == "bindu-pi3-uno" || "$FACILITY" == "bindu-pi3" ]]; then
      PRODUCER="freezer_valencia_03"
    else
      PRODUCER="freezer_generic_01"
    fi
  fi

  if [[ -z "${WHERE}" ]]; then
    if [[ "$FACILITY" == "node-1" || "$FACILITY" == "node1" ]]; then
      WHERE="/warehouse-a/node-1/aisle-1/rack-1"
    elif [[ "$FACILITY" == "bindu-minis" ]]; then
      WHERE="/warehouse-b/bindu-minis/aisle-4/rack-2"
    elif [[ "$FACILITY" == "bindu-pi3-uno" || "$FACILITY" == "bindu-pi3" ]]; then
      WHERE="/warehouse-c/bindu-pi3-uno/aisle-7/rack-3"
    else
      WHERE="/warehouse-generic/node-generic/aisle-1/rack-1"
    fi
  fi

  # Format WHERE to a valid LocaGrama path if needed
  if [[ ! "$WHERE" =~ ^/ ]]; then
    WHERE="/$(echo "$WHERE" | tr '.' '/' | tr '_' '-')"
  fi

  echo "=== Telemetry Simulation: Node: $FACILITY | Freezer: $PRODUCER | Location: $WHERE ==="
  echo "Broker: tcp://$MQTT_HOST:$MQTT_PORT | User: $MQTT_USER"

  # Generate the 4 chronologically structured FreezerTelemetry events
  MSG1='{"$type":"FreezerTelemetry","$group":"mesh","$user":"admin","$producer":"'${PRODUCER}'","$when":"'$(date -u -d "3 minutes ago" +"%Y-%m-%dT%H:%M:%SZ")'","$where":{"locagram":"'${WHERE}'"},"fields":{"temperature":-18.2,"door_open":false,"compressor_current_amps":8.1}}'
  MSG2='{"$type":"FreezerTelemetry","$group":"mesh","$user":"admin","$producer":"'${PRODUCER}'","$when":"'$(date -u -d "2 minutes ago" +"%Y-%m-%dT%H:%M:%SZ")'","$where":{"locagram":"'${WHERE}'"},"fields":{"temperature":-3.5,"door_open":true,"compressor_current_amps":12.4}}'
  MSG3='{"$type":"FreezerTelemetry","$group":"mesh","$user":"admin","$producer":"'${PRODUCER}'","$when":"'$(date -u -d "1 minute ago" +"%Y-%m-%dT%H:%M:%SZ")'","$where":{"locagram":"'${WHERE}'"},"fields":{"temperature":9.4,"door_open":true,"compressor_current_amps":14.1}}'
  MSG4='{"$type":"FreezerTelemetry","$group":"mesh","$user":"admin","$producer":"'${PRODUCER}'","$when":"'$(date -u +"%Y-%m-%dT%H:%M:%SZ")'","$where":{"locagram":"'${WHERE}'"},"fields":{"temperature":22.1,"door_open":true,"compressor_current_amps":14.8}}'

  # Generate the 4 chronologically structured FreezerMaintenanceTelemetry events
  M_MSG1='{"$type":"FreezerMaintenanceTelemetry","$group":"mesh","$user":"admin","$producer":"'${PRODUCER}'","$when":"'$(date -u -d "3 minutes ago" +"%Y-%m-%dT%H:%M:%SZ")'","$where":{"locagram":"'${WHERE}'"},"fields":{"vibration_amplitude_g":0.5,"motor_temperature":45.2,"refrigerant_pressure_psi":120.0}}'
  M_MSG2='{"$type":"FreezerMaintenanceTelemetry","$group":"mesh","$user":"admin","$producer":"'${PRODUCER}'","$when":"'$(date -u -d "2 minutes ago" +"%Y-%m-%dT%H:%M:%SZ")'","$where":{"locagram":"'${WHERE}'"},"fields":{"vibration_amplitude_g":1.4,"motor_temperature":68.4,"refrigerant_pressure_psi":110.5}}'
  M_MSG3='{"$type":"FreezerMaintenanceTelemetry","$group":"mesh","$user":"admin","$producer":"'${PRODUCER}'","$when":"'$(date -u -d "1 minute ago" +"%Y-%m-%dT%H:%M:%SZ")'","$where":{"locagram":"'${WHERE}'"},"fields":{"vibration_amplitude_g":2.3,"motor_temperature":82.1,"refrigerant_pressure_psi":95.2}}'
  M_MSG4='{"$type":"FreezerMaintenanceTelemetry","$group":"mesh","$user":"admin","$producer":"'${PRODUCER}'","$when":"'$(date -u +"%Y-%m-%dT%H:%M:%SZ")'","$where":{"locagram":"'${WHERE}'"},"fields":{"vibration_amplitude_g":3.1,"motor_temperature":91.5,"refrigerant_pressure_psi":75.0}}'

  skip_node=false

  if [[ "$SIM_MODE" == "telemetry" || "$SIM_MODE" == "all" ]]; then
    echo "🚀 Publishing telemetry messages sequentially to topic 'bindu/in'..."
    for msg in "$MSG1" "$MSG2" "$MSG3" "$MSG4"; do
      if ! go run "${SCRIPT_DIR}/publish_mqtt.go" \
        -host "$MQTT_HOST" \
        -port "$MQTT_PORT" \
        -user "$MQTT_USER" \
        -pass "$MQTT_PASS" \
        -topic "bindu/in" \
        -msg "$msg"; then
        echo "⚠️ Failed to publish telemetry to $FACILITY (broker unreachable). Skipping remaining messages."
        skip_node=true
        break
      fi
      sleep 0.2
    done
  fi

  if [ "$skip_node" = false ] && [[ "$SIM_MODE" == "maintenance" || "$SIM_MODE" == "all" ]]; then
    echo "⚙️ Publishing maintenance messages sequentially to topic 'bindu/in'..."
    for msg in "$M_MSG1" "$M_MSG2" "$M_MSG3" "$M_MSG4"; do
      if ! go run "${SCRIPT_DIR}/publish_mqtt.go" \
        -host "$MQTT_HOST" \
        -port "$MQTT_PORT" \
        -user "$MQTT_USER" \
        -pass "$MQTT_PASS" \
        -topic "bindu/in" \
        -msg "$msg"; then
        echo "⚠️ Failed to publish maintenance to $FACILITY (broker unreachable). Skipping remaining messages."
        skip_node=true
        break
      fi
      sleep 0.2
    done
  fi

  if [ "$skip_node" = false ]; then
    echo "✓ Telemetry replayed successfully for $FACILITY."
  else
    echo "❌ Skipped telemetry simulation for offline node: $FACILITY."
  fi
  echo "--------------------------------------------------------"
done

echo "🎉 All simulations complete. AI Reactors are now processing..."
