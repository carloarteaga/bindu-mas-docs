#!/usr/bin/env bash
# Universal Deployment Script for Bluetooth Low Energy (BLE) Spatial Presence MAS Solution
# Follows the Universal 5-Phase Deployment Pipeline defined in public/console/docs/AGENTS.md
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../../.." && pwd)"

AUTH_CLI=""
ROOT_URL_CLI=""
USER_CLI=""
PASS_CLI=""
BASE_DIR_CLI=""
AUTO_CONFIRM=false
TARGET_GATEWAY="node-dell-2"

while [[ "$#" -gt 0 ]]; do
  case "$1" in
    -h|--help|help)
      echo "Usage: $0 [options]"
      echo ""
      echo "Deploys the BLE Spatial Presence & Reference Plane MAS solution."
      echo ""
      echo "Options:"
      echo "  --base-dir <dir>       Bindu base directory (default: auto-detected ./.e2e-root/current or repo root)"
      echo "  --root-url <url>       Bindu Root API endpoint (default: http://127.0.0.1:8080)"
      echo "  --auth <user:pass>     Basic auth credentials (default: console:StrongPass123)"
      echo "  --user <user>          Basic auth username"
      echo "  --pass <pass>          Basic auth password"
      echo "  --target <node>        Target edge gateway node ID (default: node-dell-2)"
      echo "  -y, --yes              Confirm execution without interactive prompting"
      echo "  -h, --help             Show this help message"
      exit 0
      ;;
    --base-dir) BASE_DIR_CLI="$2"; shift 2 ;;
    --root-url) ROOT_URL_CLI="$2"; shift 2 ;;
    --auth) AUTH_CLI="$2"; shift 2 ;;
    --user) USER_CLI="$2"; shift 2 ;;
    --pass) PASS_CLI="$2"; shift 2 ;;
    --target) TARGET_GATEWAY="$2"; shift 2 ;;
    -y|--yes) AUTO_CONFIRM=true; shift ;;
    *)
      echo "Unknown argument: $1"
      echo "Use --help for usage information."
      exit 1
      ;;
  esac
done

# Resolve Base Dir
BASE_DIR="${BASE_DIR_CLI:-${BASE_DIR:-}}"
if [ -z "${BASE_DIR}" ]; then
  if [ -d "${REPO_ROOT}/.e2e-root/current" ]; then
    BASE_DIR="${REPO_ROOT}/.e2e-root/current"
  else
    BASE_DIR="${REPO_ROOT}"
  fi
fi

BASE_DIR_FLAG=()
if [ -n "${BASE_DIR}" ]; then
  BASE_DIR_FLAG=("--base-dir" "${BASE_DIR}")
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

# Resolve Auth
AUTH="${AUTH_CLI:-${AUTH:-}}"
if [ -z "${AUTH}" ]; then
  if [ -n "${USER_CLI}" ] && [ -n "${PASS_CLI}" ]; then
    AUTH="${USER_CLI}:${PASS_CLI}"
  elif [ -n "${REPORT_FILE}" ]; then
    REP_USER="$(report_value "${REPORT_FILE}" "console_user")"
    REP_PASS="$(report_value "${REPORT_FILE}" "console_password")"
    if [ -n "${REP_USER}" ] && [ -n "${REP_PASS}" ]; then
      AUTH="${REP_USER}:${REP_PASS}"
    fi
  fi
fi
if [ -z "${AUTH}" ]; then
  AUTH="console:StrongPass123"
fi

# Resolve Root URL
ROOT_URL="${ROOT_URL_CLI:-${ROOT_URL:-}}"
if [ -z "${ROOT_URL}" ] && [ -n "${REPORT_FILE}" ]; then
  REP_CONSOLE_URL="$(report_value "${REPORT_FILE}" "root_console_url")"
  if [ -n "${REP_CONSOLE_URL}" ]; then
    ROOT_URL="${REP_CONSOLE_URL}"
  fi
fi
if [ -z "${ROOT_URL}" ]; then
  ROOT_URL="http://127.0.0.1:8080"
fi

echo "================================================================="
echo "📡 BINDU MAS DEPLOYMENT: BLE SPATIAL PRESENCE & DIGITAL TWINS"
echo "   (Target Gateway: ${TARGET_GATEWAY} | Reference Plane: operation-room)"
echo "================================================================="
echo "Target BaseDir: ${BASE_DIR}"
echo "Target RootURL: ${ROOT_URL}"
echo "Auth Principal: ${AUTH%%:*}:******"
echo "-----------------------------------------------------------------"

# Phase 1: Register Ontology & Message Types (Level 0 - Schemas First)
echo "📜 Phase 1: Merging 3 BLE application message schemas into Root..."
cd "$REPO_ROOT"
go run ./cmd/bindu-studio message-types merge \
    --in-path "$SCRIPT_DIR/poc_message_types.json" \
    --all \
    --auth "$AUTH" \
    --root-url "$ROOT_URL" \
    "${BASE_DIR_FLAG[@]}"

# Phase 2: Compile, Dry-Run and Certify BLE BPR Plugin (Level 1)
echo "📦 Phase 2: Simulating dry-run and certifying bindu-ble-presence-plugin..."
cd "$REPO_ROOT"
echo "  • [1/2] Executing plugin dry-run (handshake, stimulus & clean shutdown)..."
go run ./cmd/bindu-studio plugin dry-run \
    --source "$SCRIPT_DIR/bindu-ble-presence-plugin" \
    --timeout 3s

