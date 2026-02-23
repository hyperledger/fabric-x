/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package cmd implements the command-line interface for fxconfig.
// It provides commands for namespace management and configuration handling.
package cmd

import (
	"github.com/spf13/cobra"

	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/config"
)

// configKeyType is a custom type for context keys to avoid collisions.
type configKeyType string

// configKey is the context key used to store and retrieve configuration.
const configKey configKeyType = "app-config"

// getConfig retrieves the configuration from the command context.
// Panics if the configuration is not found, indicating a programming error.
func getConfig(cmd *cobra.Command) *config.Config {
	cfg, ok := cmd.Context().Value(configKey).(*config.Config)
	if !ok {
		panic("programming error: extracting config from context failed")
	}
	return cfg
}
