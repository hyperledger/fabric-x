/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/hyperledger/fabric-x-common/api/applicationpb"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/cli/v1/io"
)

// newTxMergeCommand creates a command to merge multiple endorsed transactions
// with the same transaction ID into a single transaction.
func newTxMergeCommand(ctx *CLIContext) *cobra.Command {
	var outputFlag OutputFlag

	cmd := &cobra.Command{
		Use:   "merge [tx1.json] [tx2.json] [txN.json...]",
		Short: "Merge multiple endorsed transactions",
		Long: `Combine endorsements from multiple organizations into a single transaction.

All input transactions must:
  • Have the same transaction ID
  • Contain the same transaction data
  • Have endorsements from different organizations

The merged transaction will contain all endorsement signatures, making it
ready for submission if the endorsement policy is satisfied.

This command is essential for multi-organization workflows where each
organization endorses independently and endorsements must be collected
before submission.

Examples:
  # Merge two endorsed transactions
  fxconfig tx merge tx_org1.json tx_org2.json --output merged_tx.json

  # Merge three organizations' endorsements
  fxconfig tx merge tx_org1.json tx_org2.json tx_org3.json --output merged_tx.json

  # Merge and output to stdout
  fxconfig tx merge tx_org1.json tx_org2.json > merged_tx.json`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			txs := make([]*applicationpb.Tx, 0, len(args))
			txIDs := make(map[string]struct{})
			var txID string

			for _, arg := range args {
				input, err := io.ResolveInput(cmd, arg)
				if err != nil {
					return err
				}

				id, tx, err := ctx.IOTransactionCodec.Decode(input)
				if err != nil {
					return err
				}

				txIDs[txID] = struct{}{}
				if txID == "" {
					txID = id
				}

				txs = append(txs, tx)
			}

			if len(txIDs) != 1 {
				return fmt.Errorf("all transaction must have the same txID, found %d different txIDs", len(txIDs))
			}

			mergedTx, err := ctx.App.MergeTransactions(cmd.Context(), txs)
			if err != nil {
				return err
			}

			o, err := ctx.IOTransactionCodec.Encode(txID, mergedTx)
			if err != nil {
				return err
			}

			return io.WriteOutput(cmd, string(outputFlag), o)
		},
	}
	outputFlag.Bind(cmd)

	return cmd
}
