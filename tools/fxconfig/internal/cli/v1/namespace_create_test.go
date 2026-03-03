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

	policy := cmd.Flag("policy")
	require.NotNil(t, policy, "policy flag should exist")
}

type testApp struct {
	mock.Mock
}

func (*testApp) MergeTransactions(_ context.Context, _ []*applicationpb.Tx) (*applicationpb.Tx, error) {
	// TODO implement me
	panic("implement me")
}

func (t *testApp) DeployNamespace(
	ctx context.Context,
	input *app.DeployNamespaceInput,
) (*app.DeployNamespaceOutput, app.TxStatus, error) {
	args := t.Called(ctx, input)
	return nil, 0, args.Error(1)
}

func (t *testApp) ListNamespaces(ctx context.Context) ([]app.NamespaceQueryResult, error) {
	args := t.Called(ctx)
	return nil, args.Error(1)
}

func (*testApp) EndorseTransaction(_ context.Context, _ string, _ *applicationpb.Tx) (*applicationpb.Tx, error) {
	// TODO implement me
	panic("implement me")
}

func (*testApp) SubmitTransaction(_ context.Context, _ string, _ *applicationpb.Tx) error {
	// TODO implement me
	panic("implement me")
}

func (*testApp) SubmitTransactionWithWait(_ context.Context, _ string, _ *applicationpb.Tx) (app.TxStatus, error) {
	// TODO implement me
	panic("implement me")
}
