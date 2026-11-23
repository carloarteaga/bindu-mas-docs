#!/usr/bin/env bash
# ==============================================================================
# 🥔 BINDU POTATO PROCESSING PIPELINE: MOBILE INTAKE SCANNER SIMULATOR
# ==============================================================================
# Uploads optical slice images out-of-band to MinIO S3 and publishes PotatoBatchIngress to MQTT bindu/in.
# Follows the standard Bindu Out-of-Band Media Storage pattern (media-storage-tutorial).
#
# Usage:
#   ./simulate_mobile_ingress.sh [1|2|quarantine|direct] [optional_lot_id] [options]
#
# Examples:
#   ./simulate_mobile_ingress.sh 1                      # Cold Damaged CIS -> Quarantine Reconditioning
#   ./simulate_mobile_ingress.sh quarantine             # Alias for Profile 1
#   ./simulate_mobile_ingress.sh 2                      # Fresh Harvest -> Immediate Fryer Setpoints
#   ./simulate_mobile_ingress.sh direct                 # Alias for Profile 2
#   ./simulate_mobile_ingress.sh 1 lot-potato-cold-005  # Custom lot ID
#   ./simulate_mobile_ingress.sh --host 192.168.12.160 --port 1885
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
PROFILE=""
LOT_ID_CLI=""
HOST_CLI=""
PORT_CLI=""
USER_CLI=""
PASS_CLI=""
AUTH_CLI=""
IMAGE_CLI=""

# MinIO S3 Defaults (Aligned with media-storage-tutorial)
S3_ENDPOINT="http://192.168.12.160:9000"
ACCESS_KEY="bindu-core-agent"
SECRET_KEY="bindu-media-123"
PROVIDER="minio_local"
BUCKET="raw"
SRC_DIR="${SCRIPT_DIR}/pictures"

# Parse arguments
while [[ "$#" -gt 0 ]]; do
  case "$1" in
    -h|--help|help)
      echo "================================================================="
      echo "🥔 BINDU POTATO PROCESSING PIPELINE: MOBILE SCANNER SIMULATOR"
      echo "================================================================="
      echo "Usage: $0 [1|2|quarantine|direct] [optional_lot_id] [options]"
      echo ""
      echo "Profiles:"
      echo "  1 (or 'quarantine', 'cis', 'cold') : papas-1.png (Cold Damaged CIS -> Reconditioning Schedule)"
      echo "  2 (or 'direct', 'fresh', 'fryer')  : Papas-2.png (Fresh Harvest -> Direct Fryer Actuation)"
      echo ""
      echo "Options:"
      echo "  --host <ip>          MQTT Broker Host (default: 192.168.12.160 or auto-detected)"
      echo "  --port <port>        MQTT Broker Port (default: 1885 or auto-detected)"
      echo "  --auth <user:pass>   MQTT Basic Auth credentials"
      echo "  --user <user>        MQTT Username (default: console)"
      echo "  --pass <pass>        MQTT Password (default: StrongPass123)"
      echo "  --s3-endpoint <url>  MinIO / S3 endpoint URL (default: http://192.168.12.160:9000)"
      echo "  --access-key <key>   MinIO Access Key (default: bindu-core-agent)"
      echo "  --secret-key <sec>   MinIO Secret Key (default: bindu-media-123)"
      echo "  --provider <id>      Storage Provider ID (default: minio_local)"
      echo "  --bucket <name>      Storage Bucket (default: raw)"
      echo "  --src-dir <path>     Source pictures directory (default: script-relative pictures/)"
      echo "  --lot-id <id>        Custom Batch / Lot Identifier"
      echo "  --image <path>       Custom slice photo path"
      echo "  -h, --help           Show this help message"
      exit 0
      ;;
    --host) HOST_CLI="$2"; shift 2 ;;
    --port) PORT_CLI="$2"; shift 2 ;;
    --auth) AUTH_CLI="$2"; shift 2 ;;
    --user) USER_CLI="$2"; shift 2 ;;
    --pass) PASS_CLI="$2"; shift 2 ;;
    --s3-endpoint) S3_ENDPOINT="$2"; shift 2 ;;
    --access-key) ACCESS_KEY="$2"; shift 2 ;;
    --secret-key) SECRET_KEY="$2"; shift 2 ;;
    --provider) PROVIDER="$2"; shift 2 ;;
    --bucket) BUCKET="$2"; shift 2 ;;
    --src-dir) SRC_DIR="$2"; shift 2 ;;
    --lot-id) LOT_ID_CLI="$2"; shift 2 ;;
    --image) IMAGE_CLI="$2"; shift 2 ;;
    1|papas-1|papas1|quarantine|quarentine|cis|cold)
      PROFILE="1"
      shift
      ;;
    2|papas-2|papas2|direct|fresh|fryer)
      PROFILE="2"
      shift
      ;;
    *)
      if [[ -z "$PROFILE" ]]; then
        PROFILE="$1"
      elif [[ -z "$LOT_ID_CLI" ]]; then
        LOT_ID_CLI="$1"
      fi
      shift
      ;;
  esac
done

# Default to profile 1 if none specified
if [[ -z "$PROFILE" ]]; then
  PROFILE="1"
fi

