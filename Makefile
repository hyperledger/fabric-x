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

TOOLS_EXES = configtxgen configtxlator cryptogen

pkgmap.configtxgen    := $(PKGNAME2)/configtxgen
pkgmap.configtxlator  := $(PKGNAME2)/configtxlator
pkgmap.cryptogen      := $(PKGNAME2)/cryptogen

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
	cd tools && $(go_test) ./... | gotestfmt ${GO_TEST_FMT_FLAGS}

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

.PHONY: lint
lint: FORCE
	@echo "Running Go Linters..."
	cd tools && golangci-lint run --color=always --new-from-rev=main --timeout=4m ./...
	@echo "Running License Header Linters..."
	scripts/license-lint.sh

# ---------------------------------------------------------------------
# Container image build settings
# ---------------------------------------------------------------------
IMAGE_NAME ?= docker.io/hyperledger/fabric-x-tools
IMAGE_TAG  ?= latest
DOCKERFILE ?= ./tools/images/Dockerfile
PLATFORMS  ?= linux/amd64,linux/arm64,linux/s390x
PROJECT_DIR := $(shell dirname $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST)))))

# Detect available container runtime (docker preferred, fallback podman)
CONTAINER_RUNTIME ?= $(shell command -v docker 2>/dev/null || command -v podman 2>/dev/null)

# Build the fabric-x-tools image for the current machine platform.
.PHONY: build-fabric-x-tools-image
build-fabric-x-tools-image: ## Build the fabric-x-tools image for the current machine platform
	$(CONTAINER_RUNTIME) build -f $(DOCKERFILE) \
		-t $(IMAGE_NAME):$(IMAGE_TAG) \
		--load \
		$(PROJECT_DIR)

# Build the fabric-x-tools image for multiple platforms.
.PHONY: build-fabric-x-tools-multiplatform-image
build-fabric-x-tools-multiplatform-image: ## Build the fabric-x-tools image for multiple platforms
ifeq ($(CONTAINER_RUNTIME),docker)
	$(CONTAINER_RUNTIME) buildx build \
		-f $(DOCKERFILE) \
		--platform $(PLATFORMS) \
		-t $(IMAGE_NAME):$(IMAGE_TAG) \
		--push \
		$(PROJECT_DIR)
else ifeq ($(CONTAINER_RUNTIME),podman)
	$(CONTAINER_RUNTIME) manifest create $(IMAGE_NAME):$(IMAGE_TAG) || true
	$(CONTAINER_RUNTIME) build \
		-f $(DOCKERFILE) \
		--platform $(PLATFORMS) \
		--manifest $(IMAGE_NAME):$(IMAGE_TAG) \
		$(PROJECT_DIR)
	$(CONTAINER_RUNTIME) manifest push $(IMAGE_NAME):$(IMAGE_TAG)
else
	@echo "Error: Neither Docker nor Podman is installed."
	@exit 1
endif

FORCE: