/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package app

import (
	"context"
	"fmt"

	"github.com/hyperledger/fabric-x-common/api/applicationpb"
)

// SubmitTransaction receives a transaction and sends it to the ordering service.
func (d *AdminApp) SubmitTransaction(ctx context.Context, txID string, tx *applicationpb.Tx) error {
	return d.submitTransaction(ctx, txID, tx)
}

// SubmitTransactionWithWait receives a transaction and sends it to the ordering service.
func (d *AdminApp) SubmitTransactionWithWait(ctx context.Context, txID string, tx *applicationpb.Tx) error {
	// TODO:  implement me
	// get notification service

	// subscribe

	err := d.submitTransaction(ctx, txID, tx)
	if err != nil {
		return err
	}

	// wait

	return nil
}

func (d *AdminApp) submitTransaction(ctx context.Context, txID string, tx *applicationpb.Tx) error {
	// msp validation
	if err := d.MspProvider.Validate(); err != nil {
		return fmt.Errorf("invalid msp configuration: %w", err)
	}

	// get signing identity
	sid, err := d.MspProvider.Get()
	if err != nil {
		return err
	}

	// orderer validation
	if err := d.OrdererProvider.Validate(); err != nil {
		return fmt.Errorf("invalid ordering service configuration: %w", err)
	}

	// get orderer client
	oc, err := d.OrdererProvider.Get()
	if err != nil {
		return err
	}

	if err := oc.Broadcast(ctx, sid, txID, tx); err != nil {
		return err
	}

	return nil
}
