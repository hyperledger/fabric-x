/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"fmt"

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

type dryRunPreview struct {
	operation string
	namespace string
	txID      string
}

func printDryRun(cmd *cobra.Command, ctx *CLIContext, preview dryRunPreview) {
	message := fmt.Sprintf(
		"=== DRY RUN ===\n\nNamespace operation prepared successfully.\n\n"+
			"Namespace: %s\nOperation: %s\nTxID: %s\n\nTransaction was NOT submitted.\n",
		preview.namespace,
		preview.operation,
		preview.txID,
	)
	if ctx.Printer != nil {
		ctx.Printer.Print(message)
		return
	}

	_, _ = fmt.Fprint(cmd.OutOrStdout(), message)
}
