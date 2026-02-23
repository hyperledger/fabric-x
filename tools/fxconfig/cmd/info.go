/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cmd

import (
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// NewInfoCommand returns a command that displays the effective configuration.
// The configuration is shown as YAML after applying all overrides from
// flags, environment variables, and config files.
func NewInfoCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info",
		Short: "Display system-wide information",
		Long:  ``,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg := getConfig(cmd)

			out, err := yaml.Marshal(cfg)
			if err != nil {
				return err
			}

			cmd.Println(string(out))
			return nil
		},
	}

	return cmd
}
