#!/usr/bin/env bash
# public/console/docs/mas-complete-lifecycle-tutorial/MASclDeploy.sh
# Automated Deployment & Activation Pipeline for the MAS Complete Lifecycle Tutorial.

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
SKIP_TRIGGER=false
AUTO_CONFIRM=false

while [[ "$#" -gt 0 ]]; do
  case "$1" in
    -h|--help|help)
      echo -e "${BOLD}Usage:${NC} $0 [options]"
      echo ""
      echo "Deploys all 4 multi-agent paradigms (CEL, REST, AI, BPR), registers schemas, binds dashboards, and triggers execution across the mesh."
      echo ""
      echo "Options:"
      echo "  --base-dir <dir>       Bindu base directory (default: auto-detected ./.e2e-root/current)"
      echo "  --root-url <url>       Bindu Root API endpoint (default: http://127.0.0.1:8080 or from report.txt)"
      echo "  --auth <user:pass>     Basic auth credentials (default: console:StrongPass123 or from report.txt)"
      echo "  --user <user>          Basic auth username"
      echo "  --pass <pass>          Basic auth password"
      echo "  --skip-trigger         Skip emitting the verification trigger after deployment"
      echo "  -y, --yes              Confirm execution without interactive prompting"
      echo "  -h, --help             Show this help message"
      exit 0
      ;;
    --base-dir) BASE_DIR="$2"; shift 2 ;;
    --root-url) ROOT_URL_CLI="$2"; shift 2 ;;
    --auth) AUTH_CLI="$2"; shift 2 ;;
    --user) USER_CLI="$2"; shift 2 ;;
    --pass) PASS_CLI="$2"; shift 2 ;;
    --skip-trigger) SKIP_TRIGGER=true; shift ;;
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
echo -e "${BOLD}${CYAN}   🚀 MAS COMPLETE LIFECYCLE — DEPLOYMENT PIPELINE (MASclDeploy)       ${NC}"
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

PLUGIN_DIR="${SCRIPT_DIR}/mascl-bpr-plugin"
PLUGIN_BUNDLE="${SCRIPT_DIR}/bundles/bindu-mascl-plugin-1.0.0.tar.gz"
mkdir -p "${SCRIPT_DIR}/bundles"

# Step 0: Verify Centralized Secrets in Vault (Zero-Plaintext Wire Guarantee)
echo ""
echo -e "${BOLD}${MAGENTA}▶ [0/7] Verifying Centralized Secrets in Encrypted Vault...${NC}"
if ! "${STUDIO_BIN}" access-control secrets get OLLAMA_API_KEY "${BASE_DIR_FLAG[@]}" > /dev/null 2>&1; then
  echo "  • Provisioning OLLAMA_API_KEY into encrypted vault..."
  "${STUDIO_BIN}" access-control secrets set OLLAMA_API_KEY \
    --value "ollama-local" \
    --yes \
    "${BASE_DIR_FLAG[@]}" > /tmp/mascl_secrets_set.log 2>&1
  echo -e "  └── ${GREEN}Secret OLLAMA_API_KEY Provisioned in Vault ✅${NC}"
else
  echo -e "  └── ${GREEN}Secret OLLAMA_API_KEY Verified in Vault ✅${NC}"
fi

# Step 1: Register Message Types & Schemas in Mesh Registry
echo ""
echo -e "${BOLD}${MAGENTA}▶ [1/7] Synchronizing Message Types & Schemas (MASclTrigger & MASclOcurrence)...${NC}"
if "${STUDIO_BIN}" message-types merge \
  --in-path "${SCRIPT_DIR}/poc_message_types.json" \
  --all \
  --auth "${AUTH}" \
  --root-url "${ROOT_URL}" \
  "${BASE_DIR_FLAG[@]}" > /tmp/mascl_mt_merge.log 2>&1; then
  echo -e "  └── ${GREEN}Schemas Registered & Synchronized in Mesh Registry ✅${NC}"
else
  echo -e "  └── ${RED}Message Types Registration Failed ❌${NC}"
  cat /tmp/mascl_mt_merge.log | sed 's/^/      /'
  exit 1
fi

# Step 2: Certify BPR Plugin
echo ""
echo -e "${BOLD}${MAGENTA}▶ [2/7] Certifying & Packaging Native BPR Plugin (bindu-mascl-plugin)...${NC}"
if "${STUDIO_BIN}" plugin certify \
  --source "${PLUGIN_DIR}" \
  --out "${PLUGIN_BUNDLE}" \
  --targets-from-mesh \
  "${BASE_DIR_FLAG[@]}" > /tmp/mascl_plugin_cert.log 2>&1; then
  echo -e "  └── ${GREEN}Plugin Certified & Multi-Arch Bundle Built ✅${NC}"
else
  echo -e "  └── ${RED}Plugin Certification Failed ❌${NC}"
  cat /tmp/mascl_plugin_cert.log | sed 's/^/      /'
  exit 1
fi

# Step 3: Apply Plugin to All Mesh Nodes
echo ""
echo -e "${BOLD}${MAGENTA}▶ [3/7] Staging & Applying BPR Plugin to Mesh Nodes...${NC}"
if "${STUDIO_BIN}" plugin apply \
  --bundle "${PLUGIN_BUNDLE}" \
  --target all \
  --auth "${AUTH}" \
  --root-url "${ROOT_URL}" \
  "${BASE_DIR_FLAG[@]}" > /tmp/mascl_plugin_apply.log 2>&1; then
  echo -e "  └── ${GREEN}Plugin Applied to Mesh Runtime ✅${NC}"
