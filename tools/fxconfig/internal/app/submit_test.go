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

	"github.com/hyperledger/fabric-x-common/api/applicationpb"
	"github.com/hyperledger/fabric-x-common/msp"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/adapters"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/config"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/provider"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockOrdererClient struct {
	broadcastErr error
}

func (m *mockOrdererClient) Broadcast(_ context.Context, _ msp.SigningIdentity, _ string, _ *applicationpb.Tx) error {
	return m.broadcastErr
}

func (*mockOrdererClient) Close() error { return nil }

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
	defer cancel()
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

// Additional mocks and tests for context cancellation behavior

type MockOrdererClient struct {
	mock.Mock
}

func (m *MockOrdererClient) Broadcast(
	ctx context.Context,
	_ msp.SigningIdentity,
	txID string,
	tx *applicationpb.Tx,
) error {
	args := m.Called(ctx, txID, tx)
	return args.Error(0)
}

func (*MockOrdererClient) Close() error { return nil }

type MockNotificationClient struct {
	mock.Mock
}

func (m *MockNotificationClient) Subscribe(ctx context.Context, txID string) (chan int, error) {
	args := m.Called(ctx, txID)
	v := args.Get(0)
	if v == nil {
		return nil, args.Error(1)
	}
	ch, ok := v.(chan int)
	if !ok {
		return nil, args.Error(1)
	}
	return ch, args.Error(1)
}

func (m *MockNotificationClient) WaitForEvent(ctx context.Context, ch chan int) (int, error) {
	args := m.Called(ctx, ch)
	return args.Int(0), args.Error(1)
}

func (*MockNotificationClient) Close() error { return nil }

// removed: helper not used anymore

func TestSubmitTransaction_ContextCanceledBeforeStart(t *testing.T) {
	t.Parallel()

	mockClient := new(MockOrdererClient)

	a := &AdminApp{
		MspProvider:     makeMSPProvider(&testSigningIdentity{}, nil),
		OrdererProvider: makeOrdererProvider(mockClient, nil),
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	cancel()

	err := a.SubmitTransaction(ctx, "tx-1", someTx())
	require.Error(t, err)
	require.ErrorIs(t, err, context.Canceled)

	// Ensure no broadcast attempt happened
	mockClient.AssertNotCalled(t, "Broadcast", mock.Anything, mock.Anything, mock.Anything)
}

func TestSubmitTransactionWithWait_ContextCanceledBeforeStart(t *testing.T) {
	t.Parallel()

	mockClient := new(MockOrdererClient)

	a := &AdminApp{
		MspProvider:          makeMSPProvider(&testSigningIdentity{}, nil),
		OrdererProvider:      makeOrdererProvider(mockClient, nil),
		NotificationProvider: makeNotificationProvider(&mockNotificationClient{}, nil),
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	cancel()

	status, err := a.SubmitTransactionWithWait(ctx, "tx-1", someTx())
	require.Error(t, err)
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, UnknownStatus, status)
	// Ensure no broadcast attempt happened
	mockClient.AssertNotCalled(t, "Broadcast", mock.Anything, mock.Anything, mock.Anything)
}

func TestSubmitTransaction_ContextCanceledDuringRetry(t *testing.T) {
	t.Parallel()

	mockClient := new(MockOrdererClient)

	// Simulate a broadcast that blocks until the context is cancelled
	mockClient.On("Broadcast", mock.Anything, "tx-1", mock.Anything).Run(func(args mock.Arguments) {
		v := args.Get(0)
		if ctxArg, ok := v.(context.Context); ok {
			<-ctxArg.Done()
		}
	}).Return(context.Canceled).Once()

	a := &AdminApp{
		MspProvider:     makeMSPProvider(&testSigningIdentity{}, nil),
		OrdererProvider: makeOrdererProvider(mockClient, nil),
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	err := a.SubmitTransaction(ctx, "tx-1", someTx())
	require.Error(t, err)
	require.ErrorIs(t, err, context.Canceled)
	// Ensure Broadcast was called exactly once (no accidental retries)
	mockClient.AssertNumberOfCalls(t, "Broadcast", 1)
}