# Normalize profile names and common typos (e.g. 'quarentine')
case "${PROFILE,,}" in
  1|papas-1|papas1|quarantine|quarentine|cis|cold)
    PROFILE="1"
    ;;
  2|papas-2|papas2|direct|fresh|fryer)
    PROFILE="2"
    ;;
  *)
    echo "❌ Unknown profile: '$PROFILE'"
    echo "Valid choices are: '1' (quarantine/cis) or '2' (direct/fresh)"
    exit 1
    ;;
esac

# Resolve Credentials
USER="${USER_CLI}"
PASS="${PASS_CLI}"
if [[ -n "$AUTH_CLI" ]]; then
  USER="${AUTH_CLI%%:*}"
  PASS="${AUTH_CLI##*:}"
elif [[ -z "$USER" && -n "$REPORT_FILE" ]]; then
  LOGIN_STR="$(report_value "${REPORT_FILE}" root_console_login)"
  if [[ -n "$LOGIN_STR" ]]; then
    USER="${LOGIN_STR%%:*}"
    PASS="${LOGIN_STR##*:}"
  fi
fi
if [[ -z "$USER" ]]; then
  USER="console"
  PASS="StrongPass123"
fi

# Resolve Host & Port
HOST="${HOST_CLI}"
PORT="${PORT_CLI}"

if [[ -z "$HOST" ]]; then
  # Check inventory nodes.tsv
  inventory_file="${REPO_ROOT}/.e2e-ssh/nodes.tsv"
  if [[ -f "${inventory_file}" ]]; then
    inv_host="$(awk -F'\t' '$1 == "bindu-minis" {print $2}' "${inventory_file}")"
    if [[ -n "$inv_host" ]]; then
      HOST="$inv_host"
    fi
  fi
fi

if [[ -z "$HOST" && -n "$REPORT_FILE" ]]; then
  REP_MQTT="$(report_value "$REPORT_FILE" "root_mqtt_integration")"
  if [[ -n "$REP_MQTT" ]]; then
    HOST="${REP_MQTT%%:*}"
    if [[ -z "$PORT" ]]; then
      PORT="${REP_MQTT##*:}"
    fi
  fi
fi

if [[ -z "$HOST" ]]; then
  HOST="192.168.12.160"
fi

if [[ -z "$PORT" ]]; then
  if [[ "$HOST" == "192.168.12.160" ]]; then
    PORT="1885"
  else
    PORT="1883"
  fi
fi

# Resolve Image Path relative to SRC_DIR if not given explicitly
IMAGE_PATH="${IMAGE_CLI}"
if [[ -z "$IMAGE_PATH" ]]; then
  if [[ "$PROFILE" == "1" ]]; then
    IMAGE_PATH="${SRC_DIR}/papas-1.png"
  else
    IMAGE_PATH="${SRC_DIR}/Papas-2.png"
  fi
fi

echo "================================================================="
echo "🥔 BINDU POTATO PROCESSING PIPELINE: MOBILE SCANNER SIMULATOR"
echo "================================================================="
echo "Target Broker:  ssl://${HOST}:${PORT}"
echo "Auth Principal: ${USER}"
echo "MinIO Endpoint: ${S3_ENDPOINT} (Bucket: ${BUCKET})"
echo "Slice Image:    ${IMAGE_PATH}"
echo "-----------------------------------------------------------------"

if [[ "$PROFILE" == "1" ]]; then
  echo "📋 Selected Profile: 1 (papas-1.png - Cold Damaged Batch, Transit Temp: 4.8°C < 8.0°C)"
  echo "   -> Out-of-Band S3 Upload: minio_local://raw/PotatoBatchIngress/lot-potato-cold-001/papas-1.png"
  echo "   -> Triggers llava-phi3 / CEL Optical AI C_RS ≈ 0.41"
  echo "   -> Emits SystemScheduledEvent (Silo Reconditioning Timer for 30s)"
  echo "   -> Writes Silo Climate Controller Modbus Reg 100 = 180 (18.0 °C)"
  echo "   -> After 30s, SystemCronJob wakes up digital twin -> Fryer Modbus Reg 200 = 1785 (178.5 °C)"
else
  echo "📋 Selected Profile: 2 (Papas-2.png - Fresh Optimal Harvest, Transit Temp: 11.5°C >= 8.0°C)"
  echo "   -> Out-of-Band S3 Upload: minio_local://raw/PotatoBatchIngress/lot-potato-fresh-002/Papas-2.png"
  echo "   -> Triggers llava-phi3 / CEL Optical AI C_RS ≈ 0.12 (<= 0.20 threshold)"
  echo "   -> Direct Frying Approved (No Silo Quarantine Needed)"
  echo "   -> Writes Fryer Controller Modbus Reg 200 = 1814 (181.4 °C) & Reg 201 = 35 mm/s"
fi

cd "$REPO_ROOT"
go run "$SCRIPT_DIR/simulate_mobile_ingress.go" \
  -host "$HOST" \
  -port "$PORT" \
  -user "$USER" \
  -pass "$PASS" \
  -s3-endpoint "$S3_ENDPOINT" \
  -access-key "$ACCESS_KEY" \
  -secret-key "$SECRET_KEY" \
  -provider "$PROVIDER" \
  -bucket "$BUCKET" \
  -src-dir "$SRC_DIR" \
  -potato "$PROFILE" \
  -image "$IMAGE_PATH" \
  ${LOT_ID_CLI:+-lot-id "$LOT_ID_CLI"}

