/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package app

import (
	"context"
	"fmt"

	"github.com/hyperledger/fabric-x-common/api/applicationpb"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/api"
)

type TxStatus = int

const UnknownStatus TxStatus = 0

// SubmitTransaction receives a transaction and sends it to the ordering service.
func (d *AdminApp) SubmitTransaction(ctx context.Context, txID string, tx *applicationpb.Tx) error {
	_, err := d.submitTransaction(ctx, txID, tx, false)
	return err
}

// SubmitTransactionWithWait receives a transaction and sends it to the ordering service.
func (d *AdminApp) SubmitTransactionWithWait(ctx context.Context, txID string, tx *applicationpb.Tx) (TxStatus, error) {
	return d.submitTransaction(ctx, txID, tx, true)
}

func (d *AdminApp) submitTransaction(ctx context.Context, txID string, tx *applicationpb.Tx, waitForFinality bool) (TxStatus, error) {
	var (
		nc           api.NotificationClient
		subscription chan int
		err          error
	)
	if waitForFinality {
		// notification config validation
		if err := d.NotificationProvider.Validate(); err != nil {
			return 0, fmt.Errorf("invalid notifications configuration: %w", err)
		}

		// get notification client
		var err error
		nc, err = d.NotificationProvider.Get()
		if err != nil {
			return UnknownStatus, err
		}

		defer func() {
			_ = nc.Close()
		}()
	}

	// msp config validation
	if err := d.MspProvider.Validate(); err != nil {
		return UnknownStatus, fmt.Errorf("invalid msp configuration: %w", err)
	}

	// get signing identity
	sid, err := d.MspProvider.Get()
	if err != nil {
		return UnknownStatus, err
	}

	if waitForFinality {
		subscription, err = nc.Subscribe(ctx, txID)
		if err != nil {
			return UnknownStatus, err
		}
	}

	// orderer config validation
	if err := d.OrdererProvider.Validate(); err != nil {
		return UnknownStatus, fmt.Errorf("invalid ordering service configuration: %w", err)
	}

	// get orderer client
	oc, err := d.OrdererProvider.Get()
	if err != nil {
		return UnknownStatus, err
	}
	defer func() {
		_ = oc.Close()
	}()

	if err := oc.Broadcast(ctx, sid, txID, tx); err != nil {
		return UnknownStatus, err
	}

	if waitForFinality {
		return nc.WaitForEvent(ctx, subscription)
	}

	return UnknownStatus, nil
}
