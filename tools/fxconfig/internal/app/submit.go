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

	"github.com/hyperledger/fabric-x-common/api/applicationpb"
	"github.com/hyperledger/fabric-x-common/msp"

	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/adapters"
)

const (
	maxBroadcastAttempts = 3
	broadcastRetryDelay  = 100 * time.Millisecond
)

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
		return broadcastErr
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
		return UnknownStatus, broadcastErr
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
}

func broadcastTransaction(
	ctx context.Context,
	sc *submissionContext,
	txID string,
	tx *applicationpb.Tx,
) error {
	var lastErr error

	for attempt := 1; attempt <= maxBroadcastAttempts; attempt++ {
		err := sc.ordererClient.Broadcast(ctx, sc.signingIdentity, txID, tx)
		if err == nil {
			return nil
		}

		if !isRetryable(err) {
			return err
		}

		lastErr = err

		if attempt == maxBroadcastAttempts {
			break
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(broadcastRetryDelay):
		}
	}

	return lastErr
}

func isRetryable(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
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
	}, nil
}
