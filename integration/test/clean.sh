#!/bin/bash
#
# Copyright IBM Corp. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
# =============================================================================
# E2E Integration Test - Cleanup
# =============================================================================
#
# Stops and removes all Docker containers and networks created by the E2E test,
# then deletes all generated artifacts from the filesystem.
#
# Safe to run multiple times (idempotent).
#
# What gets cleaned:
#   - Docker: containers (arma, committer, loadgen), volumes, and e2e network
#   - artifacts/     - crypto material, configs, genesis block
#   - arma-storage/  - orderer ledger data (persistent storage for 4 parties)
#   - .build/        - cloned repos from build-e2e.sh (if used)
#

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Stop and remove all containers, volumes, and networks from docker-compose
docker compose -f "${SCRIPT_DIR}/docker-compose.yaml" down -v 2>/dev/null || true

# Remove temp directories (may be overridden via env vars).
# Resolve symlinks (e.g., /tmp -> /private/tmp on macOS) so we clean the
# actual directories, matching how run-e2e.sh resolves them via pwd -P.
ARTIFACTS_DIR="${ARTIFACTS_DIR:-/tmp/fabric-x-test/artifacts}"
STORAGE_DIR="${STORAGE_DIR:-/tmp/fabric-x-test/arma-storage}"
if [ -d "${ARTIFACTS_DIR}" ]; then
  ARTIFACTS_DIR="$(cd "${ARTIFACTS_DIR}" && pwd -P)"
fi
if [ -d "${STORAGE_DIR}" ]; then
  STORAGE_DIR="$(cd "${STORAGE_DIR}" && pwd -P)"
fi
rm -rf "${ARTIFACTS_DIR}" "${STORAGE_DIR}" "${SCRIPT_DIR}/.build"
