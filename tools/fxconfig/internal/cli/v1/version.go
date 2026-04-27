/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/hyperledger/fabric-x-common/common/metadata"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/cli/v1/cliio"
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
			osArch := fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
			p := cliio.NewCLIPrinter(cmd.OutOrStdout(), cmd.ErrOrStderr(), cliio.FormatTable)
			p.Print("fxconfig\n")
			p.Print(fmt.Sprintf(" %-16s %s\n", "Version:", metadata.Version))
			p.Print(fmt.Sprintf(" %-16s %s\n", "Go Version:", runtime.Version()))
			p.Print(fmt.Sprintf(" %-16s %s\n", "Commit:", metadata.CommitSHA))
			p.Print(fmt.Sprintf(" %-16s %s\n", "OS/Arch:", osArch))
		},
	}

	return cmd
}
