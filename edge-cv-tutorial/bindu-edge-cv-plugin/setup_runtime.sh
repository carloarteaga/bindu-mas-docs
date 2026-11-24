#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# BINDU TUTORIAL: Edge CV Plugin Runtime Asset Setup
#
# Downloads native C++ ONNX Runtime libraries and pre-trained YOLOX-S weights
# on-demand, keeping the Git repository lightweight (<15MB).
# ==============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [ -d "${SCRIPT_DIR}/runtime" ]; then
  PLUGIN_DIR="${SCRIPT_DIR}"
elif [ -d "${SCRIPT_DIR}/bindu-edge-cv-plugin" ]; then
  PLUGIN_DIR="${SCRIPT_DIR}/bindu-edge-cv-plugin"
else
  PLUGIN_DIR="${SCRIPT_DIR}"
fi
RUNTIME_LIB="${PLUGIN_DIR}/runtime/lib"
RUNTIME_MODELS="${PLUGIN_DIR}/runtime/models"

ORT_VERSION="1.24.1"
YOLOX_VERSION="0.1.1rc0"

ARCH="$(uname -m)"
case "${ARCH}" in
  x86_64|amd64)
    ORT_ARCH="x64"
    ;;
  aarch64|arm64)
    ORT_ARCH="aarch64"
    ;;
  *)
    echo "❌ Unsupported architecture for prebuilt ONNX Runtime: ${ARCH}"
    echo "Supported architectures: x86_64, aarch64"
    exit 1
    ;;
esac

echo "================================================================================"
echo "🎯 Setup Edge CV Runtime Assets (ONNX Runtime + YOLOX-S)"
echo "================================================================================"
echo "Architecture: ${ARCH} (ORT target: linux-${ORT_ARCH})"
echo "Target Dir:   ${PLUGIN_DIR}/runtime"
echo "--------------------------------------------------------------------------------"

mkdir -p "${RUNTIME_LIB}" "${RUNTIME_MODELS}"

# 1. Download and install ONNX Runtime C++ Shared Library
if [ ! -f "${RUNTIME_LIB}/libonnxruntime.so" ] && [ ! -f "${RUNTIME_LIB}/libonnxruntime.so.${ORT_VERSION}" ]; then
  echo "📥 Downloading ONNX Runtime v${ORT_VERSION} for linux-${ORT_ARCH}..."
  WORK_DIR="$(mktemp -d /tmp/ort-setup-XXXXXX)"
  trap 'rm -rf "${WORK_DIR}"' EXIT

  ORT_TGZ="onnxruntime-linux-${ORT_ARCH}-${ORT_VERSION}.tgz"
  ORT_URL="https://github.com/microsoft/onnxruntime/releases/download/v${ORT_VERSION}/${ORT_TGZ}"

  curl -L --fail --progress-bar -o "${WORK_DIR}/${ORT_TGZ}" "${ORT_URL}"
  tar -xzf "${WORK_DIR}/${ORT_TGZ}" -C "${WORK_DIR}"

  FOUND_SO="$(find "${WORK_DIR}" -type f \( -name "libonnxruntime.so.${ORT_VERSION}" -o -name "libonnxruntime.so" \) | head -n 1)"
  if [ -z "${FOUND_SO}" ]; then
    echo "❌ Failed to locate libonnxruntime.so in extracted archive"
    exit 1
  fi

  cp "${FOUND_SO}" "${RUNTIME_LIB}/libonnxruntime.so.${ORT_VERSION}"
  cp "${FOUND_SO}" "${RUNTIME_LIB}/libonnxruntime.so"
  chmod 755 "${RUNTIME_LIB}/libonnxruntime.so" "${RUNTIME_LIB}/libonnxruntime.so.${ORT_VERSION}"
  echo "✓ ONNX Runtime library installed in: ${RUNTIME_LIB}/"
else
  echo "✓ ONNX Runtime library already present in: ${RUNTIME_LIB}/"
fi

# 2. Download and install YOLOX-S neural weights
if [ ! -f "${RUNTIME_MODELS}/yolox_s.onnx" ] || [ ! -s "${RUNTIME_MODELS}/yolox_s.onnx" ]; then
  echo "📥 Downloading pre-trained YOLOX-S ONNX model (Megvii release v${YOLOX_VERSION})..."
  YOLOX_URL="https://github.com/Megvii-BaseDetection/YOLOX/releases/download/${YOLOX_VERSION}/yolox_s.onnx"

  curl -L --fail --progress-bar -o "${RUNTIME_MODELS}/yolox_s.onnx" "${YOLOX_URL}"
  chmod 644 "${RUNTIME_MODELS}/yolox_s.onnx"
  echo "✓ YOLOX-S model installed in: ${RUNTIME_MODELS}/yolox_s.onnx"
else
  echo "✓ YOLOX-S model already present in: ${RUNTIME_MODELS}/yolox_s.onnx"
fi

# 3. Check for FFmpeg (runtime video stream demuxing)
if command -v ffmpeg >/dev/null 2>&1; then
  echo "✓ FFmpeg found: $(which ffmpeg)"
else
  echo "⚠️ Warning: FFmpeg is not installed on this system. RTSP stream capture requires ffmpeg."
  echo "  Install via: sudo apt install -y ffmpeg"
fi

echo "--------------------------------------------------------------------------------"
echo "✅ Runtime assets ready for bindu-edge-cv-plugin!"
echo "================================================================================"
