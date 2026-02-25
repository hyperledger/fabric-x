/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cmd

import (
	"github.com/spf13/cobra"

	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/namespace"
)

// NewNamespaceCommand returns the namespace command group.
// This command provides subcommands for namespace lifecycle operations:
// create, update, and list.
func NewNamespaceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "namespace",
		Short: "Perform namespace operations",
		Long:  "",
	}

	cmd.AddCommand(
		newCreateCommand(namespace.DeployNamespace),
		newListCommand(namespace.ListNamespaces),
		newUpdateCommand(namespace.DeployNamespace),
	)

	return cmd
}
