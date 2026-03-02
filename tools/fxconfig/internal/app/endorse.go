/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package app

import (
	"context"

	"github.com/hyperledger/fabric-x-common/api/applicationpb"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/transaction"
)

// EndorseTransaction receives a transaction as input and endorses it.
func (d *AdminApp) EndorseTransaction(ctx context.Context, txID string, tx *applicationpb.Tx) (*applicationpb.Tx, error) {
	return d.endorseTransaction(ctx, txID, tx)
}

func (d *AdminApp) endorseTransaction(_ context.Context, txID string, tx *applicationpb.Tx) (*applicationpb.Tx, error) {
	// msp validation
	if err := d.MspProvider.Validate(); err != nil {
		return nil, err
	}

	// get signing identity
	sid, err := d.MspProvider.Get()
	if err != nil {
		return nil, err
	}

	// Endorse transaction
	tx, err = transaction.Endorse(sid, txID, tx)
	if err != nil {
		return nil, err
	}

	return tx, nil
}
