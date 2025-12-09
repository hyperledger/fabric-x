#
# Copyright IBM Corp. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

# exported vars
FABRIC_SAMPLES := $(abspath fabric-samples)
export FABRIC_SAMPLES

CONF_ROOT=conf-f3
export CONF_ROOT

# Makefile vars
PLAYBOOK_PATH := $(CURDIR)/ansible/playbooks
TARGET_HOSTS ?= all
CONTAINER_CLI ?= docker

# Install the utilities needed to run the components on the targeted remote hosts (e.g. make install-prerequisites).
.PHONY: install-prerequisites-fabric
install-prerequisites-fabric:
	@echo "(nothing to do for fabric 3)"

# Build all the artifacts, the binaries and transfer them to the remote hosts (e.g. make setup).
.PHONY: setup-fabric
setup-fabric:
	@echo "(nothing to do for fabric 3)"

# Build the config artifacts on the controller node (e.g. make build-configs).
.PHONY: build-fabric
build-fabric:
	ansible-playbook "$(PLAYBOOK_PATH)/20-build.yaml"

# Clean all the artifacts (configs and bins) built on the controller node (e.g. make clean).
.PHONY: clean-fabric
clean-fabric:
	@echo "(nothing to do for fabric 3)"

# Start the targeted hosts (e.g. make fabric-fabric start).
.PHONY: start-fabric
start-fabric:
	"$(FABRIC_SAMPLES)/test-network/network.sh" up createChannel -i 3.1.1
	./scripts/cp_fabric3.sh

# Create a namespace in Fabric for the tokens. We install a chaincode but don't need it because endorsements are created by the Fabric Smart Client nodes.
.PHONY: create-namespace
create-namespace:
	INIT_REQUIRED="--init-required" "$(FABRIC_SAMPLES)/test-network/network.sh" deployCCAAS  -ccn token_namespace -ccp "$(abspath $$CONF_ROOT)/namespace" -cci "init"

# Stop the targeted hosts (e.g. make fabric-x stop).
.PHONY: stop-fabric
stop-fabric:
  @echo "use: make teardown-fabric"

# Teardown the targeted hosts (e.g. make fabric-x teardown).
.PHONY: teardown-fabric
teardown-fabric:
	@"$(FABRIC_SAMPLES)/test-network/network.sh" down
	@$(CONTAINER_CLI) rm -f peer0org1_token_namespace_ccaas peer0org2_token_namespace_ccaas
	@for d in "$(CONF_ROOT)"/*/ ; do \
		rm -rf "$$d/keys/fabric" "$$d/data"; \
	done
	@$(CONTAINER_CLI) network inspect fabric_test >/dev/null 2>&1 && $(CONTAINER_CLI) network rm fabric_test || true

# Restart the targeted hosts (e.g. make fabric-x restart).
.PHONY: restart-fabric
restart-fabric: teardown-fabric start-fabric
