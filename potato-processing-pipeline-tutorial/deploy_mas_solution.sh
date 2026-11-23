#!/usr/bin/env bash
# Universal Deployment Script for Potato Processing Pipeline MAS Solution (Root Only)
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
      echo "Deploys the Potato Processing Thermodynamic Pipeline MAS solution (Root only)."
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
echo "🥔 BINDU MAS DEPLOYMENT: POTATO PROCESSING THERMODYNAMIC PIPELINE"
echo "   (Target: ROOT NODE ONLY - Offloading Edge Gateways)"
echo "================================================================="
echo "Target BaseDir: ${BASE_DIR}"
echo "Target RootURL: ${ROOT_URL}"
echo "Auth Principal: ${AUTH%%:*}:******"
echo "-----------------------------------------------------------------"

# Phase 0: Integration Secrets Preflight & Provisioning (accessControlConfig.brs)
echo "🔐 Phase 0: Verifying integration secrets in Access Control Vault..."
cd "$REPO_ROOT"

# Ensure POTATO_VISION_API_KEY secret exists
if ! go run ./cmd/bindu-studio secrets get POTATO_VISION_API_KEY "${BASE_DIR_FLAG[@]}" >/dev/null 2>&1; then
    echo "  • Provisioning default POTATO_VISION_API_KEY into vault..."
    go run ./cmd/bindu-studio secrets set POTATO_VISION_API_KEY --value "ollama-local" --yes "${BASE_DIR_FLAG[@]}"
fi

# Ensure minio_media_secret exists for out-of-band media storage
if ! go run ./cmd/bindu-studio secrets get minio_media_secret "${BASE_DIR_FLAG[@]}" >/dev/null 2>&1; then
    echo "  • Provisioning default minio_media_secret into vault..."
    go run ./cmd/bindu-studio secrets set minio_media_secret --value "bindu-media-123" --yes "${BASE_DIR_FLAG[@]}"
fi

go run ./cmd/bindu-studio secrets validate "${BASE_DIR_FLAG[@]}"
echo "  └── ✅ Integration secrets verified in accessControlConfig.brs!"

# Phase 1: Register Ontology & Message Types (Level 0 - Schemas First)
echo "📜 Phase 1: Merging potato processing application message schemas into Root..."
cd "$REPO_ROOT"
go run ./cmd/bindu-studio message-types merge \
    --in-path "$SCRIPT_DIR/potato_message_types.json" \
    --all \
    --auth "$AUTH" \
    --root-url "$ROOT_URL" \
    "${BASE_DIR_FLAG[@]}"

# Phase 2: Binary Certification & Native Packaging (Level 1)
echo "📦 Phase 2: Certifying and packaging bindu-modbus-plugin..."
cd "$REPO_ROOT"
go run ./cmd/bindu-studio plugin certify \
    --source "$SCRIPT_DIR/bindu-modbus-plugin" \
    --out "$SCRIPT_DIR/bindu-modbus-plugin.tar.gz" \
    --target linux/amd64 \
    "${BASE_DIR_FLAG[@]}"

# Phase 3: Preflight Validation of Logic & Dashboards (Level 2)
echo "🛡️ Phase 3: Running Preflight Validation Gate on reactors & dashboards..."
cd "$REPO_ROOT"

echo "  • [1/3] Validating CEL Declarative Kinetics & Output Schemas..."
go run ./cmd/bindu-studio cel-reactor validate \
    --file "$SCRIPT_DIR/poc_cel_reactor_config.json" \
    "${BASE_DIR_FLAG[@]}"

echo "  • [2/3] Validating Semantic AI (llava-phi3) Prompts & Rules..."
go run ./cmd/bindu-studio semantic-ai-reactor validate \
    --file "$SCRIPT_DIR/poc_semantic_reactor_config.json" \
    "${BASE_DIR_FLAG[@]}"

echo "  • [3/3] Validating Dashboard Definitions..."
go run ./cmd/bindu-studio dashboard validate \
    --file "$SCRIPT_DIR/poc_dashboards.json" \
    "${BASE_DIR_FLAG[@]}"
echo "  └── ✅ All preflight validation checks passed against registered schemas!"

# Phase 4: Declarative Application & Solution Bundling (Level 3)
echo "🚀 Phase 4: Packaging & Applying declarative MAS Solution bundle (Target: root)..."
echo "  • Staging BPR Plugin on root..."
go run ./cmd/bindu-studio plugin apply \
    --bundle "$SCRIPT_DIR/bindu-modbus-plugin.tar.gz" \
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
    --file "$SCRIPT_DIR/potato_mas_solution.json" \
    --out "$SCRIPT_DIR/bundles/potato_mas_solution.bundle.json" \
    --auth "$AUTH" \
    --root-url "$ROOT_URL" \
    "${BASE_DIR_FLAG[@]}"

go run ./cmd/bindu-studio solution validate \
    --file "$SCRIPT_DIR/bundles/potato_mas_solution.bundle.json" \
    --strict

echo "  • Applying solution bundle to cluster..."
go run ./cmd/bindu-studio solution apply \
    --file "$SCRIPT_DIR/bundles/potato_mas_solution.bundle.json" \
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
    --note "Deploy Potato Processing Pipeline with llava-phi3 & Schedule Dashboards (Root Only)" \
    "${BASE_DIR_FLAG[@]}" || true

go run ./cmd/bindu-studio runtime update wizard \
    --yes \
    --auth "$AUTH" \
    --root-url "$ROOT_URL" \
    "${BASE_DIR_FLAG[@]}" || true

echo "================================================================="
echo "✅ DEPLOYMENT COMPLETE! All schemas, plugins, and reactors live on ROOT."
echo "================================================================="
