#!/bin/bash
#
# Copyright IBM Corp. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
# =============================================================================
# E2E Integration Test - Docker Cleanup
# =============================================================================
#
# Stops/removes only Docker resources used by the integration E2E test.
# Does not remove local filesystem artifacts or .build outputs.
#
# Safe to run multiple times (idempotent).

set -euo pipefail

# Stop and remove containers directly — avoids 'docker compose down -v' hanging
# indefinitely on macOS + Podman due to network cleanup blocking in compose CLI.
docker rm -f arma committer loadgen 2>/dev/null || true

# Remove the compose-managed bridge network (project name defaults to dir name "test").
docker network rm test_e2e 2>/dev/null || true
