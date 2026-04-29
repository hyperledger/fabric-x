/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/hyperledger/fabric-x-common/api/committerpb"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/app"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/cli/v1/cliio"
)

// newNsCreateCommand creates a command for creating new namespaces.
// The deployNamespace function is injected to enable testing with mock implementations.
func newNsCreateCommand(ctx *CLIContext) *cobra.Command {
	var (
		policy    policyFlag
		output    outputFlag
		namespace namespaceDeployFlags
	)

	cmd := &cobra.Command{
		Use:   "create [name]",
		Short: "Create new namespace",
		Long: `Create a new namespace with an endorsement policy.

The endorsement policy defines which organizations must sign transactions
in this namespace. Policies use MSP identifiers and logical operators.

Policy Syntax:
  • OR('Org1MSP.member')                    - Any member of Org1
  • AND('Org1MSP.member', 'Org2MSP.member') - Both Org1 and Org2
  • OutOf(2, 'Org1MSP.member', 'Org2MSP.member', 'Org3MSP.member') - 2 of 3 orgs

Transaction Lifecycle Flags:
  --endorse  Collect endorsement from local MSP
  --submit   Submit transaction to ordering service
  --wait     Wait for transaction finalization (implies --submit)

Examples:
  # Create namespace with single org policy (save to file)
  fxconfig namespace create hello --policy="OR('Org1MSP.member')" --output=tx.json

  # Create and immediately deploy (endorse + submit + wait)
  fxconfig namespace create hello --policy="OR('Org1MSP.member')" --endorse --submit --wait

  # Multi-org policy requiring both organizations
  fxconfig namespace create payments \
    --policy="AND('Org1MSP.member', 'Org2MSP.member')" \
    --endorse --submit

  # Complex policy with threshold
  fxconfig namespace create voting \
    --policy="OutOf(2, 'Org1MSP.member', 'Org2MSP.member', 'Org3MSP.member')" \
    --output=tx.json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p := app.PolicyConfig{}
			p.Set(string(policy))

			input := app.DeployNamespaceInput{
				NsID:    args[0],
				Version: -1, // Set version to -1 to indicate this is a create operation (not an update)
				Policy:  p,
				Endorse: namespace.endorse,
				Submit:  namespace.submit,
				Wait:    namespace.wait,
			}

			res, status, err := ctx.App.DeployNamespace(cmd.Context(), &input)
			if err != nil {
				return err
			}

			if res == nil {
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

			o, err := ctx.IOTransactionCodec.Encode(res.TxID, res.Tx)
			if err != nil {
				return err
			}

			return cliio.WriteOutput(cmd, string(output), o)
		},
	}

	// adds flags related to namespaces
	policy.bind(cmd)
	output.bind(cmd)
	namespace.bind(cmd)

	return cmd
}
