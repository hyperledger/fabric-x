/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/hyperledger/fabric-x-common/common/metadata"
)

// NewVersionCommand returns a command that displays version information.
// It shows the fxconfig version, Go version, commit SHA, and OS/architecture.
func NewVersionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Display version information",
		Long: `Display detailed version information including:
  • fxconfig version
  • Go compiler version
  • Git commit SHA
  • Operating system and architecture`,
		Run: func(cmd *cobra.Command, _ []string) {
			// TODO: use printer
			cmd.Printf("fxconfig\n")
			showLine(cmd, "Version", metadata.Version)
			showLine(cmd, "Go version", runtime.Version())
			showLine(cmd, "Commit", metadata.CommitSHA)
			showLine(cmd, "OS/Arch", fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH))
		},
	}

	return cmd
}

// showLine formats and prints a single line of version information.
// The title is capitalized and right-padded to 16 characters for alignment.
func showLine(cmd *cobra.Command, title, value string) {
	cmd.Printf(" %-16s %s\n", fmt.Sprintf("%s:", cases.Title(language.Und, cases.NoLower).String(title)), value)
}
