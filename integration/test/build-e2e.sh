#!/bin/bash
#
# Copyright IBM Corp. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
# Build Docker images for E2E integration testing.
#
# This script builds the orderer (arma-4p1s) and committer (committer-test-node)
# Docker images needed by run-e2e.sh. It clones the repos at specific refs
# and builds the images locally.
#
# Usage:
#   ./build-e2e.sh                              # build using refs from refs.conf
#   ./build-e2e.sh --fabric-x-ref=abc123        # override fabric-x tools ref
#   ./build-e2e.sh --committer-ref=v1.2.3       # override committer ref
#   ./build-e2e.sh --orderer-ref=abc123         # override orderer ref
#   ./build-e2e.sh --fabric-x-repo=URL          # custom fabric-x repo URL
#
# Output:
#   Prints export commands for ORDERER_IMAGE and COMMITTER_IMAGE.
#   When GITHUB_OUTPUT is set (CI), also writes image names there.
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BUILD_DIR="${SCRIPT_DIR}/.build"

# ──────────────────────────────────────────────────────────────────────────────
# Prerequisite checks
# ──────────────────────────────────────────────────────────────────────────────
require_bin() {
  local bin="$1"
  command -v "${bin}" >/dev/null 2>&1 || {
    echo "ERROR: '${bin}' is required but not found in PATH"
    exit 1
  }
}

for bin in git docker make; do
  require_bin "${bin}"
done

# ──────────────────────────────────────────────────────────────────────────────
# Load default refs from configuration file
# ──────────────────────────────────────────────────────────────────────────────
REFS_CONF="${SCRIPT_DIR}/refs.conf"
if [ -f "${REFS_CONF}" ]; then
  # shellcheck source=refs.conf
  source "${REFS_CONF}"
fi

# ──────────────────────────────────────────────────────────────────────────────
# Argument parsing
# ──────────────────────────────────────────────────────────────────────────────
for arg in "$@"; do
  case "${arg}" in
  --fabric-x-ref=*) FABRIC_X_REF="${arg#*=}" ;;
  --committer-ref=*) COMMITTER_REF="${arg#*=}" ;;
  --orderer-ref=*) ORDERER_REF="${arg#*=}" ;;
  --fabric-x-repo=*) FABRIC_X_REPO="${arg#*=}" ;;
  --committer-repo=*) COMMITTER_REPO="${arg#*=}" ;;
  --orderer-repo=*) ORDERER_REPO="${arg#*=}" ;;
  --help)
    echo "Usage: ./build-e2e.sh [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --fabric-x-ref=REF    Tag, branch, or commit for fabric-x tools"
    echo "  --committer-ref=REF   Tag, branch, or commit for fabric-x-committer"
    echo "  --orderer-ref=REF     Tag, branch, or commit for fabric-x-orderer"
    echo "  --fabric-x-repo=URL   Override default fabric-x GitHub repo URL"
    echo "  --committer-repo=URL  Override default committer GitHub repo URL"
    echo "  --orderer-repo=URL    Override default orderer GitHub repo URL"
    echo ""
    echo "Refs are loaded from refs.conf by default and can be overridden via CLI."
    exit 0
    ;;
  *)
    echo "Unknown argument: ${arg}"
    exit 1
    ;;
  esac
done

# Verify required refs are set
require_var() {
  local name="$1" message="$2"
  if [ -z "${!name:-}" ]; then
    echo "ERROR: ${message}"
    exit 1
  fi
}

require_var "FABRIC_X_REF" "FABRIC_X_REF is not set. Please specify --fabric-x-ref or set it in refs.conf"
require_var "COMMITTER_REF" "COMMITTER_REF is not set. Please specify --committer-ref or set it in refs.conf"
require_var "ORDERER_REF" "ORDERER_REF is not set. Please specify --orderer-ref or set it in refs.conf"
require_var "ORDERER_IMAGE_NAME" "ORDERER_IMAGE_NAME is not set. Check refs.conf"
require_var "COMMITTER_IMAGE_NAME" "COMMITTER_IMAGE_NAME is not set. Check refs.conf"

# ──────────────────────────────────────────────────────────────────────────────
# Helper functions
# ──────────────────────────────────────────────────────────────────────────────

# Images are always built locally for determinism and to avoid registry access
# issues (private/unpublished tags, auth, rate limits).
# clone_at_ref clones a repository at a specific ref into the build directory.
# First attempts a shallow clone (--depth 1) for tags/branches. If that fails
# (e.g., for commit hashes which can't be shallow-cloned), does a full clone
# and checks out the ref.
clone_at_ref() {
  local repo="$1" ref="$2" dest="$3"
  echo "Cloning ${repo} at ${ref}..."
  rm -rf "${dest}"
  git clone --depth 1 --branch "${ref}" "${repo}" "${dest}" 2>/dev/null || {
    git clone "${repo}" "${dest}"
    git -C "${dest}" checkout "${ref}"
  }
}

mkdir -p "${BUILD_DIR}"

