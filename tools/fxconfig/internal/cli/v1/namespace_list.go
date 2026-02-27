/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/config"
)

// listFunc is a function type for listing namespaces.
// It queries the committer service, formats the results, and writes them to the provided writer.
// This abstraction enables dependency injection for testing.
type listFunc func(vctx config.ValidationContext, cfg config.QueriesConfig, out io.Writer) error

// newListCommand creates a command for listing installed namespaces.
// It connects to the query service and displays namespace names, versions, and policies.
// The listFunc is injected to enable testing with mock implementations.
func newListCommand(listFunc listFunc) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List installed Namespaces",
		Long:  "",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg := getConfig(cmd)

			// validate config
			vctx := *getConfigValidatorContext(cmd)
			if err := cfg.Queries.Validate(vctx); err != nil {
				return fmt.Errorf("query service configuration invalid: %w", err)
			}

			return listFunc(vctx, cfg.Queries, cmd.OutOrStdout())
		},
	}

	return cmd
}
