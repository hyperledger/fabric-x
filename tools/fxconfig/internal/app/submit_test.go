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
	"github.com/hyperledger/fabric-x-common/msp"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/adapters"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/config"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/provider"
)

type mockOrdererClient struct {
	broadcastErr error
	closed       bool
}

func (m *mockOrdererClient) Broadcast(_ context.Context, _ msp.SigningIdentity, _ string, _ *applicationpb.Tx) error {
	if m.closed {
		return errors.New("orderer client closed")
	}
	return m.broadcastErr
}

func (m *mockOrdererClient) Close() error {
	m.closed = true
	return nil
}

type mockNotificationClient struct {
	subscribeErr error
	waitErr      error
	status       int
	closed       bool
}

func (m *mockNotificationClient) Subscribe(_ context.Context, _ string) (chan int, error) {
	if m.closed {
		return nil, errors.New("notification client closed")
	}
	if m.subscribeErr != nil {
		return nil, m.subscribeErr
	}
	ch := make(chan int, 1)
	ch <- m.status
	return ch, nil
}

func (m *mockNotificationClient) WaitForEvent(_ context.Context, ch chan int) (int, error) {
	if m.closed {
		return 0, errors.New("notification client closed")
	}
	if m.waitErr != nil {
		return 0, m.waitErr
	}
	return <-ch, nil
}

func (m *mockNotificationClient) Close() error {
	m.closed = true
	return nil
}

func makeOrdererProvider(
	client adapters.OrdererClient,
	err error,
) *provider.Provider[adapters.OrdererClient, *config.OrdererConfig] {
	cfg := &config.OrdererConfig{
		EndpointServiceConfig: config.EndpointServiceConfig{
			Address:           "localhost:7050",
			ConnectionTimeout: 30 * time.Second,
		},
		Channel: "mychannel",
	}
	return provider.New(func(_ *config.OrdererConfig) (adapters.OrdererClient, error) {
		return client, err
	}, cfg, fakeValidationContext())
}

func makeManagedOrdererProvider(
	factory func(*config.OrdererConfig) (adapters.OrdererClient, error),
) *provider.Provider[adapters.OrdererClient, *config.OrdererConfig] {
	cfg := &config.OrdererConfig{
		EndpointServiceConfig: config.EndpointServiceConfig{
			Address:           "localhost:7050",
			ConnectionTimeout: 30 * time.Second,
		},
		Channel: "mychannel",
	}
	return provider.New(factory, cfg, fakeValidationContext())
}

func makeNotificationProvider(
	client adapters.NotificationClient,
	err error,
) *provider.Provider[adapters.NotificationClient, *config.NotificationsConfig] {
	cfg := &config.NotificationsConfig{
		EndpointServiceConfig: config.EndpointServiceConfig{
			Address:           "localhost:9000",
			ConnectionTimeout: 30 * time.Second,
		},
	}
	return provider.New(func(_ *config.NotificationsConfig) (adapters.NotificationClient, error) {
		return client, err
	}, cfg, fakeValidationContext())
}

func makeManagedNotificationProvider(
	factory func(*config.NotificationsConfig) (adapters.NotificationClient, error),
) *provider.Provider[adapters.NotificationClient, *config.NotificationsConfig] {
	cfg := &config.NotificationsConfig{
		EndpointServiceConfig: config.EndpointServiceConfig{
			Address:           "localhost:9000",
			ConnectionTimeout: 30 * time.Second,
		},
	}
	return provider.New(factory, cfg, fakeValidationContext())
}

func someTx() *applicationpb.Tx {
	return &applicationpb.Tx{Namespaces: []*applicationpb.TxNamespace{{NsId: "ns1"}}}
}

// SubmitTransaction tests

func TestSubmitTransaction_MspProviderError(t *testing.T) {
	t.Parallel()

	a := &AdminApp{
		MspProvider:     makeMSPProvider(nil, errors.New("msp unavailable")),
		OrdererProvider: makeOrdererProvider(&mockOrdererClient{}, nil),
	}

	err := a.SubmitTransaction(t.Context(), "tx-1", someTx())
	require.Error(t, err)
}

