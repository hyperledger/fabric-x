// Copyright IBM Corp. All Rights Reserved.
//
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/app"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/cli/v1/cliio"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/config"
)

// CLIContext holds shared dependencies for CLI commands.
// It provides access to configuration, output formatting, transaction encoding,
// and the application layer for executing operations.
type CLIContext struct {
	Config             *config.Config
	Printer            cliio.Printer
	IOTransactionCodec cliio.Codec
	// Logger  logger.Logger

	App app.Application
}
