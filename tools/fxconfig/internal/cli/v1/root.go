/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package v1 implements the command-line interface for fxconfig.
// It provides commands for namespace management and configuration handling.
package v1

import (
	"github.com/spf13/cobra"

	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/app"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/cli/v1/io"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/config"
)

// NewRootCommand constructs and returns the root command for fxconfig.
// It sets up configuration loading, flag registration, and all subcommands.
// Configuration is loaded in PersistentPreRunE.
func NewRootCommand(cliCtx *CLIContext, buildApp func(cfg *config.Config) (app.Application, error)) *cobra.Command {
	// cli flags
	var cfgFile string
	rootCmd := &cobra.Command{
		Use:   "fxconfig",
		Short: "Yet another admin tool for Fabric-X",
		Long: `fxconfig is a CLI tool for managing Fabric-X namespaces.
This application enables creating, updating, and querying namespaces with flexible endorsement policies.`,
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

			// set our config and printer in our context
			cliCtx.Config = cfg
			cliCtx.Printer = io.NewCLIPrinter(cmd.OutOrStdout(), cmd.ErrOrStderr(), io.FormatTable)

			// output coded
			cliCtx.IOTransactionCodec = &io.JsonCodec{}

			// set application in context
			cliCtx.App, err = buildApp(cfg)
			if err != nil {
				return err
			}

			return nil
		},
	}

	// config parameter
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.fxconfig/config.yaml)")

	// Register all subcommands
	rootCmd.AddCommand(NewVersionCommand())
	rootCmd.AddCommand(NewInfoCommand(cliCtx))
	rootCmd.AddCommand(NewNsRootCommand(cliCtx))
	rootCmd.AddCommand(NewTxRootCommand(cliCtx))

	rootCmd.SilenceUsage = true

	return rootCmd
}
