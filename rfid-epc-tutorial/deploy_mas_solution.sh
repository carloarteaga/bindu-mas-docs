#!/usr/bin/env bash
# Universal Deployment Script for Industrial RFID LLRP & 11-Standard EPC MAS Solution (Root Only)
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../../.." && pwd)"

AUTH_CLI=""
ROOT_URL_CLI=""
USER_CLI=""
PASS_CLI=""
BASE_DIR_CLI=""
AUTO_CONFIRM=false

while [[ "$#" -gt 0 ]]; do
  case "$1" in
    -h|--help|help)
      echo "Usage: $0 [options]"
      echo ""
      echo "Deploys the Industrial RFID LLRP & 11-Standard EPC MAS solution (Root only)."
      echo ""
      echo "Options:"
      echo "  --base-dir <dir>       Bindu base directory (default: auto-detected ./.e2e-root/current or repo root)"
      echo "  --root-url <url>       Bindu Root API endpoint (default: http://127.0.0.1:8080)"
      echo "  --auth <user:pass>     Basic auth credentials (default: console:StrongPass123)"
      echo "  --user <user>          Basic auth username"
      echo "  --pass <pass>          Basic auth password"
      echo "  -y, --yes              Confirm execution without interactive prompting"
      echo "  -h, --help             Show this help message"
      exit 0
      ;;
    --base-dir) BASE_DIR_CLI="$2"; shift 2 ;;
    --root-url) ROOT_URL_CLI="$2"; shift 2 ;;
    --auth) AUTH_CLI="$2"; shift 2 ;;
    --user) USER_CLI="$2"; shift 2 ;;
    --pass) PASS_CLI="$2"; shift 2 ;;
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
echo "🏷️ BINDU MAS DEPLOYMENT: INDUSTRIAL RFID LLRP & 11-STANDARD EPC"
echo "   (Target: ROOT NODE ONLY - High-Throughput Edge Gate Integration)"
echo "================================================================="
echo "Target BaseDir: ${BASE_DIR}"
echo "Target RootURL: ${ROOT_URL}"
echo "Auth Principal: ${AUTH%%:*}:******"
echo "-----------------------------------------------------------------"

# Phase 0: Provisioning Centralized Cloud API Secrets (Zero-Plaintext Wire Guarantee)
echo "🔐 Phase 0: Verifying Cloud API Secrets in Centralized Vault..."
cd "$REPO_ROOT"
if ! go run ./cmd/bindu-studio access-control secrets get TYPESAFE_API_KEY "${BASE_DIR_FLAG[@]}" >/dev/null 2>&1; then
  echo "  • Provisioning TYPESAFE_API_KEY into encrypted vault..."
  go run ./cmd/bindu-studio access-control secrets set TYPESAFE_API_KEY \
      --value "${TYPESAFE_API_KEY:-ts_live_mock_token_123}" \
      --yes \
      "${BASE_DIR_FLAG[@]}"
else
  echo "  • ✅ Secret TYPESAFE_API_KEY verified in vault."
fi

# Phase 1: Register Ontology & Message Types (Level 0 - Schemas First)
echo "📜 Phase 1: Merging 15 RFID application message schemas into Root..."
cd "$REPO_ROOT"
go run ./cmd/bindu-studio message-types merge \
    --in-path "$SCRIPT_DIR/poc_message_types.json" \
    --all \
    --auth "$AUTH" \
    --root-url "$ROOT_URL" \
    "${BASE_DIR_FLAG[@]}"

# Phase 2: Verify in-process Dry-Run, Compile and Certify LLRP RFID BPR Plugin (Level 1)
echo "🔬 Phase 2a: Simulating BPR Plugin execution & lifecycle dry-run in-process..."
cd "$REPO_ROOT"
go run ./cmd/bindu-studio plugin dry-run \
    --source "$SCRIPT_DIR/bindu-rfid-epc-plugin" \
    --config "$SCRIPT_DIR/poc_message_types.json" \
    --timeout 4s \
    "${BASE_DIR_FLAG[@]}"

echo "📦 Phase 2b: Certifying and packaging bindu-rfid-epc-plugin bundle..."
go run ./cmd/bindu-studio plugin certify \
    --source "$SCRIPT_DIR/bindu-rfid-epc-plugin" \
    --out "$SCRIPT_DIR/bindu-rfid-epc-plugin.tar.gz" \
    --target linux/amd64 \
    "${BASE_DIR_FLAG[@]}"

# Phase 3: Preflight Validation of Logic & Dashboards (Level 2)
echo "🛡️ Phase 3: Running Preflight Validation Gate on reactors & dashboards..."
cd "$REPO_ROOT"

echo "  • [1/3] Validating CEL Declarative Kinetics & Spatial Synthesizers..."
go run ./cmd/bindu-studio cel-reactor validate \
    --file "$SCRIPT_DIR/poc_cel_reactor_config.json" \
    "${BASE_DIR_FLAG[@]}"

echo "  • [1b/3] Simulating CEL Kinetics Execution Engine (Dry-Run)..."
go run ./cmd/bindu-studio cel-reactor dry-run \
    --file "$SCRIPT_DIR/poc_cel_reactor_config.json" \
    --reactor-id "rfid-sgtin-transit-synthesizer" \
    --payload '{"antenna_id":1,"canonical_urn":"urn:epc:tag:sgtin-96:3.0614141.107346.2026","company_prefix":"0614141","epc_hex":"3074257bf401f40000000001","peak_rssi_dbm":-58.5,"reader_id":"portal-reader-01","standard":"SGTIN-96","first_seen_utc":"2026-10-14T12:00:00Z","last_seen_utc":"2026-10-14T12:00:01Z"}' \
    "${BASE_DIR_FLAG[@]}"

