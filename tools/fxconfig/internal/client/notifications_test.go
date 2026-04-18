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
	"google.golang.org/grpc/metadata"

	"github.com/hyperledger/fabric-x-common/api/committerpb"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/config"
)

func newTestNotificationClient(waitingTimeout time.Duration) *NotificationClient {
	return &NotificationClient{
		cfg: config.NotificationsConfig{
			WaitingTimeout: waitingTimeout,
		},
		requestQueue:  make(chan *committerpb.NotificationRequest),
		responseQueue: make(chan *committerpb.NotificationResponse),
		subscribers:   make(map[string][]chan int),
	}
}

type mockNotificationStream struct {
	sendErr  error
	recvResp *committerpb.NotificationResponse
	recvErr  error
}

func (m *mockNotificationStream) Send(req *committerpb.NotificationRequest) error { return m.sendErr }
func (m *mockNotificationStream) Recv() (*committerpb.NotificationResponse, error) {
	return m.recvResp, m.recvErr
}
func (*mockNotificationStream) Header() (metadata.MD, error) { return nil, nil }
func (*mockNotificationStream) Trailer() metadata.MD         { return nil }
func (*mockNotificationStream) CloseSend() error             { return nil }
func (*mockNotificationStream) Context() context.Context     { return context.Background() }
func (*mockNotificationStream) SendMsg(_ any) error          { return nil }
func (*mockNotificationStream) RecvMsg(_ any) error          { return nil }

type mockNotifierClient struct {
	stream  grpc.BidiStreamingClient[committerpb.NotificationRequest, committerpb.NotificationResponse]
	openErr error
}

func (m *mockNotifierClient) OpenNotificationStream(
	_ context.Context,
	_ ...grpc.CallOption,
) (grpc.BidiStreamingClient[committerpb.NotificationRequest, committerpb.NotificationResponse], error) {
	return m.stream, m.openErr
}

// parseResponse tests

func TestParseResponse_Empty(t *testing.T) {
	t.Parallel()

	result := parseResponse(&committerpb.NotificationResponse{})
	require.Empty(t, result)
}

func TestParseResponse_TimeoutTxIDs(t *testing.T) {
	t.Parallel()

	resp := &committerpb.NotificationResponse{
		TimeoutTxIds: []string{"tx1", "tx2"},
	}
	result := parseResponse(resp)
	require.Len(t, result, 2)
	require.Equal(t, int(committerpb.Status_STATUS_UNSPECIFIED), result["tx1"])
	require.Equal(t, int(committerpb.Status_STATUS_UNSPECIFIED), result["tx2"])
}

func TestParseResponse_TxStatusEvents(t *testing.T) {
	t.Parallel()

	resp := &committerpb.NotificationResponse{
		TxStatusEvents: []*committerpb.TxStatus{
			{
				Ref:    &committerpb.TxRef{TxId: "tx1"},
				Status: committerpb.Status_COMMITTED,
			},
		},
	}
	result := parseResponse(resp)
	require.Len(t, result, 1)
	require.Equal(t, int(committerpb.Status_COMMITTED), result["tx1"])
}

func TestParseResponse_StatusOverridesTimeout(t *testing.T) {
	t.Parallel()

	resp := &committerpb.NotificationResponse{
		TimeoutTxIds: []string{"tx1"},
		TxStatusEvents: []*committerpb.TxStatus{
			{
				Ref:    &committerpb.TxRef{TxId: "tx1"},
				Status: committerpb.Status_COMMITTED,
			},
		},
	}
	result := parseResponse(resp)
	require.Len(t, result, 1)
	require.Equal(t, int(committerpb.Status_COMMITTED), result["tx1"])
}

// wait tests

func TestWait_ContextAlreadyCanceled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := wait(ctx, make(chan int, 1))
	require.ErrorIs(t, err, context.Canceled)
}

func TestWait_StatusReceived(t *testing.T) {
	t.Parallel()

	ch := make(chan int, 1)
	ch <- 42

	status, err := wait(t.Context(), ch)
	require.NoError(t, err)
	require.Equal(t, 42, status)
}

// WaitForEvent tests

