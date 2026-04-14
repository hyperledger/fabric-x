/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/hyperledger/fabric-x-common/api/committerpb"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/cli/v1/cliio"
)

// newTxSubmitCommand creates a command for submitting transactions.
func newTxSubmitCommand(ctx *CLIContext) *cobra.Command {
	var wait waitFlag

	cmd := &cobra.Command{
		Use:   "submit [file]",
		Short: "Submit transaction to ordering service",
		Long: `Submit an endorsed transaction to the Fabric-X ordering service.

The transaction must have sufficient endorsements to satisfy its endorsement
policy. Use 'fxconfig tx merge' to combine endorsements from multiple
organizations before submission.

Status Codes (with --wait):
  0 - Transaction successfully committed
  1 - Transaction failed (see error message for details)

Examples:
  # Submit transaction (returns immediately)
  fxconfig tx submit merged_tx.json

  # Submit and wait for finalization
  fxconfig tx submit merged_tx.json --wait

  # Submit with custom config
  fxconfig tx submit merged_tx.json --config /path/to/config.yaml --wait

  # Submit and capture status
  if fxconfig tx submit merged_tx.json --wait; then
    echo "Transaction committed successfully"
  else
    echo "Transaction failed"
  fi`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			input, err := cliio.ResolveInput(cmd, args[0])
			if err != nil {
				return err
			}

			txID, tx, err := ctx.IOTransactionCodec.Decode(input)
			if err != nil {
				return err
			}

			if wait {
				status, err := ctx.App.SubmitTransactionWithWait(cmd.Context(), txID, tx)
				if err != nil {
					return err
				}
				ctx.Printer.Print(
					fmt.Sprintf("Transaction status: %s", committerpb.Status_name[int32(status)]), //nolint:gosec
				)
				if status != int(committerpb.Status_COMMITTED) {
					return fmt.Errorf(
						"transaction failed with status: %s",
						committerpb.Status_name[int32(status)], //nolint:gosec
					)
				}
				return nil
			}

			return ctx.App.SubmitTransaction(cmd.Context(), txID, tx)
		},
	}
	wait.bind(cmd)

	return cmd
}
