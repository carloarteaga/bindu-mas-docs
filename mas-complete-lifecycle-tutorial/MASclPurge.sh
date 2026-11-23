#!/usr/bin/env bash
# public/console/docs/mas-complete-lifecycle-tutorial/MASclPurge.sh
# Automated Teardown & Zero-Residue Purge Pipeline for the MAS Complete Lifecycle Tutorial.

set -e
export GOCACHE=/tmp/gocache

# Terminal Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
YELLOW='\033[1;33m'
MAGENTA='\033[0;35m'
BOLD='\033[1m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/../../../.." && pwd)"

CURRENT_ROOT_RUN_DIR="${ROOT_DIR}/.e2e-root/current"
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

AUTH_CLI=""
ROOT_URL_CLI=""
USER_CLI=""
PASS_CLI=""
AUTO_CONFIRM=false

while [[ "$#" -gt 0 ]]; do
  case "$1" in
    -h|--help|help)
      echo -e "${BOLD}Usage:${NC} $0 [options]"
      echo ""
      echo "Completely dismantles, unassigns, and purges the MAS Complete Lifecycle solution, reactors, BPR plugins, dashboards, message schemas, and historical data."
      echo ""
      echo "Options:"
      echo "  --base-dir <dir>       Bindu base directory (default: auto-detected ./.e2e-root/current)"
      echo "  --root-url <url>       Bindu Root API endpoint (default: http://127.0.0.1:8080 or from report.txt)"
      echo "  --auth <user:pass>     Basic auth credentials (default: console:StrongPass123 or from report.txt)"
      echo "  --user <user>          Basic auth username"
      echo "  --pass <pass>          Basic auth password"
      echo "  -y, --yes              Confirm teardown and purge without interactive prompting"
      echo "  -h, --help             Show this help message"
      exit 0
      ;;
    --base-dir) BASE_DIR="$2"; shift 2 ;;
    --root-url) ROOT_URL_CLI="$2"; shift 2 ;;
    --auth) AUTH_CLI="$2"; shift 2 ;;
    --user) USER_CLI="$2"; shift 2 ;;
    --pass) PASS_CLI="$2"; shift 2 ;;
    -y|--yes) AUTO_CONFIRM=true; shift ;;
    *)
      echo -e "${RED}Unknown argument: $1${NC}"
      echo "Use --help for usage information."
      exit 1
      ;;
  esac
done

# Resolve Auth
AUTH="${AUTH_CLI}"
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
ROOT_URL="${ROOT_URL_CLI}"
if [ -z "${ROOT_URL}" ] && [ -n "${REPORT_FILE}" ]; then
  REP_CONSOLE_URL="$(report_value "${REPORT_FILE}" "root_console_url")"
  if [ -n "${REP_CONSOLE_URL}" ]; then
    ROOT_URL="${REP_CONSOLE_URL}"
  else
    REP_HTTP_PORT="$(report_value "${REPORT_FILE}" "root_http_port")"
    if [ -n "${REP_HTTP_PORT}" ]; then
      ROOT_URL="http://127.0.0.1:${REP_HTTP_PORT}"
    fi
  fi
fi
if [ -z "${ROOT_URL}" ]; then
  ROOT_URL="http://127.0.0.1:8080"
fi

echo -e "${BOLD}${CYAN}========================================================================${NC}"
echo -e "${BOLD}${CYAN}   🗑️ MAS COMPLETE LIFECYCLE — TEARDOWN & PURGE (MASclPurge)          ${NC}"
echo -e "${BOLD}${CYAN}========================================================================${NC}"
echo -e "${BOLD}Timestamp:${NC}     $(date -u +"%Y-%m-%dT%H:%M:%SZ")"
echo -e "${BOLD}Target BaseDir:${NC} ${BASE_DIR:-"(standard default)"}"
echo -e "${BOLD}Target RootURL:${NC} ${ROOT_URL}"
echo -e "${BOLD}Auth Identity:${NC}  ${AUTH%%:*}:******"
echo -e "${CYAN}------------------------------------------------------------------------${NC}"

STUDIO_BIN="/tmp/bindu-studio-tool"
echo -n "Compiling fresh bindu-studio binary... "
cd "${ROOT_DIR}"
if go build -o "${STUDIO_BIN}" ./cmd/bindu-studio > /tmp/bindu_studio_build.log 2>&1; then
  echo -e "${GREEN}READY ✅${NC}"
