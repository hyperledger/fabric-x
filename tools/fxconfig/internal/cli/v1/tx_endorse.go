/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"github.com/spf13/cobra"

	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/cli/v1/io"
)

// newTxEndorseCommand creates a command for endorsing transactions.
func newTxEndorseCommand(ctx *CLIContext) *cobra.Command {
	var outputInput OutputFlag

	cmd := &cobra.Command{
		Use:   "endorse",
		Short: "Endorse transaction",
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

			endorsedTx, err := ctx.App.EndorseTransaction(cmd.Context(), txID, tx)
			if err != nil {
				return err
			}

			o, err := ctx.IOTransactionCodec.Encode(txID, endorsedTx)
			if err != nil {
				return err
			}

			return io.WriteOutput(cmd, string(outputInput), o)
		},
	}
	outputInput.Bind(cmd)

	return cmd
}
