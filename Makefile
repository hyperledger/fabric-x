# Copyright IBM Corp All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#
# -------------------------------------------------------------
# Run `make help` to find the supported targets

# Disable implicit rules
.SUFFIXES:
MAKEFLAGS += --no-builtin-rules

PKGNAME = github.com/hyperledger/fabric-x-common
PKGNAME2 = github.com/hyperledger/fabric-x/tools

GO_TAGS ?=

go_cmd          ?= go

TOOLS_EXES = configtxgen configtxlator cryptogen fxconfig fxadmin

pkgmap.configtxgen    := $(PKGNAME2)/configtxgen
pkgmap.configtxlator  := $(PKGNAME2)/configtxlator
pkgmap.cryptogen      := $(PKGNAME2)/cryptogen
pkgmap.fxconfig		  := $(PKGNAME2)/fxconfig
pkgmap.fxadmin		  := $(PKGNAME2)/fxadmin

.DEFAULT_GOAL := help

MAKEFLAGS += --jobs=16

# Use gotestsum (same style as fabric-x-committer):
# - compact output format
# - does not rerun failed tests
TEST_METHOD = $(go_cmd) tool gotestsum --rerun-fails=0 --format dots --packages "$(1)" -- -v -timeout 30m $(2)

BUILD_DIR ?= bin
RELEASE_DIR ?= release

# Resolved on first use and then cached, so `go env` is never invoked for
# targets that don't need it (help, lint, test) nor when GOOS/GOARCH are preset.
GOOS   ?= $(eval GOOS := $(shell go env GOOS))$(GOOS)
GOARCH ?= $(eval GOARCH := $(shell go env GOARCH))$(GOARCH)
RELEASE_BIN_DIR = $(RELEASE_DIR)/$(GOOS)-$(GOARCH)/bin


## List all commands with documentation
.PHONY: help
help:
	@echo "Available commands:"
	@awk '/^## / {doc = substr($$0, 4)} /^[a-zA-Z_-]+:/ && doc {split($$1, t, ":"); printf "\033[36m%-15s\033[0m %s\n", t[1], doc; doc = ""}' $(MAKEFILE_LIST)

## Builds all tools
.PHONY: tools
tools: $(TOOLS_EXES)

## Run generate
.PHONY: generate
generate: FORCE
	go generate ./...

## Run all tests
.PHONY: test
test: FORCE
	@echo "Running Go unit tests..."
	cd tools && $(call TEST_METHOD,./...)

## Builds a native binary
.PHONY: $(TOOLS_EXES)
$(TOOLS_EXES): %: $(BUILD_DIR)/%

$(BUILD_DIR)/%: GO_LDFLAGS = $(METADATA_VAR:%=-X $(PKGNAME)/common/metadata.%)
$(BUILD_DIR)/%: FORCE
	@echo "Building $@"
	@mkdir -p $(@D)
	@GOBIN=$(abspath $(@D)) go install -tags "$(GO_TAGS)" -ldflags "$(GO_LDFLAGS)" -buildvcs=false $(pkgmap.$(@F))
	@touch $@

## Cross-compiles all tools for $(GOOS)/$(GOARCH) into $(RELEASE_DIR)
.PHONY: release-bins
release-bins:
	@$(MAKE) --no-print-directory GOOS=$(GOOS) GOARCH=$(GOARCH) $(TOOLS_EXES:%=$(RELEASE_BIN_DIR)/%)

$(RELEASE_DIR)/%: GO_LDFLAGS = $(METADATA_VAR:%=-X $(PKGNAME)/common/metadata.%)
$(RELEASE_DIR)/%: FORCE
	@echo "Building $@"
	@mkdir -p $(@D)
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -trimpath \
		-tags "$(GO_TAGS)" -ldflags "$(GO_LDFLAGS)" -buildvcs=false \
		-o $@ $(pkgmap.$(@F))

## Cleans the build area
.PHONY: clean
clean:
	-@rm -rf $(BUILD_DIR) $(RELEASE_DIR)

## Run lint
# TODO: fix existing lint issues (to find them, remove --new-from-rev=origin/main option)
.PHONY: lint
lint: FORCE
	@echo "Running Go Linters..."
	cd tools && golangci-lint run --new-from-rev=origin/main --color=always --max-same-issues 0
	@echo "Running License Header Linters..."
	scripts/license-lint.sh
FORCE:
