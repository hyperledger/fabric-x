/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/hyperledger/fabric-x-common/api/applicationpb"

	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/cli/v1/cliio"
)

func TestNewTxEndorseCommand(t *testing.T) {
	t.Parallel()

	cmd := newTxEndorseCommand(&CLIContext{App: &testApp{}})

	require.NotNil(t, cmd)
	require.Equal(t, "endorse [file]", cmd.Use)
	require.NotEmpty(t, cmd.Short)
	require.NotNil(t, cmd.RunE)
	require.NotNil(t, cmd.Flags().Lookup("output"))
}

func TestTxEndorseCommand_Success(t *testing.T) {
	t.Parallel()

	txFile := writeTxFile(t, "test-tx-id", &applicationpb.Tx{})

	mockApp := &testApp{}
	mockApp.On("EndorseTransaction", mock.Anything, "test-tx-id", mock.AnythingOfType("*applicationpb.Tx")).
		Return(&applicationpb.Tx{}, nil)

	var outBuf bytes.Buffer
	ctx := &CLIContext{
		App:                mockApp,
		Printer:            cliio.NewCLIPrinter(&outBuf, &outBuf, cliio.FormatTable),
		IOTransactionCodec: &cliio.JSONCodec{},
	}

	cmd := newTxEndorseCommand(ctx)
	cmd.SetOut(&outBuf)
	cmd.SetArgs([]string{txFile})

	require.NoError(t, cmd.Execute())
	mockApp.AssertExpectations(t)
	require.NotEmpty(t, outBuf.String())
}

func TestTxEndorseCommand_AppError(t *testing.T) {
	t.Parallel()

	txFile := writeTxFile(t, "test-tx-id", &applicationpb.Tx{})

	mockApp := &testApp{}
	mockApp.On("EndorseTransaction", mock.Anything, "test-tx-id", mock.AnythingOfType("*applicationpb.Tx")).
		Return(nil, errors.New("endorse failed"))

	ctx := &CLIContext{
		App:                mockApp,
		IOTransactionCodec: &cliio.JSONCodec{},
	}

	cmd := newTxEndorseCommand(ctx)
	cmd.SetArgs([]string{txFile})

	err := cmd.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), "endorse failed")
	mockApp.AssertExpectations(t)
}

func TestTxEndorseCommand_MissingArg(t *testing.T) {
	t.Parallel()

	cmd := newTxEndorseCommand(&CLIContext{App: &testApp{}})
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	require.Error(t, err)
}

// writeTxFile encodes tx as JSON and writes it to a temp file, returning its path.
// Shared by tx_endorse_test.go, tx_merge_test.go, and tx_submit_test.go.
func writeTxFile(t *testing.T, txID string, tx *applicationpb.Tx) string {
	t.Helper()

	data, err := (&cliio.JSONCodec{}).Encode(txID, tx)
	require.NoError(t, err)

	path := filepath.Join(t.TempDir(), "tx.json")
	require.NoError(t, os.WriteFile(path, data, 0o600))

	return path
}
