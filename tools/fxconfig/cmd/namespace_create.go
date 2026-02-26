/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cmd

import (
	"github.com/spf13/cobra"

	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/config"
)

// deployF is a function type for deploying namespace transactions.
// It takes namespace configuration, orderer configuration, and MSP configuration,
// and returns an error if the deployment fails.
type deployF func(vctx config.ValidationContext, cfg config.Config, nsCfg config.NsConfig) error

// newCreateCommand creates a command for creating new namespaces.
// The deployNamespace function is injected to enable testing with mock implementations.
func newCreateCommand(deployNamespace deployF) *cobra.Command {
	var (
		nsCfg  config.NsConfig
		policy string
	)

	cmd := &cobra.Command{
		Use:   "create NAMESPACE_NAME",
		Short: "Create Namespace",
		Long:  "",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := getConfig(cmd)

			nsCfg.NamespaceID = args[0]
			nsCfg.Policy.Set(policy)
			// Set version to -1 to indicate this is a create operation (not an update)
			nsCfg.Version = -1

			vctx := *getConfigValidatorContext(cmd)
			return deployNamespace(vctx, *cfg, nsCfg)
		},
	}

	// adds flags related to namespaces
	cmd.PersistentFlags().StringVar(&policy,
		"policy",
		"",
		"The endorsement policy",
	)
	_ = cmd.MarkPersistentFlagRequired("policy")

	return cmd
}
