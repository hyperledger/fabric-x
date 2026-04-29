/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewVerifyCommand returns the verify subcommand.
// Post-migration integrity verification is tracked in a follow-up PR once
// the committer bootstrap feature lands.
func NewVerifyCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "verify",
		Short: "Verify integrity between a Fabric snapshot and a Fabric-X state DB (coming soon)",
		Long: `verify compares key-value counts and cryptographic hashes between
a source Fabric peer snapshot and a live Fabric-X state database.

It produces a human-readable integrity report with:
  - Row count comparison per namespace
  - Block height match
  - Sampled key-value deep comparison
  - Policy registration check

Note: this command is a placeholder — full implementation follows once the
committer --init-from-snapshot bootstrap feature is merged.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), "verify: not yet implemented — tracked in follow-up PR")
			return nil
		},
	}
}
