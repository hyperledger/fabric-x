/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"fmt"
	"reflect"
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
				env := structToEnv("FXCONFIG", ctx.Config)
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

// structToEnv converts a struct to a flat list of KEY=value environment variable
// strings by walking all exported fields via reflection. Field names are derived
// from mapstructure tags. Unlike the previous YAML round-trip approach, this
// preserves zero-value fields (false, 0s, empty strings) so the full effective
// configuration is always visible.
func structToEnv(prefix string, cfg any) []string {
	v := reflect.ValueOf(cfg)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil
	}

	return flattenStruct(prefix, v)
}

// flattenStruct recursively walks the struct fields and builds env var entries.
func flattenStruct(prefix string, v reflect.Value) []string {
	var result []string
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		tag := field.Tag.Get("mapstructure")
		if tag == "-" {
			continue
		}

		name, squash := parseMapstructureTag(tag)
		if name == "" {
			name = field.Name
		}

		fieldVal := v.Field(i)

		// squash means the embedded struct's fields are promoted into the parent
		if squash {
			inner := reflect.Indirect(fieldVal)
			if inner.IsValid() && inner.Kind() == reflect.Struct {
				result = append(result, flattenStruct(prefix, inner)...)
			}
			continue
		}

		key := envKey(prefix, name)
		result = append(result, flattenValue(key, fieldVal)...)
	}

	return result
}

// flattenValue converts a single reflected value into env var entries.
func flattenValue(key string, v reflect.Value) []string {
	// dereference pointers — nil pointers are skipped
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Struct:
		// types like time.Duration implement Stringer; print them as values
		if stringer, ok := v.Interface().(fmt.Stringer); ok {
			return []string{fmt.Sprintf("%s=%s", key, stringer.String())}
		}
		return flattenStruct(key, v)

	case reflect.Slice:
		items := make([]string, v.Len())
		for i := 0; i < v.Len(); i++ {
			items[i] = fmt.Sprintf("%v", v.Index(i).Interface())
		}
		return []string{fmt.Sprintf("%s=%s", key, strings.Join(items, ","))}

	default:
		return []string{fmt.Sprintf("%s=%v", key, v.Interface())}
	}
}

// envKey builds an environment variable key from a prefix and field name.
func envKey(prefix, name string) string {
	key := strings.ToUpper(strings.ReplaceAll(name, "-", "_"))
	if prefix != "" {
		return prefix + "_" + key
	}
	return key
}

// parseMapstructureTag extracts the field name and squash flag from a
// mapstructure struct tag. Examples:
//
//	"address"   → ("address", false)
//	",squash"   → ("", true)
//	""          → ("", false)
func parseMapstructureTag(tag string) (name string, squash bool) {
	parts := strings.Split(tag, ",")
	if len(parts) > 0 {
		name = parts[0]
	}
	for _, p := range parts[1:] {
		if p == "squash" {
			squash = true
		}
	}
	return name, squash
}
