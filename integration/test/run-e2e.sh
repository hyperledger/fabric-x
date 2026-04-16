#!/bin/bash
#
# Copyright IBM Corp. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
# =============================================================================
# E2E Integration Test: Arma Orderer + Committer Pipeline + Loadgen
# =============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
LOGS_DUMPED=0

dump_failure_logs() {
  if [ "${LOGS_DUMPED}" -eq 1 ]; then
    return 0
  fi
  LOGS_DUMPED=1

  echo "=== FAILURE: dumping docker compose logs before cleanup ==="
  if [ -f "${SCRIPT_DIR}/docker-compose.yaml" ]; then
    docker compose -f "${SCRIPT_DIR}/docker-compose.yaml" logs --no-color arma committer loadgen || true
  fi
}

# Runs final cleanup for containers/artifacts on script exit.
# Bound to EXIT trap to handle both success and failure paths.
# On successful test completion, prompts user before cleanup so they can
# view Grafana dashboards. On failure, cleans up immediately.
cleanup() {
  local exit_code=$?
  if [ "${exit_code}" -eq 0 ] && [ "${SKIP_CLEANUP_PROMPT:-}" != "1" ]; then
    echo ""
    echo "============================================================"
    echo "  E2E test PASSED!"
    echo ""
    echo "  Grafana dashboard: http://localhost:3000  (admin/admin)"
    echo "  Prometheus:        http://localhost:9090"
    echo "============================================================"
    echo ""
    echo "Press ENTER to clean up containers and artifacts, or Ctrl+C to keep them running..."
    read -r
  fi
  "${SCRIPT_DIR}/clean.sh"
}
trap dump_failure_logs ERR
trap cleanup EXIT

# Polls 127.0.0.1 TCP port until reachable or timeout expires.
# Uses 127.0.0.1 (not localhost) to avoid IPv6 resolution on macOS + Podman
# where IPv6 port-forwarding can cause TLS handshake failures.
# Used for deterministic service readiness checks before progressing.
wait_for_port() {
  local port="$1" name="$2"
  echo "Waiting for ${name} (port ${port})..."
  local i
  for ((i = 1; i <= HEALTH_TIMEOUT; i++)); do
    nc -z 127.0.0.1 "${port}" 2>/dev/null && return 0
    sleep 1
  done
  return 1
}

# Polls an mTLS endpoint until the TLS handshake completes.
# Verifies that the server's TLS stack is fully initialized, not just the
# TCP port. The Arma router and committer sidecar both require client certs,
# so readiness must use the generated mTLS credentials. Use openssl instead
# of curl because these endpoints are gRPC/HTTP2, not regular HTTPS paths.
wait_for_mtls() {
  local port="$1" name="$2" cert="$3" key="$4" ca="$5"
  echo "Waiting for TLS readiness of ${name} (port ${port})..."
  local i
  for ((i = 1; i <= HEALTH_TIMEOUT; i++)); do
    if echo | openssl s_client \
      -connect "127.0.0.1:${port}" \
      -cert "${cert}" \
      -key "${key}" \
      -CAfile "${ca}" \
      -servername 127.0.0.1 \
      2>/dev/null | grep -q "Verify return code: 0 (ok)"; then
      return 0
    fi
    sleep 1
  done
  return 1
}

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------
export ARTIFACTS_DIR="/tmp/fabric-x-test/artifacts"
export STORAGE_DIR="/tmp/fabric-x-test/arma-storage"
DEFAULT_FABRIC_X_BIN="${SCRIPT_DIR}/.build/fabric-x/bin"

REFS_CONF="${SCRIPT_DIR}/refs.conf"
if [ ! -f "${REFS_CONF}" ]; then
  echo "ERROR: refs.conf not found at ${REFS_CONF}"
  exit 1
fi
# shellcheck source=refs.conf
source "${REFS_CONF}"

export ORDERER_IMAGE="${ORDERER_IMAGE:-docker.io/hyperledger/${ORDERER_IMAGE_NAME}:${ORDERER_REF}}"
export COMMITTER_IMAGE="${COMMITTER_IMAGE:-docker.io/hyperledger/${COMMITTER_IMAGE_NAME}:${COMMITTER_REF}}"
export LOADGEN_IMAGE="${LOADGEN_IMAGE:-docker.io/hyperledger/${LOADGEN_IMAGE_NAME}:${COMMITTER_REF}}"
FABRIC_X_BIN="${FABRIC_X_BIN:-${DEFAULT_FABRIC_X_BIN}}"

