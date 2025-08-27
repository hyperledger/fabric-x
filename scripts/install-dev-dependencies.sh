#!/bin/bash

#
# Copyright IBM Corp. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

set -e

echo "Installing goimports"
go install "golang.org/x/tools/cmd/goimports@v0.33.0"

echo "Installing golangci-lint"
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest

echo "Installing gotestfmt"
go install "github.com/gotesttools/gotestfmt/v2/cmd/gotestfmt@v2.5.0"
