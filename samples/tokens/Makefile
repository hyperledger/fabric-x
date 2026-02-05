#
# Copyright IBM Corp. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
PLATFORM ?= fabricx
CONF_ROOT=conf

ifeq ($(PLATFORM),fabricx)
include ./fabricx_ansible.mk
else ifeq ($(PLATFORM),xdev)
include ./fabricx_dev.mk
else
include ./fabric3.mk
endif

include ./app.mk

# Install the utilities needed to run the components on the targeted remote hosts (e.g. make install-prerequisites).
.PHONY: install-prerequisites
install-prerequisites: install-prerequisites-fabric
	./install-fabric.sh --fabric-version 3.1.1
	go mod tidy

# Build all the artifacts and binaries, and copy them to the application folders
.PHONY: setup
setup: clean setup-fabric setup-app

# Start a Fabric and token network.
.PHONY: start
start: start-fabric start-app

# Teardown Fabric and the token network.
.PHONY: teardown
teardown: teardown-app teardown-fabric

# Test Fabric and the token network.
.PHONY: test
test:
	@./scripts/test.sh

# Stop Fabric and the token network.
.PHONY: stop
stop: stop-app stop-fabric

# Remove all generated crypto. Requires a `make setup` to restore.
.PHONY: clean
clean: clean-app clean-fabric

# Print the list of supported commands.
.PHONY: help
help:
	@awk ' \
		/^#/ { \
			sub(/^#[ \t]*/, "", $$0); \
			help_msg = $$0; \
		} \
		/^[a-zA-Z0-9][^ :]*:/ { \
			if (help_msg) { \
				split($$1, target, ":"); \
				printf "  %-40s %s\n", target[1], help_msg; \
				help_msg = ""; \
			} \
		} \
	' $(MAKEFILE_LIST)
