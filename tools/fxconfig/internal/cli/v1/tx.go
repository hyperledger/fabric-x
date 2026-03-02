/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"github.com/spf13/cobra"
)

// NewTxRootCommand returns the namespace command group.
// This command provides subcommands for transaction operations:
// endorse, merge, and submit.
func NewTxRootCommand(ctx *CLIContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tx",
		Short: "Perform transaction operations",
		Long:  "",
	}

	cmd.AddCommand(
		newTxMergeCommand(ctx),
		newTxEndorseCommand(ctx),
		newTxSubmitCommand(ctx),
	)

	return cmd
}
