/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"github.com/spf13/cobra"

	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/config"
)

// newUpdateCommand creates a command for updating existing namespaces.
// It accepts a namespace name as argument and requires the --version flag to specify
// the current version number, preventing concurrent modification conflicts.
// The deployNamespace function is injected to enable testing with mock implementations.
func newUpdateCommand(deployNamespace deployF) *cobra.Command {
	var (
		nsCfg  config.NsConfig
		policy string
	)

	cmd := &cobra.Command{
		Use:   "update NAMESPACE_NAME",
		Short: "Update Namespace",
		Long:  "",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := getConfig(cmd)

			nsCfg.NamespaceID = args[0]
			nsCfg.Policy.Set(policy)

			vctx := *getConfigValidatorContext(cmd)
			return deployNamespace(vctx, *cfg, nsCfg)
		},
	}

	// adds flags related to namespaces
	cmd.Flags().IntVar(&nsCfg.Version,
		"version",
		0,
		"The current namespace version",
	)

	cmd.PersistentFlags().StringVar(&policy,
		"policy",
		"",
		"The new endorsement policy",
	)
	cmd.MarkFlagsRequiredTogether("policy", "version")

	return cmd
}
