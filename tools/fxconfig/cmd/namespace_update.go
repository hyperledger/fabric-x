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

// newUpdateCommand creates a command for updating existing namespaces.
// It accepts a namespace name as argument and requires the --version flag to specify
// the current version number, preventing concurrent modification conflicts.
// The deployNamespace function is injected to enable testing with mock implementations.
func newUpdateCommand(deployNamespace deployF) *cobra.Command {
	var nsCfg config.NsConfig

	cmd := &cobra.Command{
		Use:   "update NAMESPACE_NAME",
		Short: "Update Namespace",
		Long:  "",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := getConfig(cmd)

			nsCfg.NamespaceID = args[0]

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
	cmd.Flags().IntVar(&nsCfg.Version,
		"version",
		0,
		"The version",
	)
	_ = cmd.MarkFlagRequired("version")

	cmd.PersistentFlags().StringVar(&nsCfg.ThresholdPolicyVerificationKeyPath,
		"policy-ecdsa-threshold",
		"",
		"The path to the ecdsa threshold verification key",
	)

	cmd.Flags().StringVar(&nsCfg.Channel,
		"channel",
		"",
		"The channel name",
	)

	return cmd
}