COMPOSE_FILES=("-f" "${SCRIPT_DIR}/docker-compose.yaml")
HEALTH_TIMEOUT="120"
ENABLE_MONITORING="${ENABLE_MONITORING:-true}"

# Validates required host binaries and generated fabric-x tools are present.
# Fails fast with clear errors to avoid partial E2E setup.
check_prereqs() {
  for bin in docker nc curl openssl; do
    command -v "${bin}" >/dev/null 2>&1 || {
      echo "ERROR: '${bin}' is required but not found in PATH"
      exit 1
    }
  done
  for bin in cryptogen configtxgen fxconfig; do
    [ -x "${FABRIC_X_BIN}/${bin}" ] || {
      echo "ERROR: '${FABRIC_X_BIN}/${bin}' not found. Build binaries under ./.build/fabric-x/bin first."
      exit 1
    }
  done
}

# Prints resolved image references used by docker compose services.
# Helps debugging by making refs.conf resolution explicit in logs.
print_images() {
  echo "Using images:"
  echo "  orderer:   ${ORDERER_IMAGE}"
  echo "  committer: ${COMMITTER_IMAGE}"
  echo "  loadgen:   ${LOADGEN_IMAGE}"
}

# Cleans stale docker resources and prepares runtime artifact/storage directories.
# Removes any leftover artifacts from a previous run before regenerating crypto
# to prevent cryptogen's extend-mode from mixing old and new certificates.
# Normalizes paths and applies macOS-friendly permissions for mounted storage.
step_0_prepare_dirs() {
  echo "=== Step 0: Clean previous run and prepare directories ==="

  "${SCRIPT_DIR}/clean-docker.sh"

  # Remove stale artifacts from previous runs. cryptogen's extend-mode skips
  # existing nodes, so leftover files can produce inconsistent TLS certificates.
  ARTIFACTS_DIR_RESOLVED="${ARTIFACTS_DIR}"
  if [ -d "${ARTIFACTS_DIR_RESOLVED}" ]; then
    ARTIFACTS_DIR_RESOLVED="$(cd "${ARTIFACTS_DIR_RESOLVED}" && pwd -P)"
  fi
  rm -rf "${ARTIFACTS_DIR_RESOLVED}" 2>/dev/null || true

  STORAGE_DIR_RESOLVED="${STORAGE_DIR}"
  if [ -d "${STORAGE_DIR_RESOLVED}" ]; then
    STORAGE_DIR_RESOLVED="$(cd "${STORAGE_DIR_RESOLVED}" && pwd -P)"
  fi
  rm -rf "${STORAGE_DIR_RESOLVED}" 2>/dev/null || true

  mkdir -p "${ARTIFACTS_DIR}"
  ARTIFACTS_DIR="$(cd "${ARTIFACTS_DIR}" && pwd -P)"
  export ARTIFACTS_DIR

  mkdir -p "${STORAGE_DIR}"
  STORAGE_DIR="$(cd "${STORAGE_DIR}" && pwd -P)"
  export STORAGE_DIR

  for i in 1 2 3 4; do
    for role in router assembler batcher consenter; do
      mkdir -p "${STORAGE_DIR}/party${i}/${role}"
      if [ "$(uname)" = "Darwin" ]; then
        chmod 777 "${STORAGE_DIR}/party${i}/${role}"
      fi
    done
  done
}

# Generates PKI material for orderer and peer organizations via cryptogen.
# Writes certificates/keys under the runtime artifacts directory.
step_1_generate_crypto() {
  echo "=== Step 1: Generate crypto material ==="
  "${FABRIC_X_BIN}/cryptogen" generate \
    --config="${SCRIPT_DIR}/networkconfig/crypto-config.yaml" \
    --output="${ARTIFACTS_DIR}"
}