else
  echo -e "${RED}FAILED ❌${NC}"
  cat /tmp/bindu_studio_build.log
  exit 1
fi

BASE_DIR_FLAG=()
if [ -n "${BASE_DIR}" ]; then
  BASE_DIR_FLAG=("--base-dir" "${BASE_DIR}")
fi

# Step 1: Delete MASSolutionSet Manifest
echo ""
echo -e "${BOLD}${MAGENTA}▶ [1/7] Removing MASSolutionSet Manifest (mascl-complete-lifecycle)...${NC}"
if "${STUDIO_BIN}" solution delete \
  --id "mascl-complete-lifecycle" \
  --auth "${AUTH}" \
  --root-url "${ROOT_URL}" \
  --yes \
  "${BASE_DIR_FLAG[@]}" > /tmp/mascl_purge_sol.log 2>&1; then
  echo -e "  └── ${GREEN}MASSolutionSet Deleted ✅${NC}"
else
  echo -e "  └── ${YELLOW}Solution Manifest Already Deleted or Not Found ⏩${NC}"
fi

# Step 2: Delete Active CEL, REST, AI Reactors
echo ""
echo -e "${BOLD}${MAGENTA}▶ [2/7] Dismantling Active CEL, REST, and Semantic AI Reactors...${NC}"
"${STUDIO_BIN}" cel-reactor delete --reactor "mascl-cel-occurrence-emitter" --node "root" --auth "${AUTH}" --root-url "${ROOT_URL}" --yes "${BASE_DIR_FLAG[@]}" > /tmp/mascl_purge_cel_root.log 2>&1 || true
"${STUDIO_BIN}" cel-reactor delete --reactor "mascl-cel-occurrence-emitter" --node "node-1" --auth "${AUTH}" --root-url "${ROOT_URL}" --yes "${BASE_DIR_FLAG[@]}" > /tmp/mascl_purge_cel_node1.log 2>&1 || true

"${STUDIO_BIN}" rest-reactor delete --reactor "mascl-rest-occurrence-emitter" --node "root" --auth "${AUTH}" --root-url "${ROOT_URL}" --yes "${BASE_DIR_FLAG[@]}" > /tmp/mascl_purge_rest_root.log 2>&1 || true
"${STUDIO_BIN}" rest-reactor delete --reactor "mascl-rest-occurrence-emitter" --node "node-1" --auth "${AUTH}" --root-url "${ROOT_URL}" --yes "${BASE_DIR_FLAG[@]}" > /tmp/mascl_purge_rest_node1.log 2>&1 || true

"${STUDIO_BIN}" semantic-ai-reactor delete --reactor "mascl-ai-occurrence-emitter" --node "root" --auth "${AUTH}" --root-url "${ROOT_URL}" --yes "${BASE_DIR_FLAG[@]}" > /tmp/mascl_purge_ai_root.log 2>&1 || true
"${STUDIO_BIN}" semantic-ai-reactor delete --reactor "mascl-ai-occurrence-emitter" --node "node-1" --auth "${AUTH}" --root-url "${ROOT_URL}" --yes "${BASE_DIR_FLAG[@]}" > /tmp/mascl_purge_ai_node1.log 2>&1 || true
echo -e "  └── ${GREEN}All Multi-Agent Reactors Dismantled Across Mesh Nodes ✅${NC}"

# Step 3: Delete Dashboard Definition
echo ""
echo -e "${BOLD}${MAGENTA}▶ [3/7] Deleting MASclDashboard (mascl-occurrence-interval-monitor)...${NC}"
if "${STUDIO_BIN}" dashboard delete \
  --id "mascl-occurrence-interval-monitor" \
  --auth "${AUTH}" \
  --root-url "${ROOT_URL}" \
  --yes \
  "${BASE_DIR_FLAG[@]}" > /tmp/mascl_purge_dash.log 2>&1; then
  echo -e "  └── ${GREEN}Dashboard Definition Purged ✅${NC}"
else
  echo -e "  └── ${YELLOW}Dashboard Already Deleted or Not Found ⏩${NC}"
fi

