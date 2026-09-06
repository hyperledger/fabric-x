/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package app

import (
	"context"
	"fmt"

	"github.com/hyperledger/fabric-x-common/api/applicationpb"
	"github.com/hyperledger/fabric-x-common/msp"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/adapters"
)

// TxStatus represents the finality status of a submitted transaction.
type TxStatus = int

// UnknownStatus indicates transaction status is not yet determined.
const UnknownStatus TxStatus = 0

// SubmitTransaction receives a transaction and sends it to the ordering service.
func (d *AdminApp) SubmitTransaction(ctx context.Context, txID string, tx *applicationpb.Tx) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	// get orderer client and signing identity
	sc, err := d.prepareSubmission(ctx)
	if err != nil {
		return fmt.Errorf("failed to prepare submission: %w", err)
	}
	defer func() {
		_ = sc.ordererClient.Close()
	}()

	if err := sc.ordererClient.Broadcast(ctx, sc.signingIdentity, txID, tx); err != nil {
		return fmt.Errorf("failed to broadcast transaction: %w", err)
	}

	return nil
}

// SubmitTransactionWithWait receives a transaction and sends it to the ordering service.
func (d *AdminApp) SubmitTransactionWithWait(ctx context.Context, txID string, tx *applicationpb.Tx) (TxStatus, error) {
	if err := ctx.Err(); err != nil {
		return UnknownStatus, err
	}
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

	if err := sc.ordererClient.Broadcast(ctx, sc.signingIdentity, txID, tx); err != nil {
		return UnknownStatus, fmt.Errorf("failed to broadcast transaction: %w", err)
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
