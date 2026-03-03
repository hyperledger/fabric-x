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

// newNsUpdateCommand creates a command for updating existing namespaces.
// It accepts a namespace name as argument and requires the --version flag to specify
// the current version number, preventing concurrent modification conflicts.
// The deployNamespace function is injected to enable testing with mock implementations.
func newNsUpdateCommand(ctx *CLIContext) *cobra.Command {
	var (
		// flag variables
		versionFlag    VersionFlag
		policyFlag     PolicyFlag
		outputFlag     OutputFlag
		namespaceFlags NamespaceDeployFlags
	)

	cmd := &cobra.Command{
		Use:   "update [name]",
		Short: "Update existing namespace",
		Long: `Update an existing namespace's endorsement policy.

The --version flag is required to prevent concurrent modification conflicts.
Use 'fxconfig namespace list' to find the current version number.

Version numbers increment with each successful update. If the version you
specify doesn't match the current version, the update will fail.

Examples:
  # Update namespace policy (check version first with 'list')
  fxconfig namespace update hello \
    --policy="OR('Org2MSP.member')" \
    --version=0 \
    --endorse --submit --wait

  # Change from single-org to multi-org policy
  fxconfig namespace update hello \
    --policy="AND('Org1MSP.member', 'Org2MSP.member')" \
    --version=1 \
    --output=tx.json

  # Update and save transaction for multi-org endorsement
  fxconfig namespace update payments \
    --policy="OutOf(2, 'Org1MSP.member', 'Org2MSP.member', 'Org3MSP.member')" \
    --version=2 \
    --output=update_tx.json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p := app.PolicyConfig{}
			p.Set(string(policyFlag))

			input := app.DeployNamespaceInput{
				NsID:    args[0],
				Version: int(versionFlag),
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
	versionFlag.Bind(cmd)
	policyFlag.Bind(cmd)
	outputFlag.Bind(cmd)
	namespaceFlags.Bind(cmd)

	return cmd
}
