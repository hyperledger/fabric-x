/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Option is a functional option for configuring the configuration loader.
type Option func(*viper.Viper)

// WithConfigFile specifies an explicit configuration file path to load.
// This takes precedence over project and user config files.
func WithConfigFile(path string) Option {
	return func(v *viper.Viper) {
		v.SetConfigFile(path)
	}
}

// WithOverride sets a configuration value that overrides all other sources.
// This is typically used for command-line flag values.
func WithOverride(key string, value any) Option {
	return func(v *viper.Viper) {
		v.Set(key, value)
	}
}

// Load loads configuration from multiple sources with a defined precedence hierarchy.
// Configuration is merged in order: user config, project config, environment variables,
// explicit config file (via WithConfigFile), and finally CLI flag overrides (via WithOverride).
// Returns a fully resolved Config with TLS settings inherited and merged across services.
func Load(opts ...Option) (*Config, error) {
	v := viper.New()

	// Register all configuration keys from the Config struct
	registerStructKeys(v, "", reflect.TypeFor[Config]())

	// Set up environment variable binding with FXCONFIG_ prefix
	v.SetEnvPrefix("FXCONFIG")
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	v.AutomaticEnv()

	// define our default config file to be "config.yaml"
	v.SetConfigName("config")
	v.SetConfigType("yaml")

	// Attempt to load user-level config (~/.fxconfig/config.yaml)
	if home, err := os.UserHomeDir(); err == nil {
		v.AddConfigPath(filepath.Join(home, ".fxconfig"))
		if err := v.MergeInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				return nil, fmt.Errorf("error loading user config: %w", err)
			}
		}
	}

	// Attempt to load project-level config (.fxconfig/config.yaml)
	v.AddConfigPath(".fxconfig")
	if err := v.MergeInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error loading project config: %w", err)
		}
	}

	// Apply functional options (flags, explicit config file)
	for _, opt := range opts {
		opt(v)
	}

	// Read explicit config file if specified via WithConfigFile
	if v.ConfigFileUsed() != "" {
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("error reading config file %s: %w", v.ConfigFileUsed(), err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	// Resolve TLS sections for each service
	cfg.ResolveTLS()

	return &cfg, nil
}

// registerStructKeys recursively registers all configuration fields with viper.
// It traverses the struct hierarchy and registers each field with its full path.
// Supports: int, []int, string, []string, bool, []bool, time.Duration, and nested structs.
func registerStructKeys(v *viper.Viper, viperPrefix string, t reflect.Type) { //nolint:gocognit
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	for i := range t.NumField() {
		field := t.Field(i)

		if !field.IsExported() {
			continue
		}

		tag := field.Tag.Get("mapstructure")
		if tag == "" || tag == "-" {
			continue
		}

		// Handle squash
		if tag == ",squash" {
			registerStructKeys(v, viperPrefix, field.Type)
			continue
		}

		viperKey := joinViperKey(viperPrefix, tag)

		defaultValue := field.Tag.Get("default")

		ft := field.Type

		// Recurse into nested struct (except time.Duration)
		isPtr := ft.Kind() == reflect.Pointer
		if isPtr {
			ft = ft.Elem()
		}
		if ft.Kind() == reflect.Struct && !isDuration(ft) {
			if !isPtr {
				registerStructKeys(v, viperKey, ft)
			}
			continue
		}

		registerSingleFlag(v, ft, viperKey, defaultValue)
	}
}

func registerSingleFlag(
	v *viper.Viper,
	fieldType reflect.Type,
	viperKey string,
	defaultValue string,
) {
	switch {
	case fieldType.Kind() == reflect.String:
		v.SetDefault(viperKey, defaultValue)

	case fieldType.Kind() == reflect.Int:
		k, _ := strconv.Atoi(defaultValue)
		v.SetDefault(viperKey, k)

	case fieldType.Kind() == reflect.Bool:
		b, _ := strconv.ParseBool(defaultValue)
		v.SetDefault(viperKey, b)

	case isDuration(fieldType):
		d, _ := time.ParseDuration(defaultValue)
		v.SetDefault(viperKey, d)

	case fieldType.Kind() == reflect.Slice:
		elem := fieldType.Elem()
		switch elem.Kind() {
		case reflect.String:
			// Register as empty slice for strings
			v.SetDefault(viperKey, []string{})
		case reflect.Int:
			v.SetDefault(viperKey, []int{})
		case reflect.Bool:
			v.SetDefault(viperKey, []bool{})
		default:
			return
		}

	default:
		return
	}
}

func joinViperKey(prefix, key string) string {
	if prefix == "" {
		return key
	}
	return prefix + "." + key
}

func isDuration(t reflect.Type) bool {
	return t == reflect.TypeFor[time.Duration]()
}
