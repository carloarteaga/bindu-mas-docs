#!/usr/bin/env bash
# public/console/docs/mas-solution-tutorial/deploy_all_solutions.sh
# Builds, validates, and deploys all 10 canonical Multi-Agent System (MASSolutionSet) domain solutions.

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

# Defaults
AUTH_CLI=""
ROOT_URL_CLI=""
USER_CLI=""
PASS_CLI=""
VALIDATE_ONLY=false
BUILD_ONLY=false
SKIP_BUILD=false
SPECIFIC_SOLUTION=""
AUTO_CONFIRM=false

while [[ "$#" -gt 0 ]]; do
  case "$1" in
    -h|--help|help)
      echo -e "${BOLD}Usage:${NC} $0 [options]"
      echo ""
      echo "Builds, validates, and deploys all 10 canonical MASSolutionSet domain manifests into Bindu."
      echo ""
      echo "Options:"
      echo "  --base-dir <dir>       Bindu base directory (default: auto-detected ./.e2e-root/current)"
      echo "  --root-url <url>       Bindu Root API endpoint (default: from config or http://127.0.0.1:8080)"
      echo "  --auth <user:pass>     Basic auth credentials (default: console:StrongPass123 or from report.txt)"
      echo "  --user <user>          Basic auth username"
      echo "  --pass <pass>          Basic auth password"
      echo "  --solution <id_or_num> Process only a specific solution (e.g. 01, sentinel, albarino)"
      echo "  --build-only           Export and compile solution bundles only without validating or deploying"
      echo "  --validate-only        Build and run validation checks only without publishing to the mesh"
      echo "  --skip-build           Skip bundle compilation and use existing bundles in bundles/ directory"
      echo "  -y, --yes              Confirm execution without interactive prompting"
      echo "  -h, --help             Show this help message"
      exit 0
      ;;
    --base-dir)
      BASE_DIR="$2"
      shift 2
      ;;
    --root-url)
      ROOT_URL_CLI="$2"
      shift 2
      ;;
    --auth)
      AUTH_CLI="$2"
      shift 2
      ;;
    --user)
      USER_CLI="$2"
      shift 2
      ;;
    --pass)
      PASS_CLI="$2"
      shift 2
      ;;
    --solution)
      SPECIFIC_SOLUTION="$2"
      shift 2
      ;;
    --build-only)
      BUILD_ONLY=true
      shift
      ;;
    --validate-only)
      VALIDATE_ONLY=true
      shift
      ;;
    --skip-build)
      SKIP_BUILD=true
      shift
      ;;
    -y|--yes)
      AUTO_CONFIRM=true
      shift
      ;;
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
echo -e "${BOLD}${CYAN}   📦 BINDU MULTI-AGENT SYSTEM (MAS) ALL-SOLUTIONS PIPELINE           ${NC}"
echo -e "${BOLD}${CYAN}========================================================================${NC}"
echo -e "${BOLD}Timestamp:${NC}     $(date -u +"%Y-%m-%dT%H:%M:%SZ")"
echo -e "${BOLD}Target BaseDir:${NC} ${BASE_DIR:-"(standard default)"}"
echo -e "${BOLD}Target RootURL:${NC} ${ROOT_URL}"
echo -e "${BOLD}Auth Identity:${NC}  ${AUTH%%:*}:******"
echo -e "${BOLD}Skip Build:${NC}     ${SKIP_BUILD}"
echo -e "${BOLD}Build Only:${NC}     ${BUILD_ONLY}"
echo -e "${BOLD}Validate Only:${NC}  ${VALIDATE_ONLY}"
if [ -n "${SPECIFIC_SOLUTION}" ]; then
  echo -e "${BOLD}Filter Solution:${NC}${SPECIFIC_SOLUTION}"
fi
echo -e "${CYAN}------------------------------------------------------------------------${NC}"

# Canonical Solution Manifest Names
SOLUTIONS=(
  "solution_01_albarino_vineyard"
  "solution_02_coldchain_watchdog"
  "solution_03_financial_audit"
  "solution_04_hello_world"
  "solution_05_patio_media_vision"
  "solution_06_marine_fisheries"
  "solution_07_pharma_coldchain_ai"
  "solution_08_system_email_gateway"
  "solution_09_fishing_poc"
  "solution_10_sentinel"
)

TOTAL_COUNT=0
BUILT_COUNT=0
VALID_COUNT=0
DEPLOYED_COUNT=0
FAILED_COUNT=0

mkdir -p "${SCRIPT_DIR}/bundles"
cd "${ROOT_DIR}"

STUDIO_BIN="/tmp/bindu-studio-tool"
echo -n "Building fresh bindu-studio binary... "
if go build -o "${STUDIO_BIN}" ./cmd/bindu-studio > /tmp/bindu_build.log 2>&1; then
  echo -e "${GREEN}READY ✅${NC}"
else
  echo -e "${RED}FAILED ❌${NC}"
  cat /tmp/bindu_build.log
  exit 1
fi

