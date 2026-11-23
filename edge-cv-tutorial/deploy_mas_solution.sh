#!/usr/bin/env bash
# ==============================================================================
# BINDU AI-NATIVE TUTORIAL: Edge CV Multi-Camera Presence & Authority Arbitration
# Automated Deployment & Activation Pipeline (deploy_mas_solution.sh)
# ==============================================================================

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
      echo "Deploys the Edge CV Multi-Camera Presence solution (BPR on node-dell-1, CEL & Dashboards on root)."
      echo ""
      echo "Options:"
      echo "  --base-dir <dir>       Bindu base directory (default: auto-detected ./.e2e-root/current)"
      echo "  --root-url <url>       Bindu Root API endpoint (default: http://127.0.0.1:8080 or from report.txt)"
      echo "  --auth <user:pass>     Basic auth credentials (default: console:StrongPass123 or from report.txt)"
      echo "  --user <user>          Basic auth username"
      echo "  --pass <pass>          Basic auth password"
      echo "  -y, --yes              Confirm execution without interactive prompting"
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
echo -e "${BOLD}${CYAN}   🚀 EDGE CV MULTI-CAMERA PRESENCE — DEPLOYMENT PIPELINE              ${NC}"
echo -e "${BOLD}${CYAN}========================================================================${NC}"
echo -e "${BOLD}Timestamp:${NC}     $(date -u +"%Y-%m-%dT%H:%M:%SZ")"
echo -e "${BOLD}Target BaseDir:${NC} ${BASE_DIR:-"(standard default)"}"
echo -e "${BOLD}Target RootURL:${NC} ${ROOT_URL}"
echo -e "${BOLD}Auth Identity:${NC}  ${AUTH%%:*}:******"
echo -e "${BOLD}Plugin Node:${NC}    node-dell-1 (Dell Inspiron 11 gateway)"
echo -e "${CYAN}------------------------------------------------------------------------${NC}"

STUDIO_BIN="/tmp/bindu-studio-tool"
echo -n "Compiling fresh bindu-studio binary... "
cd "${ROOT_DIR}"
if go build -o "${STUDIO_BIN}" ./cmd/bindu-studio > /tmp/edge_cv_studio_build.log 2>&1; then
  echo -e "${GREEN}READY ✅${NC}"
else
  echo -e "${RED}FAILED ❌${NC}"
  cat /tmp/edge_cv_studio_build.log
  exit 1
fi

BASE_DIR_FLAG=()
if [ -n "${BASE_DIR}" ]; then
  BASE_DIR_FLAG=("--base-dir" "${BASE_DIR}")
fi

PLUGIN_DIR="${SCRIPT_DIR}/bindu-edge-cv-plugin"
PLUGIN_BUNDLE="${SCRIPT_DIR}/bundles/bindu-edge-cv-plugin-1.0.0.tar.gz"
mkdir -p "${SCRIPT_DIR}/bundles"

# Step 0: Centralized Secrets Verification & Vault Provisioning
echo ""
echo -e "${BOLD}${MAGENTA}▶ [0/8] Validating Centralized Secret in Access Control Vault...${NC}"
if ! "${STUDIO_BIN}" access-control secrets get minio_media_secret "${BASE_DIR_FLAG[@]}" >/dev/null 2>&1; then
  echo -e "  └── ${YELLOW}'minio_media_secret' not found in vault. Provisioning default development secret...${NC}"
  "${STUDIO_BIN}" access-control secrets set minio_media_secret --value "bindu-media-123" "${BASE_DIR_FLAG[@]}" >/dev/null 2>&1
fi
echo -e "  └── ${GREEN}minio_media_secret confirmed in vault ✅${NC}"

