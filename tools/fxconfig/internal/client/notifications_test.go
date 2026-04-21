/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

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
		done:          make(chan struct{}),
	}
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

// Stream termination tests

// TestNotificationClient_ListenExit_UnblocksWaitForEvent verifies that when the
// listener goroutine terminates (simulated by closing subscriber channels and done),
// WaitForEvent returns ErrNotificationStreamClosed instead of hanging until the
// caller's context deadline.
func TestNotificationClient_ListenExit_UnblocksWaitForEvent(t *testing.T) {
	t.Parallel()

	nc := newTestNotificationClient(10 * time.Second)

	// Register a subscriber as if Subscribe() had been called.
	ch := make(chan int, 1)
	nc.subscribersMu.Lock()
	nc.subscribers["tx-orphan"] = []chan int{ch}
	nc.subscribersMu.Unlock()

	// Simulate listen() exiting: close subscriber channels, clear map, close done.
	nc.subscribersMu.Lock()
	for _, receivers := range nc.subscribers {
		for _, c := range receivers {
			close(c)
		}
	}
	clear(nc.subscribers)
	nc.subscribersMu.Unlock()
	close(nc.done)

	// WaitForEvent must return ErrNotificationStreamClosed immediately, not block
	// for the 10-second WaitingTimeout.
	_, err := nc.WaitForEvent(t.Context(), ch)
	require.ErrorIs(t, err, ErrNotificationStreamClosed)
}

// TestNotificationClient_SubscribeAfterListenExit_ReturnsSentinel verifies that
// Subscribe fails fast with ErrNotificationStreamClosed after the listener has
// exited, rather than blocking forever on the unbuffered requestQueue.
func TestNotificationClient_SubscribeAfterListenExit_ReturnsSentinel(t *testing.T) {
	t.Parallel()

	nc := newTestNotificationClient(10 * time.Second)
	close(nc.done)

	_, err := nc.Subscribe(t.Context(), "tx-dead")
	require.ErrorIs(t, err, ErrNotificationStreamClosed)
}