func TestSubmitTransaction_OrdererProviderError(t *testing.T) {
	t.Parallel()

	a := &AdminApp{
		MspProvider:     makeMSPProvider(&testSigningIdentity{}, nil),
		OrdererProvider: makeOrdererProvider(nil, errors.New("orderer unavailable")),
	}

	err := a.SubmitTransaction(t.Context(), "tx-1", someTx())
	require.Error(t, err)
}

func TestSubmitTransaction_BroadcastError(t *testing.T) {
	t.Parallel()

	a := &AdminApp{
		MspProvider:     makeMSPProvider(&testSigningIdentity{}, nil),
		OrdererProvider: makeOrdererProvider(&mockOrdererClient{broadcastErr: errors.New("broadcast failed")}, nil),
	}

	err := a.SubmitTransaction(t.Context(), "tx-1", someTx())
	require.Error(t, err)
}

func TestSubmitTransaction_Success(t *testing.T) {
	t.Parallel()

	a := &AdminApp{
		MspProvider:     makeMSPProvider(&testSigningIdentity{}, nil),
		OrdererProvider: makeOrdererProvider(&mockOrdererClient{}, nil),
	}

	err := a.SubmitTransaction(t.Context(), "tx-1", someTx())
	require.NoError(t, err)
}

func TestSubmitTransaction_ContextCancelled(t *testing.T) {
	t.Parallel()

	a := &AdminApp{
		MspProvider:     makeMSPProvider(&testSigningIdentity{}, nil),
		OrdererProvider: makeOrdererProvider(&mockOrdererClient{broadcastErr: context.Canceled}, nil),
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := a.SubmitTransaction(ctx, "tx-1", someTx())
	require.Error(t, err)
	require.ErrorIs(t, err, context.Canceled)
}

// SubmitTransactionWithWait tests

func TestSubmitTransactionWithWait_MspProviderError(t *testing.T) {
	t.Parallel()

	a := &AdminApp{
		MspProvider:          makeMSPProvider(nil, errors.New("msp unavailable")),
		OrdererProvider:      makeOrdererProvider(&mockOrdererClient{}, nil),
		NotificationProvider: makeNotificationProvider(&mockNotificationClient{}, nil),
	}

	_, err := a.SubmitTransactionWithWait(t.Context(), "tx-1", someTx())
	require.Error(t, err)
}

func TestSubmitTransactionWithWait_NotificationProviderError(t *testing.T) {
	t.Parallel()

	a := &AdminApp{
		MspProvider:          makeMSPProvider(&testSigningIdentity{}, nil),
		OrdererProvider:      makeOrdererProvider(&mockOrdererClient{}, nil),
		NotificationProvider: makeNotificationProvider(nil, errors.New("notification unavailable")),
	}

	status, err := a.SubmitTransactionWithWait(t.Context(), "tx-1", someTx())
	require.Error(t, err)
	require.Equal(t, UnknownStatus, status)
}

func TestSubmitTransactionWithWait_SubscribeError(t *testing.T) {
	t.Parallel()

	a := &AdminApp{
		MspProvider:     makeMSPProvider(&testSigningIdentity{}, nil),
		OrdererProvider: makeOrdererProvider(&mockOrdererClient{}, nil),
		NotificationProvider: makeNotificationProvider(
			&mockNotificationClient{subscribeErr: errors.New("subscribe failed")}, nil,
		),
	}

	status, err := a.SubmitTransactionWithWait(t.Context(), "tx-1", someTx())
	require.Error(t, err)
	require.Equal(t, UnknownStatus, status)
}

func TestSubmitTransactionWithWait_BroadcastError(t *testing.T) {
	t.Parallel()

	a := &AdminApp{
		MspProvider: makeMSPProvider(&testSigningIdentity{}, nil),
		OrdererProvider: makeOrdererProvider(
			&mockOrdererClient{broadcastErr: errors.New("broadcast failed")}, nil,
		),
		NotificationProvider: makeNotificationProvider(&mockNotificationClient{}, nil),
	}

	status, err := a.SubmitTransactionWithWait(t.Context(), "tx-1", someTx())
	require.Error(t, err)
	require.Equal(t, UnknownStatus, status)
}

func TestSubmitTransactionWithWait_WaitForEventError(t *testing.T) {
	t.Parallel()

	a := &AdminApp{
		MspProvider:     makeMSPProvider(&testSigningIdentity{}, nil),
		OrdererProvider: makeOrdererProvider(&mockOrdererClient{}, nil),
		NotificationProvider: makeNotificationProvider(
			&mockNotificationClient{waitErr: errors.New("wait failed")}, nil,
		),
	}

	status, err := a.SubmitTransactionWithWait(t.Context(), "tx-1", someTx())
	require.Error(t, err)
	require.Equal(t, UnknownStatus, status)
}

func TestSubmitTransactionWithWait_Success(t *testing.T) {
	t.Parallel()

	const expectedStatus = 42

	a := &AdminApp{
		MspProvider:          makeMSPProvider(&testSigningIdentity{}, nil),
		OrdererProvider:      makeOrdererProvider(&mockOrdererClient{}, nil),
		NotificationProvider: makeNotificationProvider(&mockNotificationClient{status: expectedStatus}, nil),
	}

	status, err := a.SubmitTransactionWithWait(t.Context(), "tx-1", someTx())
	require.NoError(t, err)
	require.Equal(t, expectedStatus, status)
}

func TestSubmitTransaction_ReusesProviderManagedOrdererClient(t *testing.T) {
	t.Parallel()

	client := &mockOrdererClient{}
	a := &AdminApp{
		MspProvider: makeMSPProvider(&testSigningIdentity{}, nil),
		OrdererProvider: makeManagedOrdererProvider(func(_ *config.OrdererConfig) (adapters.OrdererClient, error) {
			return client, nil
		}),
	}

	err := a.SubmitTransaction(t.Context(), "tx-1", someTx())
	require.NoError(t, err)

	err = a.SubmitTransaction(t.Context(), "tx-2", someTx())
	require.NoError(t, err)
	require.False(t, client.closed)
}

func TestSubmitTransactionWithWait_ReusesProviderManagedClients(t *testing.T) {
	t.Parallel()

	ordererClient := &mockOrdererClient{}
	notificationClient := &mockNotificationClient{status: 7}

	a := &AdminApp{
		MspProvider: makeMSPProvider(&testSigningIdentity{}, nil),
		OrdererProvider: makeManagedOrdererProvider(func(_ *config.OrdererConfig) (adapters.OrdererClient, error) {
			return ordererClient, nil
		}),
		NotificationProvider: makeManagedNotificationProvider(
			func(_ *config.NotificationsConfig) (adapters.NotificationClient, error) {
				return notificationClient, nil
			},
		),
	}

	status, err := a.SubmitTransactionWithWait(t.Context(), "tx-1", someTx())
	require.NoError(t, err)
	require.Equal(t, 7, status)

	status, err = a.SubmitTransactionWithWait(t.Context(), "tx-2", someTx())
	require.NoError(t, err)
	require.Equal(t, 7, status)
	require.False(t, ordererClient.closed)
	require.False(t, notificationClient.closed)
}

func TestAdminApp_Close_ClosesProviderManagedClients(t *testing.T) {
	t.Parallel()

	ordererClient := &mockOrdererClient{}
	notificationClient := &mockNotificationClient{}
	queryClient := &mockQueryClient{policies: &applicationpb.NamespacePolicies{}}

	a := &AdminApp{
		MspProvider: makeMSPProvider(&testSigningIdentity{}, nil),
		OrdererProvider: makeManagedOrdererProvider(func(_ *config.OrdererConfig) (adapters.OrdererClient, error) {
			return ordererClient, nil
		}),
		NotificationProvider: makeManagedNotificationProvider(
			func(_ *config.NotificationsConfig) (adapters.NotificationClient, error) {
				return notificationClient, nil
			},
		),
		QueryProvider: makeQueryProvider(queryClient, nil),
	}

	require.NoError(t, a.SubmitTransaction(t.Context(), "tx-1", someTx()))
	_, err := a.SubmitTransactionWithWait(t.Context(), "tx-2", someTx())
	require.NoError(t, err)
	_, err = a.ListNamespaces(t.Context())
	require.NoError(t, err)

	require.NoError(t, a.Close())
	require.True(t, ordererClient.closed)
	require.True(t, notificationClient.closed)
	require.True(t, queryClient.closed)
}
