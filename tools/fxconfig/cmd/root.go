/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/config"
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

			// Store configuration in command context for access by subcommands
			ctx := context.WithValue(cmd.Context(), configKey, cfg)
			cmd.SetContext(ctx)

			return nil
		},
	}

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file")

	// Register all subcommands
	rootCmd.AddCommand(NewVersionCommand())
	rootCmd.AddCommand(NewInfoCommand())
	rootCmd.AddCommand(NewNamespaceCommand())
	return rootCmd
}
