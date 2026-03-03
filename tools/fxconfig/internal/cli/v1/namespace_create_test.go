/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package v1

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/hyperledger/fabric-x-common/api/applicationpb"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/app"
)

func TestNewCreateCommand(t *testing.T) {
	t.Parallel()

	// Execute
	cmd := newNsCreateCommand(&CLIContext{App: &testApp{}})

	// Assert
	require.NotNil(t, cmd, "newNsCreateCommand should return a non-nil command")
	require.Equal(t, "create [name]", cmd.Use, "command use should be 'create [name]'")
	require.NotEmpty(t, cmd.Short, "command should have a short description")
	require.NotNil(t, cmd.RunE, "command should have a RunE function")

	policyFlag := cmd.Flag("policy")
	require.NotNil(t, policyFlag, "policy flag should exist")
}

type testApp struct {
	mock.Mock
}

func (t *testApp) MergeTransactions(ctx context.Context, txs []*applicationpb.Tx) (*applicationpb.Tx, error) {
	// TODO implement me
	panic("implement me")
}

func (t *testApp) DeployNamespace(ctx context.Context, input *app.DeployNamespaceInput) (*app.DeployNamespaceOutput, app.TxStatus, error) {
	args := t.Called(ctx, input)
	return nil, 0, args.Error(1)
}

func (t *testApp) ListNamespaces(ctx context.Context) ([]app.NamespaceQueryResult, error) {
	args := t.Called(ctx)
	return nil, args.Error(1)
}

func (t *testApp) EndorseTransaction(ctx context.Context, txID string, tx *applicationpb.Tx) (*applicationpb.Tx, error) {
	// TODO implement me
	panic("implement me")
}

func (t *testApp) SubmitTransaction(ctx context.Context, txID string, tx *applicationpb.Tx) error {
	// TODO implement me
	panic("implement me")
}

func (t *testApp) SubmitTransactionWithWait(ctx context.Context, txID string, tx *applicationpb.Tx) (app.TxStatus, error) {
	// TODO implement me
	panic("implement me")
}
