#
# Copyright IBM Corp. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

# exported vars
ANSIBLE_CONFIG ?= ./ansible/ansible.cfg
export ANSIBLE_CONFIG
PROJECT_DIR := $(CURDIR)
export PROJECT_DIR

CONF_ROOT=conf
export CONF_ROOT

# Makefile vars
PLAYBOOK_PATH := $(CURDIR)/ansible/playbooks
TARGET_HOSTS ?= all

# Install the utilities needed to run the components on the targeted remote hosts (e.g. make install-prerequisites).
.PHONY: install-prerequisites-fabric
install-prerequisites-fabric:
	ansible-playbook hyperledger.fabricx.install_prerequisites

# Build all the artifacts, the binaries and transfer them to the remote hosts (e.g. make setup).
.PHONY: setup-fabric
setup-fabric:
	ansible-playbook "$(PLAYBOOK_PATH)/20-build.yaml"
	./scripts/cp_fabricx.sh

# Clean all the artifacts (configs and bins) built on the controller node (e.g. make clean).
.PHONY: clean-fabric
clean-fabric:
	rm -rf ./out
	@for d in "$(CONF_ROOT)"/*/ ; do \
		rm -rf "$$d/keys/fabric" "$$d/data"; \
	done

# Start fabric-x on the targeted hosts.
.PHONY: start-fabric
start-fabric:
	@docker network inspect fabric_test >/dev/null 2>&1 || docker network create fabric_test
	ansible-playbook "$(PLAYBOOK_PATH)/60-start.yaml"

# Create a namespace in fabric-x for the tokens.
.PHONY: create-namespace
create-namespace:
	@echo "install namespace:"
	go tool fxconfig namespace create token_namespace --channel=arma --orderer=localhost:7050 --mspID=Org1MSP  --mspConfigPath=conf/endorser1/keys/fabric/admin --pk=conf/endorser1/keys/fabric/endorser/signcerts/endorser@org1.example.com-cert.pem 2> /dev/null
	@until go tool fxconfig namespace list --endpoint=localhost:5500 | grep -q token_namespace; do \
		sleep 2; \
		echo "waiting for namespace to be created..."; \
	done
	go tool fxconfig namespace list --endpoint=localhost:5500

# Stop the targeted hosts (e.g. make fabric-x stop).
.PHONY: stop-fabric
stop-fabric:
	ansible-playbook "$(PLAYBOOK_PATH)/70-stop.yaml"

# Teardown the targeted hosts (e.g. make fabric-x teardown).
.PHONY: teardown-fabric
teardown-fabric:
	ansible-playbook "$(PLAYBOOK_PATH)/80-teardown.yaml"
	@docker network inspect fabric_test >/dev/null 2>&1 && docker network rm fabric_test || true

# Restart the targeted hosts (e.g. make fabric-x restart).
.PHONY: restart-fabric
restart-fabric: teardown-fabric start-fabric