# Renders Arma shared config YAML and compiles it into bootstrap protobuf.
# Executes armageddon inside orderer image with platform-specific user mapping.
step_2_generate_shared_config() {
  echo "=== Step 2: Generate Arma shared config proto ==="

  sed "s|ARTIFACTS_DIR|/artifacts|g" \
    "${SCRIPT_DIR}/networkconfig/arma_config.yaml" >"${ARTIFACTS_DIR}/shared_config.yaml"

  mkdir -p "${ARTIFACTS_DIR}/bootstrap"

  DOCKER_USER_ARGS=()
  if [ "$(uname)" = "Linux" ]; then
    DOCKER_USER_ARGS=("--user" "$(id -u):$(id -g)")
  fi

  docker run --rm --entrypoint armageddon \
    ${DOCKER_USER_ARGS[@]+"${DOCKER_USER_ARGS[@]}"} \
    -v "${ARTIFACTS_DIR}:/artifacts" \
    "${ORDERER_IMAGE}" \
    createSharedConfigProto \
    --sharedConfigYaml="/artifacts/shared_config.yaml" \
    --output="/artifacts/bootstrap"
}

# Builds per-party, per-role orderer local config files from templates.
# Substitutes ports, MSP metadata, paths, and client root CAs via sed.
step_3_generate_local_configs() {
  echo "=== Step 3: Generate local configs ==="

  CONTAINER_ARTIFACTS="/tmp/arma-all-in-one"
  PEER_CA_EXTRA=$(printf '\\\n      - %s\\\n      - %s' \
    "${CONTAINER_ARTIFACTS}/peerOrganizations/peer-org-0/msp/tlscacerts/tlsca.peer-org-0-cert.pem" \
    "${CONTAINER_ARTIFACTS}/peerOrganizations/peer-org-1/msp/tlscacerts/tlsca.peer-org-1-cert.pem")

  for i in 1 2 3 4; do
    PARTY_DIR="${ARTIFACTS_DIR}/config/party${i}"
    mkdir -p "${PARTY_DIR}"

    OFFSET=$(((i - 1) * 100))
    ORG_DOMAIN="orderer-org-${i}"
    PARTY="party${i}"

    for role_tpl in router assembler batcher consenter; do
      case ${role_tpl} in
      router) PORT=$((6022 + OFFSET)) ;;
      assembler) PORT=$((6023 + OFFSET)) ;;
      batcher) PORT=$((6024 + OFFSET)) ;;
      consenter) PORT=$((6025 + OFFSET)) ;;
      esac

      if [ "${role_tpl}" = "batcher" ]; then
        NODE_DIR="batcher1.${ORG_DOMAIN}"
      else
        NODE_DIR="${role_tpl}.${ORG_DOMAIN}"
      fi

      EXTRA_CAS=""
      if [ "${role_tpl}" = "router" ] || [ "${role_tpl}" = "assembler" ]; then
        EXTRA_CAS="${PEER_CA_EXTRA}"
      fi

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

      # Add Operations section for Prometheus metrics endpoint
      OPERATIONS_PORT=$((8000 + OFFSET))
      case ${role_tpl} in
      router) OPERATIONS_PORT=$((OPERATIONS_PORT + 22)) ;;
      assembler) OPERATIONS_PORT=$((OPERATIONS_PORT + 23)) ;;
      batcher) OPERATIONS_PORT=$((OPERATIONS_PORT + 24)) ;;
      consenter) OPERATIONS_PORT=$((OPERATIONS_PORT + 25)) ;;
      esac

      printf '\nOperations:\n  ListenAddress: 0.0.0.0\n  ListenPort: %d\n' "${OPERATIONS_PORT}" \
        >>"${PARTY_DIR}/local_config_${NODE_DIR%%.*}.yaml"
    done
  done
}

# Generates channel configtx input and creates genesis config block.
# Applies read/execute permissions so containers can consume artifacts.
step_4_generate_config_block() {
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

  find "${ARTIFACTS_DIR}" -type d -exec chmod a+rx {} +
  find "${ARTIFACTS_DIR}" -type f -exec chmod a+r {} +
}

# Starts Arma and committer services in detached mode using compose.
# Applies Linux-specific read permission for loadgen config bind mount.
step_5_start_services() {
  echo "=== Step 5: Start arma and committer ==="

  if [ "$(uname)" = "Linux" ]; then
    chmod a+r "${SCRIPT_DIR}/loadgen.yaml"
  fi

  docker compose "${COMPOSE_FILES[@]}" up -d arma committer
}

