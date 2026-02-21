/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cmd

import (
	"github.com/spf13/cobra"

	"github.com/hyperledger/fabric-x/tools/fxconfig/cmd/namespace"
)

// Execute is the entry point of fxconfig and collects all commands.
func Execute() error {
	return RootCmd().Execute()
}

// RootCmd returns our root command.
func RootCmd() *cobra.Command {
	rootCmd := &cobra.Command{Use: "fxconfig"}
	rootCmd.AddCommand(NewVersionCommand())
	rootCmd.AddCommand(namespace.NewNamespaceCommand())
	return rootCmd
}
