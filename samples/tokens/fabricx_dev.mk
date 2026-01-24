#
# Copyright IBM Corp. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

# exported vars
CONF_ROOT=conf
export CONF_ROOT

# Install the utilities needed to run the components on the targeted remote hosts (e.g. make install-prerequisites).
.PHONY: install-prerequisites-fabric
install-prerequisites-fabric:

# Build all the artifacts, the binaries and transfer them to the remote hosts (e.g. make setup).
.PHONY: setup-fabric
setup-fabric:
	@go tool cryptogen generate --config crypto-config.yaml --output crypto
	@go tool configtxgen --channelID mychannel --profile OrgsChannel --outputBlock crypto/sc-genesis-block.proto.bin
	@CRYPTO_DIR=crypto ./scripts/cp_fabricx.sh

# Clean all the artifacts (configs and bins) built on the controller node (e.g. make clean).
.PHONY: clean-fabric
clean-fabric:
	@rm -rf ./crypto
	@for d in "$(CONF_ROOT)"/*/ ; do \
		rm -rf "$$d/keys/fabric" "$$d/data"; \
	done

# Start fabric-x
.PHONY: start-fabric
start-fabric:
	@$(CONTAINER_CLI) network inspect fabric_test >/dev/null 2>&1 || $(CONTAINER_CLI) network create fabric_test
	@$(CONTAINER_CLI) compose -f compose-xdev.yml up -d --wait && sleep 2
	@echo "install namespace:"
	@go tool fxconfig namespace create token_namespace --channel=mychannel --orderer=localhost:7050 --mspID=Org1MSP \
		--mspConfigPath=crypto/peerOrganizations/org1.example.com/users/channel_admin@org1.example.com/msp \
		--pk=crypto/peerOrganizations/org1.example.com/users/endorser@org1.example.com/msp/signcerts/endorser@org1.example.com-cert.pem
	@until go tool fxconfig namespace list --endpoint=localhost:7001 2>/dev/null | grep -q token_namespace; do sleep 1; echo "waiting for namespace to be created..."; done
	@go tool fxconfig namespace list --endpoint=localhost:7001

# Stop the docker container.
.PHONY: stop-fabric
stop-fabric:
	@$(CONTAINER_CLI) compose -f compose-xdev.yml down

# Teardown fabric-x
.PHONY: teardown-fabric
teardown-fabric: stop-fabric
	@$(CONTAINER_CLI) network inspect fabric_test >/dev/null 2>&1 && $(CONTAINER_CLI) network rm fabric_test


# Restart fabric. This deletes the ledger; the app must be cleaned as well.
.PHONY: restart-fabric
restart-fabric: teardown-fabric start-fabric
