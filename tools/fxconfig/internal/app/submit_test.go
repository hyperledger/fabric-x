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

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/hyperledger/fabric-x-common/api/applicationpb"
	"github.com/hyperledger/fabric-x-common/msp"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/adapters"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/config"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/provider"
)

type MockOrdererClient struct {
	mock.Mock
}

func (m *MockOrdererClient) Broadcast(
	ctx context.Context,
	id msp.SigningIdentity,
	txID string,
	tx *applicationpb.Tx,
) error {
	args := m.Called(ctx, id, txID, tx)
	return args.Error(0)
}

func (*MockOrdererClient) Close() error {
	return nil
}

type mockNotificationClient struct {
	subscribeErr error
	waitErr      error
	status       int
}

func (m *mockNotificationClient) Subscribe(_ context.Context, _ string) (chan int, error) {
	if m.subscribeErr != nil {
		return nil, m.subscribeErr
	}
	ch := make(chan int, 1)
	ch <- m.status
	return ch, nil
}

func (m *mockNotificationClient) WaitForEvent(_ context.Context, ch chan int) (int, error) {
	if m.waitErr != nil {
		return 0, m.waitErr
	}
	return <-ch, nil
}

func (*mockNotificationClient) Close() error { return nil }

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

func expectBroadcasts(t *testing.T, mockClient *MockOrdererClient, txID any, errs ...error) {
	t.Helper()

	for _, err := range errs {
		mockClient.
			On("Broadcast", mock.Anything, mock.Anything, txID, mock.Anything).
			Return(err).
			Once()
	}

	t.Cleanup(func() {
		mockClient.AssertExpectations(t)
	})
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

func someTx() *applicationpb.Tx {
	return &applicationpb.Tx{Namespaces: []*applicationpb.TxNamespace{{NsId: "ns1"}}}
}

// SubmitTransaction tests

func TestSubmitTransaction_MspProviderError(t *testing.T) {
	t.Parallel()

	a := &AdminApp{
		MspProvider:     makeMSPProvider(nil, errors.New("msp unavailable")),
		OrdererProvider: makeOrdererProvider(&MockOrdererClient{}, nil),
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

	mockClient := new(MockOrdererClient)
	expectBroadcasts(t, mockClient, "tx-1",
		errors.New("broadcast failed"),
		errors.New("broadcast failed"),
		errors.New("broadcast failed"),
	)

	a := &AdminApp{
		MspProvider:     makeMSPProvider(&testSigningIdentity{}, nil),
		OrdererProvider: makeOrdererProvider(mockClient, nil),
	}

	err := a.SubmitTransaction(t.Context(), "tx-1", someTx())
	require.Error(t, err)
}

func TestSubmitTransaction_BroadcastRetrySucceeds(t *testing.T) {
	t.Parallel()

	mockClient := new(MockOrdererClient)
	expectBroadcasts(t, mockClient, "tx-1", errors.New("temporary error"), nil)

	a := &AdminApp{
		MspProvider:     makeMSPProvider(&testSigningIdentity{}, nil),
		OrdererProvider: makeOrdererProvider(mockClient, nil),
	}

	err := a.SubmitTransaction(t.Context(), "tx-1", someTx())
	require.NoError(t, err)
}

func TestSubmitTransaction_BroadcastRetryExhausted(t *testing.T) {
	t.Parallel()

	mockClient := new(MockOrdererClient)
	expectBroadcasts(t, mockClient, "tx-1",
		errors.New("broadcast failed"),
		errors.New("broadcast failed"),
		errors.New("broadcast failed"),
	)

	a := &AdminApp{
		MspProvider:     makeMSPProvider(&testSigningIdentity{}, nil),
		OrdererProvider: makeOrdererProvider(mockClient, nil),
	}

	err := a.SubmitTransaction(t.Context(), "tx-1", someTx())
	require.Error(t, err)
	mockClient.AssertNumberOfCalls(t, "Broadcast", maxBroadcastAttempts)
}

func TestSubmitTransaction_Success(t *testing.T) {
	t.Parallel()

	mockClient := new(MockOrdererClient)
	expectBroadcasts(t, mockClient, "tx-1", nil)

	a := &AdminApp{
		MspProvider:     makeMSPProvider(&testSigningIdentity{}, nil),
		OrdererProvider: makeOrdererProvider(mockClient, nil),
	}

	err := a.SubmitTransaction(t.Context(), "tx-1", someTx())
	require.NoError(t, err)
}

func TestSubmitTransaction_ContextCancelled(t *testing.T) {
	t.Parallel()

	mockClient := new(MockOrdererClient)
	expectBroadcasts(t, mockClient, "tx-1", context.Canceled)

	a := &AdminApp{
		MspProvider:     makeMSPProvider(&testSigningIdentity{}, nil),
		OrdererProvider: makeOrdererProvider(mockClient, nil),
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := a.SubmitTransaction(ctx, "tx-1", someTx())
	require.Error(t, err)
	require.ErrorIs(t, err, context.Canceled)
}

func TestSubmitTransaction_NonRetryableError_NoRetry(t *testing.T) {
	t.Parallel()

	mockClient := new(MockOrdererClient)
	expectBroadcasts(t, mockClient, "tx-1", context.Canceled)

	a := &AdminApp{
		MspProvider:     makeMSPProvider(&testSigningIdentity{}, nil),
		OrdererProvider: makeOrdererProvider(mockClient, nil),
	}

	err := a.SubmitTransaction(t.Context(), "tx-1", someTx())
	require.Error(t, err)
	mockClient.AssertNumberOfCalls(t, "Broadcast", 1)
}

// SubmitTransactionWithWait tests

func TestSubmitTransactionWithWait_MspProviderError(t *testing.T) {
	t.Parallel()

	a := &AdminApp{
		MspProvider:          makeMSPProvider(nil, errors.New("msp unavailable")),
		OrdererProvider:      makeOrdererProvider(&MockOrdererClient{}, nil),
		NotificationProvider: makeNotificationProvider(&mockNotificationClient{}, nil),
	}

	_, err := a.SubmitTransactionWithWait(t.Context(), "tx-1", someTx())
	require.Error(t, err)
}

func TestSubmitTransactionWithWait_NotificationProviderError(t *testing.T) {
	t.Parallel()

	a := &AdminApp{
		MspProvider:          makeMSPProvider(&testSigningIdentity{}, nil),
		OrdererProvider:      makeOrdererProvider(&MockOrdererClient{}, nil),
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
		OrdererProvider: makeOrdererProvider(&MockOrdererClient{}, nil),
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

	mockClient := new(MockOrdererClient)
	expectBroadcasts(t, mockClient, "tx-1",
		errors.New("broadcast failed"),
		errors.New("broadcast failed"),
		errors.New("broadcast failed"),
	)

	a := &AdminApp{
		MspProvider:          makeMSPProvider(&testSigningIdentity{}, nil),
		OrdererProvider:      makeOrdererProvider(mockClient, nil),
		NotificationProvider: makeNotificationProvider(&mockNotificationClient{}, nil),
	}

	status, err := a.SubmitTransactionWithWait(t.Context(), "tx-1", someTx())
	require.Error(t, err)
	require.Equal(t, UnknownStatus, status)
}

func TestSubmitTransactionWithWait_WaitForEventError(t *testing.T) {
	t.Parallel()

	mockClient := new(MockOrdererClient)
	expectBroadcasts(t, mockClient, "tx-1", nil)

	a := &AdminApp{
		MspProvider:     makeMSPProvider(&testSigningIdentity{}, nil),
		OrdererProvider: makeOrdererProvider(mockClient, nil),
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

	mockClient := new(MockOrdererClient)
	expectBroadcasts(t, mockClient, "tx-1", nil)

	a := &AdminApp{
		MspProvider:          makeMSPProvider(&testSigningIdentity{}, nil),
		OrdererProvider:      makeOrdererProvider(mockClient, nil),
		NotificationProvider: makeNotificationProvider(&mockNotificationClient{status: expectedStatus}, nil),
	}

	status, err := a.SubmitTransactionWithWait(t.Context(), "tx-1", someTx())
	require.NoError(t, err)
	require.Equal(t, expectedStatus, status)
}
