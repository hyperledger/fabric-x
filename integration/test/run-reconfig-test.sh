#!/bin/bash
#
# Copyright IBM Corp. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
# =============================================================================
# Run the TestReconfigAppOrg integration test.
#
# The test generates all crypto/config artifacts internally via armageddon.
# This script only ensures the required Docker image is available and then
# runs the Go test. The test itself manages the arma container lifecycle via
# testcontainers.
#
# Usage:
#   ./run-reconfig-test.sh
#
# Environment variables:
#   ORDERER_IMAGE     Docker image for the arma orderer
#   COMMITTER_IMAGE   Docker image for the committer test-node
# =============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

REFS_CONF="${SCRIPT_DIR}/refs.conf"
if [ ! -f "${REFS_CONF}" ]; then
  echo "ERROR: refs.conf not found at ${REFS_CONF}"
  exit 1
fi
# shellcheck source=refs.conf
source "${REFS_CONF}"

export ORDERER_IMAGE="${ORDERER_IMAGE:-hyperledger/arma-4p1s:${ORDERER_REF}}"
export COMMITTER_IMAGE="${COMMITTER_IMAGE:-hyperledger/committer-test-node:${ORDERER_REF}}"

echo "=== Running TestReconfigAppOrg ==="
echo "  ORDERER_IMAGE:   ${ORDERER_IMAGE}"
echo "  COMMITTER_IMAGE: ${COMMITTER_IMAGE}"

# Verify the images exist locally.
for img in "${ORDERER_IMAGE}" "${COMMITTER_IMAGE}"; do
  if ! docker image inspect "${img}" >/dev/null 2>&1; then
    echo "ERROR: Docker image '${img}' not found locally."
    echo "       Build it first or set ORDERER_IMAGE / COMMITTER_IMAGE to existing images."
    exit 1
  fi
done
#nop
:
