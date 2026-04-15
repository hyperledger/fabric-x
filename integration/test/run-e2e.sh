#!/bin/bash
#
# Copyright IBM Corp. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
# =============================================================================
# E2E Integration Test: Arma Orderer + Committer Pipeline + Loadgen
# =============================================================================
#
# This script runs a full end-to-end integration test that exercises the entire
# Fabric-X transaction lifecycle:
#
#   Loadgen --> Arma Routers (BFT broadcast)
#       --> Arma Consensus (4 parties, SmartBFT)
#       --> Arma Assemblers --> Committer Sidecar (BFT block delivery)
#       --> Coordinator --> Verifier --> Validator-Committer (VC)
#
# The test generates crypto material, config blocks, and orderer configs on the
# host, then runs the orderer, committer, and loadgen as Docker containers.
# After the loadgen completes, it verifies that the VC committed >= 5000 TXs.
#
# Prerequisites:
#   - Docker (with compose plugin)
#   - cryptogen, configtxgen, and fxconfig binaries in fabric-x/bin/ (built from this repo)
#   - curl (for metrics verification)
#   - nc (netcat, for health checks)
#
# Usage:
#   ./run-e2e.sh                                    # use default images
#   ORDERER_IMAGE=myimage:tag ./run-e2e.sh          # custom orderer image
#   COMMITTER_IMAGE=myimage:tag ./run-e2e.sh        # custom committer image
#
# Inputs (env vars with defaults):
#   ORDERER_IMAGE    - Arma 4-party-1-shard Docker image (default: localhost/arma-4p1s)
#   COMMITTER_IMAGE  - Committer test node image (default: docker.io/hyperledger/committer-test-node)
#   LOADGEN_IMAGE    - Load generator image (default: docker.io/hyperledger/fabric-x-loadgen)
#   FABRIC_X_BIN     - Path to cryptogen/configtxgen/fxconfig binaries (default: ../../bin)
#
# See also:
#   build-e2e.sh  - Build Docker images from specific commits/tags
#   clean.sh      - Remove all generated artifacts and containers
#
set -euo pipefail

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------

# Resolve script directory (works regardless of where the script is invoked from)
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# All generated artifacts (crypto, configs, blocks) go here.
# This directory is mounted into containers and cleaned between runs.
# Uses OS temp directory by default, can be overridden via env var.
export ARTIFACTS_DIR="${ARTIFACTS_DIR:-/tmp/fabric-x-test/artifacts}"
export STORAGE_DIR="${STORAGE_DIR:-/tmp/fabric-x-test/arma-storage}"

# Path to fabric-x host binaries (cryptogen, configtxgen).
# These are built from the fabric-x repo and must match the host OS.
FABRIC_X_BIN="${FABRIC_X_BIN:-${SCRIPT_DIR}/../../bin}"

# Docker images — override via env vars for testing different versions.
# These are exported so docker-compose.yaml can reference them.
export ORDERER_IMAGE="${ORDERER_IMAGE:-localhost/arma-4p1s}"
export COMMITTER_IMAGE="${COMMITTER_IMAGE:-docker.io/hyperledger/committer-test-node}"
export LOADGEN_IMAGE="${LOADGEN_IMAGE:-docker.io/hyperledger/fabric-x-loadgen}"

# Docker Compose file list. On Linux, include a user-mapping override so
# containers run as the host user and avoid permission issues with bind-mounts.
COMPOSE_FILES=("-f" "${SCRIPT_DIR}/docker-compose.yaml")
if [ "$(uname)" = "Linux" ]; then
  export COMPOSE_UID="$(id -u)"
  export COMPOSE_GID="$(id -g)"
  COMPOSE_FILES+=("-f" "${SCRIPT_DIR}/docker-compose.linux.yaml")
fi

echo "Using images:"
echo "  orderer:   ${ORDERER_IMAGE}"
echo "  committer: ${COMMITTER_IMAGE}"
echo "  loadgen:   ${LOADGEN_IMAGE}"

# ---------------------------------------------------------------------------
# Step 0: Clean previous run and create directory structure
# ---------------------------------------------------------------------------

# Remove any leftover containers, volumes, and artifacts from a previous run
"${SCRIPT_DIR}/clean.sh"

# Create the artifacts directory (crypto, configs, blocks will go here)
mkdir -p "${ARTIFACTS_DIR}"
# Resolve symlinks (e.g., /tmp -> /private/tmp on macOS) so Docker
# bind-mounts work correctly.
ARTIFACTS_DIR="$(cd "${ARTIFACTS_DIR}" && pwd -P)"
export ARTIFACTS_DIR

