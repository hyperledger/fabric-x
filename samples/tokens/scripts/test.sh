#!/usr/bin/env bash

#
# Copyright IBM Corp. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

# Function to print a section title
function print_section_header() {
    echo "# ========================="
    echo "$1"
    echo "# ========================="
}

# Function to handle cleanup when the script is aborted
function cleanup() {
    # stop any spinner message
    print_section_header "Stopping tests..."
    # remove lock if we own it
    make teardown clean
    exit 1
}

# init FabricX
function init_fabricx() {
    if [[ "$PLATFORM" == "fabricx" ]]; then
        print_section_header "Initializing ${PLATFORM}..."
        curl -X POST http://localhost:9300/endorser/init
    fi
}

# Function to run tests to verify whether the network works as expected
function run_test() {
    # test application
    print_section_header "Run tests"

    # currently we wait manually with a sleep. TODO: add an healthcheck within the `docker-compose`
    sleep 10
    curl http://localhost:9100/issuer/issue -d '{
        "amount": {"code": "TOK","value": 1000},
        "counterparty": {"node": "owner1","account": "alice"},
        "message": "hello world!"
    }'
    curl http://localhost:9500/owner/accounts/alice | jq
    curl http://localhost:9600/owner/accounts/dan | jq
    curl http://localhost:9500/owner/accounts/alice/transfer -d '{
        "amount": {"code": "TOK","value": 100},
        "counterparty": {"node": "owner2","account": "dan"},
        "message": "hello dan!"
    }'
    curl -X GET http://localhost:9600/owner/accounts/dan/transactions | jq
    curl -X GET http://localhost:9500/owner/accounts/alice/transactions | jq
}

# prepare script
# trap signals to ensure cleanup and lock release
# note: INT/TERM go to cleanup (which exits).
trap cleanup INT TERM

print_section_header "Setup and start the network..."
make setup start

run_test
make teardown clean