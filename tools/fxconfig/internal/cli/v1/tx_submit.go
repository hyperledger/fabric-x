/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/cli/v1/io"
)

// newTxSubmitCommand creates a command for submitting transactions.
func newTxSubmitCommand(ctx *CLIContext) *cobra.Command {
	var waitFlag WaitFlag

	cmd := &cobra.Command{
		Use:   "submit",
		Short: "Submit transaction",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			input, err := io.ResolveInput(cmd, args[0])
			if err != nil {
				return err
			}

			txID, tx, err := ctx.IOTransactionCodec.Decode(input)
			if err != nil {
				return err
			}

			if waitFlag {
				status, err := ctx.App.SubmitTransactionWithWait(cmd.Context(), txID, tx)
				if err != nil {
					return err
				}
				ctx.Printer.Print(fmt.Sprintf("Status Code: %d", status))
				return nil
			}

			return ctx.App.SubmitTransaction(cmd.Context(), txID, tx)
		},
	}
	waitFlag.Bind(cmd)

	return cmd
}
