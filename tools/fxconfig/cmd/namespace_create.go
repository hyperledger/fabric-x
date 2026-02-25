/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cmd

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/config"
)

// deployF is a function type for deploying namespace transactions.
// It takes namespace configuration, orderer configuration, and MSP configuration,
// and returns an error if the deployment fails.
type deployF func(nsCfg config.NsConfig, ordererCfg config.OrdererConfig, mspCfg config.MSPConfig) error

// newCreateCommand creates a command for creating new namespaces.
// The deployNamespace function is injected to enable testing with mock implementations.
func newCreateCommand(deployNamespace deployF) *cobra.Command {
	var nsCfg config.NsConfig

	cmd := &cobra.Command{
		Use:   "create NAMESPACE_NAME",
		Short: "Create Namespace",
		Long:  "",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := getConfig(cmd)

			nsCfg.NamespaceID = args[0]

			// Set version to -1 to indicate this is a create operation (not an update)
			nsCfg.Version = -1

			// validate config
			err := errors.Join(
				config.ValidateMSPConfig(cfg.MSP),
				config.ValidateOrdererConfig("orderer", cfg.Orderer),
				config.ValidateNsConfig(nsCfg),
			)
			if err != nil {
				return err
			}

			return deployNamespace(nsCfg, cfg.Orderer, cfg.MSP)
		},
	}

	// adds flags related to namespaces
	cmd.PersistentFlags().StringVar(&nsCfg.ThresholdPolicyVerificationKeyPath,
		"policy-ecdsa-threshold",
		"",
		"The path to the ecdsa threshold verification key",
	)

	cmd.PersistentFlags().StringVar(&nsCfg.Policy,
		"policy",
		"",
		"The MSP-based endorsement policy",
	)

	cmd.Flags().StringVar(&nsCfg.Channel,
		"channel",
		"",
		"[DEPRECATED] The channel",
	)

	return cmd
}