for sol in "${SOLUTIONS[@]}"; do
  manifest="${SCRIPT_DIR}/${sol}.json"
  bundle="${SCRIPT_DIR}/bundles/${sol}.bundle.json"
  
  if [ -n "${SPECIFIC_SOLUTION}" ]; then
    if [[ "${sol}" != *"${SPECIFIC_SOLUTION}"* ]]; then
      continue
    fi
  fi

  TOTAL_COUNT=$((TOTAL_COUNT + 1))
  echo ""
  echo -e "${BOLD}${MAGENTA}▶ [${TOTAL_COUNT}] Processing Solution: ${NC}${BOLD}${sol}${NC}"

  # Step 1: Build / Export Bundle
  if [ "${SKIP_BUILD}" = false ]; then
    echo -n "  ├── [1/3] Compiling Self-Contained Bundle... "
    if "${STUDIO_BIN}" solution export --file "${manifest}" --out "${bundle}" > /tmp/bindu_exp.log 2>&1; then
      echo -e "${GREEN}BUILT ✅${NC}"
      BUILT_COUNT=$((BUILT_COUNT + 1))
    else
      echo -e "${RED}FAILED ❌${NC}"
      cat /tmp/bindu_exp.log | sed 's/^/  │   /'
      FAILED_COUNT=$((FAILED_COUNT + 1))
      continue
    fi
  else
    echo -e "  ├── [1/3] Bundle Compilation Skipped (--skip-build active) ⏩"
    BUILT_COUNT=$((BUILT_COUNT + 1))
  fi

  if [ "${BUILD_ONLY}" = true ]; then
    echo -e "  └── [2/3 & 3/3] Validation & Deployment Skipped (--build-only active) ⏩"
    continue
  fi

  # Step 2: Validate Bundle
  echo -n "  ├── [2/3] Validating Bundle Schemas & Artifacts... "
  if "${STUDIO_BIN}" solution validate --file "${bundle}" --strict > /tmp/bindu_val.log 2>&1; then
    echo -e "${GREEN}VALID ✅${NC}"
    VALID_COUNT=$((VALID_COUNT + 1))
  else
    echo -e "${RED}FAILED ❌${NC}"
    cat /tmp/bindu_val.log | sed 's/^/  │   /'
    FAILED_COUNT=$((FAILED_COUNT + 1))
    continue
  fi

  # Step 3: Apply / Ingest Bundle if not validate-only
  if [ "${VALIDATE_ONLY}" = false ]; then
    echo -n "  └── [3/3] Applying Complete Bundle to Mesh Runtime... "
    CMD_ARGS=(
      "${STUDIO_BIN}" "solution" "apply"
      "--file" "${bundle}"
      "--auth" "${AUTH}"
      "--root-url" "${ROOT_URL}"
      "--yes"
    )
    if [ -n "${BASE_DIR}" ]; then
      CMD_ARGS+=("--base-dir" "${BASE_DIR}")
    fi

    if "${CMD_ARGS[@]}" > /tmp/bindu_reg.log 2>&1; then
      echo -e "${GREEN}DEPLOYED & ACTIVE 🚀${NC}"
      DEPLOYED_COUNT=$((DEPLOYED_COUNT + 1))
    else
      echo -e "${YELLOW}OFFLINE / SKIPPED ⚠️${NC}"
      cat /tmp/bindu_reg.log | sed 's/^/      /'
    fi
  else
    echo -e "  └── [3/3] Deployment Skipped (--validate-only flag active) ⏩"
  fi
done

echo ""
echo -e "${BOLD}${CYAN}========================================================================${NC}"
echo -e "${BOLD}${CYAN}   📊 DEPLOYMENT & VALIDATION SUMMARY REPORT                           ${NC}"
echo -e "${BOLD}${CYAN}========================================================================${NC}"
echo -e "  • Total Solutions Processed: ${BOLD}${TOTAL_COUNT}${NC}"
echo -e "  • Bundles Built / Compiled:  ${BOLD}${GREEN}${BUILT_COUNT}${NC} / ${TOTAL_COUNT}"
if [ "${BUILD_ONLY}" = false ]; then
  echo -e "  • Bundles Validated:         ${BOLD}${GREEN}${VALID_COUNT}${NC} / ${TOTAL_COUNT}"
  if [ "${VALIDATE_ONLY}" = false ]; then
    echo -e "  • Solutions Applied:         ${BOLD}${GREEN}${DEPLOYED_COUNT}${NC} / ${TOTAL_COUNT}"
  fi
fi
if [ "${FAILED_COUNT}" -gt 0 ]; then
  echo -e "  • Failed Operations:         ${BOLD}${RED}${FAILED_COUNT}${NC}"
fi
echo -e "${CYAN}------------------------------------------------------------------------${NC}"

if [ "${FAILED_COUNT}" -eq 0 ]; then
  echo -e "${BOLD}${GREEN}All MAS Solution operations completed successfully! 🎓🎉${NC}"
else
  echo -e "${BOLD}${RED}One or more solution operations encountered errors.${NC}"
  exit 1
fi
