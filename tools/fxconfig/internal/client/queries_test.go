/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/hyperledger/fabric-x-common/api/applicationpb"
	"github.com/hyperledger/fabric-x-common/api/committerpb"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/config"
)

// mockQueryServiceClient implements committerpb.QueryServiceClient for testing.
type mockQueryServiceClient struct {
	policies *applicationpb.NamespacePolicies
	err      error
}

func (m *mockQueryServiceClient) GetNamespacePolicies(
	_ context.Context,
	_ *emptypb.Empty,
	_ ...grpc.CallOption,
) (*applicationpb.NamespacePolicies, error) {
	return m.policies, m.err
}

func (*mockQueryServiceClient) GetRows(
	_ context.Context,
	_ *committerpb.Query,
	_ ...grpc.CallOption,
) (*committerpb.Rows, error) {
	return nil, nil
}

func (*mockQueryServiceClient) BeginView(
	_ context.Context,
	_ *committerpb.ViewParameters,
	_ ...grpc.CallOption,
) (*committerpb.View, error) {
	return nil, nil
}

func (*mockQueryServiceClient) EndView(
	_ context.Context,
	_ *committerpb.View,
	_ ...grpc.CallOption,
) (*emptypb.Empty, error) {
	return nil, nil
}

func (*mockQueryServiceClient) GetConfigTransaction(
	_ context.Context,
	_ *emptypb.Empty,
	_ ...grpc.CallOption,
) (*applicationpb.ConfigTransaction, error) {
	return nil, nil
}

func (*mockQueryServiceClient) GetTransactionStatus(
	_ context.Context,
	_ *committerpb.TxStatusQuery,
	_ ...grpc.CallOption,
) (*committerpb.TxStatusResponse, error) {
	return nil, nil
}

func newTestQueryClient(mock committerpb.QueryServiceClient) *QueryClient {
	return &QueryClient{
		cfg: config.QueriesConfig{
			EndpointServiceConfig: config.EndpointServiceConfig{
				ConnectionTimeout: time.Second,
			},
		},
		client: mock,
	}
}

func TestQueryClient_GetNamespacePolicies_NilClient(t *testing.T) {
	t.Parallel()

	qc := &QueryClient{cfg: config.QueriesConfig{}}
	_, err := qc.GetNamespacePolicies(t.Context())
	require.Error(t, err)
}

func TestQueryClient_GetNamespacePolicies_Error(t *testing.T) {
	t.Parallel()

	qc := newTestQueryClient(&mockQueryServiceClient{err: errors.New("rpc error")})
	_, err := qc.GetNamespacePolicies(t.Context())
	require.Error(t, err)
}

func TestQueryClient_GetNamespacePolicies_Success(t *testing.T) {
	t.Parallel()

	expected := &applicationpb.NamespacePolicies{}
	qc := newTestQueryClient(&mockQueryServiceClient{policies: expected})
	result, err := qc.GetNamespacePolicies(t.Context())
	require.NoError(t, err)
	require.Equal(t, expected, result)
}

func TestQueryClient_Close_CallsCloseFunc(t *testing.T) {
	t.Parallel()

	closed := false
	qc := &QueryClient{closeF: func() { closed = true }}
	require.NoError(t, qc.Close())
	require.True(t, closed)
}

func TestQueryClient_Close_NilFunc(t *testing.T) {
	t.Parallel()

	qc := &QueryClient{}
	require.NoError(t, qc.Close())
}
