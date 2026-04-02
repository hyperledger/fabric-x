/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/hyperledger/fabric-x-common/api/applicationpb"

	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/cli/v1/cliio"
)

func TestNewTxMergeCommand(t *testing.T) {
	t.Parallel()

	cmd := newTxMergeCommand(&CLIContext{App: &testApp{}})

	require.NotNil(t, cmd)
	require.Equal(t, "merge [tx1.json] [tx2.json] [txN.json...]", cmd.Use)
	require.NotEmpty(t, cmd.Short)
	require.NotNil(t, cmd.RunE)
	require.NotNil(t, cmd.Flags().Lookup("output"))
}

// TestResolveInputs_SameTxID verifies that two files with the same txID are merged.
//
// NOTE: this test currently fails due to a bug in resolveInputs (tx_merge.go line ~91)
// where `txIDs[txID]` uses the outer variable (initially "") instead of `txIDs[id]`
// (the decoded ID), causing a spurious "" entry in the map for every merge call.
func TestResolveInputs_SameTxID(t *testing.T) {
	t.Parallel()

	file1 := writeTxFile(t, "tx-abc", &applicationpb.Tx{})
	file2 := writeTxFile(t, "tx-abc", &applicationpb.Tx{})

	ctx := &CLIContext{
		App:                &testApp{},
		IOTransactionCodec: &cliio.JSONCodec{},
	}

	txID, txs, err := resolveInputs(ctx, &cobra.Command{}, []string{file1, file2})
	require.NoError(t, err)
	require.Equal(t, "tx-abc", txID)
	require.Len(t, txs, 2)
}

func TestResolveInputs_DifferentTxIDs(t *testing.T) {
	t.Parallel()

	file1 := writeTxFile(t, "tx-1", &applicationpb.Tx{})
	file2 := writeTxFile(t, "tx-2", &applicationpb.Tx{})

	ctx := &CLIContext{
		App:                &testApp{},
		IOTransactionCodec: &cliio.JSONCodec{},
	}

	_, _, err := resolveInputs(ctx, &cobra.Command{}, []string{file1, file2})
	require.Error(t, err)
	require.Contains(t, err.Error(), "txID")
}

func TestResolveInputs_ThreeFilesSameTxID(t *testing.T) {
	t.Parallel()

	file1 := writeTxFile(t, "tx-xyz", &applicationpb.Tx{})
	file2 := writeTxFile(t, "tx-xyz", &applicationpb.Tx{})
	file3 := writeTxFile(t, "tx-xyz", &applicationpb.Tx{})

	ctx := &CLIContext{
		App:                &testApp{},
		IOTransactionCodec: &cliio.JSONCodec{},
	}

	txID, txs, err := resolveInputs(ctx, &cobra.Command{}, []string{file1, file2, file3})
	require.NoError(t, err)
	require.Equal(t, "tx-xyz", txID)
	require.Len(t, txs, 3)
}

func TestResolveInputs_NonExistentFile(t *testing.T) {
	t.Parallel()

	ctx := &CLIContext{
		App:                &testApp{},
		IOTransactionCodec: &cliio.JSONCodec{},
	}

	_, _, err := resolveInputs(ctx, &cobra.Command{}, []string{"/nonexistent/tx.json"})
	require.Error(t, err)
}

func TestTxMergeCommand(t *testing.T) {
	t.Parallel()

	file1 := writeTxFile(t, "tx-abc", &applicationpb.Tx{})
	file2 := writeTxFile(t, "tx-abc", &applicationpb.Tx{})

	mergedTx := &applicationpb.Tx{}
	mockApp := &testApp{}
	mockApp.On("MergeTransactions", mock.Anything, mock.Anything).Return(mergedTx, nil)

	var outBuf bytes.Buffer
	ctx := &CLIContext{
		App:                mockApp,
		Printer:            cliio.NewCLIPrinter(&outBuf, &outBuf, cliio.FormatTable),
		IOTransactionCodec: &cliio.JSONCodec{},
	}

	cmd := newTxMergeCommand(ctx)
	cmd.SetOut(&outBuf)
	cmd.SetArgs([]string{file1, file2})

	require.NoError(t, cmd.Execute())
	mockApp.AssertExpectations(t)
	require.Contains(t, outBuf.String(), "tx-abc")
}

func TestTxMergeCommand_RequiresMinTwoArgs(t *testing.T) {
	t.Parallel()

	cmd := newTxMergeCommand(&CLIContext{App: &testApp{}})
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	require.Error(t, err)
}
