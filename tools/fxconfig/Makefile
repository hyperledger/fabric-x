# Copyright IBM Corp All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0

go_cmd ?= go
go_test ?= $(go_cmd) test -race -cover -v -timeout 30m

.PHONY: test
test:
	$(go_test) ./...
