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
		Long: `Perform transaction operations such as endorsement, merging, and submission.

Transaction Lifecycle:
  1. Create - Generate transaction (e.g., namespace create/update)
  2. Endorse - Collect signatures from required organizations
  3. Merge - Combine endorsements from multiple organizations
  4. Submit - Send to ordering service for finalization

Multi-Organization Workflow:
  1. Org1 creates transaction: fxconfig namespace create ... --output tx.json
  2. Org1 endorses: fxconfig tx endorse tx.json --output tx_org1.json
  3. Org2 endorses: fxconfig tx endorse tx.json --output tx_org2.json
  4. Merge endorsements: fxconfig tx merge tx_org1.json tx_org2.json --output merged.json
  5. Submit: fxconfig tx submit merged.json --wait`,
	}

	cmd.AddCommand(
		newTxMergeCommand(ctx),
		newTxEndorseCommand(ctx),
		newTxSubmitCommand(ctx),
	)

	return cmd
}