# Create persistent storage directories for each of the 4 Arma parties.
# Each party has 4 roles: router, assembler, batcher, consenter.
# These are mounted into the arma container at /storage/partyN/role/.
#
# Platform-specific handling:
# - Linux: Containers run with --user flag (host UID:GID)
# - macOS: Docker Desktop runs in a VM, so --user is ignored. We make
#   directories world-writable to allow root containers to write.
mkdir -p "${STORAGE_DIR}"
STORAGE_DIR="$(cd "${STORAGE_DIR}" && pwd -P)"
export STORAGE_DIR

for i in 1 2 3 4; do
  for role in router assembler batcher consenter; do
    mkdir -p "${STORAGE_DIR}/party${i}/${role}"
    # On macOS Docker Desktop, containers run as root in a VM.
    # Files created by root appear as root-owned on the host.
    # 777 allows host user to delete files during cleanup (directory write perm).
    if [ "$(uname)" = "Darwin" ]; then
      chmod 777 "${STORAGE_DIR}/party${i}/${role}"
    fi
  done
done

# ---------------------------------------------------------------------------
# Step 1: Generate crypto material (certificates and keys)
# ---------------------------------------------------------------------------
# Uses cryptogen with crypto-config.yaml to generate:
#   - 4 orderer organizations (orderer-org-1 through orderer-org-4)
#     Each with router, assembler, batcher, consenter node identities
#   - 2 peer organizations (peer-org-0, peer-org-1)
#     Each with client, admin, and loadgen identities
echo "=== Step 1: Generate crypto material ==="
"${FABRIC_X_BIN}/cryptogen" generate \
  --config="${SCRIPT_DIR}/networkconfig/crypto-config.yaml" \
  --output="${ARTIFACTS_DIR}"

# ---------------------------------------------------------------------------
# Step 2: Generate Arma shared config protobuf
# ---------------------------------------------------------------------------
# The shared config defines the Arma network topology: which parties exist,
# their roles (router, assembler, batcher, consenter), endpoints, and certs.
#
# arma_config.yaml uses "ARTIFACTS_DIR" as a placeholder for cert paths.
# We replace it with "/artifacts" (the container mount path) since armageddon
# runs inside a container where artifacts are mounted at /artifacts.
echo "=== Step 2: Generate Arma shared config proto ==="
sed "s|ARTIFACTS_DIR|/artifacts|g" \
  "${SCRIPT_DIR}/networkconfig/arma_config.yaml" >"${ARTIFACTS_DIR}/shared_config.yaml"

# Run armageddon inside the orderer image to generate the shared config protobuf.
# Unlike cryptogen, configtxgen, and fxconfig (which are fabric-x binaries available
# on the host), armageddon is a fabric-x-orderer binary and is only available inside
# the orderer image. The --entrypoint flag overrides the default entrypoint.sh which
# would try to start the full orderer.
mkdir -p "${ARTIFACTS_DIR}/bootstrap"
DOCKER_USER_FLAG=""
if [ "$(uname)" = "Linux" ]; then
  DOCKER_USER_FLAG="--user $(id -u):$(id -g)"
fi
docker run --rm --entrypoint armageddon \
  ${DOCKER_USER_FLAG} \
  -v "${ARTIFACTS_DIR}:/artifacts" \
  "${ORDERER_IMAGE}" \
  createSharedConfigProto \
  --sharedConfigYaml="/artifacts/shared_config.yaml" \
  --output="/artifacts/bootstrap"

# ---------------------------------------------------------------------------
# Step 3: Generate orderer local configs (one per party per role)
# ---------------------------------------------------------------------------
# Each Arma party has 4 roles, each needing its own local config YAML:
#   - router:    accepts client broadcast requests (port 60x2)
#   - assembler: delivers blocks to sidecars (port 60x3)
#   - batcher:   batches transactions for consensus (port 60x4)
#   - consenter: participates in SmartBFT consensus (port 60x5)
#
# Configs are built by concatenating a base template with a role-specific
# snippet, then replacing placeholders with actual values.
#
# IMPORTANT: These configs use CONTAINER-INTERNAL paths (/tmp/arma-all-in-one)
# because they run inside the arma container, not on the host.
echo "=== Step 3: Generate local configs ==="
CONTAINER_ARTIFACTS="/tmp/arma-all-in-one"

