// Copyright IBM Corp. All Rights Reserved.
//
// SPDX-License-Identifier: Apache-2.0

// Package main provides the fxconfig CLI tool for managing Fabric-X namespaces.
// It supports creating, updating, and listing namespaces with flexible policy configurations.
package main

import (
	"os"

	v1 "github.com/hyperledger/fabric-x/tools/fxconfig/internal/cli/v1"
)

func main() {
	if err := v1.Execute(); err != nil {
		os.Exit(1)
	}
}
