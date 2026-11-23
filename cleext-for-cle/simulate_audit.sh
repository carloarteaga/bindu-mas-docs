#!/usr/bin/env bash
# public/console/docs/cleext-for-cle/simulate_audit.sh
# Validates and triggers a CEL & celext Fleet Topology Audit.

set -e

# Colors for terminal output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
YELLOW='\033[1;33m'
BOLD='\033[1m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/../../../.." && pwd)"

CURRENT_ROOT_RUN_DIR="${ROOT_DIR}/.e2e-root/current"
RUN_DIR=""
if [ -d "${CURRENT_ROOT_RUN_DIR}" ]; then
  RUN_DIR="$(readlink -f "${CURRENT_ROOT_RUN_DIR}" | sed 's#/root$##')"
fi
REPORT_FILE="${RUN_DIR}/report.txt"

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
TARGET_NODE="root"
AUDIT_LEVEL="DEEP"
TARGET_CLUSTER="/galicia"

while [[ "$#" -gt 0 ]]; do
  case "$1" in
    -h|--help|help)
      echo -e "${BOLD}Usage:${NC} $0 [options]"
      echo "Options:"
      echo "  --node <id>         Target gateway node ID (default: root)"
      echo "  --host <ip>         MQTT broker host (defaults to auto-detected local/e2e)"
      echo "  --port <port>       MQTT broker port (defaults to auto-detected from mesh.json/report.txt)"
      echo "  --auth <u:p>        MQTT credentials (default: console:StrongPass123 or report.txt)"
      echo "  --level <level>     Audit level: STANDARD, DEEP, SECURITY (default: DEEP)"
      echo "  --cluster <path>    Target cluster locagram (default: /galicia)"
      echo "  -h, --help          Show this help message"
      exit 0
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
    --level)
      AUDIT_LEVEL="$2"
      shift 2
      ;;
    --cluster)
      TARGET_CLUSTER="$2"
      shift 2
      ;;
    *)
      echo -e "${RED}Unknown parameter:${NC} $1"
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

echo -e "${BOLD}${CYAN}======================================================${NC}"
echo -e "${BOLD}${CYAN} 🧭 Bindu CEL & celext Fleet Topology Audit Simulator ${NC}"
echo -e "${BOLD}${CYAN}======================================================${NC}"

# 1. Offline Validation of Tutorial Configurations
echo -e "\n${BOLD}${BLUE}[Step 1/3] Validating Tutorial Reactor Configurations Offline...${NC}"

echo -e "${CYAN}Checking CEL Reactor Config (poc_cel_reactor_config.json):${NC}"
GOCACHE=/tmp/gocache go run "${ROOT_DIR}/cmd/bindu-studio" cel-reactor validate --file "${SCRIPT_DIR}/poc_cel_reactor_config.json"

echo -e "\n${CYAN}Checking REST Reactor Config (poc_rest_reactor_config.json):${NC}"
GOCACHE=/tmp/gocache go run "${ROOT_DIR}/cmd/bindu-studio" rest-reactor validate --file "${SCRIPT_DIR}/poc_rest_reactor_config.json"

# 2. Broker Connection Resolution
echo -e "\n${BOLD}${BLUE}[Step 2/3] Resolving MQTT Broker Connection...${NC}"

RESOLVED_HOST="$MQTT_HOST_CLI"
RESOLVED_PORT="$MQTT_PORT_CLI"

# Resolve Host if not CLI-specified
if [[ -z "$RESOLVED_HOST" ]]; then
  INVENTORY_FILE="${ROOT_DIR}/.e2e-ssh/nodes.tsv"
  if [[ -f "${INVENTORY_FILE}" ]]; then
    INV_HOST="$(awk -F'\t' -v node="${TARGET_NODE}" '$1 == node {print $2}' "${INVENTORY_FILE}")"
    if [[ -n "$INV_HOST" ]]; then
      RESOLVED_HOST="$INV_HOST"
    fi
  fi
  if [[ -z "$RESOLVED_HOST" ]]; then
    if [[ "$TARGET_NODE" == "node-dell-2" || "$TARGET_NODE" == "dell-debian1" ]]; then
      RESOLVED_HOST="192.168.12.185"
    elif [[ "$TARGET_NODE" == "bindu-minis" ]]; then
      RESOLVED_HOST="192.168.12.160"
    elif [[ "$TARGET_NODE" == "bindu-pi3-uno" || "$TARGET_NODE" == "bindu-pi3" ]]; then
      RESOLVED_HOST="192.168.12.108"
    else
      RESOLVED_HOST="127.0.0.1"
    fi
  fi