func TestWaitForEvent_Success(t *testing.T) {
	t.Parallel()

	nc := newTestNotificationClient(time.Second)
	ch := make(chan int, 1)
	ch <- 7

	status, err := nc.WaitForEvent(t.Context(), ch)
	require.NoError(t, err)
	require.Equal(t, 7, status)
}

func TestWaitForEvent_Timeout(t *testing.T) {
	t.Parallel()

	nc := newTestNotificationClient(time.Millisecond)

	_, err := nc.WaitForEvent(t.Context(), make(chan int))
	require.Error(t, err)
}

// Subscribe tests

func TestNotificationClient_Subscribe_DuplicateSubscription(t *testing.T) {
	t.Parallel()

	nc := newTestNotificationClient(time.Second)
	// Pre-populate a subscriber so the second subscribe is a duplicate
	// and won't send to the requestQueue (which would block without a listener).
	nc.subscribers["tx1"] = []chan int{make(chan int, 1)}

	ch, err := nc.Subscribe(t.Context(), "tx1")
	require.NoError(t, err)
	require.NotNil(t, ch)
}

func TestNotificationClient_Subscribe_ContextCanceled(t *testing.T) {
	t.Parallel()

	nc := newTestNotificationClient(time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := nc.Subscribe(ctx, "tx1")
	require.ErrorIs(t, err, context.Canceled)
}

func TestNotificationClient_Subscribe_SendsRequest(t *testing.T) {
	t.Parallel()

	nc := newTestNotificationClient(time.Second)

	// Consume the request in a background goroutine to unblock Subscribe.
	go func() { <-nc.requestQueue }()

	ch, err := nc.Subscribe(t.Context(), "tx1")
	require.NoError(t, err)
	require.NotNil(t, ch)
}

func TestNotificationClient_Subscribe_StreamError(t *testing.T) {
	t.Parallel()

	nc := newTestNotificationClient(time.Second)
	// Simulate a stream error
	expectedErr := errors.New("stream connection lost")
	nc.streamErr.Store(&expectedErr)

	// Verify that subscribers map is NOT modified when streamErr is set
	_, err := nc.Subscribe(t.Context(), "tx1")
	require.ErrorIs(t, err, expectedErr)
	require.Empty(t, nc.subscribers, "subscribers map should not be modified when streamErr is set")
}

func TestNotificationClient_Subscribe_Timeout(t *testing.T) {
	t.Parallel()

	nc := newTestNotificationClient(time.Millisecond)
	// Don't consume requestQueue — Subscribe should time out waiting to send.

	_, err := nc.Subscribe(t.Context(), "tx1")
	require.Error(t, err)
	// Should get DeadlineExceeded since the context timeout is short.
	require.ErrorContains(t, err, "deadline exceeded")
}

func TestNewNotificationClient_FailsFastWhenStreamStartupFails(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("stream unavailable")
	closed := false

	nc, err := newNotificationClient(
		t.Context(),
		config.NotificationsConfig{WaitingTimeout: time.Second},
		&mockNotifierClient{openErr: expectedErr},
		func() { closed = true },
	)

	require.Nil(t, nc)
	require.ErrorIs(t, err, expectedErr)
	require.True(t, closed)
}

func TestNotificationClient_Start_StoresStartupError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("stream unavailable")
	nc := &NotificationClient{
		cfg:           config.NotificationsConfig{WaitingTimeout: time.Second},
		notifyClient:  &mockNotifierClient{openErr: expectedErr},
		requestQueue:  make(chan *committerpb.NotificationRequest),
		responseQueue: make(chan *committerpb.NotificationResponse),
		subscribers:   make(map[string][]chan int),
	}

	err := nc.start(t.Context())
	require.ErrorIs(t, err, expectedErr)

	stored := nc.streamErr.Load()
	require.NotNil(t, stored)
	require.ErrorIs(t, *stored, expectedErr)
}

// Close tests

func TestNotificationClient_Close_CallsCloseFunc(t *testing.T) {
	t.Parallel()

	closed := false
	nc := &NotificationClient{closeF: func() { closed = true }}
	require.NoError(t, nc.Close())
	require.True(t, closed)
}

func TestNotificationClient_Close_NilFunc(t *testing.T) {
	t.Parallel()

	nc := &NotificationClient{}
	require.NoError(t, nc.Close())
}
