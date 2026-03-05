/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/hyperledger/fabric-x-common/api/applicationpb"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/adapters"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/config"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/provider"
)

type mockQueryClient struct {
	policies *applicationpb.NamespacePolicies
	err      error
}

func (m *mockQueryClient) GetNamespacePolicies(_ context.Context) (*applicationpb.NamespacePolicies, error) {
	return m.policies, m.err
}

func (*mockQueryClient) Close() error { return nil }

func makeQueryProvider(
	client adapters.QueryClient,
	err error,
) *provider.Provider[adapters.QueryClient, *config.QueriesConfig] {
	cfg := &config.QueriesConfig{
		EndpointServiceConfig: config.EndpointServiceConfig{
			Address:           "localhost:7050",
			ConnectionTimeout: 30 * time.Second,
		},
	}
	return provider.New(func(_ *config.QueriesConfig) (adapters.QueryClient, error) {
		return client, err
	}, cfg, fakeValidationContext())
}

func TestListNamespaces_Empty(t *testing.T) {
	t.Parallel()

	a := &AdminApp{
		QueryProvider: makeQueryProvider(&mockQueryClient{policies: &applicationpb.NamespacePolicies{}}, nil),
	}

	results, err := a.ListNamespaces(t.Context())
	require.NoError(t, err)
	require.Empty(t, results)
}

func TestListNamespaces_WithResults(t *testing.T) {
	t.Parallel()

	policies := &applicationpb.NamespacePolicies{
		Policies: []*applicationpb.PolicyItem{
			{Namespace: "Ns1", Version: 1, Policy: []byte("policy1")},
			{Namespace: "Ns2", Version: 2, Policy: []byte("policy2")},
		},
	}
	a := &AdminApp{
		QueryProvider: makeQueryProvider(&mockQueryClient{policies: policies}, nil),
	}

	results, err := a.ListNamespaces(t.Context())
	require.NoError(t, err)
	require.Len(t, results, 2)
	require.Equal(t, "Ns1", results[0].NsID)
	require.Equal(t, 1, results[0].Version)
	require.Equal(t, "Ns2", results[1].NsID)
	require.Equal(t, 2, results[1].Version)
}

func TestListNamespaces_QueryError(t *testing.T) {
	t.Parallel()

	a := &AdminApp{
		QueryProvider: makeQueryProvider(nil, errors.New("connection refused")),
	}

	_, err := a.ListNamespaces(t.Context())
	require.Error(t, err)
}
