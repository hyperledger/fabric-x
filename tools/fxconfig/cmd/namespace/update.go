/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package namespace

import (
	"errors"

	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/namespace"
	"github.com/spf13/cobra"
)

func newUpdateCommand(deployNamespace deployF) *cobra.Command {
	var (
		ordererCfg namespace.OrdererConfig
		mspCfg     namespace.MSPConfig
		nsCfg      namespace.NsConfig
		err        error
	)

	cmd := &cobra.Command{
		Use:   "update NAMESPACE_NAME",
		Short: "Update Namespace",
		Long:  "",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			nsCfg.NamespaceID = args[0]

			nsCfg.Channel, err = cmd.Flags().GetString("channel")
			if err != nil {
				return err
			}
			if nsCfg.Channel == "" {
				return errors.New("you must specify a channel name '--channel channelName'")
			}

			nsCfg.VerificationKeyPath, err = cmd.Flags().GetString("pk")
			if err != nil {
				return err
			}

			nsCfg.Version, err = cmd.Flags().GetInt("version")
			if err != nil {
				return err
			}

			return deployNamespace(nsCfg, ordererCfg, mspCfg)
		},
	}

	ordererFlags(cmd, &ordererCfg)
	mspFlags(cmd, &mspCfg)

	// adds flags related to namespaces
	cmd.PersistentFlags().String(
		"pk",
		"",
		"The path to the public key of the endorser",
	)
	_ = cmd.PersistentFlags().MarkDeprecated(
		"pk",
		"This flag is deprecated and will be removed in future versions",
	)

	cmd.PersistentFlags().Int(
		"version",
		0,
		"The version of this namespace definition",
	)

	return cmd
}
