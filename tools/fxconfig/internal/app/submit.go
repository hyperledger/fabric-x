/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v5"
	"github.com/hyperledger/fabric-lib-go/common/flogging"

	"github.com/hyperledger/fabric-x-common/api/applicationpb"
	"github.com/hyperledger/fabric-x-common/msp"

	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/adapters"
)

var logger = flogging.MustGetLogger("app")

const defaultBroadcastRetryTimeout = 30 * time.Second

// TxStatus represents the finality status of a submitted transaction.
type TxStatus = int

// UnknownStatus indicates transaction status is not yet determined.
const UnknownStatus TxStatus = 0

// SubmitTransaction receives a transaction and sends it to the ordering service.
func (d *AdminApp) SubmitTransaction(ctx context.Context, txID string, tx *applicationpb.Tx) error {
	// get orderer client and signing identity
	sc, err := d.prepareSubmission(ctx)
	if err != nil {
		return fmt.Errorf("failed to prepare submission: %w", err)
	}
	defer func() {
		_ = sc.ordererClient.Close()
	}()

	broadcastErr := broadcastTransaction(ctx, sc, txID, tx)
	if broadcastErr != nil {
		return fmt.Errorf("failed to broadcast transaction: %w", broadcastErr)
	}

	return nil
}

// SubmitTransactionWithWait receives a transaction and sends it to the ordering service.
func (d *AdminApp) SubmitTransactionWithWait(ctx context.Context, txID string, tx *applicationpb.Tx) (TxStatus, error) {
	// get orderer client and signing identity
	sc, err := d.prepareSubmission(ctx)
	if err != nil {
		return UnknownStatus, fmt.Errorf("failed to prepare submission: %w", err)
	}
	defer func() {
		_ = sc.ordererClient.Close()
	}()

	// get notification client
	nc, err := d.NotificationProvider.Get()
	if err != nil {
		return UnknownStatus, fmt.Errorf("failed to get notification client: %w", err)
	}

	defer func() {
		_ = nc.Close()
	}()

	subscription, err := nc.Subscribe(ctx, txID)
	if err != nil {
		return UnknownStatus, fmt.Errorf("failed to subscribe to transaction events: %w", err)
	}

	broadcastErr := broadcastTransaction(ctx, sc, txID, tx)
	if broadcastErr != nil {
		return UnknownStatus, fmt.Errorf("failed to broadcast transaction: %w", broadcastErr)
	}

	status, err := nc.WaitForEvent(ctx, subscription)
	if err != nil {
		return UnknownStatus, fmt.Errorf("failed to wait for transaction status event: %w", err)
	}

	return status, nil
}

type submissionContext struct {
	signingIdentity msp.SigningIdentity
	ordererClient   adapters.OrdererClient
	retryTimeout    time.Duration
}

func broadcastTransaction(
	ctx context.Context,
	sc *submissionContext,
	txID string,
	tx *applicationpb.Tx,
) error {
	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = 100 * time.Millisecond
	bo.Multiplier = 2
	bo.MaxInterval = 2 * time.Second
	maxElapsedTime := retryTimeout(sc.retryTimeout)

	attempt := 0
	// backoff/v5 binds cancellation via Retry(ctx, ...); there is no WithContext helper in v5.
	_, err := backoff.Retry(ctx, func() (struct{}, error) {
		attempt++

		broadcastErr := sc.ordererClient.Broadcast(ctx, sc.signingIdentity, txID, tx)
		if broadcastErr == nil {
			if attempt > 1 {
				logger.Info("transaction broadcast succeeded after retry",
					"attempt", attempt,
				)
			}
			return struct{}{}, nil
		}

		if !isRetryable(broadcastErr) {
			logger.Error("non-retryable broadcast error",
				"attempt", attempt,
				"error", broadcastErr,
			)
			return struct{}{}, backoff.Permanent(broadcastErr)
		}

		return struct{}{}, broadcastErr
	},
		backoff.WithBackOff(bo),
		backoff.WithMaxElapsedTime(maxElapsedTime),
		backoff.WithNotify(func(retryErr error, nextBackOff time.Duration) {
			if errors.Is(retryErr, context.Canceled) || errors.Is(retryErr, context.DeadlineExceeded) {
				logger.Info("transaction broadcast canceled",
					"attempt", attempt,
					"error", retryErr,
				)
				return
			}

			logger.Warn("transaction broadcast failed",
				"attempt", attempt,
				"error", retryErr,
				"next_retry_in", nextBackOff,
			)
		}),
	)
	if err != nil {
		return fmt.Errorf("broadcast failed after %d attempts: %w", attempt, err)
	}

	return nil
}

func retryTimeout(timeout time.Duration) time.Duration {
	if timeout <= 0 {
		return defaultBroadcastRetryTimeout
	}

	return timeout
}

func isRetryable(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// TODO: refine retryable error classification based on orderer error types.

	return true
}

func (d *AdminApp) prepareSubmission(_ context.Context) (*submissionContext, error) {
	// get signing identity
	sid, err := d.MspProvider.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to get signing identity: %w", err)
	}

	// get orderer client
	oc, err := d.OrdererProvider.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to get orderer client: %w", err)
	}

	return &submissionContext{
		signingIdentity: sid,
		ordererClient:   oc,
		retryTimeout:    oc.ConnectionTimeout(),
	}, nil
}
