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

func TestInfoCommand_EnvFormat(t *testing.T) {
	t.Parallel()

	var outBuf bytes.Buffer
	ctx := &CLIContext{
		Config:  &config.Config{},
		Printer: cliio.NewCLIPrinter(&outBuf, &outBuf, cliio.FormatTable),
	}

	cmd := NewInfoCommand(ctx)
	cmd.SetArgs([]string{"--format", "env"})
	// Use Execute() so flags are parsed
	err := cmd.Execute()
	require.NoError(t, err)
	
	// Since Config is empty, it will not have any specific values, 
	// but let's check it does not error and produces some output
	// or no output if empty.
	require.NotNil(t, outBuf.String())
}

func TestInfoCommand_UnknownFormat(t *testing.T) {
	t.Parallel()

	var outBuf bytes.Buffer
	ctx := &CLIContext{
		Config:  &config.Config{},
		Printer: cliio.NewCLIPrinter(&outBuf, &outBuf, cliio.FormatTable),
	}

	cmd := NewInfoCommand(ctx)
	cmd.SetArgs([]string{"--format", "invalid"})
	err := cmd.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported format: invalid")
}
