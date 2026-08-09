// Copyright IBM Corp. All Rights Reserved.
//
// SPDX-License-Identifier: Apache-2.0

// Package main is the entry point for the fxadmin CLI. It wires the
// command tree from the fxadmin core cli package and delegates execution to
// it; all command logic lives in fabric-x-common/tools/fxadmin.
package main

import (
	"fmt"
	"os"

	"github.com/hyperledger/fabric-x-common/common/metadata"
	"github.com/hyperledger/fabric-x-common/tools/fxadmin/core/cli"
	"github.com/hyperledger/fabric-x-common/tools/fxadmin/core/decode"
	"github.com/hyperledger/fabric-x-common/tools/fxadmin/core/follow"
	"github.com/hyperledger/fabric-x-common/tools/fxadmin/core/ledger"
	"github.com/hyperledger/fabric-x-common/tools/fxadmin/core/tx"
	"github.com/hyperledger/fabric-x-common/tools/fxadmin/core/update"
)

func main() {
	handlers := cli.Handlers{
		Ledger: ledger.New(),
		Decode: decode.New(),
		Update: update.New(),
		Tx:     tx.New(),
		Follow: follow.New(),
	}

	app := cli.New(handlers, metadata.Version)

	if err := app.Run(os.Args[1:]); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "fxadmin: %v\n", err)
		os.Exit(1)
	}
}