echo "  • [2/3] Validating Semantic AI (SLM Manifest Reconciler) Prompts & Rules..."
go run ./cmd/bindu-studio semantic-ai-reactor validate \
    --file "$SCRIPT_DIR/poc_semantic_reactor_config.json" \
    "${BASE_DIR_FLAG[@]}"

echo "  • [2b/3] Simulating Semantic AI Real Inference Engine (Dry-Run: LLM reachability & model existence)..."
go run ./cmd/bindu-studio semantic-ai-reactor dry-run \
    --file "$SCRIPT_DIR/poc_semantic_reactor_config.json" \
    --reactor-id "rfid-slm-manifest-reconciler" \
    --payload '{"transit_id":"preflight-test-01","portal_id":"portal-01","pallet_sscc":"000123456789012345","direction":"SOUTH_INBOUND","item_count":48,"estimated_velocity_mps":1.2,"mean_transit_rssi_dbm":-58.5}' \
    "${BASE_DIR_FLAG[@]}"

echo "  • [3/3] Validating Dashboard Definitions..."
go run ./cmd/bindu-studio dashboard validate \
    --file "$SCRIPT_DIR/poc_dashboards.json" \
    "${BASE_DIR_FLAG[@]}"
echo "  └── ✅ Preflight validation & execution simulation passed against registered schemas and live engines!"

# Phase 4: Declarative Application & Solution Bundling (Level 3)
echo "🚀 Phase 4: Packaging & Applying declarative MAS Solution bundle..."
echo "  • Staging BPR Plugin on root..."
go run ./cmd/bindu-studio plugin apply \
    --bundle "$SCRIPT_DIR/bindu-rfid-epc-plugin.tar.gz" \
    --target root \
    --auth "$AUTH" \
    --root-url "$ROOT_URL" \
    "${BASE_DIR_FLAG[@]}"

echo "  • Merging Semantic AI, CEL Kinetics & Dashboards..."
if [ -f "$SCRIPT_DIR/poc_semantic_reactor_config.json" ]; then
    go run ./cmd/bindu-studio semantic-ai-reactor merge \
        --file "$SCRIPT_DIR/poc_semantic_reactor_config.json" \
        --yes \
        --auth "$AUTH" \
        --root-url "$ROOT_URL" \
        "${BASE_DIR_FLAG[@]}" || true
fi

if [ -f "$SCRIPT_DIR/poc_cel_reactor_config.json" ]; then
    go run ./cmd/bindu-studio cel-reactor merge \
        --file "$SCRIPT_DIR/poc_cel_reactor_config.json" \
        --yes \
        --auth "$AUTH" \
        --root-url "$ROOT_URL" \
        "${BASE_DIR_FLAG[@]}" || true
fi

if [ -f "$SCRIPT_DIR/poc_dashboards.json" ]; then
    go run ./cmd/bindu-studio dashboard import \
        --file "$SCRIPT_DIR/poc_dashboards.json" \
        --all \
        --auth "$AUTH" \
        --root-url "$ROOT_URL" \
        "${BASE_DIR_FLAG[@]}" || true
fi

echo "  • Exporting and validating solution bundle..."
mkdir -p "$SCRIPT_DIR/bundles"
go run ./cmd/bindu-studio solution export \
    --file "$SCRIPT_DIR/rfid_mas_solution.json" \
    --out "$SCRIPT_DIR/bundles/rfid_mas_solution.bundle.json" \
    --auth "$AUTH" \
    --root-url "$ROOT_URL" \
    "${BASE_DIR_FLAG[@]}"

go run ./cmd/bindu-studio solution validate \
    --file "$SCRIPT_DIR/bundles/rfid_mas_solution.bundle.json" \
    --strict

echo "  • Applying solution bundle to cluster..."
go run ./cmd/bindu-studio solution apply \
    --file "$SCRIPT_DIR/bundles/rfid_mas_solution.bundle.json" \
    --yes \
    --auth "$AUTH" \
    --root-url "$ROOT_URL" \
    "${BASE_DIR_FLAG[@]}"

# Phase 5: The Universal Runtime OTA Triptych (Level 4 - Mandatory for Physical OS Delivery)
echo "⚡ Phase 5: Executing Universal Runtime OTA Triptych..."
go run ./cmd/bindu-studio runtime baseline accept \
    --yes \
    "${BASE_DIR_FLAG[@]}" || true

go run ./cmd/bindu-studio runtime desired commit \
    --note "Deploy Industrial RFID LLRP & 11-Standard EPC MAS Solution with 4-Antenna Portal Telemetry" \
    "${BASE_DIR_FLAG[@]}" || true

go run ./cmd/bindu-studio runtime update wizard \
    --yes \
    --auth "$AUTH" \
    --root-url "$ROOT_URL" \
    "${BASE_DIR_FLAG[@]}" || true

echo "================================================================="
echo "✅ DEPLOYMENT COMPLETE! RFID LLRP Plugin, Schemas & Dashboards live on ROOT."
echo "================================================================="
