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
	"github.com/hyperledger/fabric-x-common/cmd/common/comm"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/app/api"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/config"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/validation"
)

type NotificationProvider struct {
	ValidationContext validation.Context
	Cfg               config.NotificationsConfig
	// TODO make this provide once
}

func (f *NotificationProvider) Validate() error {
	return f.Cfg.Validate(f.ValidationContext)
}

func (f *NotificationProvider) Get() (api.NotificationClient, error) {
	return NewNotificationClient(f.Cfg)
}

type NotificationClient struct {
	cfg    config.NotificationsConfig
	closeF func()

	notifyClient  committerpb.NotifierClient
	requestQueue  chan *committerpb.NotificationRequest
	responseQueue chan *committerpb.NotificationResponse

	subscribers   map[string][]chan int
	subscribersMu sync.RWMutex
}

func NewNotificationClient(cfg config.NotificationsConfig) (*NotificationClient, error) {
	clientCfg := comm.Config{
		Timeout: cfg.ConnectionTimeout,
	}

	// TLS config
	if cfg.TLS.IsEnabled() {
		clientCfg.CertPath = cfg.TLS.ClientCertPath
		clientCfg.KeyPath = cfg.TLS.ClientKeyPath
		clientCfg.PeerCACertPath = cfg.TLS.RootCertPaths[0]
	}

	cl, err := comm.NewClient(clientCfg)
	if err != nil {
		return nil, fmt.Errorf("cannot get grpc client: %w", err)
	}

	conn, err := cl.NewDialer(cfg.Address)()
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
			fmt.Printf("error: Notification listener stream terminated unexpectedly: %s\n", err)
			// logger.Errorf("Notification listener stream terminated unexpectedly for %s: %s", key, err)
		}
	}()

	return nc, nil
}

func (n *NotificationClient) Close() error {
	if n.closeF != nil {
		n.closeF()
	}
	return nil
}

func (n *NotificationClient) Subscribe(ctx context.Context, txID string) (chan int, error) {
	receiverCh := make(chan int, 1)

	n.subscribersMu.Lock()
	defer n.subscribersMu.Unlock()

	subscribers := n.subscribers[txID]
	n.subscribers[txID] = append(subscribers, receiverCh)

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

	// check if our ctx is still open
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// try to push to request queue
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case n.requestQueue <- req:
	}

	return receiverCh, nil
}

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

// Listen is a blocking method that runs the notification listener stream.
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
			res, err := notifyStream.Recv()
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return nil
				}
				return err
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

			if err := notifyStream.Send(req); err != nil {
				return err
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

	// Cleanup subscribers map when listen() exits
	n.subscribersMu.Lock()
	clear(n.subscribers)
	n.subscribersMu.Unlock()

	return err
}

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
