/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package cmd implements the command-line interface for fxconfig.
// It provides commands for namespace management and configuration handling.
package cmd

import (
	"context"
	"os"
	"os/signal"

	"github.com/spf13/cobra"

	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/config"
)

// contextKeyType is a custom type for context keys to avoid collisions.
type contextKeyType string

const (
	// configKey is the context key used to store and retrieve configuration.
	configKey contextKeyType = "app-config"

	// configValidatorKey is the context key used to store and retrieve configuration.
	configValidatorKey contextKeyType = "app-config-validator"
)

// Execute is the entry point for the fxconfig CLI application.
// It builds the command tree and executes the root command.
func Execute() error {
	return RootCmd().Execute()
}

// RootCmd constructs and returns the root command for fxconfig.
// It sets up configuration loading, flag registration, and all subcommands.
// Configuration is loaded in PersistentPreRunE.
func RootCmd() *cobra.Command {
	var cfgFile string

	rootCmd := &cobra.Command{
		Use: "fxconfig",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			var opts []config.Option

			// Add config file option if specified
			if cfgFile != "" {
				opts = append(opts, config.WithConfigFile(cfgFile))
			}

			// Load configuration with all overrides applied
			cfg, err := config.Load(opts...)
			if err != nil {
				return err
			}

			// setup config validator
			vctx := config.NewValidationContext()

			// Store configuration and validator in command context for access by subcommands
			ctx := cmd.Context()
			ctx = context.WithValue(ctx, configKey, cfg)
			ctx = context.WithValue(ctx, configValidatorKey, &vctx)
			cmd.SetContext(ctx)

			return nil
		},
	}

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file")

	// Register all subcommands
	rootCmd.AddCommand(NewVersionCommand())
	rootCmd.AddCommand(NewInfoCommand())
	rootCmd.AddCommand(NewNamespaceCommand())

	//
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	rootCmd.SetContext(ctx)

	return rootCmd
}

// getConfig retrieves the configuration from the command context.
// Panics if the configuration is not found, indicating a programming error.
func getConfig(cmd *cobra.Command) *config.Config {
	cfg, ok := cmd.Context().Value(configKey).(*config.Config)
	if !ok {
		panic("programming error: extracting config from context failed")
	}
	return cfg
}

// getConfigValidatorContext retrieves the configuration validator from the command context.
// Panics if the configuration validator is not found, indicating a programming error.
func getConfigValidatorContext(cmd *cobra.Command) *config.ValidationContext {
	vctx, ok := cmd.Context().Value(configValidatorKey).(*config.ValidationContext)
	if !ok {
		panic("programming error: extracting config validator from context failed")
	}
	return vctx
}