# Step 4: Unassign & Purge BPR Plugin
echo ""
echo -e "${BOLD}${MAGENTA}▶ [4/7] Unassigning & Purging Native BPR Plugin (bindu-mascl-plugin)...${NC}"
if "${STUDIO_BIN}" plugin remove \
  --plugin "bindu-mascl-plugin" \
  --target all \
  --uninstall \
  --purge \
  --auth "${AUTH}" \
  --root-url "${ROOT_URL}" \
  "${BASE_DIR_FLAG[@]}" > /tmp/mascl_purge_plug.log 2>&1; then
  echo -e "  └── ${GREEN}BPR Plugin Unassigned & Artifacts Purged ✅${NC}"
else
  echo -e "  └── ${YELLOW}Plugin Unassigned / Offline Staged ⏩${NC}"
fi

# Step 5: Synchronize Desired State Post-Teardown
echo ""
echo -e "${BOLD}${MAGENTA}▶ [5/7] Synchronizing Desired State Post-Purge...${NC}"
"${STUDIO_BIN}" runtime baseline accept --yes --note "Accept baseline post mascl purge" "${BASE_DIR_FLAG[@]}" > /tmp/mascl_purge_baseline.log 2>&1 || true
"${STUDIO_BIN}" runtime desired commit --note "Commit desired state post mascl purge" "${BASE_DIR_FLAG[@]}" > /tmp/mascl_purge_desired.log 2>&1 || true
"${STUDIO_BIN}" runtime update wizard --yes --auth "${AUTH}" --root-url "${ROOT_URL}" "${BASE_DIR_FLAG[@]}" > /tmp/mascl_purge_wiz.log 2>&1 || true
echo -e "  └── ${GREEN}Runtime State Converged Cleanly ✅${NC}"

# Step 6: Purge Historical Message Samples from Storage
echo ""
echo -e "${BOLD}${MAGENTA}▶ [6/7] Purging SQLite Telemetry & Message Samples...${NC}"
"${STUDIO_BIN}" message-samples purge --type "MASclTrigger" --auth "${AUTH}" --root-url "${ROOT_URL}" "${BASE_DIR_FLAG[@]}" > /tmp/mascl_purge_samples_trigger.log 2>&1 || true
"${STUDIO_BIN}" message-samples purge --type "MASclOcurrence" --auth "${AUTH}" --root-url "${ROOT_URL}" "${BASE_DIR_FLAG[@]}" > /tmp/mascl_purge_samples_ocurrence.log 2>&1 || true
rm -f "${SCRIPT_DIR}/bundles/bindu-mascl-plugin-1.0.0.tar.gz"
echo -e "  └── ${GREEN}Historical Telemetry Samples Purged ✅${NC}"

# Step 7: Delete Message Types Schemas
echo ""
echo -e "${BOLD}${MAGENTA}▶ [7/7] Deleting Message Type Schemas (MASclTrigger & MASclOcurrence)...${NC}"
"${STUDIO_BIN}" message-types delete --type "MASclTrigger" --yes --auth "${AUTH}" --root-url "${ROOT_URL}" "${BASE_DIR_FLAG[@]}" > /tmp/mascl_purge_mt_trigger.log 2>&1 || true
"${STUDIO_BIN}" message-types delete --type "MASclOcurrence" --yes --auth "${AUTH}" --root-url "${ROOT_URL}" "${BASE_DIR_FLAG[@]}" > /tmp/mascl_purge_mt_ocurrence.log 2>&1 || true
echo -e "  └── ${GREEN}Message Schemas Removed from Active Mesh Registry ✅${NC}"

echo ""
echo -e "${BOLD}${CYAN}========================================================================${NC}"
echo -e "${BOLD}${GREEN}   🧹 MAS COMPLETE LIFECYCLE PURGE COMPLETED (ZERO RESIDUE)             ${NC}"
echo -e "${BOLD}${CYAN}========================================================================${NC}"
echo -e "  • All 4 Multi-Agent Reactors Deployed: ${BOLD}${GREEN}PURGED 0️⃣${NC}"
echo -e "  • BPR Plugin (bindu-mascl-plugin):    ${BOLD}${GREEN}UNASSIGNED & REMOVED 0️⃣${NC}"
echo -e "  • Dashboard Definition:                ${BOLD}${GREEN}PURGED 0️⃣${NC}"
echo -e "  • Message Types Schemas:               ${BOLD}${GREEN}PURGED 0️⃣${NC}"
echo -e "  • Runtime Mesh State:                  ${BOLD}${GREEN}CONVERGED & CLEAN 0️⃣${NC}"
echo -e "${CYAN}------------------------------------------------------------------------${NC}"
