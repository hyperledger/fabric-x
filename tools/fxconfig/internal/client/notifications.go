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

	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/hyperledger/fabric-x-common/api/committerpb"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/config"
)

// ErrNotificationStreamClosed is returned when the notification stream has
// terminated, either because the gRPC stream failed or the client was closed.
// Callers receive this from Subscribe or WaitForEvent instead of blocking until
// their own context deadline expires.
var ErrNotificationStreamClosed = errors.New("notification stream closed")

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

	// done is closed when listen() returns. Subscribers use it to fail fast
	// instead of blocking on requestQueue or their own context deadline.
	done chan struct{}
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
		done:          make(chan struct{}),
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
// If the listener has terminated, Subscribe fails fast with ErrNotificationStreamClosed
// instead of blocking on requestQueue.
func (n *NotificationClient) Subscribe(ctx context.Context, txID string) (chan int, error) {
	// fast-path checks before mutating state
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	receiverCh := make(chan int, 1)

	n.subscribersMu.Lock()
	// Recheck done under the lock so we serialize with listen()'s cleanup: if
	// cleanup has already closed subscriber channels and cleared the map, we
	// must not append a stale entry that will never be delivered or closed.
	select {
	case <-n.done:
		n.subscribersMu.Unlock()
		return nil, ErrNotificationStreamClosed
	default:
	}

	subscribers := n.subscribers[txID]
	n.subscribers[txID] = append(subscribers, receiverCh)
	n.subscribersMu.Unlock()

	if len(subscribers) > 0 {
		// we already have an active subscription for this txID
		return receiverCh, nil
	}

	// setup request
	req := &committerpb.NotificationRequest{
		TxStatusRequest: &committerpb.TxIDsBatch{
			TxIds: []string{txID},
		},
		Timeout: durationpb.New(n.cfg.WaitingTimeout),
	}

	// try to push to request queue
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-n.done:
		return nil, ErrNotificationStreamClosed
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

	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case status, ok := <-subscription:
		if !ok {
			return 0, ErrNotificationStreamClosed
		}
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
		type notificationCall struct {
			receiverQueue chan int
			status        int
		}

		var resp *committerpb.NotificationResponse
		for {
			select {
			case <-gCtx.Done():
				return gCtx.Err()
			case resp = <-n.responseQueue:
			}

			res := parseResponse(resp)

			// Collect subscribers under lock, then release before spawning goroutines.
			// This minimizes lock hold time — only map lookups and deletes happen
			// under the lock. Goroutine scheduling happens entirely outside.
			var notifications []notificationCall

			n.subscribersMu.Lock()
			for txID, v := range res {
				receivers, ok := n.subscribers[txID]
				if !ok {
					continue
				}
				delete(n.subscribers, txID)
				for _, q := range receivers {
					notifications = append(notifications, notificationCall{receiverQueue: q, status: v})
				}
			}
			n.subscribersMu.Unlock()

			for _, c := range notifications {
				select {
				case c.receiverQueue <- c.status:
				default:
					// message dropped
				}
			}
		}
	})

	err = g.Wait()

	// Signal termination first so Subscribe() can detect stream death under
	// the lock before any new entries are appended to the map.
	close(n.done)

	// Close every waiting subscriber channel so callers in wait() unblock
	// with ErrNotificationStreamClosed immediately, without waiting for their
	// own context deadline to expire.
	n.subscribersMu.Lock()
	for _, receivers := range n.subscribers {
		for _, ch := range receivers {
			close(ch)
		}
	}
	clear(n.subscribers)
	n.subscribersMu.Unlock()

	return err
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
