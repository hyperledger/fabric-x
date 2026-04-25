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
    curl_with_retry -X POST http://localhost:9300/endorser/init
}

## Wait for an API endpoint to report ready
function wait_until_ready() {
    local service_name="$1"
    local url="$2"
    local max_attempts="${MAX_READY_ATTEMPTS:-30}"
    local sleep_seconds="${READY_RETRY_SLEEP_SECONDS:-2}"

    echo "Waiting for ${service_name} readiness: ${url}" >&2
    curl -fsS \
        --retry "$max_attempts" \
        --retry-delay "$sleep_seconds" \
        --retry-all-errors \
        --retry-connrefused \
        "$url" >/dev/null
}

## Wait for all services needed by the test run
function wait_for_services() {
    print_section_header "Waiting for services to become ready..."

    if [[ "$PLATFORM" == "fabricx" || "$PLATFORM" == "xdev" ]]; then
        wait_until_ready "endorser" "http://localhost:9300/readyz"
    fi

    wait_until_ready "issuer" "http://localhost:9100/readyz"
    wait_until_ready "owner1" "http://localhost:9500/readyz"
    wait_until_ready "owner2" "http://localhost:9600/readyz"
}

## Run curl with retries for transient startup errors
function curl_with_retry() {
    curl -fS \
        --retry "${CURL_MAX_ATTEMPTS:-6}" \
        --retry-delay "${CURL_RETRY_SLEEP_SECONDS:-2}" \
        --retry-all-errors \
        --retry-connrefused \
        "$@"
}

## Run tests to verify the network
function run_test() {
    # test application
    print_section_header "Run tests"

    curl_with_retry -X POST http://localhost:9100/issuer/issue -d '{
        "amount": {"code": "TOK","value": 1000},
        "counterparty": {"node": "owner1","account": "alice"},
        "message": "hello world!"
    }'
    curl_with_retry -X GET http://localhost:9500/owner/accounts/alice | jq
    curl_with_retry -X GET http://localhost:9600/owner/accounts/dan | jq
    curl_with_retry -X POST http://localhost:9500/owner/accounts/alice/transfer -d '{
        "amount": {"code": "TOK","value": 100},
        "counterparty": {"node": "owner2","account": "dan"},
        "message": "hello dan!"
    }'
    curl_with_retry -X GET http://localhost:9600/owner/accounts/dan/transactions | jq
    curl_with_retry -X GET http://localhost:9500/owner/accounts/alice/transactions | jq
}

# Script Start
set -eE
set -o pipefail
trap cleanup INT ERR
PLATFORM="${PLATFORM:-fabric3}"
export PLATFORM

run_network
# # currently we wait manually with a sleep.
# # TODO: add an healthcheck within the `docker-compose`
sleep 10
wait_for_services
if [[ "$PLATFORM" == "fabricx" || "$PLATFORM" == "xdev" ]]; then
    init_fabricx
fi
run_test
stop_network