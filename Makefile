# Copyright IBM Corp All Rights Reserved.
# Copyright London Stock Exchange Group All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
# -------------------------------------------------------------
# Run `make help` to find the supported targets

# Disable implicit rules
.SUFFIXES:
MAKEFLAGS += --no-builtin-rules

BUILD_DIR ?= bin

PKGNAME = github.com/hyperledger/fabric-x-common
PKGNAME2 = github.com/hyperledger/fabric-x/tools

GO_TAGS ?=

go_cmd          ?= go
go_test         ?= $(go_cmd) test -json -v -timeout 30m

TOOLS_EXES = configtxgen configtxlator cryptogen fxconfig

pkgmap.configtxgen    := $(PKGNAME2)/configtxgen
pkgmap.configtxlator  := $(PKGNAME2)/configtxlator
pkgmap.cryptogen      := $(PKGNAME2)/cryptogen
pkgmap.fxconfig		  := $(PKGNAME2)/fxconfig

.DEFAULT_GOAL := help

MAKEFLAGS += --jobs=16

.PHONY: help
# List all commands with documentation
help: ## List all commands with documentation
	@echo "Available commands:"
	@awk 'BEGIN {FS = ":.*?## "}; /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: tools
tools: $(TOOLS_EXES) ## Builds all tools

GO_TEST_FMT_FLAGS := -hide empty-packages

## Run all tests
.PHONY: test
test: FORCE
	@echo "Running Go unit tests..."
	cd tools && $(go_test) ./... | go tool gotestfmt ${GO_TEST_FMT_FLAGS}

.PHONY: $(TOOLS_EXES)
$(TOOLS_EXES): %: $(BUILD_DIR)/% ## Builds a native binary

$(BUILD_DIR)/%: GO_LDFLAGS = $(METADATA_VAR:%=-X $(PKGNAME)/common/metadata.%)
$(BUILD_DIR)/%:
	@echo "Building $@"
	@mkdir -p $(@D)
	@GOBIN=$(abspath $(@D)) go install -tags "$(GO_TAGS)" -ldflags "$(GO_LDFLAGS)" -buildvcs=false $(pkgmap.$(@F))
	@touch $@

.PHONY: clean
clean: ## Cleans the build area
	-@rm -rf $(BUILD_DIR)

# Run lint
# TODO: fix existing lint issues (to find them, remove --new-from-rev=origin/main option)
.PHONY: lint
lint: FORCE
	@echo "Running Go Linters..."
	cd tools && golangci-lint run --new-from-rev=origin/main --color=always --max-same-issues 0
	@echo "Running License Header Linters..."
	scripts/license-lint.sh

FORCE: