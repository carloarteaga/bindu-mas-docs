#!/usr/bin/env bash
# ==============================================================================
# 🚀 Deployment Script for Heavy Equipment CBM & Fleet Health MAS Solution
#
# Sequence:
#   1) Merge message types from poc_message_types.json
#   2) Merge CEL reactors from poc_cel_reactor_config.json
#   3) Merge Semantic AI reactors from poc_semantic_reactor_config.json
#   4) Merge dashboards from poc_dashboards.json
#   5) Register MAS solution from mining_mas_solution.json
# ==============================================================================
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
      echo "Deploys the Heavy Equipment Vibration CBM & Fleet Utilization MAS solution."
      echo ""
      echo "Sequence:"
      echo "  1) Merge message types (poc_message_types.json)"
      echo "  2) Merge CEL-reactors (poc_cel_reactor_config.json)"
      echo "  3) Merge Semantic AI reactors (poc_semantic_reactor_config.json)"
      echo "  4) Merge dashboards (poc_dashboards.json)"
      echo "  5) Register MAS solution (mining_mas_solution.json)"
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
    REP_CONSOLE_LOGIN="$(report_value "${REPORT_FILE}" "root_console_login")"
    if [ -n "${REP_CONSOLE_LOGIN}" ]; then
      AUTH="${REP_CONSOLE_LOGIN}"
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
echo "🚜 BINDU MAS DEPLOYMENT: HEAVY EQUIPMENT CBM & FLEET HEALTH"
echo "================================================================="
echo "Target BaseDir: ${BASE_DIR}"
echo "Target RootURL: ${ROOT_URL}"
echo "Auth Principal: ${AUTH%%:*}:******"
echo "-----------------------------------------------------------------"

cd "$REPO_ROOT"

# Step 1: Merge Message Types
echo "📜 [1/5] Merging mining machinery message schemas into Root..."
GOCACHE=/tmp/gocache go run ./cmd/bindu-studio message-types merge \
    --in-path "$SCRIPT_DIR/poc_message_types.json" \
    --all \
    --auth "$AUTH" \
    --root-url "$ROOT_URL" \
    "${BASE_DIR_FLAG[@]}"

# Step 2: Merge CEL Reactors
if [ -f "$SCRIPT_DIR/poc_cel_reactor_config.json" ]; then
  echo "⚡ [2/5] Merging CEL kinetic reactors from poc_cel_reactor_config.json..."
  GOCACHE=/tmp/gocache go run ./cmd/bindu-studio cel-reactor merge \
      --file "$SCRIPT_DIR/poc_cel_reactor_config.json" \
      --yes \
      --auth "$AUTH" \
      --root-url "$ROOT_URL" \
      "${BASE_DIR_FLAG[@]}"
fi

# Step 3: Merge Semantic AI Reactors
if [ -f "$SCRIPT_DIR/poc_semantic_reactor_config.json" ]; then
  echo "🧠 [3/5] Merging Semantic AI reactors from poc_semantic_reactor_config.json..."
  GOCACHE=/tmp/gocache go run ./cmd/bindu-studio semantic-ai-reactor merge \
      --file "$SCRIPT_DIR/poc_semantic_reactor_config.json" \
      --yes \
      --auth "$AUTH" \
      --root-url "$ROOT_URL" \
      "${BASE_DIR_FLAG[@]}"
fi

# Step 4: Merge Dashboards
if [ -f "$SCRIPT_DIR/poc_dashboards.json" ]; then
  echo "📊 [4/5] Merging dashboards from poc_dashboards.json..."
  GOCACHE=/tmp/gocache go run ./cmd/bindu-studio dashboard merge \
      --file "$SCRIPT_DIR/poc_dashboards.json" \
      --yes \
      --auth "$AUTH" \
      --root-url "$ROOT_URL" \
      "${BASE_DIR_FLAG[@]}"
fi

# Step 5: Register MAS Solution
echo "🧩 [5/5] Registering mining MAS solution from mining_mas_solution.json..."
GOCACHE=/tmp/gocache go run ./cmd/bindu-studio solution register \
    --file "$SCRIPT_DIR/mining_mas_solution.json" \
    --yes \
    --auth "$AUTH" \
    --root-url "$ROOT_URL" \
    "${BASE_DIR_FLAG[@]}"

echo "================================================================="
echo "✅ DEPLOYMENT COMPLETE! Heavy Equipment CBM MAS Solution is live."
echo "================================================================="
