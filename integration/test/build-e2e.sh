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
if [ -z "${FABRIC_X_REF}" ]; then
  echo "ERROR: FABRIC_X_REF is not set. Please specify --fabric-x-ref or set it in refs.conf"
  exit 1
fi
if [ -z "${COMMITTER_REF}" ]; then
  echo "ERROR: COMMITTER_REF is not set. Please specify --committer-ref or set it in refs.conf"
  exit 1
fi
if [ -z "${ORDERER_REF}" ]; then
  echo "ERROR: ORDERER_REF is not set. Please specify --orderer-ref or set it in refs.conf"
  exit 1
fi
if [ -z "${ORDERER_IMAGE_NAME:-}" ]; then
  echo "ERROR: ORDERER_IMAGE_NAME is not set. Check refs.conf"
  exit 1
fi
if [ -z "${COMMITTER_IMAGE_NAME:-}" ]; then
  echo "ERROR: COMMITTER_IMAGE_NAME is not set. Check refs.conf"
  exit 1
fi

# ──────────────────────────────────────────────────────────────────────────────
# Helper functions
# ──────────────────────────────────────────────────────────────────────────────

# try_pull attempts to pull a pre-built image from a registry.
# Returns 0 on success so callers can skip the local build.
try_pull() {
  local image="$1"
  echo "Trying to pull ${image}..."
  if docker pull "${image}" 2>/dev/null; then
    echo "Pulled ${image}"
    return 0
  fi
  return 1
}

# is_release_tag returns 0 if the ref matches the semver pattern vX.Y.Z.
# Only release tags are candidates for pre-built registry pulls.
is_release_tag() {
  [[ "$1" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]
}

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
ORDERER_IMAGE=""

# For release tags, try pulling the pre-built image first to avoid a full build.
if is_release_tag "${ORDERER_REF}"; then
  if try_pull "docker.io/hyperledger/${ORDERER_IMAGE_NAME}:${ORDERER_REF}"; then
    ORDERER_IMAGE="docker.io/hyperledger/${ORDERER_IMAGE_NAME}:${ORDERER_REF}"
  fi
fi

# Fall back to building locally if no pre-built image was found.
if [ -z "${ORDERER_IMAGE}" ]; then
  ORDERER_DIR="${BUILD_DIR}/fabric-x-orderer"
  clone_at_ref "${ORDERER_REPO}" "${ORDERER_REF}" "${ORDERER_DIR}"

  echo "Building ${ORDERER_IMAGE_NAME} image from ${ORDERER_DIR}..."
  docker build -t "localhost/${ORDERER_IMAGE_NAME}" -f "${ORDERER_DIR}/node/examples/all-in-one/Dockerfile" "${ORDERER_DIR}"
  ORDERER_IMAGE="localhost/${ORDERER_IMAGE_NAME}"
fi

# ──────────────────────────────────────────────────────────────────────────────
# Build committer image (committer-test-node)
#
# The committer image packages all committer pipeline services (sidecar,
# coordinator, verifier, validator-committer, query) into a single container.
# It uses the committer Makefile target `build-image-test-node`.
# ──────────────────────────────────────────────────────────────────────────────
COMMITTER_IMAGE=""

# For release tags, try pulling the pre-built image first.
if is_release_tag "${COMMITTER_REF}"; then
  if try_pull "docker.io/hyperledger/${COMMITTER_IMAGE_NAME}:${COMMITTER_REF}"; then
    COMMITTER_IMAGE="docker.io/hyperledger/${COMMITTER_IMAGE_NAME}:${COMMITTER_REF}"
  fi
fi

# Fall back to building locally if no pre-built image was found.
if [ -z "${COMMITTER_IMAGE}" ]; then
  COMMITTER_DIR="${BUILD_DIR}/fabric-x-committer"
  clone_at_ref "${COMMITTER_REPO}" "${COMMITTER_REF}" "${COMMITTER_DIR}"

  echo "Building ${COMMITTER_IMAGE_NAME} image from ${COMMITTER_DIR}..."
  make -C "${COMMITTER_DIR}" build-image-test-node
  COMMITTER_IMAGE="docker.io/hyperledger/${COMMITTER_IMAGE_NAME}"
fi

# ──────────────────────────────────────────────────────────────────────────────
# Summary
# ──────────────────────────────────────────────────────────────────────────────
echo ""
echo "=== Build complete ==="
echo "Run the E2E test with:"
echo ""
echo "  export ORDERER_IMAGE=${ORDERER_IMAGE}"
echo "  export COMMITTER_IMAGE=${COMMITTER_IMAGE}"
echo "  ./run-e2e.sh"

# When running in GitHub Actions, write resolved image names to GITHUB_OUTPUT
# so downstream workflow steps can reference them without parsing stdout.
if [ -n "${GITHUB_OUTPUT:-}" ]; then
  echo "orderer_image=${ORDERER_IMAGE}" >> "${GITHUB_OUTPUT}"
  echo "committer_image=${COMMITTER_IMAGE}" >> "${GITHUB_OUTPUT}"
fi

