/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// NewInfoCommand returns a command that displays the effective configuration.
// The configuration is shown in the requested format (yaml or env) after applying
// all overrides from flags, environment variables, and config files.
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
			switch format {
			case "env":
				env := toEnv("FXCONFIG", ctx.Config)
				slices.Sort(env)
				ctx.Printer.Print(strings.Join(env, "\n") + "\n")
			case "yaml":
				out, err := yaml.Marshal(ctx.Config)
				if err != nil {
					return err
				}
				ctx.Printer.Print(string(out))
			default:
				return fmt.Errorf("invalid --format: %s (want yaml|env)", format)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&format, "format", "yaml", "Output format (yaml|env)")

	return cmd
}

func toEnv(prefix string, cfg any) []string {
	// we use yaml marshaling as a shortcut to get a map representation of the config
	// that respects all yaml tags and omitempty.
	out, err := yaml.Marshal(cfg)
	if err != nil {
		return nil
	}

	var m map[string]any
	if err := yaml.Unmarshal(out, &m); err != nil {
		return nil
	}

	return flatten(prefix, m)
}

func flatten(prefix string, m map[string]any) []string {
	var result []string
	for k, v := range m {
		key := strings.ToUpper(strings.ReplaceAll(k, "-", "_"))
		if prefix != "" {
			key = prefix + "_" + key
		}

		switch val := v.(type) {
		case map[string]any:
			result = append(result, flatten(key, val)...)
		case []any:
			strVals := make([]string, 0, len(val))
			for _, item := range val {
				strVals = append(strVals, fmt.Sprintf("%v", item))
			}
			result = append(result, fmt.Sprintf("%s=%s", key, strings.Join(strVals, ",")))
		default:
			result = append(result, fmt.Sprintf("%s=%v", key, val))
		}
	}
	return result
}
