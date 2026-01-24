#!/usr/bin/env bash

set -exo pipefail

#
# Copyright IBM Corp. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

CONF_ROOT=$(realpath "${CONF_ROOT:-$(pwd)/conf}")
CRYPTO_DIR="${CRYPTO_DIR:-./out/control-node/config/cryptogen-artifacts/crypto}"

echo "$CRYPTO_DIR"

for d in "${CONF_ROOT}"/*/ ; do
    rm -rf "$d/keys/fabric"
    rm -rf "$d/data" # data will be useless when we deleted the ledger
done

set -e

# Org1MSP fabric user and peer TLS (for simplicity we use Org1MSP for all nodes)
nodes=(issuer owner1 owner2 endorser1)
for node in "${nodes[@]}"; do
    dir="${CONF_ROOT}/${node}/keys/fabric"
    mkdir -p "${CONF_ROOT}/${node}/data"
    mkdir -p "$dir"

    cp -r "${CRYPTO_DIR}/peerOrganizations/org1.example.com/users/User1@org1.example.com/msp" "$dir/user"
done

# Endorser (see: https://github.com/hyperledger-labs/fabric-token-sdk/blob/main/docs/core-token.md?plain=1#L109).
dir="${CONF_ROOT}/endorser1/keys/fabric" 
mkdir -p "$dir"
cp -r "${CRYPTO_DIR}/peerOrganizations/org1.example.com/users/endorser@org1.example.com/msp" "${dir}/endorser"
cp -r "${CRYPTO_DIR}/peerOrganizations/org1.example.com/users/channel_admin@org1.example.com/msp" "${dir}/admin"

dir="${CONF_ROOT}/endorser2/keys/fabric" 
mkdir -p "$dir"
cp -r "${CRYPTO_DIR}/peerOrganizations/org1.example.com/users/endorser@org1.example.com/msp" "${dir}/endorser"
cp -r "${CRYPTO_DIR}/peerOrganizations/org1.example.com/users/channel_admin@org1.example.com/msp" "${dir}/admin"
