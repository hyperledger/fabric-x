/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package v1 implements the command-line interface for fxmigrate.
package v1

import (
	"github.com/spf13/cobra"
)

// NewRootCommand constructs and returns the root cobra command for fxmigrate.
func NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "fxmigrate",
		Short: "CLI tool for migrating Hyperledger Fabric ledger state to Fabric-X",
		Long: `fxmigrate is a command-line tool for migrating world state from a
Hyperledger Fabric network to Fabric-X.

It reads a standard Fabric peer snapshot directory, applies namespace
mapping, strips private-data-collection artefacts, and produces a
verifiable genesis-data file that the Fabric-X committer can ingest
via its --init-from-snapshot bootstrap mode.

Commands:
  export   Export a Fabric peer snapshot into a Fabric-X genesis-data file
  verify   Verify integrity between a Fabric snapshot and a live Fabric-X state DB
  version  Print the fxmigrate version`,
	}

	rootCmd.SilenceUsage = true

	rootCmd.AddCommand(NewVersionCommand())
	rootCmd.AddCommand(NewExportCommand())
	rootCmd.AddCommand(NewVerifyCommand())

	return rootCmd
}
