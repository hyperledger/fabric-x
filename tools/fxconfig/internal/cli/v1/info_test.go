/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/cli/v1/cliio"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/config"
)

func TestNewInfoCommand(t *testing.T) {
	t.Parallel()

	cmd := NewInfoCommand(&CLIContext{})

	require.NotNil(t, cmd)
	require.Equal(t, "info", cmd.Use)
	require.NotEmpty(t, cmd.Short)
	require.NotNil(t, cmd.RunE)
}

func TestInfoCommand_PrintsYAMLConfig(t *testing.T) {
	t.Parallel()

	var outBuf bytes.Buffer
	ctx := &CLIContext{
		Config:  &config.Config{},
		Printer: cliio.NewCLIPrinter(&outBuf, &outBuf, cliio.FormatTable),
	}

	cmd := NewInfoCommand(ctx)
	err := cmd.RunE(cmd, nil)
	require.NoError(t, err)
	require.NotEmpty(t, outBuf.String())
}

func TestInfoCommand_NilConfigPrintsNull(t *testing.T) {
	t.Parallel()

	var outBuf bytes.Buffer
	ctx := &CLIContext{
		Config:  nil,
		Printer: cliio.NewCLIPrinter(&outBuf, &outBuf, cliio.FormatTable),
	}

	cmd := NewInfoCommand(ctx)
	err := cmd.RunE(cmd, nil)
	require.NoError(t, err)
	require.Contains(t, outBuf.String(), "null")
}
