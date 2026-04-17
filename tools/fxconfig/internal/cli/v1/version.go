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
func NewVersionCommand(cliCtx *CLIContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Display version information",
		Long: `Display detailed version information including:
  • fxconfig version
  • Go compiler version
  • Git commit SHA
  • Operating system and architecture`,
		Run: func(cmd *cobra.Command, _ []string) {
			cliCtx.Printer.Print("fxconfig\n")
			cliCtx.Printer.Print(formatLine("Version", metadata.Version))
			cliCtx.Printer.Print(formatLine("Go version", runtime.Version()))
			cliCtx.Printer.Print(formatLine("Commit", metadata.CommitSHA))
			cliCtx.Printer.Print(formatLine("OS/Arch", fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)))
		},
	}

	return cmd
}

// formatLine formats and returns a single line of version information.
// The title is capitalized and right-padded to 16 characters for alignment.
func formatLine(title, value string) string {
	return fmt.Sprintf(" %-16s %s\n", fmt.Sprintf("%s:", cases.Title(language.Und, cases.NoLower).String(title)), value)
}
