package v1

import (
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/app"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/cli/v1/io"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/config"
)

// CLIContext holds shared dependencies for CLI commands.
// It provides access to configuration, output formatting, transaction encoding,
// and the application layer for executing operations.
type CLIContext struct {
	Config             *config.Config
	Printer            io.Printer
	IOTransactionCodec io.Codec
	// Logger  logger.Logger

	App app.Application
}
