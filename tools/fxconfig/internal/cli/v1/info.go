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
		Short: "Display effective configuration",
		Long: `Display the effective configuration after applying all overrides.

Configuration is loaded and merged in the following order (later sources override earlier):
  1. Default values
  2. User config file ($HOME/.fxconfig/config.yaml)
  3. Project config file (.fxconfig/config.yaml)
  4. Config file (--config flag)
  5. Environment variables (FXCONFIG_*)

Use this command to verify your configuration before executing operations.

Examples:
  # Show current configuration
  fxconfig info

  # Show configuration from specific file
  fxconfig info --config /path/to/config.yaml

  # Show configuration as environment variables
  fxconfig info --format env`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// TODO: add flag to show yaml/env config
			out, err := yaml.Marshal(ctx.Config)
			if err != nil {
				return err
			}
			return ctx.Printer.Print(string(out))
		},
	}

	return cmd
}