# Router and assembler accept connections from peer orgs (sidecar, loadgen),
# so their ClientRootCAs must include the peer org TLS CA certificates.
# Use \<newline> escaping so sed replacement works on both BSD (macOS) and GNU (Linux).
PEER_CA_EXTRA=$(printf '\\\n      - %s\\\n      - %s' \
  "${CONTAINER_ARTIFACTS}/peerOrganizations/peer-org-0/msp/tlscacerts/tlsca.peer-org-0-cert.pem" \
  "${CONTAINER_ARTIFACTS}/peerOrganizations/peer-org-1/msp/tlscacerts/tlsca.peer-org-1-cert.pem")

for i in 1 2 3 4; do
  PARTY_DIR="${ARTIFACTS_DIR}/config/party${i}"
  mkdir -p "${PARTY_DIR}"

  # Port offset: party1=0, party2=100, party3=200, party4=300
  # e.g., party1 router=6022, party2 router=6122, etc.
  OFFSET=$(((i - 1) * 100))
  ORG_DOMAIN="orderer-org-${i}"
  PARTY="party${i}"

  for role_tpl in router assembler batcher consenter; do
    # Each role gets a unique port within the party's port range
    case ${role_tpl} in
    router) PORT=$((6022 + OFFSET)) ;;
    assembler) PORT=$((6023 + OFFSET)) ;;
    batcher) PORT=$((6024 + OFFSET)) ;;
    consenter) PORT=$((6025 + OFFSET)) ;;
    esac

    # Cryptogen creates node directories in Hostname.Domain format
    # e.g., router.orderer-org-1/, assembler.orderer-org-2/
    # (cryptogen creates node dirs without suffix; only one batcher exists)
    if [ "${role_tpl}" = "batcher" ]; then
      NODE_DIR="batcher1.${ORG_DOMAIN}"
    else
      NODE_DIR="${role_tpl}.${ORG_DOMAIN}"
    fi

    # Only router and assembler need peer org CAs in their ClientRootCAs
    # (they accept client connections from sidecar and loadgen)
    EXTRA_CAS=""
    if [ "${role_tpl}" = "router" ] || [ "${role_tpl}" = "assembler" ]; then
      EXTRA_CAS="${PEER_CA_EXTRA}"
    fi

    # Build the config by concatenating base template + role snippet,
    # then replacing all placeholders with actual values via sed
    cat "${SCRIPT_DIR}/ordererconfig/base.yaml.tpl" \
      "${SCRIPT_DIR}/ordererconfig/role_${role_tpl}.yaml" |
      sed \
        -e "s|ARTIFACTS_DIR|${CONTAINER_ARTIFACTS}|g" \
        -e "s|PORT|${PORT}|g" \
        -e "s|ORG_DOMAIN|${ORG_DOMAIN}|g" \
        -e "s|ORG_MSP_ID|OrdererOrg${i}MSP|g" \
        -e "s|PARTY_ID|${i}|g" \
        -e "s|PARTY|${PARTY}|g" \
        -e "s|NODE_DIR|${NODE_DIR}|g" \
        -e "s|STORAGE_DIR|/storage/party${i}/${role_tpl}|g" \
        -e "s|CLIENT_ROOT_CAS_EXTRA|${EXTRA_CAS}|g" \
        >"${PARTY_DIR}/local_config_${NODE_DIR%%.*}.yaml"
  done
done

# ---------------------------------------------------------------------------
# Step 4: Generate the channel genesis (config) block
# ---------------------------------------------------------------------------
# configtxgen reads networkconfig/configtx.yaml which defines:
#   - 4 orderer orgs + 2 peer orgs with MSP directories
#   - Channel capabilities (V3_0 for BFT support)
#   - ConsenterMapping (4 consenters with signing certs)
#   - BlockValidation policy (ImplicitMeta: ANY Writers)
#   - Arma shared config path for the orderer section
#
# The output config-block.pb.bin is the genesis block that bootstraps the
# orderer and is used by the sidecar and loadgen for channel configuration.
echo "=== Step 4: Generate config block ==="
CONFIGTX_DIR="${ARTIFACTS_DIR}/networkconfig"
mkdir -p "${CONFIGTX_DIR}"
sed "s|ARTIFACTS_DIR|${ARTIFACTS_DIR}|g" \
  "${SCRIPT_DIR}/networkconfig/configtx.yaml" >"${CONFIGTX_DIR}/configtx.yaml"

