package v1

import (
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/app"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/cli/v1/io"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/config"
)

type CLIContext struct {
	Config             *config.Config
	Printer            io.Printer
	IOTransactionCodec io.Codec
	// Logger  logger.Logger

	App app.Application
}
