/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/hyperledger/fabric-x-common/api/applicationpb"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/app"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/cli/v1/cliio"
)

const testNamespace = "my-namespace"

func TestNewCreateCommand(t *testing.T) {
	t.Parallel()

	cmd := newNsCreateCommand(&CLIContext{App: &testApp{}})

	require.NotNil(t, cmd, "newNsCreateCommand should return a non-nil command")
	require.Equal(t, "create [name]", cmd.Use, "command use should be 'create [name]'")
	require.NotEmpty(t, cmd.Short, "command should have a short description")
	require.NotNil(t, cmd.RunE, "command should have a RunE function")

	policy := cmd.Flag("policy")
	require.NotNil(t, policy, "policy flag should exist")
}

func TestNewCreateCommandRun_TxReturned(t *testing.T) {
	t.Parallel()

	mockApp := &testApp{}
	deployOut := &app.DeployNamespaceOutput{
		TxID: "tx-123",
		Tx:   &applicationpb.Tx{},
	}
	mockApp.On("DeployNamespace", mock.Anything, mock.Anything).Return(deployOut, app.UnknownStatus, nil)

	var printerOut, printerErr bytes.Buffer
	printer := cliio.NewCLIPrinter(&printerOut, &printerErr, cliio.FormatTable)
	cmd := newNsCreateCommand(&CLIContext{
		App:                mockApp,
		Printer:            printer,
		IOTransactionCodec: &cliio.JSONCodec{},
	})

	var cmdOut bytes.Buffer
	cmd.SetOut(&cmdOut)
	require.NoError(t, cmd.Flags().Set("policy", "OR('Org1MSP.member')"))

	err := cmd.RunE(cmd, []string{testNamespace})

	require.NoError(t, err)
	require.Contains(t, cmdOut.String(), "tx-123")
	mockApp.AssertExpectations(t)
}

func TestNewCreateCommandRun_NoTx(t *testing.T) {
	t.Parallel()

	mockApp := &testApp{}
	mockApp.On("DeployNamespace", mock.Anything, mock.Anything).Return(nil, app.UnknownStatus, nil)

	var printerOut, printerErr bytes.Buffer
	printer := cliio.NewCLIPrinter(&printerOut, &printerErr, cliio.FormatTable)
	cmd := newNsCreateCommand(&CLIContext{
		App:     mockApp,
		Printer: printer,
	})
	require.NoError(t, cmd.Flags().Set("policy", "OR('Org1MSP.member')"))

	err := cmd.RunE(cmd, []string{testNamespace})

	require.NoError(t, err)
	require.Contains(t, printerOut.String(), "Transaction status: STATUS_UNSPECIFIED")
	mockApp.AssertExpectations(t)
}

type testApp struct {
	mock.Mock
}

func (t *testApp) MergeTransactions(ctx context.Context, txs []*applicationpb.Tx) (*applicationpb.Tx, error) {
	args := t.Called(ctx, txs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*applicationpb.Tx), args.Error(1) //nolint:errcheck,revive,forcetypeassert
}

func (t *testApp) DeployNamespace(
	ctx context.Context,
	input *app.DeployNamespaceInput,
) (*app.DeployNamespaceOutput, app.TxStatus, error) {
	args := t.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).(*app.DeployNamespaceOutput), args.Int(1), args.Error(2) //nolint:errcheck,revive,forcetypeassert
}

func (t *testApp) ListNamespaces(ctx context.Context) ([]app.NamespaceQueryResult, error) {
	args := t.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]app.NamespaceQueryResult), args.Error(1) //nolint:errcheck,revive,forcetypeassert
}

func (t *testApp) EndorseTransaction(
	ctx context.Context,
	txID string,
	tx *applicationpb.Tx,
) (*applicationpb.Tx, error) {
	args := t.Called(ctx, txID, tx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*applicationpb.Tx), args.Error(1) //nolint:errcheck,revive,forcetypeassert
}

func (t *testApp) SubmitTransaction(ctx context.Context, txID string, tx *applicationpb.Tx) error {
	args := t.Called(ctx, txID, tx)
	return args.Error(0)
}

func (t *testApp) SubmitTransactionWithWait(
	ctx context.Context,
	txID string,
	tx *applicationpb.Tx,
) (app.TxStatus, error) {
	args := t.Called(ctx, txID, tx)
	return args.Int(0), args.Error(1)
}