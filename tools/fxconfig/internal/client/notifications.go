/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/hyperledger/fabric-x-common/api/committerpb"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/config"
)

// NotificationClient provides a gRPC client for receiving transaction status notifications.
// It manages bidirectional streaming with the committer notification service and multiplexes
// notifications to multiple subscribers per transaction ID.
type NotificationClient struct {
	cfg    config.NotificationsConfig
	closeF func()

	notifyClient  committerpb.NotifierClient
	requestQueue  chan *committerpb.NotificationRequest
	responseQueue chan *committerpb.NotificationResponse

	subscribers   map[string][]chan int
	subscribersMu sync.RWMutex

	// streamErr holds the error that caused the stream to terminate.
	// Atomically stored; checked by Subscribe() before sending requests.
	streamErr atomic.Pointer[error]

	// droppedNotifications counts status events that the dispatcher could not
	// deliver because the subscriber's channel buffer was full or the receiver
	// had already given up. Exposed via DroppedNotifications for diagnostics
	// and tests.
	droppedNotifications atomic.Uint64
}

// NewNotificationClient creates a notification client with the provided configuration.
// It establishes a gRPC connection with optional TLS and starts a background listener.
func NewNotificationClient(cfg config.NotificationsConfig) (*NotificationClient, error) {
	conn, err := newClientConn(&cfg.EndpointServiceConfig)
	if err != nil {
		return nil, fmt.Errorf("cannot get grpc client: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	nc := &NotificationClient{
		cfg:          cfg,
		notifyClient: committerpb.NewNotifierClient(conn),
		closeF: func() {
			cancel()
			_ = conn.Close()
		},
		requestQueue:  make(chan *committerpb.NotificationRequest),
		responseQueue: make(chan *committerpb.NotificationResponse),
		subscribers:   make(map[string][]chan int),
	}

	go func() {
		if err := nc.listen(ctx); err != nil && !errors.Is(err, context.Canceled) {
			logger.Errorf("Notification listener stream terminated unexpectedly: %s", err)
		}
	}()

	return nc, nil
}

// Close terminates the gRPC connection and cancels the background listener.
func (n *NotificationClient) Close() error {
	if n.closeF != nil {
		n.closeF()
	}
	return nil
}

// Subscribe registers interest in a transaction's status and returns a channel for notifications.
// Multiple subscribers to the same txID share a single upstream subscription.
func (n *NotificationClient) Subscribe(ctx context.Context, txID string) (chan int, error) {
	// Apply timeout to prevent blocking on requestQueue send.
	ctx, cancel := context.WithTimeout(ctx, n.cfg.WaitingTimeout)
	defer cancel()

	// Fail fast if the stream has previously failed — check before any state mutation.
	if err := n.streamErr.Load(); err != nil {
		return nil, *err
	}

	receiverCh := make(chan int, 1)
	isFirst := func() bool {
		n.subscribersMu.Lock()
		defer n.subscribersMu.Unlock()

		subscribers := n.subscribers[txID]
		n.subscribers[txID] = append(subscribers, receiverCh)

		return len(subscribers) == 0
	}()

	if !isFirst {
		// we already have an active subscription for this txID
		return receiverCh, nil
	}

	rollback := func() {
		// rollback can race logically with dispatcher cleanup in listen(),
		// where completed txIDs are also deleted from n.subscribers. The shared
		// subscribersMu lock plus the missing-key guard below makes this idempotent
		// and safe regardless of which path removes the entry first.
		n.subscribersMu.Lock()
		defer n.subscribersMu.Unlock()

		subscribers, ok := n.subscribers[txID]
		if !ok {
			return
		}

		for i, ch := range subscribers {
			if ch != receiverCh {
				continue
			}

			subscribers = append(subscribers[:i], subscribers[i+1:]...)
			if len(subscribers) == 0 {
				delete(n.subscribers, txID)
				return
			}

			n.subscribers[txID] = subscribers
			return
		}
	}

	// setup request
	req := &committerpb.NotificationRequest{
		TxStatusRequest: &committerpb.TxIDsBatch{
			TxIds: []string{txID},
		},
		Timeout: durationpb.New(n.cfg.WaitingTimeout),
	}

	// check if our ctx is still open
	select {
	case <-ctx.Done():
		rollback()
		return nil, ctx.Err()
	default:
	}

	// try to push to request queue
	select {
	case <-ctx.Done():
		rollback()
		return nil, ctx.Err()
	case n.requestQueue <- req:
	}

	return receiverCh, nil
}

// WaitForEvent blocks until a status notification arrives or the timeout expires.
// Returns the transaction status code or an error if the context is canceled.
func (n *NotificationClient) WaitForEvent(ctx context.Context, subscription chan int) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, n.cfg.WaitingTimeout)
	defer cancel()
	return wait(ctx, subscription)
}

