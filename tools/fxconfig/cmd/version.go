/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/hyperledger/fabric-x-common/common/metadata"
)

// NewVersionCommand returns a version command.
func NewVersionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version of fxconfig",
		Long:  ``,
		Run: func(cmd *cobra.Command, _ []string) {
			cmd.Printf("fxconfig\n")
			showLine(cmd, "Version", metadata.Version)
			showLine(cmd, "Go version", runtime.Version())
			showLine(cmd, "Commit", metadata.CommitSHA)
			showLine(cmd, "OS/Arch", fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH))
		},
	}

	return cmd
}

func showLine(cmd *cobra.Command, title, value string) {
	cmd.Printf(" %-16s %s\n", fmt.Sprintf("%s:", cases.Title(language.Und, cases.NoLower).String(title)), value)
}