echo "  • [2/2] Certifying and packaging linux/amd64 bundle..."
go run ./cmd/bindu-studio plugin certify \
    --source "$SCRIPT_DIR/bindu-ble-presence-plugin" \
    --out "$SCRIPT_DIR/bindu-ble-presence-plugin.tar.gz" \
    --target linux/amd64 \
    --config "$SCRIPT_DIR/poc_message_types.json" \
    "${BASE_DIR_FLAG[@]}"

# Phase 3: Spatial Topology & Reference Node Configuration (Level 0 Spatial Model)
echo "🌐 Phase 3: Setting up Mesh reference node & 2D spatial Cartesian plane..."
echo "  • Adding reference node 'operation-room' under '${TARGET_GATEWAY}'..."
go run ./cmd/bindu-studio mesh add \
    --id operation-room \
    --parent "$TARGET_GATEWAY" \
    --connectivity reference \
    --name "Operation Room 101" \
    --yes \
    "${BASE_DIR_FLAG[@]}" || true

go run ./cmd/bindu-studio mesh apply "${BASE_DIR_FLAG[@]}" || true

echo "  • Establishing '${TARGET_GATEWAY}' as Cartesian plane root (origin: 0,0, unit: meters)..."
go run ./cmd/bindu-studio geo-locagram local-plane \
    --node "$TARGET_GATEWAY" \
    --units meters \
    --scale 1.0 \
    --auth "$AUTH" \
    --root-url "$ROOT_URL" \
    "${BASE_DIR_FLAG[@]}" || true

echo "  • Assigning 12m x 8m polygon geometry to 'operation-room'..."
go run ./cmd/bindu-studio geo-locagram local-set \
    --node operation-room \
    --plane "$TARGET_GATEWAY" \
    --role zone \
    --geometry '{"type":"Polygon","coordinates":[[[0,0],[12,0],[12,8],[0,8],[0,0]]]}' \
    --color "#3B82F6" \
    --auth "$AUTH" \
    --root-url "$ROOT_URL" \
    "${BASE_DIR_FLAG[@]}" || true

# Phase 4: Declarative Application & Solution Bundling (Levels 2 & 3)
echo "🚀 Phase 4: Packaging & Applying declarative MAS Solution bundle..."
echo "  • [1/4] Validating Dashboard Definitions..."
go run ./cmd/bindu-studio dashboard validate \
    --file "$SCRIPT_DIR/poc_dashboards.json" \
    "${BASE_DIR_FLAG[@]}"

echo "  • [2/4] Staging BLE Presence Plugin on target gateway (${TARGET_GATEWAY})..."
PLUGIN_TARGET_FLAGS=(--target node --node "$TARGET_GATEWAY")
if [ "$TARGET_GATEWAY" = "root" ]; then
  PLUGIN_TARGET_FLAGS=(--target root)
fi

go run ./cmd/bindu-studio plugin apply \
    --bundle "$SCRIPT_DIR/bindu-ble-presence-plugin.tar.gz" \
    "${PLUGIN_TARGET_FLAGS[@]}" \
    --auth "$AUTH" \
    --root-url "$ROOT_URL" \
    "${BASE_DIR_FLAG[@]}"

echo "  • [3/4] Importing Table & Operational Map Dashboards..."
go run ./cmd/bindu-studio dashboard import \
    --file "$SCRIPT_DIR/poc_dashboards.json" \
    --all \
    --auth "$AUTH" \
    --root-url "$ROOT_URL" \
    "${BASE_DIR_FLAG[@]}" || true

echo "  • [4/4] Exporting, validating, and applying MAS Solution Set..."
mkdir -p "$SCRIPT_DIR/bundles"
go run ./cmd/bindu-studio solution export \
    --file "$SCRIPT_DIR/ble_mas_solution.json" \
    --out "$SCRIPT_DIR/bundles/ble_mas_solution.bundle.json" \
    --auth "$AUTH" \
    --root-url "$ROOT_URL" \
    "${BASE_DIR_FLAG[@]}" || true

go run ./cmd/bindu-studio solution validate \
    --file "$SCRIPT_DIR/ble_mas_solution.json" \
    "${BASE_DIR_FLAG[@]}"

go run ./cmd/bindu-studio solution apply \
    --file "$SCRIPT_DIR/ble_mas_solution.json" \
    --yes \
    --auth "$AUTH" \
    --root-url "$ROOT_URL" \
    "${BASE_DIR_FLAG[@]}" || true

# Phase 5: The Universal Runtime OTA Triptych (Level 4 - Physical OS Delivery)
echo "⚡ Phase 5: Executing Universal Runtime OTA Triptych..."
go run ./cmd/bindu-studio runtime baseline accept \
    --yes \
    "${BASE_DIR_FLAG[@]}" || true

go run ./cmd/bindu-studio runtime desired commit \
    --note "Deploy BLE Spatial Presence & Reference Plane MAS Solution on ${TARGET_GATEWAY}" \
    "${BASE_DIR_FLAG[@]}" || true

go run ./cmd/bindu-studio runtime update wizard \
    --yes \
    --auth "$AUTH" \
    --root-url "$ROOT_URL" \
    "${BASE_DIR_FLAG[@]}" || true

echo "================================================================="
echo "✅ DEPLOYMENT COMPLETE! BLE Plugin, Reference Plane & Dashboards live."
echo "================================================================="