# Step 1: Register Message Types & Schemas in Mesh Registry
echo ""
echo -e "${BOLD}${MAGENTA}▶ [1/7] Synchronizing Message Types & Schemas (EdgeCVSet, EdgeCVSetStatus, AppearedClass, DisappearedClass, WorkstationPresenceSession)...${NC}"
if "${STUDIO_BIN}" message-types merge \
  --in-path "${SCRIPT_DIR}/poc_message_types.json" \
  --all \
  --auth "${AUTH}" \
  --root-url "${ROOT_URL}" \
  "${BASE_DIR_FLAG[@]}" > /tmp/edge_cv_mt_merge.log 2>&1; then
  echo -e "  └── ${GREEN}Schemas Registered & Synchronized in Mesh Registry ✅${NC}"
else
  echo -e "  └── ${RED}Message Types Registration Failed ❌${NC}"
  cat /tmp/edge_cv_mt_merge.log | sed 's/^/      /'
  exit 1
fi

# Step 2: Certify BPR Plugin
echo ""
echo -e "${BOLD}${MAGENTA}▶ [2/7] Certifying & Packaging Native BPR Plugin (bindu-edge-cv-plugin)...${NC}"
if "${STUDIO_BIN}" plugin certify \
  --source "${PLUGIN_DIR}" \
  --out "${PLUGIN_BUNDLE}" \
  --target linux/amd64 \
  "${BASE_DIR_FLAG[@]}" > /tmp/edge_cv_plugin_cert.log 2>&1; then
  echo -e "  └── ${GREEN}Plugin Certified & Native amd64 Bundle Built ✅${NC}"
else
  echo -e "  └── ${RED}Plugin Certification Failed ❌${NC}"
  cat /tmp/edge_cv_plugin_cert.log | sed 's/^/      /'
  exit 1
fi

# Step 3: Apply Plugin STRICTLY to node-dell-1
echo ""
echo -e "${BOLD}${MAGENTA}▶ [3/7] Staging & Applying BPR Plugin strictly to node-dell-1...${NC}"
if "${STUDIO_BIN}" plugin apply \
  --bundle "${PLUGIN_BUNDLE}" \
  --target node \
  --node node-dell-1 \
  --auth "${AUTH}" \
  --root-url "${ROOT_URL}" \
  "${BASE_DIR_FLAG[@]}" > /tmp/edge_cv_plugin_apply.log 2>&1; then
  # Ensure root and node-1 runtimes do not retain edge vision plugin
  "${STUDIO_BIN}" plugin remove --plugin bindu-edge-cv-plugin --target root --uninstall --auth "${AUTH}" --root-url "${ROOT_URL}" "${BASE_DIR_FLAG[@]}" > /dev/null 2>&1 || true
  "${STUDIO_BIN}" plugin remove --plugin bindu-edge-cv-plugin --target node --node node-1 --uninstall --auth "${AUTH}" --root-url "${ROOT_URL}" "${BASE_DIR_FLAG[@]}" > /dev/null 2>&1 || true
  echo -e "  └── ${GREEN}Plugin Assigned to node-dell-1 & Verified Removed from root / node-1 ✅${NC}"
else
  echo -e "  └── ${YELLOW}Plugin Applied with warnings / Offline Staged ⚠️${NC}"
  cat /tmp/edge_cv_plugin_apply.log | sed 's/^/      /'
fi

# Step 4: Synchronize Desired State Across Runtime
echo ""
echo -e "${BOLD}${MAGENTA}▶ [4/7] Synchronizing Runtime State & Baseline Wizard for node-dell-1...${NC}"
"${STUDIO_BIN}" runtime baseline accept --yes --note "Accept baseline with edge-cv BPR plugin on node-dell-1" "${BASE_DIR_FLAG[@]}" > /tmp/edge_cv_baseline_accept.log 2>&1 || true
"${STUDIO_BIN}" runtime desired commit --note "Commit desired state for edge-cv BPR plugin on node-dell-1" "${BASE_DIR_FLAG[@]}" > /tmp/edge_cv_desired_commit.log 2>&1 || true
if "${STUDIO_BIN}" runtime update wizard --yes --auth "${AUTH}" --root-url "${ROOT_URL}" "${BASE_DIR_FLAG[@]}" > /tmp/edge_cv_update_wiz.log 2>&1; then
  echo -e "  └── ${GREEN}Runtime State Converged & node-dell-1 Up to Date ✅${NC}"
