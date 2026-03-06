/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newNsListCommand creates a command for listing installed namespaces.
// It connects to the query service and displays namespace names, versions, and policies.
// The listFunc is injected to enable testing with mock implementations.
func newNsListCommand(ctx *CLIContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List installed Namespaces",
		Long: `Query and display all installed namespaces with their configurations.

For each namespace, displays:
  • Name (namespace identifier)
  • Version (current version number)
  • Policy (endorsement policy in hexadecimal format)

Use this command to:
  • Verify namespace deployment
  • Check current version before updates
  • Audit endorsement policies

Examples:
  # List all namespaces
  fxconfig namespace list

  # List with custom config
  fxconfig namespace list --config /path/to/config.yaml

  # List and save output to file
  fxconfig namespace list > namespaces.txt`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			result, err := ctx.App.ListNamespaces(cmd.Context())
			if err != nil {
				return err
			}

			// print namespace policy information to the Output writer.
			// Each namespace is displayed with its index, name, version, and policy in hexadecimal format.
			ctx.Printer.Print(fmt.Sprintf("Installed namespaces (%d total):\n", len(result)))
			for i, p := range result {
				ctx.Printer.Print(fmt.Sprintf("%d) %v: version %d policy: %x\n", i, p.NsID, p.Version, p.Policy))
			}

			return nil
		},
	}

	return cmd
}
