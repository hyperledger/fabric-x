/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// NewInfoCommand returns a command that displays the effective configuration.
// The configuration is shown as YAML after applying all overrides from
// flags, environment variables, and config files.
func NewInfoCommand(ctx *CLIContext) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info",
		Short: "Display system-wide information",
		Long:  ``,
		RunE: func(cmd *cobra.Command, _ []string) error {
			out, err := yaml.Marshal(ctx.Config)
			if err != nil {
				return err
			}
			return ctx.Printer.Print(string(out))
		},
	}

	return cmd
}