fi

# Resolve Port if not CLI-specified
if [[ -z "$RESOLVED_PORT" ]]; then
  # 1. Check report.txt
  if [[ -f "${REPORT_FILE}" ]]; then
    if [[ "$TARGET_NODE" == "root" ]]; then
      INTEG_VAL="$(report_value "${REPORT_FILE}" "root_mqtt_integration")"
      [[ -z "$INTEG_VAL" ]] && INTEG_VAL="$(report_value "${REPORT_FILE}" "root_mesh_mqtt")"
      if [[ -n "$INTEG_VAL" ]]; then
        RESOLVED_PORT="${INTEG_VAL##*:}"
      fi
    elif [[ "$TARGET_NODE" == "node-1" ]]; then
      INTEG_VAL="$(report_value "${REPORT_FILE}" "node_mqtt_integration")"
      [[ -z "$INTEG_VAL" ]] && INTEG_VAL="$(report_value "${REPORT_FILE}" "node_mesh_mqtt")"
      if [[ -n "$INTEG_VAL" ]]; then
        RESOLVED_PORT="${INTEG_VAL##*:}"
      fi
    fi
  fi

  # 2. Check mesh.json
  if [[ -z "$RESOLVED_PORT" ]]; then
    MESH_FILE="${CURRENT_ROOT_RUN_DIR}/runtime/config/mesh.json"
    if [[ -f "$MESH_FILE" ]]; then
      P_FOUND="$(jq -r --arg n "$TARGET_NODE" '.. | objects | select(.id == $n or .bindu?.node_id == $n) | .bindu.channels.mqtt_integration.port // empty' "$MESH_FILE" 2>/dev/null | head -n 1 || true)"
      if [[ -z "$P_FOUND" || "$P_FOUND" == "null" ]]; then
        P_FOUND="$(jq -r --arg n "$TARGET_NODE" '.. | objects | select(.id == $n or .bindu?.node_id == $n) | .bindu.channels.mesh_mqtt.port // empty' "$MESH_FILE" 2>/dev/null | head -n 1 || true)"
      fi
      if [[ -n "$P_FOUND" && "$P_FOUND" != "null" ]]; then
        RESOLVED_PORT="$P_FOUND"
      fi
    fi
  fi

  # 3. Fallbacks
  if [[ -z "$RESOLVED_PORT" ]]; then
    if [[ "$TARGET_NODE" == "root" ]]; then
      RESOLVED_PORT="38190"
    elif [[ "$TARGET_NODE" == "node-1" ]]; then
      RESOLVED_PORT="39188"
    elif [[ "$TARGET_NODE" == "node-dell-2" ]]; then
      RESOLVED_PORT="1887"
    elif [[ "$TARGET_NODE" == "bindu-minis" || "$TARGET_NODE" == "bindu-pi3-uno" || "$TARGET_NODE" == "node-dell-1" ]]; then
      RESOLVED_PORT="1885"
    else
      RESOLVED_PORT="1883"
    fi
  fi
fi

echo -e "Target Node:   ${BOLD}${TARGET_NODE}${NC}"
echo -e "Target Broker: ${BOLD}ssl://${RESOLVED_HOST}:${RESOLVED_PORT}${NC}"
echo -e "Audit Level:   ${BOLD}${AUDIT_LEVEL}${NC}"
echo -e "Cluster Path:  ${BOLD}${TARGET_CLUSTER}${NC}"

# 3. Publish Audit Trigger
echo -e "\n${BOLD}${BLUE}[Step 3/3] Publishing MeshAuditTrigger event via MQTT...${NC}"

cd "${SCRIPT_DIR}"
GOCACHE=/tmp/gocache go run publish_audit_trigger.go \
  -host "${RESOLVED_HOST}" \
  -port "${RESOLVED_PORT}" \
  -user "${MQTT_USER}" \
  -pass "${MQTT_PASS}" \
  -level "${AUDIT_LEVEL}" \
  -cluster "${TARGET_CLUSTER}"

echo -e "\n${BOLD}${GREEN}🎉 Fleet Audit Simulation complete!${NC}"
echo -e "Observe the generated ${BOLD}MeshTopologyInspection${NC} in the Bindu Console (http://localhost:8080/console/) or via:"
echo -e "${YELLOW}go run ./cmd/bindu-studio mesh health --base-dir ./.e2e-root/current${NC}\n"
