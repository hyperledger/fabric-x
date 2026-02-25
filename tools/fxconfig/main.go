// Copyright IBM Corp. All Rights Reserved.
//
// SPDX-License-Identifier: Apache-2.0

// Package main provides the fxconfig CLI tool for managing Fabric-X namespaces.
// It supports creating, updating, and listing namespaces with flexible policy configurations.
package main

import (
	"os"

	"github.com/hyperledger/fabric-x/tools/fxconfig/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
