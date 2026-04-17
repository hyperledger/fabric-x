// Copyright IBM Corp. All Rights Reserved.
//
// SPDX-License-Identifier: Apache-2.0

// Package main provides the fxconfig CLI tool for managing Fabric-X namespaces.
// It supports creating, updating, and listing namespaces with flexible policy configurations.
package main

import (
	"context"
	"errors"
	"os"
	"os/signal"

	fmsp "github.com/hyperledger/fabric-x-common/msp"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/adapters"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/app"
	cli "github.com/hyperledger/fabric-x/tools/fxconfig/internal/cli/v1"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/client"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/config"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/msp"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/provider"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/validation"
)

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	cliCtx := &cli.CLIContext{}

	// setup our root command
	cmd := cli.NewRootCommand(
		cliCtx,
		// inject an application builder that is invoked once we have loaded the configuration
		func(cfg *config.Config) (app.Application, error) {
			vctx := validation.NewValidationContext()
			return &app.AdminApp{
				Validators: vctx,
				MspProvider: provider.New[fmsp.SigningIdentity, *config.MSPConfig](
					func(cfg *config.MSPConfig) (fmsp.SigningIdentity, error) {
						return msp.GetSignerIdentityFromMSP(*cfg)
					},
					&cfg.MSP,
					vctx,
				),
				QueryProvider: provider.New[adapters.QueryClient, *config.QueriesConfig](
					func(cfg *config.QueriesConfig) (adapters.QueryClient, error) {
						return client.NewQueryClient(*cfg)
					},
					&cfg.Queries,
					vctx,
				),
				OrdererProvider: provider.New[adapters.OrdererClient, *config.OrdererConfig](
					func(cfg *config.OrdererConfig) (adapters.OrdererClient, error) {
						return client.NewOrdererClient(*cfg)
					},
					&cfg.Orderer,
					vctx,
				),
				NotificationProvider: provider.New[adapters.NotificationClient, *config.NotificationsConfig](
					func(cfg *config.NotificationsConfig) (adapters.NotificationClient, error) {
						return client.NewNotificationClient(*cfg)
					},
					&cfg.Notifications,
					vctx,
				),
			}, nil
		},
	)

	err := cmd.ExecuteContext(ctx)
	if cliCtx.App != nil {
		err = errors.Join(err, cliCtx.App.Close())
	}

	return err
}
