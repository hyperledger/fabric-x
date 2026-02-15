/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package namespace

import (
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/namespace"
	"github.com/spf13/cobra"
)

type deployF func(nsCfg namespace.NsConfig, ordererCfg namespace.OrdererConfig, mspCfg namespace.MSPConfig) error

func newCreateCommand(deployNamespace deployF) *cobra.Command {
	var (
		ordererCfg namespace.OrdererConfig
		mspCfg     namespace.MSPConfig
		nsCfg      namespace.NsConfig
	)

	cmd := &cobra.Command{
		Use:   "create NAMESPACE_NAME",
		Short: "Create Namespace",
		Long:  "",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			nsCfg.NamespaceID = args[0]
			nsCfg.Version = -1

			return deployNamespace(nsCfg, ordererCfg, mspCfg)
		},
	}

	ordererFlags(cmd, &ordererCfg)
	mspFlags(cmd, &mspCfg)

	// adds flags related to namespaces
	cmd.PersistentFlags().StringVar(&nsCfg.ThresholdPolicyVerificationKeyPath,
		"policy-ecdsa-threshold",
		"",
		"The path to the ecdsa threshold verification key",
	)
	_ = cmd.MarkFlagRequired("policy-ecdsa-threshold")

	cmd.Flags().StringVar(&nsCfg.Channel,
		"channel",
		"",
		"The channel",
	)
	_ = cmd.MarkFlagRequired("channel")

	return cmd
}
