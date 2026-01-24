#!/usr/bin/env bash

#
# Copyright IBM Corp. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

set -e
CONF_ROOT=$(realpath "${CONF_ROOT:-$(pwd)/conf}")

fabric-ca-client version || (echo "install fabric binaries and make sure they're in your \$PATH"; exit 1)

# Start the fabric CA and enroll the admin user. We keep it running in the background until the script exits.
setup_fabric_ca() {
    # start CA to issue idemix certificates for owners (idemix) and nodes (tls).
    export FABRIC_CA_HOME="${CONF_ROOT}/ca/keys/msp"
    mkdir -p "$FABRIC_CA_HOME"
    fabric-ca-server start -b "admin:adminpw" --idemix.curve gurvy.Bn254 & pid=$!
    trap 'kill "$pid"' EXIT SIGINT SIGTERM SIGHUP SIGQUIT
    while ! fabric-ca-client getcainfo -u localhost:7054 2>/dev/null; do echo "waiting for CA to start..." && sleep 1; done

    # ca admin
    fabric-ca-client enroll -u "http://admin:adminpw@localhost:7054" -M "${CONF_ROOT}/ca/keys/ca-admin"
}

# generate an elliptic curve keypair for the issuer, and the parameters for the dlog verifications.
gen_parameters() {
    # issuer
    dir="${CONF_ROOT}/issuer/keys/wallet/iss"
    mkdir -p "${dir}/signcerts" "${dir}/keystore" 
    openssl ecparam -name prime256v1 -genkey -noout -out "${dir}/keystore/priv_sk"
    openssl req -new -x509 -key "${dir}/keystore/priv_sk" -out "${dir}/signcerts/cert.pem" -days 3650 -subj "/CN=Digital Currency Issuer"

    # encode the ca and issuer identities, and generate the cryptographic parameters needed for the endorsers to validate transactions
    go tool tokengen gen zkatdlognogh.v1 --issuers "${dir}" --idemix "${CONF_ROOT}/ca/keys" --output "${CONF_ROOT}/namespace"
}

# Enroll idemix identities for owners at the CA
enroll_token_users() {
    node=$1; shift
    owners=("$@")
    for owner in "${owners[@]}"; do
        fabric-ca-client register -u http://localhost:7054 --id.name "${owner}" --id.secret password --id.type client --enrollment.type idemix --idemix.curve gurvy.Bn254
        fabric-ca-client enroll -u "http://${owner}:password@localhost:7054"  -M "${CONF_ROOT}/${node}/keys/wallet/${owner}/msp" --enrollment.type idemix --idemix.curve gurvy.Bn254
    done
}

# FSC nodes (identity to talk to each other)
gen_node_crypto() {
    mkdir -p tmp/nodes
    nodes=("$@")
    for node in "${nodes[@]}"; do
        dir="${CONF_ROOT}/${node}/keys"
        mkdir -p "$dir/node"

        # we use the shared CA for mTLS certificates. The certificate chain is currently not verified so we could also use self-signed certificates.
        fabric-ca-client enroll -u "http://admin:adminpw@localhost:7054" -m "${node}.example.com" --enrollment.profile tls -M "${dir}/node"
        cp "${dir}"/node/keystore/* "${dir}/node.key"
        cp "${dir}/node/signcerts/cert.pem" "${dir}/node.crt"

        cp "${dir}/node.crt" "tmp/nodes/${node}.crt"
        rm -r "${dir}/node"
    done

    # copy the public certificates of the other nodes
    for node in "${nodes[@]}"; do
        mkdir -p "${CONF_ROOT}/${node}/data"
        dir="${CONF_ROOT}/${node}/keys"
        cp -r tmp/nodes "$dir/nodes"
    done
    rm -r tmp
}

setup_fabric_ca
enroll_token_users owner1 alice bob
enroll_token_users owner2 carlos dan
gen_node_crypto issuer owner1 owner2 endorser1 endorser2
gen_parameters
