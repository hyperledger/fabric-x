/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package logging initializes the default logging configuration for fxconfig.
// It sets up the Fabric logging framework with a standard format and error-level logging.
package logging

import "github.com/hyperledger/fabric-lib-go/common/flogging"

//nolint:revive,lll
const defaultFormat = "%{color}%{time:2006-01-02 15:04:05.000 MST} [%{module}] %{shortfunc} -> %{level:.4s} %{id:03x}%{color:reset} %{message}"

func init() {
	flogging.Init(flogging.Config{
		Format:  defaultFormat,
		LogSpec: "error",
	})
}