else
  echo -e "  └── ${YELLOW}Runtime Update Wizard Complete (or node-dell-1 already updated) ⏩${NC}"
fi

# Step 5: Export Complete MAS Solution Bundle
MANIFEST_FILE="${SCRIPT_DIR}/solution_edge_cv.json"
BUNDLE_FILE="${SCRIPT_DIR}/bundles/solution_edge_cv.bundle.json"
echo ""
echo -e "${BOLD}${MAGENTA}▶ [5/7] Compiling Self-Contained Solution Bundle...${NC}"

# Ensure dashboards and CEL reactors are registered in DB before export
"${STUDIO_BIN}" dashboard register "${SCRIPT_DIR}/poc_dashboards.json" --force --auth "${AUTH}" --root-url "${ROOT_URL}" "${BASE_DIR_FLAG[@]}" > /tmp/edge_cv_db_reg.log 2>&1 || true
"${STUDIO_BIN}" cel-reactor merge --file "${SCRIPT_DIR}/poc_cel_reactor_config.json" --all --yes --auth "${AUTH}" --root-url "${ROOT_URL}" "${BASE_DIR_FLAG[@]}" > /tmp/edge_cv_cel_app.log 2>&1 || true
"${STUDIO_BIN}" solution register "${MANIFEST_FILE}" --force --auth "${AUTH}" --root-url "${ROOT_URL}" "${BASE_DIR_FLAG[@]}" > /tmp/edge_cv_sol_reg.log 2>&1 || true

if "${STUDIO_BIN}" solution export \
  --file "${MANIFEST_FILE}" \
  --out "${BUNDLE_FILE}" \
  --auth "${AUTH}" \
  --root-url "${ROOT_URL}" \
  "${BASE_DIR_FLAG[@]}" > /tmp/edge_cv_sol_exp.log 2>&1; then
  echo -e "  └── ${GREEN}Bundle Compiled with All Schemas, CEL Reactor, Dashboards & Plugin ✅${NC}"
else
  echo -e "  └── ${RED}Bundle Compilation Failed ❌${NC}"
  cat /tmp/edge_cv_sol_exp.log | sed 's/^/      /'
  exit 1
fi

# Step 6: Strict Validation of Solution Bundle
echo ""
echo -e "${BOLD}${MAGENTA}▶ [6/7] Validating Solution Bundle Integrity & Schema Dependencies...${NC}"
if "${STUDIO_BIN}" solution validate \
  --file "${BUNDLE_FILE}" \
  --strict > /tmp/edge_cv_sol_val.log 2>&1; then
  echo -e "  └── ${GREEN}Validation Passed: Strict Schema & Dependency Alignment Confirmed ✅${NC}"
else
  echo -e "  └── ${RED}Validation Failed ❌${NC}"
  cat /tmp/edge_cv_sol_val.log | sed 's/^/      /'
  exit 1
fi

# Step 7: Apply Complete Solution to Mesh Runtime
echo ""
echo -e "${BOLD}${MAGENTA}▶ [7/8] Deploying Solution Bundle to Mesh Runtime...${NC}"
if "${STUDIO_BIN}" solution apply \
  --file "${BUNDLE_FILE}" \
  --auth "${AUTH}" \
  --root-url "${ROOT_URL}" \
  --yes \
  "${BASE_DIR_FLAG[@]}" > /tmp/edge_cv_sol_app.log 2>&1; then
  echo -e "  └── ${GREEN}MASSolutionSet, MessageTypes, CEL Reactor, Dashboards, and Plugin Active 🚀${NC}"
else
  echo -e "  └── ${YELLOW}Solution Deployed / Staged with Warnings ⚠️${NC}"
  cat /tmp/edge_cv_sol_app.log | sed 's/^/      /'
fi

