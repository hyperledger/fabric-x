/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/app"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/cli/v1/io"
)

// newNsCreateCommand creates a command for creating new namespaces.
// The deployNamespace function is injected to enable testing with mock implementations.
func newNsCreateCommand(ctx *CLIContext) *cobra.Command {
	var (
		policyFlag     PolicyFlag
		outputFlag     OutputFlag
		namespaceFlags NamespaceDeployFlags
	)

	cmd := &cobra.Command{
		Use:   "create [name]",
		Short: "Create new namespace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p := app.PolicyConfig{}
			p.Set(string(policyFlag))

			input := app.DeployNamespaceInput{
				NsID:    args[0],
				Version: -1, // Set version to -1 to indicate this is a create operation (not an update)
				Policy:  p,
				Endorse: namespaceFlags.endorse,
				Submit:  namespaceFlags.submit,
				Wait:    namespaceFlags.wait,
			}

			res, status, err := ctx.App.DeployNamespace(cmd.Context(), &input)
			if err != nil {
				return err
			}

			if res == nil {
				ctx.Printer.Print(fmt.Sprintf("Status code: %d", status))
				return nil
			}

			o, err := ctx.IOTransactionCodec.Encode(res.TxID, res.Tx)
			if err != nil {
				return err
			}

			return io.WriteOutput(cmd, string(outputFlag), o)
		},
	}

	// adds flags related to namespaces
	policyFlag.Bind(cmd)
	outputFlag.Bind(cmd)
	namespaceFlags.Bind(cmd)

	return cmd
}
