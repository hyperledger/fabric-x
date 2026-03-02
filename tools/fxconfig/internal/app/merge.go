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

func (d *AdminApp) MergeTransactions(_ context.Context, txs []*applicationpb.Tx) (*applicationpb.Tx, error) {
	return transaction.Merge(txs)
}
