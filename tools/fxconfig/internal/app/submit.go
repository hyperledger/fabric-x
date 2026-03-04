/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package app

import (
	"context"

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
	// get orderer client and signing identity
	sc, err := d.prepareSubmission(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = sc.ordererClient.Close()
	}()

	return sc.ordererClient.Broadcast(ctx, sc.signingIdentity, txID, tx)
}

// SubmitTransactionWithWait receives a transaction and sends it to the ordering service.
func (d *AdminApp) SubmitTransactionWithWait(ctx context.Context, txID string, tx *applicationpb.Tx) (TxStatus, error) {
	// get orderer client and signing identity
	sc, err := d.prepareSubmission(ctx)
	if err != nil {
		return UnknownStatus, err
	}
	defer func() {
		_ = sc.ordererClient.Close()
	}()

	// get notification client
	nc, err := d.NotificationProvider.Get()
	if err != nil {
		return UnknownStatus, err
	}

	defer func() {
		_ = nc.Close()
	}()

	subscription, err := nc.Subscribe(ctx, txID)
	if err != nil {
		return UnknownStatus, err
	}

	if err := sc.ordererClient.Broadcast(ctx, sc.signingIdentity, txID, tx); err != nil {
		return UnknownStatus, err
	}

	return nc.WaitForEvent(ctx, subscription)
}

type submissionContext struct {
	signingIdentity msp.SigningIdentity
	ordererClient   adapters.OrdererClient
}

func (d *AdminApp) prepareSubmission(_ context.Context) (*submissionContext, error) {
	// get signing identity
	sid, err := d.MspProvider.Get()
	if err != nil {
		return nil, err
	}

	// get orderer client
	oc, err := d.OrdererProvider.Get()
	if err != nil {
		return nil, err
	}

	return &submissionContext{
		signingIdentity: sid,
		ordererClient:   oc,
	}, nil
}
