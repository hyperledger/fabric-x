/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package namespace

import (
	"io"

	"github.com/spf13/cobra"
)

type listFunc func(out io.Writer, endpoint, cacert string) error

func newListCommand(listFunc listFunc) *cobra.Command {
	// this is our default query service endpoint
	endpoint := "localhost:7001"
	cacert := ""

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List installed Namespaces",
		Long:  "",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return listFunc(cmd.OutOrStdout(), endpoint, cacert)
		},
	}

	cmd.PersistentFlags().StringVarP(&cacert,
		"cafile",
		"",
		"",
		"Path to file containing PEM-encoded trusted certificate(s) for the committer query service endpoint",
	)

	// TODO: add client crt / key for mTLS support

	cmd.PersistentFlags().StringVar(
		&endpoint,
		"endpoint",
		"",
		"committer query service endpoint",
	)
	_ = cmd.MarkPersistentFlagRequired("endpoint")

	return cmd
}