# ──────────────────────────────────────────────────────────────────────────────
# Clone fabric-x repository for tools at FABRIC_X_REF
# ──────────────────────────────────────────────────────────────────────────────
FABRIC_X_DIR="${BUILD_DIR}/fabric-x"
echo "Cloning fabric-x repo for tools at ${FABRIC_X_REF}..."
clone_at_ref "${FABRIC_X_REPO}" "${FABRIC_X_REF}" "${FABRIC_X_DIR}"
echo "Building fabric-x tools..."
make -C "${FABRIC_X_DIR}" tools
echo "fabric-x tools ready at ${FABRIC_X_DIR}"

# ──────────────────────────────────────────────────────────────────────────────
# Build orderer image (arma-4p1s)
#
# The orderer image packages all 4 Arma roles (router, batcher, consenter,
# assembler) into a single container. The all-in-one Dockerfile in the orderer
# repo builds the armageddon binary and bundles it with the role entrypoints.
# ──────────────────────────────────────────────────────────────────────────────
ORDERER_DIR="${BUILD_DIR}/fabric-x-orderer"
clone_at_ref "${ORDERER_REPO}" "${ORDERER_REF}" "${ORDERER_DIR}"

echo "Building ${ORDERER_IMAGE_NAME} image from ${ORDERER_DIR}..."
docker build -t "localhost/${ORDERER_IMAGE_NAME}" -f "${ORDERER_DIR}/node/examples/all-in-one/Dockerfile" "${ORDERER_DIR}"
# Tag with the exact name run-e2e.sh resolves from refs.conf, so no pull is needed.
ORDERER_IMAGE="docker.io/hyperledger/${ORDERER_IMAGE_NAME}:${ORDERER_REF}"
docker tag "localhost/${ORDERER_IMAGE_NAME}" "${ORDERER_IMAGE}"

# ──────────────────────────────────────────────────────────────────────────────
# Build committer image (committer-test-node)
#
# The committer image packages all committer pipeline services (sidecar,
# coordinator, verifier, validator-committer, query) into a single container.
# It uses the committer Makefile target `build-image-test-node`.
# ──────────────────────────────────────────────────────────────────────────────
COMMITTER_DIR="${BUILD_DIR}/fabric-x-committer"
clone_at_ref "${COMMITTER_REPO}" "${COMMITTER_REF}" "${COMMITTER_DIR}"

echo "Building ${COMMITTER_IMAGE_NAME} image from ${COMMITTER_DIR}..."
make -C "${COMMITTER_DIR}" build-image-test-node
# build-image-test-node creates docker.io/hyperledger/${COMMITTER_IMAGE_NAME} locally.
# Tag with refs.conf-derived tag expected by run-e2e.sh.
COMMITTER_IMAGE_BASE="docker.io/hyperledger/${COMMITTER_IMAGE_NAME}"
COMMITTER_IMAGE="${COMMITTER_IMAGE_BASE}:${COMMITTER_REF}"
docker tag "${COMMITTER_IMAGE_BASE}" "${COMMITTER_IMAGE}"

# Loadgen image should match committer ref (as used by run-e2e.sh).
# Re-tag an existing local loadgen image to the expected refs.conf-derived tag.
LOADGEN_IMAGE_BASE="docker.io/hyperledger/fabric-x-loadgen"
LOADGEN_IMAGE="${LOADGEN_IMAGE_BASE}:${COMMITTER_REF}"
if docker image inspect "${LOADGEN_IMAGE_BASE}" >/dev/null 2>&1; then
  docker tag "${LOADGEN_IMAGE_BASE}" "${LOADGEN_IMAGE}"
elif docker image inspect "localhost/fabric-x-loadgen" >/dev/null 2>&1; then
  docker tag "localhost/fabric-x-loadgen" "${LOADGEN_IMAGE}"
else
  echo "ERROR: local loadgen image not found (${LOADGEN_IMAGE_BASE} or localhost/fabric-x-loadgen)."
  echo "build-image-test-node should produce it as part of committer build; if not, build/load it locally first."
  exit 1
fi

# ──────────────────────────────────────────────────────────────────────────────
# Summary
# ──────────────────────────────────────────────────────────────────────────────
echo ""
echo "=== Build complete ==="
echo "Run the E2E test with:"
echo ""
echo "  ./run-e2e.sh"
echo ""
echo "Resolved local tags for run-e2e.sh:"
echo "  ${ORDERER_IMAGE}"
echo "  ${COMMITTER_IMAGE}"
echo "  ${LOADGEN_IMAGE}"

# When running in GitHub Actions, write resolved image names to GITHUB_OUTPUT
# so downstream workflow steps can reference them without parsing stdout.
if [ -n "${GITHUB_OUTPUT:-}" ]; then
  echo "orderer_image=${ORDERER_IMAGE}" >>"${GITHUB_OUTPUT}"
  echo "committer_image=${COMMITTER_IMAGE}" >>"${GITHUB_OUTPUT}"
  echo "loadgen_image=${LOADGEN_IMAGE}" >>"${GITHUB_OUTPUT}"
fi
