#
# Copyright IBM Corp. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

export PLATFORM
ifneq ($(PLATFORM),fabricx)
	COMPOSE_ARGS := -f compose.yml -f compose-endorser2.yml
endif

# Setup application
.PHONY: setup-app
setup-app: build-app
	./scripts/gen_crypto.sh

# Setup application
.PHONY: build-app
build-app:
	docker-compose $(COMPOSE_ARGS) build

# Start application
.PHONY: start-app
start-app:
	docker-compose $(COMPOSE_ARGS) up -d

# Restart application
.PHONY: restart-app
restart-app:
	docker-compose $(COMPOSE_ARGS) down
	docker-compose $(COMPOSE_ARGS) up -d

# Stop application
.PHONY: stop-app
stop-app:
	PLATFORM=$(PLATFORM) docker-compose $(COMPOSE_ARGS) stop

# Teardown application
.PHONY: teardown-app
teardown-app:
	docker-compose $(COMPOSE_ARGS) down
	rm -rf "$(CONF_ROOT)"/*/data

# Clean just the databases.
.PHONY: clean-data
clean-data:
	rm -rf "$(CONF_ROOT)"/*/data

# Clean everything and remove all the keys
.PHONY: clean-app
clean-app:
	rm -rf "$(CONF_ROOT)"/*/keys "$(CONF_ROOT)"/*/data "$(CONF_ROOT)"/namespace/*.json
