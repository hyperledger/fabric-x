#!/usr/bin/env bash

#
# Copyright IBM Corp. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

# init FabricX
if [[ "$PLATFORM" == "fabricx" ]]; then
    curl -X POST http://localhost:9300/endorser/init
fi

# test application
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