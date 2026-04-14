/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/hyperledger/fabric-x-common/api/applicationpb"
	"github.com/hyperledger/fabric-x-common/api/committerpb"

	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/cli/v1/cliio"
)

func TestNewTxSubmitCommand(t *testing.T) {
	t.Parallel()

	cmd := newTxSubmitCommand(&CLIContext{App: &testApp{}})

	require.NotNil(t, cmd)
	require.Equal(t, "submit [file]", cmd.Use)
	require.NotEmpty(t, cmd.Short)
	require.NotNil(t, cmd.RunE)
	require.NotNil(t, cmd.Flags().Lookup("wait"))
}

func TestTxSubmitCommand_Submit(t *testing.T) {
	t.Parallel()

	txFile := writeTxFile(t, "tx-123", &applicationpb.Tx{})

	mockApp := &testApp{}
	mockApp.On("SubmitTransaction", mock.Anything, "tx-123", mock.AnythingOfType("*applicationpb.Tx")).
		Return(nil)

	ctx := &CLIContext{
		App:                mockApp,
		IOTransactionCodec: &cliio.JSONCodec{},
	}

	cmd := newTxSubmitCommand(ctx)
	cmd.SetArgs([]string{txFile})

	require.NoError(t, cmd.Execute())
	mockApp.AssertExpectations(t)
}

func TestTxSubmitCommand_SubmitWithWait(t *testing.T) {
	t.Parallel()

	txFile := writeTxFile(t, "tx-123", &applicationpb.Tx{})

	mockApp := &testApp{}
	mockApp.On("SubmitTransactionWithWait", mock.Anything, "tx-123", mock.AnythingOfType("*applicationpb.Tx")).
		Return(int(committerpb.Status_COMMITTED), nil)

	var outBuf bytes.Buffer
	ctx := &CLIContext{
		App:                mockApp,
		Printer:            cliio.NewCLIPrinter(&outBuf, &outBuf, cliio.FormatTable),
		IOTransactionCodec: &cliio.JSONCodec{},
	}

	cmd := newTxSubmitCommand(ctx)
	cmd.SetOut(&outBuf)
	cmd.SetArgs([]string{txFile, "--wait"})

	require.NoError(t, cmd.Execute())
	require.Contains(t, outBuf.String(), "Transaction status:")
	mockApp.AssertExpectations(t)
}

func TestTxSubmitCommand_SubmitWithWaitFailed(t *testing.T) {
	t.Parallel()

	txFile := writeTxFile(t, "tx-123", &applicationpb.Tx{})

	mockApp := &testApp{}
	mockApp.On("SubmitTransactionWithWait", mock.Anything, "tx-123", mock.AnythingOfType("*applicationpb.Tx")).
		Return(int(committerpb.Status_ABORTED_SIGNATURE_INVALID), nil)

	var outBuf bytes.Buffer
	ctx := &CLIContext{
		App:                mockApp,
		Printer:            cliio.NewCLIPrinter(&outBuf, &outBuf, cliio.FormatTable),
		IOTransactionCodec: &cliio.JSONCodec{},
	}

	cmd := newTxSubmitCommand(ctx)
	cmd.SetOut(&outBuf)
	cmd.SetArgs([]string{txFile, "--wait"})

	err := cmd.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), "transaction failed with status: ABORTED_SIGNATURE_INVALID")
	require.Contains(t, outBuf.String(), "Transaction status: ABORTED_SIGNATURE_INVALID")
	mockApp.AssertExpectations(t)
}

func TestTxSubmitCommand_AppError(t *testing.T) {
	t.Parallel()

	txFile := writeTxFile(t, "tx-123", &applicationpb.Tx{})

	mockApp := &testApp{}
	mockApp.On("SubmitTransaction", mock.Anything, "tx-123", mock.AnythingOfType("*applicationpb.Tx")).
		Return(errors.New("submit failed"))

	ctx := &CLIContext{
		App:                mockApp,
		IOTransactionCodec: &cliio.JSONCodec{},
	}

	cmd := newTxSubmitCommand(ctx)
	cmd.SetArgs([]string{txFile})

	err := cmd.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), "submit failed")
	mockApp.AssertExpectations(t)
}

func TestTxSubmitCommand_MissingArg(t *testing.T) {
	t.Parallel()

	cmd := newTxSubmitCommand(&CLIContext{App: &testApp{}})
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	require.Error(t, err)
}
