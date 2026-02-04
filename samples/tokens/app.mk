#
# Copyright IBM Corp. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

export PLATFORM
ifeq ($(PLATFORM),fabric3)
	COMPOSE_ARGS := -f compose.yml -f compose-endorser2.yml
endif
CONTAINER_CLI ?= docker

# Setup application
.PHONY: setup-app
setup-app: build-app
	./scripts/gen_crypto.sh

# Setup application
.PHONY: build-app
build-app:
	$(CONTAINER_CLI) compose $(COMPOSE_ARGS) build

# Start application
.PHONY: start-app
start-app:
	$(CONTAINER_CLI) compose $(COMPOSE_ARGS) up -d

# Test the tokens network by performing some transactions
.PHONY: test-app
test-app:
	curl http://localhost:9100/issuer/issue --json '{"amount": {"code": "TOK","value": 1000},"counterparty": {"node": "owner1","account": "alice"},"message": "hello world!"}'
	curl http://localhost:9500/owner/accounts/alice | jq
	curl http://localhost:9600/owner/accounts/dan | jq
	curl http://localhost:9500/owner/accounts/alice/transfer --json '{"amount": {"code": "TOK","value": 100}, "counterparty": {"node": "owner2","account": "dan"}, "message": "hello dan!"}'
	curl -X GET http://localhost:9600/owner/accounts/dan/transactions | jq
	curl -X GET http://localhost:9500/owner/accounts/alice/transactions | jq

# Restart application
.PHONY: restart-app
restart-app: build-app
	$(CONTAINER_CLI) compose $(COMPOSE_ARGS) down
	$(CONTAINER_CLI) compose $(COMPOSE_ARGS) up -d

# Stop application
.PHONY: stop-app
stop-app:
	PLATFORM=$(PLATFORM) $(CONTAINER_CLI) compose $(COMPOSE_ARGS) stop

# Teardown application
.PHONY: teardown-app
teardown-app:
	$(CONTAINER_CLI) compose $(COMPOSE_ARGS) down
	rm -rf "$(CONF_ROOT)"/*/data

# Clean just the databases.
.PHONY: clean-data
clean-data:
	rm -rf "$(CONF_ROOT)"/*/data

# Clean everything and remove all the keys
.PHONY: clean-app
clean-app:
	rm -rf "$(CONF_ROOT)"/*/keys "$(CONF_ROOT)"/*/data "$(CONF_ROOT)"/namespace/*.json
