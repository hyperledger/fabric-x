/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/hyperledger/fabric-x-common/api/applicationpb"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/app"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/cli/v1/cliio"
)

func TestNewUpdateCommand(t *testing.T) {
	t.Parallel()

	// Execute
	cmd := newNsUpdateCommand(&CLIContext{App: &testApp{}})

	// Assert
	require.NotNil(t, cmd, "newNsUpdateCommand should return a non-nil command")
	require.Equal(t, "update [name]", cmd.Use, "command use should be 'update [name]'")
	require.NotEmpty(t, cmd.Short, "command should have a short description")
	require.NotNil(t, cmd.RunE, "command should have a RunE function")

	// Verify command-specific required flags
	version := cmd.Flag("version")
	require.NotNil(t, version, "version flag should exist")

	policy := cmd.Flag("policy")
	require.NotNil(t, policy, "policy flag should exist")
}

func TestNsUpdateCommandRun_TxReturned(t *testing.T) {
	t.Parallel()

	mockApp := &testApp{}
	deployOut := &app.DeployNamespaceOutput{
		TxID: "tx-456",
		Tx:   &applicationpb.Tx{},
	}
	mockApp.On("DeployNamespace", mock.Anything, mock.Anything).Return(deployOut, app.UnknownStatus, nil)

	var outBuf bytes.Buffer
	cmd := newNsUpdateCommand(&CLIContext{
		App:                mockApp,
		Printer:            cliio.NewCLIPrinter(&outBuf, &outBuf, cliio.FormatTable),
		IOTransactionCodec: &cliio.JSONCodec{},
	})
	cmd.SetOut(&outBuf)
	require.NoError(t, cmd.Flags().Set("policy", "OR('Org1MSP.member')"))
	require.NoError(t, cmd.Flags().Set("version", "1"))

	err := cmd.RunE(cmd, []string{"my-namespace"})

	require.NoError(t, err)
	require.Contains(t, outBuf.String(), "tx-456")
	mockApp.AssertExpectations(t)
}

func TestNsUpdateCommandRun_NoTx(t *testing.T) {
	t.Parallel()

	mockApp := &testApp{}
	mockApp.On("DeployNamespace", mock.Anything, mock.Anything).Return(nil, app.UnknownStatus, nil)

	var printerOut bytes.Buffer
	cmd := newNsUpdateCommand(&CLIContext{
		App:     mockApp,
		Printer: cliio.NewCLIPrinter(&printerOut, &printerOut, cliio.FormatTable),
	})
	require.NoError(t, cmd.Flags().Set("policy", "OR('Org1MSP.member')"))
	require.NoError(t, cmd.Flags().Set("version", "0"))

	err := cmd.RunE(cmd, []string{"my-namespace"})

	require.NoError(t, err)
	require.Contains(t, printerOut.String(), "Transaction status: STATUS_UNSPECIFIED")
	mockApp.AssertExpectations(t)
}
