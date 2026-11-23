#!/usr/bin/env bash
# public/console/docs/rest-reactor-tutorial/simulate_vessel_gps.sh
# Simulates the navigation trajectory of fishing vessel NUNEGAL-77 across maritime waypoints
# by publishing VesselGPSPosition events sequentially to MQTT.

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

# Default simulation parameters
CLI_NODE_SPECIFIED=false
TARGET_NODE="node-dell-2"
VESSEL_ID="NUNEGAL-77"
WHERE_LOCAGRAM="/galicia/coruna/mar-artabro"
STEP_INTERVAL="1.5"

# Default MQTT parameters
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
      echo "  --node <id>        Specify target gateway node (default: node-dell-2)"
      echo "  --vessel <id>      Specify vessel identifier (default: NUNEGAL-77)"
      echo "  --interval <sec>   Delay between GPS telemetry emits in seconds (default: 1.5)"
      echo "  --host <ip>        MQTT broker host"
      echo "  --port <port>      MQTT broker port"
      echo "  --auth <u:p>       MQTT credentials (user:password)"
      echo "  -h, --help         Show this help message"
      exit 0
      ;;
    --node)
      CLI_NODE_SPECIFIED=true
      TARGET_NODE="$2"
      shift 2
      ;;
    --vessel)
      VESSEL_ID="$2"
      shift 2
      ;;
    --interval)
      STEP_INTERVAL="$2"
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

# Resolve Host and Port
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
    if [[ "$TARGET_NODE" == "node-dell-2" || "$TARGET_NODE" == "dell-debian1" || "$TARGET_NODE" == "vessel-nunegal-77" ]]; then
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
  if [[ -z "$MQTT_PORT" ]]; then
    if [[ "$TARGET_NODE" == "node-dell-2" || "$TARGET_NODE" == "dell-debian1" || "$TARGET_NODE" == "vessel-nunegal-77" ]]; then
      MQTT_PORT="1887"
    elif [[ "$TARGET_NODE" == "bindu-minis" || "$TARGET_NODE" == "bindu-pi3-uno" || "$TARGET_NODE" == "bindu-pi3" || "$TARGET_NODE" == "node-dell-1" ]]; then
      MQTT_PORT="1885"
    elif [[ "$TARGET_NODE" == "node-1" ]]; then
      MQTT_PORT="39188"
    elif [[ "$TARGET_NODE" == "root" ]]; then
      MQTT_PORT="38190"
    else
      MQTT_PORT="1883"
    fi
  fi
fi

echo "=========================================================================="
echo "🚢 Fishing Vessel GPS Trajectory Simulation: ${VESSEL_ID}"
echo "📍 Target Node: ${TARGET_NODE} | Locagram: ${WHERE_LOCAGRAM}"
echo "📡 Broker: tcp://${MQTT_HOST}:${MQTT_PORT} (mqtt_integration) | Auth User: ${MQTT_USER}"
echo "=========================================================================="

# Define the 8 sequential navigation waypoints
WAYPOINTS=(
  "43.3715|-8.3960|4.2|45.0|ZONE_CORUNA_COAST|Punto 1: Saliendo del Dique de Abrigo (Puerto de A Coruña)"
  "43.3850|-8.4065|8.5|330.0|ZONE_HERCULES|Punto 2: Navegando frente a la Torre de Hércules"
  "43.4002|-8.4201|12.1|315.0|ZONE_OPEN_SEA|Punto 3: Saliendo a mar abierto, rumbo Noroeste"
  "43.4155|-8.4350|14.8|310.0|ZONE_THERMAL_TRANSIT|Punto 4: Aumentando velocidad hacia la zona térmica"
  "43.4310|-8.4485|6.0|290.0|ZONE_FISHING_OPTIMAL|Punto 5: Entrando a la zona de pesca predicha (Óptima)"
  "43.4400|-8.4600|3.5|270.0|ZONE_FISHING_ACTIVE|Punto 6: Maniobra de pesca activa"
  "43.4350|-8.4750|5.2|245.0|ZONE_WEST_DRIFT|Punto 7: El barco empieza a desviarse ligeramente hacia el oeste"
  "43.4280|-8.4820|7.8|220.0|ZONE_CRITICAL_OFFCOURSE|Punto 8: Desvío crítico (Alerta de rumbo CEL Reactor)"
)

step=1
for wp in "${WAYPOINTS[@]}"; do
  IFS='|' read -r lat lon speed heading zone note <<< "$wp"
  clean_heading=$(echo "$heading" | sed -E 's/^0+([0-9])/\1/')
  now_iso="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

  # Construct VesselGPSPosition payload
  MSG=$(cat <<JSON
{
  "\$type": "VesselGPSPosition",
  "\$group": "mesh",
  "\$user": "${MQTT_USER}",
  "\$producer": "${VESSEL_ID}",
  "\$when": "${now_iso}",
  "\$where": {
    "locagram": "${WHERE_LOCAGRAM}"
  },
  "fields": {
    "vessel_id": "${VESSEL_ID}",
    "latitude": ${lat},
    "longitude": ${lon},
    "speed_knots": ${speed},
    "heading_deg": ${clean_heading},
    "zone_id": "${zone}"
  }
}
JSON
)

  echo -e "\n📍 [Step ${step}/8] ${note}"
  echo "   Coords: (${lat}, ${lon}) | Speed: ${speed} kts | Heading: ${heading}° | Zone: ${zone}"

  if ! go run "${SCRIPT_DIR}/publish_mqtt.go" \
    -host "$MQTT_HOST" \
    -port "$MQTT_PORT" \
    -user "$MQTT_USER" \
    -pass "$MQTT_PASS" \
    -topic "bindu/in" \
    -msg "$MSG"; then
    echo "⚠️ Failed to publish waypoint ${step} to ${TARGET_NODE} (broker unreachable). Exiting."
    exit 1
  fi

  echo "   ✓ Published VesselGPSPosition event successfully."
  
  if [ "$step" -lt 8 ]; then
    sleep "$STEP_INTERVAL"
  fi
  ((step++))
done

echo -e "\n=========================================================================="
echo "🎉 Voyage Simulation Complete!"
echo "📡 The REST Reactor on ${TARGET_NODE} has evaluated oceanographic conditions"
echo "   for each waypoint and emitted corresponding RawMarineData telemetry events."
echo "=========================================================================="
