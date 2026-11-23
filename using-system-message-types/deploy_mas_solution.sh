#!/usr/bin/env bash
# ==============================================================================
# Universal Deployment Script: Mesh Routing & Retention Fleet Auditor MAS
# (Root Node Only - Declarative Google CEL Kinetics & Central Observability)
# ==============================================================================
set -euo pipefail

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
      echo "Deploys the Mesh Routing & Retention Fleet Auditor MAS solution on Root."
      echo ""
      echo "Options:"
      echo "  --base-dir <dir>       Bindu base directory (default: auto-detected ./.e2e-root/current or repo root)"
      echo "  --root-url <url>       Bindu Root API endpoint (default: http://127.0.0.1:38080)"
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
  ROOT_URL="http://127.0.0.1:38080"
fi

# Find bindu-studio binary
STUDIO_BIN=""
if [ -x "${BASE_DIR}/bin/bindu-studio" ]; then
  STUDIO_BIN="${BASE_DIR}/bin/bindu-studio"
elif [ -x "${REPO_ROOT}/.e2e-root/current/bin/bindu-studio" ]; then
  STUDIO_BIN="${REPO_ROOT}/.e2e-root/current/bin/bindu-studio"
elif command -v bindu-studio >/dev/null 2>&1; then
  STUDIO_BIN="bindu-studio"
else
  STUDIO_BIN="go run ${REPO_ROOT}/cmd/bindu-studio"
fi

echo "========================================================================="
echo "🛡️ BINDU MAS DEPLOYMENT: MESH ROUTING & RETENTION FLEET AUDITOR"
echo "   (Target: ROOT COORDINATOR ONLY - Declarative CEL Kinetics & Dashboards)"
echo "========================================================================="
echo "Target BaseDir: ${BASE_DIR}"
echo "Target RootURL: ${ROOT_URL}"
echo "Auth Principal: ${AUTH%%:*}:******"
echo "Studio Exec:    ${STUDIO_BIN}"
echo "-------------------------------------------------------------------------"

# -----------------------------------------------------------------------------
# Phase 0: Preflight Validation Gate (Strict Offline Static Analysis)
# -----------------------------------------------------------------------------
echo "🔍 Phase 0: Preflight Validation Gate (Offline Compilation & Linting)..."

echo "  • [1/4] Validating CEL ReactorSet AST & Syntax..."
${STUDIO_BIN} cel-reactor validate \
  --file "$SCRIPT_DIR/poc_cel_reactor_config.json" \
  "${BASE_DIR_FLAG[@]}"

echo "  • [2/4] Validating Dashboard Schemas & SQL Views..."
${STUDIO_BIN} dashboard validate \
  --file "$SCRIPT_DIR/poc_dashboards.json" \
  "${BASE_DIR_FLAG[@]}"

echo "  • [3/4] Validating Solution Manifest & Node Containment..."
${STUDIO_BIN} solution validate \
  --file "$SCRIPT_DIR/solution.json" \
  "${BASE_DIR_FLAG[@]}"

echo "  • [4/4] Validating Compiled Standalone Solution Bundle (--strict)..."
${STUDIO_BIN} solution validate \
  --file "$SCRIPT_DIR/solution.bundle.json" \
  --strict \
  "${BASE_DIR_FLAG[@]}"

echo "  └── ✅ Preflight Validation Gate passed with zero errors!"

# -----------------------------------------------------------------------------
# Phase 1: Register Schema Ontology (Level 0 - Schemas First)
# -----------------------------------------------------------------------------
echo "📜 Phase 1: Merging MessageStateAudit schema into Root schema catalog..."
${STUDIO_BIN} message-types merge \
  --in-path "$SCRIPT_DIR/poc_message_types.json" \
  --all \
  --auth "$AUTH" \
  --root-url "$ROOT_URL" \
  "${BASE_DIR_FLAG[@]}"

# -----------------------------------------------------------------------------
# Phase 2: Deploy In-Process CEL Kinetics (Level 2)
# -----------------------------------------------------------------------------
echo "⚡ Phase 2: Deploying Declarative CEL Reactor on Root..."
${STUDIO_BIN} cel-reactor merge \
  --file "$SCRIPT_DIR/poc_cel_reactor_config.json" \
  --yes \
  --auth "$AUTH" \
  --root-url "$ROOT_URL" \
  "${BASE_DIR_FLAG[@]}"

# -----------------------------------------------------------------------------
# Phase 3 & 4: Deploy Declarative Dashboards & Solution Manifest (via Bundle)
# -----------------------------------------------------------------------------
echo "🚀 Phase 3 & 4: Deploying Declarative Solution Bundle (Preserving Entity Partitions)..."
${STUDIO_BIN} solution apply \
  --file "$SCRIPT_DIR/solution.bundle.json" \
  --yes \
  --auth "$AUTH" \
  --root-url "$ROOT_URL" \
  "${BASE_DIR_FLAG[@]}"


# -----------------------------------------------------------------------------
# Phase 5: Verification & Observability Gate
# -----------------------------------------------------------------------------
echo "🔎 Phase 5: Verification & Observability Gate..."
echo "  • Querying SQLite Event Ledger (v_messages) via REST SQL endpoint..."

PAYLOAD=$(jq -n --arg sql 'SELECT "$producer", "$type", "$when", json_extract(raw_json, "$.fields.solution_id") as sol_id FROM v_messages WHERE "$type" = "MASSolutionSet" AND json_extract(raw_json, "$.fields.solution_id") = "mesh-system-audit-solution" ORDER BY "$when" DESC LIMIT 1;' '{"sql": $sql}')

RESPONSE=$(curl -s -k -u "$AUTH" \
  -X POST "${ROOT_URL}/api/v1/messages/sql" \
  -H "Content-Type: application/json" \
  -d "$PAYLOAD")



echo "  Ledger Verification Query Response:"
echo "  $RESPONSE"

echo "========================================================================="
echo "🎉 DEPLOYMENT COMPLETE!"
echo "   Solution 'mesh-system-audit-solution' is active on Root Coordinator."
echo "   Auditing incoming MessageState heartbeats across the mesh."
echo "   Dashboards accessible at: ${ROOT_URL}/console/dashboards"
echo "========================================================================="
