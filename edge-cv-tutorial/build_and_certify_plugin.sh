#!/usr/bin/env bash
set -e

# ==============================================================================
# BINDU TUTORIAL: Compile & Certify Edge CV BPR Plugin via Bindu Studio Tools
# ==============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/../../../.." && pwd)"
BINDU_STUDIO="go run ./cmd/bindu-studio"

PLUGIN_SOURCE="${SCRIPT_DIR}/bindu-edge-cv-plugin"
MESSAGE_TYPES="${SCRIPT_DIR}/poc_message_types.json"
BUILD_DIR="${ROOT_DIR}/dist/plugins"
BUNDLE_OUT="${BUILD_DIR}/bindu-edge-cv-plugin-1.0.0-linux-amd64.tar.gz"

echo "================================================================================"
echo "🛠️ BINDU BPR: Compile & Certify Edge CV Plugin (bindu-edge-cv-plugin)"
echo "================================================================================"
echo "Plugin Source:  ${PLUGIN_SOURCE}"
echo "Message Schema: ${MESSAGE_TYPES}"
echo "Output Bundle:  ${BUNDLE_OUT}"
echo "--------------------------------------------------------------------------------"

# Step 0: Ensure native C++ ONNX runtime and models are present
if [ ! -f "${PLUGIN_SOURCE}/runtime/lib/libonnxruntime.so" ] || [ ! -f "${PLUGIN_SOURCE}/runtime/models/yolox_s.onnx" ]; then
  echo "📥 Step 0: Runtime assets missing. Executing setup_runtime.sh..."
  "${SCRIPT_DIR}/setup_runtime.sh"
  echo ""
fi

# Step 1: Run Plugin Doctor
echo "🩺 Step 1: Running Bindu Studio Plugin Doctor..."
(cd "${ROOT_DIR}" && $BINDU_STUDIO plugin doctor \
  --base-dir "${ROOT_DIR}" \
  --source "${PLUGIN_SOURCE}" \
  --config "${MESSAGE_TYPES}")

# Step 2: Ensure build output directory exists
mkdir -p "${BUILD_DIR}"

# Step 3: Certify and Build Plugin Package
echo ""
echo "📦 Step 2: Certifying and Packaging BPR Plugin for linux/amd64..."
(cd "${ROOT_DIR}" && $BINDU_STUDIO plugin certify \
  --base-dir "${ROOT_DIR}" \
  --source "${PLUGIN_SOURCE}" \
  --config "${MESSAGE_TYPES}" \
  --out "${BUNDLE_OUT}" \
  --target linux/amd64)

echo ""
echo "================================================================================"
echo "✅ Edge CV BPR Plugin Certified & Packaged Successfully!"
echo "--------------------------------------------------------------------------------"
echo "Target Bundle: ${BUNDLE_OUT}"
echo ""
echo "Next Steps to Deploy to Live Node:"
echo "  1. Import into cluster: go run ./cmd/bindu-studio plugin import --bundle ${BUNDLE_OUT}"
echo "  2. Assign to gateway:   go run ./cmd/bindu-studio plugin register --plugin bindu-edge-cv-plugin --target node --node node-dell-1"
echo "  3. Commit & rollout:    go run ./cmd/bindu-studio runtime desired commit --note 'Deploy CV Plugin to node-dell-1'"
echo "                          go run ./cmd/bindu-studio runtime update wizard --nodes node-dell-1 --yes"
echo "================================================================================"
