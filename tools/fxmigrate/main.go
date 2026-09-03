// Copyright IBM Corp. All Rights Reserved.
//
// SPDX-License-Identifier: Apache-2.0

// Package main provides the fxmigrate CLI tool for migrating Hyperledger Fabric
// ledger state to Fabric-X.
package main

import (
	"context"
	"os"
	"os/signal"

	cli "github.com/hyperledger/fabric-x/tools/fxmigrate/internal/cli/v1"
)

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	return cli.NewRootCommand().ExecuteContext(ctx)
}
