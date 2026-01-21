/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package namespace

import (
	"io"

	"github.com/hyperledger/fabric-x-common/cmd/common/comm"

	"github.com/spf13/cobra"
)

type listFunc func(out io.Writer, endpoint string, tlsConfig comm.Config) error

func newListCommand(listFunc listFunc) *cobra.Command {
	// this is our default query service endpoint
	endpoint := "localhost:7001"
	var tlsConfig comm.Config

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List installed Namespaces",
		Long:  "",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return listFunc(cmd.OutOrStdout(), endpoint, tlsConfig)
		},
	}

	cmd.PersistentFlags().StringVarP(&tlsConfig.PeerCACertPath,
		"cafile",
		"",
		"",
		"Path to file containing PEM-encoded trusted certificate(s) for the committer",
	)

	cmd.PersistentFlags().StringVarP(&tlsConfig.KeyPath,
		"keyfile",
		"",
		"",
		"Path to file containing PEM-encoded private key to use for mutual TLS communication with the committer",
	)
	cmd.PersistentFlags().StringVarP(&tlsConfig.CertPath,
		"certfile",
		"",
		"",
		"Path to file containing PEM-encoded public key to use for mutual TLS communication with the committer",
	)

	cmd.PersistentFlags().StringVar(
		&endpoint,
		"endpoint",
		"",
		"committer query service endpoint",
	)
	_ = cmd.MarkPersistentFlagRequired("endpoint")

	return cmd
}
