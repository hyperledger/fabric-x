/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// NewInfoCommand returns a command that displays the effective configuration.
// The configuration is shown as YAML after applying all overrides from
// flags, environment variables, and config files.
func NewInfoCommand(ctx *CLIContext) *cobra.Command {
	var format string

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
		RunE: func(_ *cobra.Command, _ []string) error {
			if format == "yaml" {
				out, err := yaml.Marshal(ctx.Config)
				if err != nil {
					return err
				}
				ctx.Printer.Print(string(out))
				return nil
			}

			if format == "env" {
				out, err := yaml.Marshal(ctx.Config)
				if err != nil {
					return err
				}
				var m map[string]interface{}
				if err := yaml.Unmarshal(out, &m); err != nil {
					return err
				}
				envVars := flattenEnv("FXCONFIG", m)
				sort.Strings(envVars)
				ctx.Printer.Print(strings.Join(envVars, "\n") + "\n")
				return nil
			}

			return fmt.Errorf("unsupported format: %s", format)
		},
	}

	cmd.Flags().StringVar(&format, "format", "yaml", "output format (yaml|env)")

	return cmd
}

// flattenEnv recursively flattens a nested map into environment variable definitions.
func flattenEnv(prefix string, v interface{}) []string {
	var result []string
	switch typedVal := v.(type) {
	case map[string]interface{}:
		for k, val := range typedVal {
			nextPrefix := prefix + "_" + strings.ToUpper(k)
			result = append(result, flattenEnv(nextPrefix, val)...)
		}
	case []interface{}:
		for i, val := range typedVal {
			nextPrefix := fmt.Sprintf("%s_%d", prefix, i)
			result = append(result, flattenEnv(nextPrefix, val)...)
		}
	default:
		result = append(result, fmt.Sprintf("%s=%v", prefix, typedVal))
	}
	return result
}
