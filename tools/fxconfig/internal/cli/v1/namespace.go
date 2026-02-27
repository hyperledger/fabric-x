/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"github.com/spf13/cobra"

	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/app"
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
		newCreateCommand(app.DeployNamespace),
		newListCommand(app.ListNamespaces),
		newUpdateCommand(app.DeployNamespace),
	)

	return cmd
}