"${FABRIC_X_BIN}/configtxgen" \
  -profile E2EProfile \
  -channelID mychannel \
  -configPath "${CONFIGTX_DIR}" \
  -outputBlock "${ARTIFACTS_DIR}/config-block.pb.bin"

# Allow container users to traverse the artifacts tree and read all files.
# The loadgen image runs as uid 10001 (non-root) and needs read access to
# certs, configs, blocks, and its own TLS private key.
find "${ARTIFACTS_DIR}" -type d -exec chmod a+rx {} +
find "${ARTIFACTS_DIR}" -type f -exec chmod a+r {} +

# ---------------------------------------------------------------------------
# Step 5: Start the orderer and committer containers
# ---------------------------------------------------------------------------
# - arma: Runs all 4 parties (16 processes) in a single container.
#   Mounts artifacts at /tmp/arma-all-in-one and storage at /storage.
#   The entrypoint.sh detects pre-generated configs and skips generation.
#
# - committer: Runs the full committer pipeline (embedded PostgreSQL + 5 services).
#   Command "run db committer" starts the DB first, then all committer services.
#   Mounts artifacts at /root/artifacts and configs at /root/config.
echo "=== Step 5: Start arma and committer ==="
docker compose "${COMPOSE_FILES[@]}" up -d arma committer

# ---------------------------------------------------------------------------
# Step 6: Wait for services to be healthy
# ---------------------------------------------------------------------------
# Check that the router (port 6022) and sidecar deliver (port 4001) are
# accepting connections before starting the loadgen.
echo "=== Step 6: Wait for health ==="

echo "Waiting for Arma (router port 6022)..."
for i in $(seq 1 60); do
  nc -z localhost 6022 2>/dev/null && break
  sleep 1
done
nc -z localhost 6022 || {
  echo "Arma failed to start"
  docker compose "${COMPOSE_FILES[@]}" logs arma
  exit 1
}

echo "Waiting for Arma (batcher port 6024)..."
for i in $(seq 1 60); do
  nc -z localhost 6024 2>/dev/null && break
  sleep 1
done
nc -z localhost 6024 || {
  echo "Arma batcher failed to start"
  docker compose "${COMPOSE_FILES[@]}" logs arma
  exit 1
}

echo "Waiting for Committer (sidecar deliver port 4001)..."
for i in $(seq 1 60); do
  nc -z localhost 4001 2>/dev/null && break
  sleep 1
done
nc -z localhost 4001 || {
  echo "Committer failed to start"
  docker compose "${COMPOSE_FILES[@]}" logs committer
  exit 1
}

# ---------------------------------------------------------------------------
# Step 7: Create namespace using fxconfig (multi-org endorsement)
# ---------------------------------------------------------------------------
# Use fxconfig (host binary) to create namespace "0" with an MSP-based
# endorsement policy requiring both peer organizations. The channel's
# LifecycleEndorsement policy is MAJORITY, so namespace creation transactions
# must be endorsed by both peer-org-0 and peer-org-1.
#
# The multi-org endorsement pipeline:
#   1. Create the unsigned namespace transaction (peer-org-0 config)
#   2. Endorse with peer-org-0
#   3. Endorse with peer-org-1
#   4. Merge both endorsements into a single transaction
#   5. Submit the merged transaction and wait for commit
#
# The fxconfig config templates use ARTIFACTS_DIR as a placeholder for cert
# paths. We replace it with the actual artifacts directory path via sed.
echo "=== Step 7: Create namespace with fxconfig (multi-org) ==="
FXCONFIG_ORG0="${ARTIFACTS_DIR}/fxconfig-peer-org-0.yaml"
FXCONFIG_ORG1="${ARTIFACTS_DIR}/fxconfig-peer-org-1.yaml"
sed "s|ARTIFACTS_DIR|${ARTIFACTS_DIR}|g" \
  "${SCRIPT_DIR}/fxconfig-peer-org-0.yaml" >"${FXCONFIG_ORG0}"
sed "s|ARTIFACTS_DIR|${ARTIFACTS_DIR}|g" \
  "${SCRIPT_DIR}/fxconfig-peer-org-1.yaml" >"${FXCONFIG_ORG1}"

FXCONFIG_TX_DIR="${ARTIFACTS_DIR}/fxconfig-tx"
mkdir -p "${FXCONFIG_TX_DIR}"

