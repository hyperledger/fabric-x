#!/usr/bin/env bash

#
# Copyright IBM Corp. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

CONTAINER_CLI="${CONTAINER_CLI:-docker}"

## Print a section title
function print_section_header() {
    echo "# ========================="
    echo "# $1"
    echo "# ========================="
}

## Cleanup and stop network on abort
function cleanup() {
    local exit_code=$?
    local failed_cmd="$BASH_COMMAND"
    trap - INT ERR
    set +e
    stop_network
    echo "Error: command '$failed_cmd' exited with status $exit_code" >&2
    exit 1
}

## Setup and start the network
function run_network() {
    print_section_header "Setup and start the network..."
    make setup start
}

## Stop and clean up the network
function stop_network() {
    print_section_header "Stopping network..."
    make teardown clean
}

## Initialize FabricX if needed
function init_fabricx() {
    print_section_header "Initializing ${PLATFORM}..."
    curl -f -X POST http://localhost:9300/endorser/init
}

## Run tests to verify the network
function run_test() {
    # test application
    print_section_header "Run tests"

    curl -f -X POST http://localhost:9100/issuer/issue -d '{
        "amount": {"code": "TOK","value": 1000},
        "counterparty": {"node": "owner1","account": "alice"},
        "message": "hello world!"
    }'
    curl -f -X GET http://localhost:9500/owner/accounts/alice | jq
    curl -f -X GET http://localhost:9600/owner/accounts/dan | jq
    curl -f -X POST http://localhost:9500/owner/accounts/alice/transfer -d '{
        "amount": {"code": "TOK","value": 100},
        "counterparty": {"node": "owner2","account": "dan"},
        "message": "hello dan!"
    }'
    curl -f -X GET http://localhost:9600/owner/accounts/dan/transactions | jq
    curl -f -X GET http://localhost:9500/owner/accounts/alice/transactions | jq
}

# Script Start
set -eE
set -o pipefail
trap cleanup INT ERR

run_network
# # currently we wait manually with a sleep.
# # TODO: add an healthcheck within the `docker-compose`
sleep 10
if [[ "$PLATFORM" == "fabricx" || "$PLATFORM" == "xdev" ]]; then
    init_fabricx
fi
run_test
stop_network