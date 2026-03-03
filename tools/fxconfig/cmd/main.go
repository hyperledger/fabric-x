// Copyright IBM Corp. All Rights Reserved.
//
// SPDX-License-Identifier: Apache-2.0

// Package main provides the fxconfig CLI tool for managing Fabric-X namespaces.
// It supports creating, updating, and listing namespaces with flexible policy configurations.
package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/app"
	cli "github.com/hyperledger/fabric-x/tools/fxconfig/internal/cli/v1"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/client"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/config"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/msp"
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
				Validators:           vctx,
				MspProvider:          &msp.SignerProvider{ValidationContext: vctx, Cfg: cfg.MSP},
				QueryProvider:        &client.QueryProvider{ValidationContext: vctx, Cfg: cfg.Queries},
				OrdererProvider:      &client.OrdererProvider{ValidationContext: vctx, Cfg: cfg.Orderer},
				NotificationProvider: &client.NotificationProvider{ValidationContext: vctx, Cfg: cfg.Notifications},
			}, nil
		},
	)

	return cmd.ExecuteContext(ctx)
}