# Step 7a: Create unsigned namespace transaction
"${FABRIC_X_BIN}/fxconfig" namespace create 0 \
  --config="${FXCONFIG_ORG0}" \
  --policy="AND('peer-org-0.member', 'peer-org-1.member')" \
  --output="${FXCONFIG_TX_DIR}/tx.json"

# Step 7b: Endorse with peer-org-0
# Note: < /dev/null closes stdin so fxconfig does not detect a pipe
# (on Linux/SSH, stdin is a pipe which conflicts with positional file args)
"${FABRIC_X_BIN}/fxconfig" tx endorse "${FXCONFIG_TX_DIR}/tx.json" \
  --config="${FXCONFIG_ORG0}" \
  --output="${FXCONFIG_TX_DIR}/tx_org0.json" < /dev/null

# Step 7c: Endorse with peer-org-1
"${FABRIC_X_BIN}/fxconfig" tx endorse "${FXCONFIG_TX_DIR}/tx.json" \
  --config="${FXCONFIG_ORG1}" \
  --output="${FXCONFIG_TX_DIR}/tx_org1.json" < /dev/null

# Step 7d: Merge both endorsements
"${FABRIC_X_BIN}/fxconfig" tx merge \
  "${FXCONFIG_TX_DIR}/tx_org0.json" \
  "${FXCONFIG_TX_DIR}/tx_org1.json" \
  --output="${FXCONFIG_TX_DIR}/tx_merged.json" < /dev/null

# Step 7e: Submit merged transaction and wait for commit
"${FABRIC_X_BIN}/fxconfig" tx submit --wait \
  "${FXCONFIG_TX_DIR}/tx_merged.json" \
  --config="${FXCONFIG_ORG0}" < /dev/null

echo "Namespace 0 created successfully"

# ---------------------------------------------------------------------------
# Step 8: Run the load generator
# ---------------------------------------------------------------------------
# The loadgen submits transactions to Arma routers via BFT broadcast and
# receives committed blocks from the committer sidecar. It runs until the
# transaction limit (configured in loadgen.yaml) is reached.
#
# Namespace creation is handled by fxconfig in Step 7 (loadgen.yaml has
# generate.namespaces: false). The loadgen only generates load.
#
# The loadgen exits with code 1 on normal shutdown due to "context canceled"
# error — this is expected behavior, so we tolerate non-zero exit.
echo "=== Step 8: Run loadgen ==="
# Timeout after 10 minutes to prevent hanging in local runs.
# CI has its own 15-minute job-level timeout, but local runs have no protection.
timeout 600 docker compose "${COMPOSE_FILES[@]}" up loadgen || true

# ---------------------------------------------------------------------------
# Step 9: Verify results via Prometheus metrics
# ---------------------------------------------------------------------------
# Query the Validator-Committer (VC) service's Prometheus metrics endpoint
# to check how many transactions were actually committed to the database.
#
# The metrics endpoint uses mTLS, so we provide peer org TLS client certs.
# Port 2116 is the VC monitoring port (mapped from the committer container).
echo "=== Step 9: Verify results ==="
COMMITTED_TXS=$(curl -sk \
  --cert "${ARTIFACTS_DIR}/peerOrganizations/peer-org-0/peers/loadgen.peer-org-0/tls/server.crt" \
  --key "${ARTIFACTS_DIR}/peerOrganizations/peer-org-0/peers/loadgen.peer-org-0/tls/server.key" \
  https://localhost:2116/metrics 2>/dev/null | grep '^vcservice_committed_transaction_total' | awk '{print $2}')
echo "Committed transactions: ${COMMITTED_TXS}"

# Expect at least 5000 committed transactions (loadgen sends ~10000).
# The threshold is intentionally lower than the limit to allow for
# batching variations and timing differences.
if [ "${COMMITTED_TXS:-0}" -ge 5000 ]; then
  echo "SUCCESS: E2E test passed (${COMMITTED_TXS} transactions committed)"
else
  echo "FAILURE: Expected >= 5000 committed transactions, got ${COMMITTED_TXS:-0}"
  docker compose "${COMPOSE_FILES[@]}" logs
  exit 1
fi

# ---------------------------------------------------------------------------
# Step 10: Cleanup
# ---------------------------------------------------------------------------
# Stop all containers, remove volumes, and delete generated artifacts.
echo "=== Step 10: Cleanup ==="
"${SCRIPT_DIR}/clean.sh"
