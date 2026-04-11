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

	flag := cmd.Flags().Lookup("format")
	require.NotNil(t, flag)                 // ensure the --format flag is defined
	require.Equal(t, "yaml", flag.DefValue) // default should be "yaml"
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

func TestInfoCommand_PrintsEnvVars(t *testing.T) {
	t.Parallel()

	var outBuf bytes.Buffer
	ctx := &CLIContext{
		Config: &config.Config{
			Logging: config.LoggingConfig{Level: "info"},
			TLS:     config.TLSConfig{ClientKeyPath: "test-client-key-path"},
		},
		Printer: cliio.NewCLIPrinter(&outBuf, &outBuf, cliio.FormatTable),
	}

	cmd := NewInfoCommand(ctx)
	cmd.SetArgs([]string{"--format", "env"})

	err := cmd.Execute()
	require.NoError(t, err)
	require.Equal(t, "env", cmd.Flag("format").Value.String()) // ensure --format env is applied

	output := outBuf.String()
	require.NotEmpty(t, output)

	require.Contains(t, output, "FXCONFIG_LOGGING_LEVEL=info")
	require.Contains(t, output, "FXCONFIG_TLS_CLIENTKEY=test-client-key-path")
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
	require.Equal(t, "yaml", cmd.Flag("format").Value.String()) // ensure --format yaml is default
	require.Contains(t, outBuf.String(), "null")

	outBuf.Reset()
	cmd.SetArgs([]string{"--format", "env"})
	err = cmd.Execute()
	require.NoError(t, err)
	require.Equal(t, "env", cmd.Flag("format").Value.String()) // ensure --format env is applied
	require.Contains(t, outBuf.String(), "null")
}
