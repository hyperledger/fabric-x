/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package util

import (
	"io"
	"os"

	"github.com/hyperledger/fabric-lib-go/common/flogging"
	"google.golang.org/grpc/grpclog"
)

// MustGetLogger creates a logger with the specified name. If an invalid name
// is provided, the operation will panic.
func MustGetLogger(loggerName string) *flogging.FabricLogger {
	ret := flogging.MustGetLogger(loggerName)
	// Due to package initialization order, the fabric-go-lib package that initialize the GRPC logger
	// might be initialized after our own package.
	// Since methods calls can happen only after package initialization,
	// This call ensures the GRPC logger will stay quiet.
	grpclog.SetLoggerV2(grpclog.NewLoggerV2(io.Discard, io.Discard, os.Stderr))
	return ret
}