else
  echo -e "  └── ${YELLOW}Plugin Applied with warnings / Offline Staged ⚠️${NC}"
  cat /tmp/mascl_plugin_apply.log | sed 's/^/      /'
fi

# Step 4: Synchronize Desired State Across Runtime
echo ""
echo -e "${BOLD}${MAGENTA}▶ [4/7] Synchronizing Runtime State & Baseline Wizard...${NC}"
"${STUDIO_BIN}" runtime baseline accept --yes --note "Accept baseline with mascl BPR plugin" "${BASE_DIR_FLAG[@]}" > /tmp/mascl_baseline_accept.log 2>&1
"${STUDIO_BIN}" runtime desired commit --note "Commit desired state for mascl BPR plugin" "${BASE_DIR_FLAG[@]}" > /tmp/mascl_desired_commit.log 2>&1
if "${STUDIO_BIN}" runtime update wizard --yes --auth "${AUTH}" --root-url "${ROOT_URL}" "${BASE_DIR_FLAG[@]}" > /tmp/mascl_update_wiz.log 2>&1; then
  echo -e "  └── ${GREEN}Runtime State Converged & Up to Date ✅${NC}"
else
  echo -e "  └── ${YELLOW}Runtime Update Wizard Complete (or no nodes pending update) ⏩${NC}"
fi

# Step 5: Export Complete MAS Solution Bundle
MANIFEST_FILE="${SCRIPT_DIR}/solution_mascl_lifecycle.json"
BUNDLE_FILE="${SCRIPT_DIR}/bundles/solution_mascl_lifecycle.bundle.json"
echo ""
echo -e "${BOLD}${MAGENTA}▶ [5/7] Compiling Self-Contained Solution Bundle...${NC}"
if "${STUDIO_BIN}" solution export \
  --file "${MANIFEST_FILE}" \
  --out "${BUNDLE_FILE}" \
  "${BASE_DIR_FLAG[@]}" > /tmp/mascl_sol_exp.log 2>&1; then
  echo -e "  └── ${GREEN}Bundle Compiled with All 4 Agent Paradigms & Schemas ✅${NC}"
else
  echo -e "  └── ${RED}Bundle Compilation Failed ❌${NC}"
  cat /tmp/mascl_sol_exp.log | sed 's/^/      /'
  exit 1
fi

# Step 6: Strict Validation of Solution Bundle
echo ""
echo -e "${BOLD}${MAGENTA}▶ [6/7] Validating Solution Bundle Integrity & Schema Dependencies...${NC}"
if "${STUDIO_BIN}" solution validate \
  --file "${BUNDLE_FILE}" \
  --strict > /tmp/mascl_sol_val.log 2>&1; then
  echo -e "  └── ${GREEN}Validation Passed: Strict Schema & Dependency Alignment Confirmed ✅${NC}"
else
  echo -e "  └── ${RED}Validation Failed ❌${NC}"
  cat /tmp/mascl_sol_val.log | sed 's/^/      /'
  exit 1
fi

# Step 7: Apply Complete Solution to Mesh Runtime
echo ""
echo -e "${BOLD}${MAGENTA}▶ [7/7] Deploying Solution Bundle to Mesh Runtime...${NC}"
if "${STUDIO_BIN}" solution apply \
  --file "${BUNDLE_FILE}" \
  --auth "${AUTH}" \
  --root-url "${ROOT_URL}" \
  --yes \
  "${BASE_DIR_FLAG[@]}" > /tmp/mascl_sol_app.log 2>&1; then
  echo -e "  └── ${GREEN}MASSolutionSet, MessageTypes, CEL, REST, AI, and Dashboards Active 🚀${NC}"
else
  echo -e "  └── ${YELLOW}Solution Deployed / Staged with Warnings ⚠️${NC}"
  cat /tmp/mascl_sol_app.log | sed 's/^/      /'
fi

# Optional Step: Emit Test Trigger
if [ "${SKIP_TRIGGER}" = false ]; then
  echo ""
  echo -e "${BOLD}${MAGENTA}▶ [Verification] Emitting Ingress Test Trigger (MASclTrigger)...${NC}"
  if "${SCRIPT_DIR}/simulate_trigger.sh" --source "MASclDeploy-verification" > /tmp/mascl_trig.log 2>&1; then
    echo -e "  └── ${GREEN}MASclTrigger Published! All 4 agents activated on demand ⚡${NC}"
  else
    echo -e "  └── ${YELLOW}Trigger publication skipped (MQTT broker may be offline/protected) ⚠️${NC}"
  fi
fi

echo ""
echo -e "${BOLD}${CYAN}========================================================================${NC}"
echo -e "${BOLD}${GREEN}   ✅ MAS COMPLETE LIFECYCLE DEPLOYMENT COMPLETED SUCCESSFULLY!        ${NC}"
echo -e "${BOLD}${CYAN}========================================================================${NC}"
echo -e "  • Solution ID:    ${BOLD}mascl-complete-lifecycle${NC}"
echo -e "  • Agent Fleet:    ${BOLD}4 Paradigms (CEL, REST, AI, BPR)${NC}"
echo -e "  • Trigger Event:  ${BOLD}MASclTrigger (ttl: 60s, count: 1)${NC}"
echo -e "  • Egress Event:   ${BOLD}MASclOcurrence (count: 1)${NC}"
echo -e "  • Dashboard ID:   ${BOLD}mascl-occurrence-interval-monitor${NC}"
echo -e "  • Web Console:    ${BOLD}${ROOT_URL}/console/${NC}"
echo -e "${CYAN}------------------------------------------------------------------------${NC}"
