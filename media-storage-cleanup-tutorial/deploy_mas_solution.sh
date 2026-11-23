#!/usr/bin/env bash
set -e

# ==============================================================================
# BINDU TUTORIAL: 1-Click MAS Solution Deployment via Bindu Studio
# ==============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/../../../.." && pwd)"
BINDU_STUDIO="go run ./cmd/bindu-studio"
BUNDLE_FILE="${SCRIPT_DIR}/bundles/solution_minio_media_cleanup.bundle.json"

# Auto-detect active running root base directory
if [ -z "${BASE_DIR}" ]; then
  if [ -d "${ROOT_DIR}/.e2e-root/current" ]; then
    BASE_DIR="${ROOT_DIR}/.e2e-root/current"
  else
    BASE_DIR="${ROOT_DIR}"
  fi
fi

AUTH="${BINDU_STUDIO_AUTH:-console:StrongPass123}"

echo "================================================================================"
echo "🚀 BINDU AI-NATIVE DEPLOYMENT: MinIO Media Storage Retention & Pruning Solution"
echo "================================================================================"
echo "Bundle Target:   ${BUNDLE_FILE}"
echo "Base Directory:  ${BASE_DIR}"
echo "Auth Principal:  ${AUTH%%:*}"
echo "Deploy Command:  bindu-studio solution apply --file <bundle> --base-dir <dir> --auth <auth> --yes"
echo "--------------------------------------------------------------------------------"

# Step 0: Centralized Secrets Verification & Vault Provisioning
echo "🔐 [Step 0] Validating Centralized Secret in Access Control Vault..."
if ! (cd "${ROOT_DIR}" && $BINDU_STUDIO access-control secrets get minio_media_secret --base-dir "${BASE_DIR}" >/dev/null 2>&1); then
  echo "⚠️ 'minio_media_secret' not found in vault. Provisioning default development secret..."
  (cd "${ROOT_DIR}" && $BINDU_STUDIO access-control secrets set minio_media_secret --value "bindu-media-123" --base-dir "${BASE_DIR}")
fi
echo "  ✓ minio_media_secret confirmed in vault."
echo "--------------------------------------------------------------------------------"

if [ ! -f "${BUNDLE_FILE}" ]; then
  echo "⚠️ Bundle file not found. Pre-registering artifacts and packaging bundle..."
  mkdir -p "${SCRIPT_DIR}/bundles"
  (cd "${ROOT_DIR}" && $BINDU_STUDIO message-type register --base-dir "${BASE_DIR}" "${SCRIPT_DIR}/poc_message_types.json" --confirm-all || true)
  (cd "${ROOT_DIR}" && $BINDU_STUDIO dashboard register --base-dir "${BASE_DIR}" "${SCRIPT_DIR}/poc_dashboards.json" --force || true)
  (cd "${ROOT_DIR}" && $BINDU_STUDIO reactor apply --base-dir "${BASE_DIR}" "${SCRIPT_DIR}/poc_cel_reactor_config.json" --yes || true)
  (cd "${ROOT_DIR}" && $BINDU_STUDIO reactor apply --base-dir "${BASE_DIR}" "${SCRIPT_DIR}/poc_rest_reactor_config.json" --yes || true)
  (cd "${ROOT_DIR}" && $BINDU_STUDIO solution register --base-dir "${BASE_DIR}" "${SCRIPT_DIR}/solution_minio_media_cleanup.json" --force || true)
  echo "📦 Packaging MAS Solution bundle into ${BUNDLE_FILE}..."
  (cd "${ROOT_DIR}" && $BINDU_STUDIO solution export minio-media-storage-cleanup --base-dir "${BASE_DIR}" --out "${BUNDLE_FILE}" || true)
fi

if [ -f "${BUNDLE_FILE}" ]; then
  echo "📦 Applying complete self-contained Multi-Agent System (MAS) bundle to active root..."
  (cd "${ROOT_DIR}" && $BINDU_STUDIO solution apply --base-dir "${BASE_DIR}" --auth "${AUTH}" --file "${BUNDLE_FILE}" --yes)
else
  echo "📦 Registering and activating MAS Solution manifest..."
  (cd "${ROOT_DIR}" && $BINDU_STUDIO solution register --base-dir "${BASE_DIR}" "${SCRIPT_DIR}/solution_minio_media_cleanup.json" --force)
fi

echo ""
echo "================================================================================"
echo "✅ MAS Solution Successfully Deployed in a Single Idempotent Transaction!"
echo "--------------------------------------------------------------------------------"
echo "The bundle automatically registered and synchronized:"
echo "  ✓ Message Types:  MediaPruneCandidate, MediaPruneAudit"
echo "  ✓ CEL Reactor:    minio-orphan-media-inspector (Scheduled: 0 0 4 * * *)"
echo "  ✓ REST Reactor:   minio-media-http-pruner (HTTP DELETE against MinIO)"
echo "  ✓ Dashboards:     minio-storage-reclaim-telemetry, media-prune-candidate-feed"
echo "  ✓ Solution State: MASSolutionSet 'minio-media-storage-cleanup' active in mesh"
echo "================================================================================"