func wait(ctx context.Context, subscription chan int) (int, error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}

	// try to push to request queue
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case status := <-subscription:
		return status, nil
	}
}

// listen runs the bidirectional gRPC stream, managing request/response queues
// and dispatching notifications to subscribers. Blocks until context is canceled.
//
//nolint:gocognit
func (n *NotificationClient) listen(ctx context.Context) error {
	notifyStream, err := n.notifyClient.OpenNotificationStream(ctx)
	if err != nil {
		n.streamErr.Store(&err)
		return err
	}

	// Use the base context for errgroup
	g, gCtx := errgroup.WithContext(ctx)

	// spawn stream receiver
	g.Go(func() error {
		for {
			res, rerr := notifyStream.Recv()
			if rerr != nil {
				if errors.Is(rerr, context.Canceled) {
					return nil
				}
				return rerr
			}
			select {
			case <-gCtx.Done():
				return gCtx.Err()
			case n.responseQueue <- res:
			}
		}
	})

	// spawn stream sender
	g.Go(func() error {
		var req *committerpb.NotificationRequest
		for {
			select {
			case <-gCtx.Done():
				return gCtx.Err()
			case req = <-n.requestQueue:
			}

			if rerr := notifyStream.Send(req); rerr != nil {
				return rerr
			}
		}
	})

	// spawn notification dispatcher
	g.Go(func() error {
		var resp *committerpb.NotificationResponse
		for {
			select {
			case <-gCtx.Done():
				return gCtx.Err()
			case resp = <-n.responseQueue:
			}

			n.dispatchNotifications(n.collectNotifications(parseResponse(resp)))
		}
	})

	err = g.Wait()

	// Capture the error from the group before cleanup.
	if err != nil && !errors.Is(err, context.Canceled) {
		n.streamErr.Store(&err)
	}

	// Cleanup subscribers map when listen() exits
	n.subscribersMu.Lock()
	clear(n.subscribers)
	n.subscribersMu.Unlock()

	return err
}

// notificationCall is a single status event ready for delivery to a subscriber.
type notificationCall struct {
	txID          string
	receiverQueue chan int
	status        int
}

// collectNotifications snapshots and removes all subscribers whose txIDs
// appear in res, returning one notificationCall per subscriber. Holding the
// subscribers lock only for the duration of the lookup/delete keeps the
// dispatcher hot path short.
func (n *NotificationClient) collectNotifications(res map[string]int) []notificationCall {
	var notifications []notificationCall

	n.subscribersMu.Lock()
	defer n.subscribersMu.Unlock()

	for txID, v := range res {
		receivers, ok := n.subscribers[txID]
		if !ok {
			continue
		}
		delete(n.subscribers, txID)
		for _, q := range receivers {
			notifications = append(notifications, notificationCall{
				txID:          txID,
				receiverQueue: q,
				status:        v,
			})
		}
	}

	return notifications
}

// dispatchNotifications sends each pending status to its subscriber. The send
// is non-blocking so a single slow subscriber cannot stall the dispatcher
// goroutine, but every dropped event is logged and counted — the txID has
// already been removed from the subscribers map, so a drop is unrecoverable
// and must be observable.
func (n *NotificationClient) dispatchNotifications(notifications []notificationCall) {
	for _, c := range notifications {
		select {
		case c.receiverQueue <- c.status:
		default:
			n.droppedNotifications.Add(1)
			logger.Warnf(
				"notification dropped (unrecoverable): txID=%s status=%d (subscriber buffer full or receiver gone)",
				c.txID, c.status,
			)
		}
	}
}

// DroppedNotifications returns the cumulative count of status events that the
// dispatcher could not deliver to a subscriber (buffer full or receiver
// already gone). Useful for end-of-run diagnostics and tests.
func (n *NotificationClient) DroppedNotifications() uint64 {
	return n.droppedNotifications.Load()
}

// parseResponse extracts transaction statuses from a notification response,
// mapping transaction IDs to their status codes (timeouts and status events).
func parseResponse(resp *committerpb.NotificationResponse) map[string]int {
	res := make(map[string]int)

	// first parse all timeouts
	for _, txID := range resp.GetTimeoutTxIds() {
		res[txID] = int(committerpb.Status_STATUS_UNSPECIFIED)
	}

	// next we parse the status events
	for _, r := range resp.GetTxStatusEvents() {
		txID := r.GetRef().GetTxId()
		status := r.GetStatus()

		res[txID] = int(status)
	}

	return res
}