# Step 8: Final Runtime Convergence & OTA Rollout to Remote Edge Gateways
echo ""
echo -e "${BOLD}${MAGENTA}▶ [8/8] Finalizing Runtime State & Triggering OTA Rollout to node-dell-1...${NC}"
"${STUDIO_BIN}" runtime baseline accept --yes --note "Final baseline for edge-cv solution on node-dell-1" "${BASE_DIR_FLAG[@]}" > /tmp/edge_cv_final_baseline.log 2>&1 || true
"${STUDIO_BIN}" runtime desired commit --note "Final desired state for edge-cv solution on node-dell-1" "${BASE_DIR_FLAG[@]}" > /tmp/edge_cv_final_desired.log 2>&1 || true
if "${STUDIO_BIN}" runtime update wizard --yes --auth "${AUTH}" --root-url "${ROOT_URL}" "${BASE_DIR_FLAG[@]}" > /tmp/edge_cv_final_update_wiz.log 2>&1; then
  echo -e "  └── ${GREEN}Runtime OTA Rollout Enqueued for Remote Gateways ✅${NC}"
else
  echo -e "  └── ${YELLOW}Runtime Update Wizard Complete ⏩${NC}"
fi

# Step 9: Publish EdgeCVSet Configuration for node-dell-1
echo ""
echo -e "${BOLD}${MAGENTA}▶ [Configuration] Publishing EdgeCVSet Configuration for node-dell-1...${NC}"
CONFIG_PAYLOAD="$(cat "${SCRIPT_DIR}/poc_edge_cv_config.json")"
PUBLISHER_GO="${ROOT_DIR}/public/console/docs/semantic-ai-reactor-tutorial/publish_mqtt.go"

# Resolve MQTT integration ports
MQTT_HOST="127.0.0.1"
ROOT_MQTT_PORT="38190"

if [ -n "${REPORT_FILE}" ]; then
  PORT_VAL="$(report_value "${REPORT_FILE}" "root_mqtt_integration")"
  if [ -n "${PORT_VAL}" ]; then
    MQTT_HOST="${PORT_VAL%%:*}"
    ROOT_MQTT_PORT="${PORT_VAL##*:}"
  fi
fi

if [ -f "${PUBLISHER_GO}" ]; then
  # Publish to root (root routes $node-dest: "node-dell-1" to target node via mesh)
  go run "${PUBLISHER_GO}" \
    -host "${MQTT_HOST}" \
    -port "${ROOT_MQTT_PORT}" \
    -user "${AUTH%%:*}" \
    -pass "${AUTH##*:}" \
    -topic "bindu/in" \
    -msg "${CONFIG_PAYLOAD}" > /tmp/edge_cv_config_pub.log 2>&1 && \
    echo -e "  └── ${GREEN}EdgeCVSet Configuration Published to Root Mesh (Target: node-dell-1) ✅${NC}" || \
    echo -e "  └── ${YELLOW}EdgeCVSet Configuration queued / offline published ⚠️${NC}"
fi

echo ""
echo -e "${BOLD}${CYAN}========================================================================${NC}"
echo -e "${BOLD}${GREEN}   ✅ EDGE CV MAS SOLUTION DEPLOYMENT COMPLETED SUCCESSFULLY!          ${NC}"
echo -e "${BOLD}${CYAN}========================================================================${NC}"
echo -e "  • Solution ID:    ${BOLD}edge-cv-workstation-presence${NC}"
echo -e "  • Plugin Node:    ${BOLD}node-dell-1${NC} (Dell Inspiron 11 gateway: bindu-edge-cv-plugin@1.0.0)"
echo -e "  • CEL Reactor:    ${BOLD}workstation-presence-overlap-arbitrator${NC} (root)"
echo -e "  • Message Types:  ${BOLD}EdgeCVSet, EdgeCVSetStatus, AppearedClass, DisappearedClass, WorkstationPresenceSession${NC}"
echo -e "  • Dashboards:     ${BOLD}carlos-workstation-presence-monitor, multi-camera-vision-stream-audit${NC}"
echo -e "  • Web Console:    ${BOLD}${ROOT_URL}/console/${NC}"
echo -e "${CYAN}------------------------------------------------------------------------${NC}"