# Starts Prometheus and Grafana monitoring stack.
# Skipped when ENABLE_MONITORING=false. Dashboards auto-provisioned from
# grafana/ directory. Grafana available at http://localhost:3000.
step_5_1_start_monitoring() {
  if [ "${ENABLE_MONITORING}" != "true" ]; then
    echo "=== Step 5.1: Monitoring disabled (ENABLE_MONITORING=${ENABLE_MONITORING}) ==="
    return 0
  fi

  echo "=== Step 5.1: Start monitoring (Prometheus + Grafana) ==="
  docker compose "${COMPOSE_FILES[@]}" up -d prometheus grafana

  echo "  Grafana:  http://localhost:3000  (admin/admin)"
  echo "  Prometheus: http://localhost:9090"
}

# Waits for key Arma/committer ports to become reachable.
# Verifies TCP port availability and TLS handshake completion.
# TLS readiness check ensures services are fully initialized before
# fxconfig attempts mTLS connections, which can fail if the server's
# TLS stack is not yet ready even though the TCP port is open.
step_6_wait_health() {
  echo "=== Step 6: Wait for health ==="

  wait_for_port 6022 "Arma router" || {
    echo "Arma failed to start"
    docker compose "${COMPOSE_FILES[@]}" logs arma
    exit 1
  }

  wait_for_port 6024 "Arma batcher" || {
    echo "Arma batcher failed to start"
    docker compose "${COMPOSE_FILES[@]}" logs arma
    exit 1
  }

  wait_for_port 4001 "Committer sidecar deliver" || {
    echo "Committer failed to start"
    docker compose "${COMPOSE_FILES[@]}" logs committer
    exit 1
  }

  echo "Waiting for TLS readiness..."
  MTLS_CERT="${ARTIFACTS_DIR}/peerOrganizations/peer-org-0/peers/loadgen.peer-org-0/tls/server.crt"
  MTLS_KEY="${ARTIFACTS_DIR}/peerOrganizations/peer-org-0/peers/loadgen.peer-org-0/tls/server.key"

  wait_for_mtls \
    6022 \
    "Arma router" \
    "${MTLS_CERT}" \
    "${MTLS_KEY}" \
    "${ARTIFACTS_DIR}/ordererOrganizations/orderer-org-1/msp/tlscacerts/tlsca.orderer-org-1-cert.pem" || {
    echo "Arma router TLS not ready"
    docker compose "${COMPOSE_FILES[@]}" logs arma
    exit 1
  }

  wait_for_mtls \
    4001 \
    "Committer sidecar" \
    "${MTLS_CERT}" \
    "${MTLS_KEY}" \
    "${ARTIFACTS_DIR}/peerOrganizations/peer-org-0/msp/tlscacerts/tlsca.peer-org-0-cert.pem" || {
    echo "Committer sidecar TLS not ready"
    docker compose "${COMPOSE_FILES[@]}" logs committer
    exit 1
  }
}

