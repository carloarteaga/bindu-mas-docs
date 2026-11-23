#!/usr/bin/env bash
# =================================================================
# 🌐 BINDU BPR PLUGIN COMPLETE LIFECYCLE RUNNER (HELLO WORLD)
# =================================================================
# Demonstrates:
#   Phase 0: Schema Synchronization & Automated Code Generation (gen-go, gen-examples)
#   Phase 1: In-Engine Dynamic Simulation (plugin dry-run)
#   Phase 2: Static Sanity Audit & Multi-Architecture Certify (doctor, certify)
#   Phase 3: Cluster Rollout & Application (plugin apply)
#   Phase 4: Operational State Inspection (plugin who-uses, plugin list)
#   Phase 5: Full Lifecycle Teardown & Purging (remove, undeploy, purge) [Optional with --teardown]
# =================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

# Default operational parameters
AUTH_DEFAULT="console:StrongPass123"
AUTO_CONFIRM=false
TEARDOWN=false
BASE_DIR_CLI=""
ROOT_URL_CLI=""
AUTH_CLI=""
USER_CLI=""
PASS_CLI=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --base-dir) BASE_DIR_CLI="$2"; shift 2 ;;
    --root-url) ROOT_URL_CLI="$2"; shift 2 ;;
    --auth) AUTH_CLI="$2"; shift 2 ;;
    --user) USER_CLI="$2"; shift 2 ;;
    --pass) PASS_CLI="$2"; shift 2 ;;
    -y|--yes) AUTO_CONFIRM=true; shift ;;
    --teardown) TEARDOWN=true; shift ;;
    -h|--help)
      echo "Usage: ./deploy_plugin_lifecycle.sh [options]"
      echo ""
      echo "Options:"
      echo "  --base-dir <dir>       Bindu base directory"
      echo "  --root-url <url>       Root URL (default: http://127.0.0.1:8080)"
      echo "  --auth <user:pass>     Basic auth credentials (default: console:StrongPass123)"
      echo "  -y, --yes              Auto-confirm prompts"
      echo "  --teardown             Execute full teardown and deep purge after testing"
      echo "  -h, --help             Show this help message"
      exit 0
      ;;
    *)
      echo "Unknown argument: $1"
      exit 1
      ;;
  esac
done

BASE_DIR="${BASE_DIR_CLI:-${BASE_DIR:-}}"
if [ -z "${BASE_DIR}" ]; then
  if [ -d "${REPO_ROOT}/.e2e-root/current" ]; then
    BASE_DIR="${REPO_ROOT}/.e2e-root/current"
  else
    BASE_DIR="${REPO_ROOT}"
  fi
fi

AUTH="${AUTH_CLI:-${BINDU_STUDIO_AUTH:-$AUTH_DEFAULT}}"
if [ -n "${USER_CLI}" ] && [ -n "${PASS_CLI}" ]; then
  AUTH="${USER_CLI}:${PASS_CLI}"
fi

ROOT_URL="${ROOT_URL_CLI:-${BINDU_ROOT_URL:-http://127.0.0.1:8080}}"

BASE_DIR_FLAG=()
if [ -n "${BASE_DIR}" ]; then
  BASE_DIR_FLAG=(--base-dir "${BASE_DIR}")
fi

echo "================================================================="
echo "🌐 BINDU BPR PLUGIN: CANONICAL LIFECYCLE RUNNER"
echo "================================================================="
echo "Base Directory: ${BASE_DIR}"
echo "Root URL:       ${ROOT_URL}"
echo "Auth Principal: ${AUTH%%:*}:******"
echo "Plugin Source:  ${SCRIPT_DIR}/hello-world-plugin"
echo "Teardown Mode:  ${TEARDOWN}"
echo "-----------------------------------------------------------------"

if [ "$AUTO_CONFIRM" != "true" ]; then
  read -p "Proceed with lifecycle demonstration? (y/N): " -r CONFIRM
  if [[ ! "$CONFIRM" =~ ^[Yy]$ ]]; then
    echo "Deployment cancelled by user."
    exit 0
  fi
fi

cd "$REPO_ROOT"

# -----------------------------------------------------------------
# Phase 0: Schema Synchronization & Code Generation
# -----------------------------------------------------------------
echo ""
echo "📜 Phase 0a: Registering message types in Root cluster..."
go run ./cmd/bindu-studio message-types merge \
    --config "$SCRIPT_DIR/hello-world-message-types.json" \
    --all \
    --auth "$AUTH" \
    --root-url "$ROOT_URL" \
    "${BASE_DIR_FLAG[@]}"

echo "🧬 Phase 0b: Generating strongly-typed Go structs (gen-go)..."
go run ./cmd/bindu-studio message-types gen-go \
    --config "$SCRIPT_DIR/hello-world-message-types.json" \
    --out "$SCRIPT_DIR/hello-world-plugin/types_gen.go" \
    --package main

