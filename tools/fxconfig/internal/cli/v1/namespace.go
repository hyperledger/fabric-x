/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"github.com/spf13/cobra"
)

// NewNsRootCommand returns the namespace command group.
// This command provides subcommands for namespace lifecycle operations:
// create, update, and list.
func NewNsRootCommand(ctx *CLIContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "namespace",
		Short: "Manage Fabric-X namespaces",
		Long: `Manage namespace lifecycle operations.

Namespaces in Fabric-X define isolated execution environments with their own
endorsement policies. Each namespace has:
  • Unique identifier (name)
  • Version number (incremented on updates)
  • Endorsement policy (defines which organizations must sign transactions)`,
	}

	cmd.AddCommand(
		newNsCreateCommand(ctx),
		newNsUpdateCommand(ctx),
		newNsListCommand(ctx),
	)

	return cmd
}
