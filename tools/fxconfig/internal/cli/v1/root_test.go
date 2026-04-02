/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/app"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/config"
)

func TestNewRootCommand(t *testing.T) {
	t.Parallel()

	ctx := &CLIContext{}
	rootCmd := NewRootCommand(ctx, func(_ *config.Config) (app.Application, error) {
		return &testApp{}, nil
	})

	require.NotNil(t, rootCmd)
	require.Equal(t, "fxconfig", rootCmd.Use)
	require.NotEmpty(t, rootCmd.Short)

	// --config flag must be registered
	require.NotNil(t, rootCmd.PersistentFlags().Lookup("config"))

	// all top-level subcommands must be present
	subCmds := make(map[string]bool)
	for _, sub := range rootCmd.Commands() {
		subCmds[sub.Name()] = true
	}
	require.True(t, subCmds["version"])
	require.True(t, subCmds["info"])
	require.True(t, subCmds["namespace"])
	require.True(t, subCmds["tx"])
}

const minimalConfig = `
msp:
  localMspID: TestMSP
`

func TestPersistentPreRunE_ViaConfigFlag(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(minimalConfig), 0o600))

	cliCtx := &CLIContext{}
	rootCmd := NewRootCommand(cliCtx, func(_ *config.Config) (app.Application, error) {
		return &testApp{}, nil
	})
	rootCmd.SetArgs([]string{"--config", configPath, "version"})

	require.NoError(t, rootCmd.Execute())

	require.NotNil(t, cliCtx.Config)
	require.NotNil(t, cliCtx.Printer)
	require.NotNil(t, cliCtx.IOTransactionCodec)
	require.NotNil(t, cliCtx.App)
	require.Equal(t, "TestMSP", cliCtx.Config.MSP.LocalMspID)
}

func TestPersistentPreRunE_ViaProjectConfig(t *testing.T) { //nolint:paralleltest
	// Not parallel: changes working directory.

	tmpDir := t.TempDir()
	projectConfigDir := filepath.Join(tmpDir, ".fxconfig")
	require.NoError(t, os.MkdirAll(projectConfigDir, 0o750))
	configPath := filepath.Join(projectConfigDir, "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(minimalConfig), 0o600))

	t.Chdir(tmpDir)

	cliCtx := &CLIContext{}
	rootCmd := NewRootCommand(cliCtx, func(_ *config.Config) (app.Application, error) {
		return &testApp{}, nil
	})
	rootCmd.SetArgs([]string{"version"})

	require.NoError(t, rootCmd.Execute())

	require.NotNil(t, cliCtx.Config)
	require.NotNil(t, cliCtx.Printer)
	require.NotNil(t, cliCtx.IOTransactionCodec)
	require.NotNil(t, cliCtx.App)
	require.Equal(t, "TestMSP", cliCtx.Config.MSP.LocalMspID)
}
