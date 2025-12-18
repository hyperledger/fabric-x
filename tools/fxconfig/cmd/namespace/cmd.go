/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package namespace

import (
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/namespace"
	"github.com/spf13/cobra"
)

// NewNamespaceCommand returns a namespace command. It collects all namespace related commands.
func NewNamespaceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "namespace",
		Short: "Perform namespace operations",
		Long:  "",
	}

	cmd.AddCommand(
		newCreateCommand(namespace.DeployNamespace),
		newListCommand(namespace.List),
		newUpdateCommand(namespace.DeployNamespace),
	)

	return cmd
}

// ordererFlags adds flags for orderer-related commands.
func ordererFlags(cmd *cobra.Command, ordererCfg *namespace.OrdererConfig) {
	cmd.PersistentFlags().String(
		"channel",
		"",
		"The name of the channel",
	)

	cmd.PersistentFlags().StringVarP(
		&ordererCfg.OrderingEndpoint,
		"orderer",
		"o",
		"",
		"Ordering service endpoint",
	)
	cmd.PersistentFlags().StringVarP(&ordererCfg.Config.PeerCACertPath,
		"cafile",
		"",
		"",
		"Path to file containing PEM-encoded trusted certificate(s) for the ordering endpoint",
	)
	cmd.PersistentFlags().StringVarP(&ordererCfg.Config.KeyPath,
		"keyfile",
		"",
		"",
		"Path to file containing PEM-encoded private key to use for mutual TLS communication with the orderer endpoint",
	)
	cmd.PersistentFlags().StringVarP(&ordererCfg.Config.CertPath,
		"certfile",
		"",
		"",
		"Path to file containing PEM-encoded public key to use for mutual TLS communication with the orderer endpoint",
	)
	cmd.PersistentFlags().DurationVarP(&ordererCfg.Config.Timeout,
		"connTimeout",
		"",
		namespace.DefaultTimeout,
		"Timeout for client to connect",
	)
}

// mspFlags adds flags to specify the MSP that will sign the requests.
func mspFlags(cmd *cobra.Command, mspCfg *namespace.MSPConfig) {
	cmd.PersistentFlags().StringVarP(&mspCfg.MSPConfigPath,
		"mspConfigPath",
		"",
		"",
		"The path to the MSP config directory",
	)
	cmd.PersistentFlags().StringVarP(&mspCfg.MSPID,
		"mspID",
		"",
		"",
		"The name of the MSP",
	)
}