# Creates namespace 0 using multi-org endorsement with fxconfig workflow.
# Renders per-org configs, endorses twice, merges, and submits transaction.
step_7_create_namespace() {
  echo "=== Step 7: Create namespace with fxconfig (multi-org) ==="

  FXCONFIG_ORG0="${ARTIFACTS_DIR}/fxconfig-peer-org-0.yaml"
  FXCONFIG_ORG1="${ARTIFACTS_DIR}/fxconfig-peer-org-1.yaml"

  sed "s|ARTIFACTS_DIR|${ARTIFACTS_DIR}|g" \
    "${SCRIPT_DIR}/fxconfig-peer-org-0.yaml" >"${FXCONFIG_ORG0}"
  sed "s|ARTIFACTS_DIR|${ARTIFACTS_DIR}|g" \
    "${SCRIPT_DIR}/fxconfig-peer-org-1.yaml" >"${FXCONFIG_ORG1}"

  FXCONFIG_TX_DIR="${ARTIFACTS_DIR}/fxconfig-tx"
  mkdir -p "${FXCONFIG_TX_DIR}"

  "${FABRIC_X_BIN}/fxconfig" namespace create 0 \
    --config="${FXCONFIG_ORG0}" \
    --policy="AND('peer-org-0.member', 'peer-org-1.member')" \
    --output="${FXCONFIG_TX_DIR}/tx.json"

  "${FABRIC_X_BIN}/fxconfig" tx endorse "${FXCONFIG_TX_DIR}/tx.json" \
    --config="${FXCONFIG_ORG0}" \
    --output="${FXCONFIG_TX_DIR}/tx_org0.json" </dev/null

  "${FABRIC_X_BIN}/fxconfig" tx endorse "${FXCONFIG_TX_DIR}/tx.json" \
    --config="${FXCONFIG_ORG1}" \
    --output="${FXCONFIG_TX_DIR}/tx_org1.json" </dev/null

  "${FABRIC_X_BIN}/fxconfig" tx merge \
    "${FXCONFIG_TX_DIR}/tx_org0.json" \
    "${FXCONFIG_TX_DIR}/tx_org1.json" \
    --output="${FXCONFIG_TX_DIR}/tx_merged.json" </dev/null

  "${FABRIC_X_BIN}/fxconfig" tx submit --wait \
    "${FXCONFIG_TX_DIR}/tx_merged.json" \
    --config="${FXCONFIG_ORG0}" </dev/null

  echo "Namespace 0 created successfully"
}

# Launches loadgen, streams progress logs, and waits for completion.
# Accepts exit codes 0/1 and preserves final logs without forced termination.
step_8_run_loadgen() {
  echo "=== Step 8: Run loadgen ==="
  docker compose "${COMPOSE_FILES[@]}" up -d loadgen

  echo "=== Step 8.1: Stream loadgen progress logs until completion ==="
  docker compose "${COMPOSE_FILES[@]}" logs -f --no-color loadgen &
  LOGS_PID=$!

  echo "=== Step 8.2: Wait for loadgen to complete ==="
  LOADGEN_EXIT_CODE="$(docker wait loadgen)"

  wait "${LOGS_PID}" 2>/dev/null || true

  if [ "${LOADGEN_EXIT_CODE}" != "0" ] && [ "${LOADGEN_EXIT_CODE}" != "1" ]; then
    echo "FAILURE: loadgen exited with unexpected code ${LOADGEN_EXIT_CODE}"
    docker compose "${COMPOSE_FILES[@]}" logs loadgen
    exit 1
  fi

  echo "Loadgen exited with code ${LOADGEN_EXIT_CODE} (accepted)"
}

# Queries VC Prometheus metrics over mTLS to read committed tx count.
# Enforces minimum threshold and dumps logs on verification failure.
step_9_verify_results() {
  echo "=== Step 9: Verify results ==="

  COMMITTED_TXS=$(curl -s \
    --cert "${ARTIFACTS_DIR}/peerOrganizations/peer-org-0/peers/loadgen.peer-org-0/tls/server.crt" \
    --key "${ARTIFACTS_DIR}/peerOrganizations/peer-org-0/peers/loadgen.peer-org-0/tls/server.key" \
    --cacert "${ARTIFACTS_DIR}/peerOrganizations/peer-org-0/msp/tlscacerts/tlsca.peer-org-0-cert.pem" \
    https://127.0.0.1:2116/metrics 2>/dev/null | grep '^vcservice_committed_transaction_total' | awk '{print $2}')

  echo "Committed transactions: ${COMMITTED_TXS}"

  if [ "${COMMITTED_TXS:-0}" -ge 5000 ]; then
    echo "SUCCESS: E2E test passed (${COMMITTED_TXS} transactions committed)"
  else
    echo "FAILURE: Expected >= 5000 committed transactions, got ${COMMITTED_TXS:-0}"
    docker compose "${COMPOSE_FILES[@]}" logs
    exit 1
  fi
}

# Orchestrates the full E2E lifecycle in deterministic step order.
# Keeps control flow concise and makes each phase easy to locate.
main() {
  check_prereqs
  print_images
  step_0_prepare_dirs
  step_1_generate_crypto
  step_2_generate_shared_config
  step_3_generate_local_configs
  step_4_generate_config_block
  step_5_start_services
  step_5_1_start_monitoring
  step_6_wait_health
  step_7_create_namespace
  step_8_run_loadgen
  step_9_verify_results
}

main "$@"
