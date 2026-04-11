/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"fmt"
	"maps"
	"reflect"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// NewInfoCommand returns a command that displays the effective configuration.
// The configuration is shown as YAML after applying all overrides from
// flags, environment variables, and config files.
func NewInfoCommand(ctx *CLIContext) *cobra.Command {
	var format formatFlag

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
			var output string

			switch format {
			case "yaml":
				out, err := yaml.Marshal(ctx.Config)
				if err != nil {
					return err
				}
				output = string(out)
			case "env":
				envVars := configToEnvVars("FXCONFIG", ctx.Config)
				if len(envVars) == 0 {
					output = "null"
					break
				}

				keys := make([]string, 0, len(envVars))
				for key := range envVars {
					keys = append(keys, key)
				}
				sort.Strings(keys)

				var b strings.Builder
				for _, key := range keys {
					_, _ = fmt.Fprintf(&b, "%s=%s\n", key, envVars[key])
				}
				output = b.String()
			default:
				return fmt.Errorf("unsupported format %q (supported formats are yaml and env)", format)
			}
			ctx.Printer.Print(output)
			return nil
		},
	}

	// adds flags related to info
	format.bind(cmd)

	return cmd
}

// configToEnvVars converts configuration to environment variable format.
// Each field is output as FXCONFIG_<FIELD>=<VALUE>.
func configToEnvVars(envPrefix string, s any) map[string]string { //nolint:gocognit
	result := make(map[string]string)
	if s == nil {
		return result
	}

	val := reflect.ValueOf(s)
	typ := reflect.TypeOf(s)

	// Dereference pointer to struct
	if val.Kind() == reflect.Pointer {
		if val.IsNil() {
			return result
		}
		val = val.Elem()
		typ = typ.Elem()
	}

	for i := range typ.NumField() {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		if !field.IsExported() {
			continue
		}

		tag := field.Tag.Get("mapstructure")
		if tag == "" || tag == "-" {
			continue
		}

		// Handle squash
		if tag == ",squash" {
			nested := configToEnvVars(envPrefix, fieldVal.Interface())
			maps.Copy(result, nested)
			continue
		}

		tagName, _, _ := strings.Cut(tag, ",")
		if tagName == "" {
			continue
		}
		envKey := envPrefix + "_" + strings.ToUpper(tagName)

		ft := field.Type
		isPtr := ft.Kind() == reflect.Pointer
		if isPtr {
			if fieldVal.IsNil() {
				continue
			}
			ft = ft.Elem()
		}

		if ft.Kind() == reflect.Slice {
			sliceValue := fieldVal
			if isPtr {
				sliceValue = fieldVal.Elem()
			}
			if sliceValue.Len() > 0 {
				var items []string
				for j := range sliceValue.Len() {
					items = append(items, fmt.Sprintf("%v",
						sliceValue.Index(j).Interface()))
				}
				result[envKey] = strings.Join(items, ",")
			}
			continue
		}

		// Recurse into nested struct
		if ft.Kind() == reflect.Struct {
			nested := configToEnvVars(envKey, fieldVal.Interface())
			maps.Copy(result, nested)
			continue
		}

		// Handle primitive values
		var value any
		if isPtr {
			value = fieldVal.Elem().Interface()
		} else {
			value = fieldVal.Interface()
		}

		// Ignore zero values such as empty strings, 0, false, etc.
		if !reflect.ValueOf(value).IsZero() {
			result[envKey] = fmt.Sprintf("%v", value)
		}
	}

	return result
}
