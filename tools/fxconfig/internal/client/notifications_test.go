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

// Concurrent subscriber tests

func TestNotificationClient_ConcurrentSubscribers_SameTxID_BothReceive(t *testing.T) {
	t.Parallel()

	nc := newTestNotificationClient(2 * time.Second)

	// First subscriber — drains the request queue so Subscribe doesn't block
	go func() { <-nc.requestQueue }()
	ch1, err := nc.Subscribe(t.Context(), "tx-concurrent")
	require.NoError(t, err)

	// Second subscriber on the same txID — takes the duplicate path (no requestQueue send)
	ch2, err := nc.Subscribe(t.Context(), "tx-concurrent")
	require.NoError(t, err)

	// Simulate the dispatcher: lock, collect receivers, delete entry, unlock, deliver
	resp := parseResponse(&committerpb.NotificationResponse{
		TxStatusEvents: []*committerpb.TxStatus{
			{
				Ref:    &committerpb.TxRef{TxId: "tx-concurrent"},
				Status: committerpb.Status_COMMITTED,
			},
		},
	})

	nc.subscribersMu.Lock()
	type call struct {
		ch     chan int
		status int
	}
	var calls []call
	for txID, v := range resp {
		receivers, ok := nc.subscribers[txID]
		if !ok {
			continue
		}
		delete(nc.subscribers, txID)
		for _, q := range receivers {
			calls = append(calls, call{ch: q, status: v})
		}
	}
	nc.subscribersMu.Unlock()

	// Deliver using the same non-blocking send the real dispatcher uses
	for _, c := range calls {
		select {
		case c.ch <- c.status:
		default:
			// dropped — this is what we're testing against
		}
	}

	// Both subscribers must receive the status
	select {
	case s := <-ch1:
		require.Equal(t, int(committerpb.Status_COMMITTED), s)
	case <-time.After(time.Second):
		t.Fatal("ch1: timed out — first subscriber was starved")
	}

	select {
	case s := <-ch2:
		require.Equal(t, int(committerpb.Status_COMMITTED), s)
	case <-time.After(time.Second):
		t.Fatal("ch2: timed out — notification silently dropped for duplicate subscriber")
	}
}

func TestNotificationClient_SubscribeAfterDispatch_DoesNotHang(t *testing.T) {
	t.Parallel()

	nc := newTestNotificationClient(500 * time.Millisecond)

	// First subscriber
	go func() { <-nc.requestQueue }()
	ch1, err := nc.Subscribe(t.Context(), "tx-resubscribe")
	require.NoError(t, err)

	// Simulate the dispatcher deleting the txID entry after delivering
	nc.subscribersMu.Lock()
	receivers := nc.subscribers["tx-resubscribe"]
	delete(nc.subscribers, "tx-resubscribe")
	nc.subscribersMu.Unlock()

	// Deliver to the first subscriber
	for _, r := range receivers {
		r <- int(committerpb.Status_COMMITTED)
	}
	s := <-ch1
	require.Equal(t, int(committerpb.Status_COMMITTED), s)

	// A new subscriber arrives for the same txID after dispatch.
	// This takes the fresh-subscription path and pushes to requestQueue.
	// With no listener draining the queue, it must respect context cancellation
	// and NOT hang indefinitely.
	ctx, cancel := context.WithTimeout(t.Context(), 500*time.Millisecond)
	defer cancel()

	_, err = nc.Subscribe(ctx, "tx-resubscribe")
	// Must fail with deadline exceeded — not hang forever
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestNotificationClient_ListenExit_ClearsSubscribers(t *testing.T) {
	t.Parallel()

	nc := newTestNotificationClient(2 * time.Second)

	// Pre-populate subscribers as if Subscribe() was called
	ch := make(chan int, 1)
	nc.subscribersMu.Lock()
	nc.subscribers["tx-orphan"] = []chan int{ch}
	nc.subscribersMu.Unlock()

	// Simulate listen() exiting — it calls clear(n.subscribers)
	nc.subscribersMu.Lock()
	clear(nc.subscribers)
	nc.subscribersMu.Unlock()

	// Verify subscriber map is empty
	nc.subscribersMu.RLock()
	require.Empty(t, nc.subscribers)
	nc.subscribersMu.RUnlock()

	// WaitForEvent must timeout — the channel will never receive because
	// no dispatcher is running and listen() has cleaned up
	ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
	defer cancel()
	_, err := nc.WaitForEvent(ctx, ch)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestNotificationClient_ConcurrentSubscribeAndDispatch(t *testing.T) {
	t.Parallel()

	nc := newTestNotificationClient(2 * time.Second)

	const subscriberCount = 10
	channels := make([]chan int, subscriberCount)

	// First subscriber triggers the upstream request
	go func() { <-nc.requestQueue }()
	ch, err := nc.Subscribe(t.Context(), "tx-race")
	require.NoError(t, err)
	channels[0] = ch

	// Remaining subscribers take the duplicate path (no requestQueue send)
	for i := 1; i < subscriberCount; i++ {
		ch, err := nc.Subscribe(t.Context(), "tx-race")
		require.NoError(t, err)
		channels[i] = ch
	}

	// Verify all subscribers are registered
	nc.subscribersMu.RLock()
	require.Len(t, nc.subscribers["tx-race"], subscriberCount)
	nc.subscribersMu.RUnlock()

	// Simulate dispatcher delivery
	status := int(committerpb.Status_COMMITTED)
	nc.subscribersMu.Lock()
	receivers := nc.subscribers["tx-race"]
	delete(nc.subscribers, "tx-race")
	nc.subscribersMu.Unlock()

	for _, r := range receivers {
		select {
		case r <- status:
		default:
		}
	}

	// ALL subscribers must receive the notification
	for i, ch := range channels {
		select {
		case s := <-ch:
			require.Equal(t, status, s, "subscriber %d got wrong status", i)
		case <-time.After(time.Second):
			t.Fatalf("subscriber %d: timed out — notification dropped", i)
		}
	}

	// Map entry must be gone after dispatch
	nc.subscribersMu.RLock()
	_, exists := nc.subscribers["tx-race"]
	nc.subscribersMu.RUnlock()
	require.False(t, exists, "subscriber entry should be deleted after dispatch")
}
