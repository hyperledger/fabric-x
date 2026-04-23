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
	"google.golang.org/protobuf/proto"

	"github.com/hyperledger/fabric-x-common/api/applicationpb"
	"github.com/hyperledger/fabric-x-common/common/policydsl"
	"github.com/hyperledger/fabric-x-common/protoutil"
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

func makeMspPolicyBytes(t *testing.T, expr string) []byte {
	t.Helper()
	env, err := policydsl.FromString(expr)
	require.NoError(t, err)
	nsPolicy := &applicationpb.NamespacePolicy{
		Rule: &applicationpb.NamespacePolicy_MspRule{
			MspRule: protoutil.MarshalOrPanic(env),
		},
	}
	b, err := proto.Marshal(nsPolicy)
	require.NoError(t, err)
	return b
}

func TestParsePolicy_MspOR(t *testing.T) {
	t.Parallel()

	b := makeMspPolicyBytes(t, "OR('Org1MSP.member', 'Org2MSP.member')")
	result, err := parsePolicy(b)
	require.NoError(t, err)
	require.Equal(t, "OR('Org1MSP.member', 'Org2MSP.member')", result)
}

func TestParsePolicy_MspAND(t *testing.T) {
	t.Parallel()

	b := makeMspPolicyBytes(t, "AND('Org1MSP.admin', 'Org2MSP.admin')")
	result, err := parsePolicy(b)
	require.NoError(t, err)
	require.Equal(t, "AND('Org1MSP.admin', 'Org2MSP.admin')", result)
}

func TestParsePolicy_MspOutOfN(t *testing.T) {
	t.Parallel()

	// OutOf(2, ...) — 2-of-3
	env, err := policydsl.FromString("OutOf(2, 'Org1MSP.member', 'Org2MSP.member', 'Org3MSP.member')")
	require.NoError(t, err)
	nsPolicy := &applicationpb.NamespacePolicy{
		Rule: &applicationpb.NamespacePolicy_MspRule{
			MspRule: protoutil.MarshalOrPanic(env),
		},
	}
	b, err := proto.Marshal(nsPolicy)
	require.NoError(t, err)

	result, err := parsePolicy(b)
	require.NoError(t, err)
	require.Contains(t, result, "OutOf(2,")
}

func TestParsePolicy_Threshold(t *testing.T) {
	t.Parallel()

	nsPolicy := &applicationpb.NamespacePolicy{
		Rule: &applicationpb.NamespacePolicy_ThresholdRule{
			ThresholdRule: &applicationpb.ThresholdRule{
				Scheme:    "ECDSA",
				PublicKey: []byte("fakepublickey"),
			},
		},
	}
	b, err := proto.Marshal(nsPolicy)
	require.NoError(t, err)

	result, err := parsePolicy(b)
	require.NoError(t, err)
	require.Contains(t, result, "Threshold(ECDSA,")
}

func TestParsePolicy_InvalidBytes(t *testing.T) {
	t.Parallel()

	_, err := parsePolicy([]byte("not a proto"))
	require.Error(t, err)
}

func TestListNamespaces_PolicyStrPopulated(t *testing.T) {
	t.Parallel()

	policyBytes := makeMspPolicyBytes(t, "OR('Org1MSP.member', 'Org2MSP.member')")
	policies := &applicationpb.NamespacePolicies{
		Policies: []*applicationpb.PolicyItem{
			{Namespace: "ns1", Version: 1, Policy: policyBytes},
		},
	}
	a := &AdminApp{
		QueryProvider: makeQueryProvider(&mockQueryClient{policies: policies}, nil),
	}

	results, err := a.ListNamespaces(t.Context())
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, "OR('Org1MSP.member', 'Org2MSP.member')", results[0].PolicyStr)
}

func TestListNamespaces_PolicyStrFallbackOnBadBytes(t *testing.T) {
	t.Parallel()

	policies := &applicationpb.NamespacePolicies{
		Policies: []*applicationpb.PolicyItem{
			{Namespace: "ns1", Version: 1, Policy: []byte("bad")},
		},
	}
	a := &AdminApp{
		QueryProvider: makeQueryProvider(&mockQueryClient{policies: policies}, nil),
	}

	results, err := a.ListNamespaces(t.Context())
	require.NoError(t, err)
	require.Len(t, results, 1)
	// falls back to hex
	require.Equal(t, "626164", results[0].PolicyStr)
}