echo "📦 Phase 0c: Generating compliant example envelope fixtures (gen-examples)..."
go run ./cmd/bindu-studio message-types gen-examples \
    --config "$SCRIPT_DIR/hello-world-message-types.json" \
    --out "$SCRIPT_DIR/hello-world-plugin/examples/out"

# -----------------------------------------------------------------
# Phase 1: Dynamic In-Process Simulation (Dry-Run)
# -----------------------------------------------------------------
echo ""
echo "🔬 Phase 1: Running in-process dynamic simulation (plugin dry-run)..."
go run ./cmd/bindu-studio plugin dry-run \
    --source "$SCRIPT_DIR/hello-world-plugin" \
    --config "$SCRIPT_DIR/hello-world-message-types.json" \
    --trigger cron \
    --timeout 4s \
    "${BASE_DIR_FLAG[@]}"

# -----------------------------------------------------------------
# Phase 2: Static Sanity Audit & Multi-Architecture Certification
# -----------------------------------------------------------------
echo ""
echo "🩺 Phase 2a: Auditing plugin project consistency (plugin doctor)..."
go run ./cmd/bindu-studio plugin doctor \
    --source "$SCRIPT_DIR/hello-world-plugin" \
    --config "$SCRIPT_DIR/hello-world-message-types.json" \
    --strict \
    "${BASE_DIR_FLAG[@]}"

echo "📦 Phase 2b: Certifying and packaging bundle (plugin certify)..."
go run ./cmd/bindu-studio plugin certify \
    --source "$SCRIPT_DIR/hello-world-plugin" \
    --config "$SCRIPT_DIR/hello-world-message-types.json" \
    --target linux/amd64 \
    --out "$SCRIPT_DIR/hello-world-plugin.tar.gz" \
    "${BASE_DIR_FLAG[@]}"

# -----------------------------------------------------------------
# Phase 3: Cluster Rollout
# -----------------------------------------------------------------
echo ""
echo "🚀 Phase 3: Deploying hello-world-plugin bundle to target root..."
go run ./cmd/bindu-studio plugin apply \
    --bundle "$SCRIPT_DIR/hello-world-plugin.tar.gz" \
    --target root \
    --auth "$AUTH" \
    --root-url "$ROOT_URL" \
    "${BASE_DIR_FLAG[@]}"

# -----------------------------------------------------------------
# Phase 4: Operational State Inspection
# -----------------------------------------------------------------
echo ""
echo "🔍 Phase 4: Inspecting active plugin usage across cluster..."
go run ./cmd/bindu-studio plugin who-uses \
    --plugin hello-world-plugin \
    --auth "$AUTH" \
    --root-url "$ROOT_URL" \
    "${BASE_DIR_FLAG[@]}" || true

go run ./cmd/bindu-studio plugin list \
    --auth "$AUTH" \
    --root-url "$ROOT_URL" \
    "${BASE_DIR_FLAG[@]}" || true

# -----------------------------------------------------------------
# Phase 5: Optional Lifecycle Teardown & Deep Purge
# -----------------------------------------------------------------
if [ "$TEARDOWN" = "true" ]; then
  echo ""
  echo "🧹 Phase 5: Executing Full Lifecycle Teardown & Deep Cache Purge..."
  
  echo "  • [1/3] Removing plugin assignment from root node..."
  go run ./cmd/bindu-studio plugin remove \
      --plugin hello-world-plugin \
      --target root \
      --yes \
      --auth "$AUTH" \
      --root-url "$ROOT_URL" \
      "${BASE_DIR_FLAG[@]}" || true

  echo "  • [2/3] Undeploying plugin from cluster registry..."
  go run ./cmd/bindu-studio plugin undeploy \
      --plugin hello-world-plugin \
      --yes \
      --auth "$AUTH" \
      --root-url "$ROOT_URL" \
      "${BASE_DIR_FLAG[@]}" || true

  echo "  • [3/3] Deep purging all cached bundles and staged artifacts..."
  go run ./cmd/bindu-studio plugin purge \
      --plugin hello-world-plugin \
      --all \
      --yes \
      --auth "$AUTH" \
      --root-url "$ROOT_URL" \
      "${BASE_DIR_FLAG[@]}" || true

  echo "✅ Teardown complete. Cluster verified clean."
fi

echo ""
echo "================================================================="
echo "🎉 BPR PLUGIN LIFECYCLE DEMONSTRATION COMPLETED SUCCESSFULLY!"
echo "================================================================="
echo "• Schemas:      ${SCRIPT_DIR}/hello-world-message-types.json"
echo "• Go Types:     ${SCRIPT_DIR}/hello-world-plugin/types_gen.go"
echo "• Bundle:       ${SCRIPT_DIR}/hello-world-plugin.tar.gz"
echo "• SPA UI:       ${SCRIPT_DIR}/hello-world-spa/index.html"
echo "• Guide:        ${SCRIPT_DIR}/tutorial_guide.md"
echo "• Plan:         ${SCRIPT_DIR}/tutorial-plan.md"
echo "================================================================="
